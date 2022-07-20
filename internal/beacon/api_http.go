package beacon

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/r3labs/sse/v2"
	"github.com/umbracle/eth2-validator/internal/server/structs"
	"go.opentelemetry.io/otel"
)

type HttpAPI struct {
	url    string
	logger hclog.Logger
}

func NewHttpAPI(url string) *HttpAPI {
	return &HttpAPI{url: url, logger: hclog.L()}
}

func (h *HttpAPI) SetLogger(logger hclog.Logger) {
	h.logger = logger.Named(h.url)
}

func (h *HttpAPI) post(path string, input interface{}, out interface{}) error {
	postBody, err := Marshal(input)
	if err != nil {
		return err
	}
	responseBody := bytes.NewBuffer(postBody)

	h.logger.Trace("Post request", "path", path, "content", string(postBody))

	resp, err := http.Post(h.url+path, "application/json", responseBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if out == nil {
		// nothing is expected, make sure its a 200 resp code
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if string(data) == `{"data":null}` {
			return nil
		}
		if string(data) == "null" {
			return nil
		}
		if string(data) == "" {
			return nil
		}
		// its a json that represnets an error, just reutrn it
		return fmt.Errorf("json failed to decode post message: '%s'", string(data))
	}
	if err := h.decodeResp(resp, out); err != nil {
		return err
	}
	return nil
}

func (h *HttpAPI) get(path string, out interface{}) error {
	resp, err := http.Get(h.url + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	h.logger.Trace("Get request", "path", path)

	if err := h.decodeResp(resp, out); err != nil {
		return err
	}
	return nil
}

func (h *HttpAPI) decodeResp(resp *http.Response, out interface{}) error {
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	h.logger.Trace("Http response", "data", string(data))

	var output struct {
		Data json.RawMessage `json:"data,omitempty"`
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return err
	}
	if err := Unmarshal(output.Data, &out); err != nil {
		return err
	}
	return nil
}

type Syncing struct {
	HeadSlot     uint64 `json:"head_slot"`
	SyncDistance string `json:"sync_distance"`
	IsSyncing    bool   `json:"is_syncing"`
}

func (h *HttpAPI) Syncing() (*Syncing, error) {
	var out Syncing
	err := h.get("/eth/v1/node/syncing", &out)
	return &out, err
}

type Genesis struct {
	Time uint64 `json:"genesis_time"`
	Root []byte `json:"genesis_validators_root"`
	Fork string `json:"genesis_fork_version"`
}

func (h *HttpAPI) Genesis(ctx context.Context) (*Genesis, error) {
	var out Genesis
	err := h.get("/eth/v1/beacon/genesis", &out)
	return &out, err
}

type HeadEvent struct {
	Slot                      string
	Block                     string
	State                     string
	EpochTransition           bool
	CurrentDutyDependentRoot  string
	PreviousDutyDependentRoot string
}

var eventValidTopics = []string{
	"head", "block", "attestation", "finalized_checkpoint",
}

func isValidTopic(str string) bool {
	for _, topic := range eventValidTopics {
		if str == topic {
			return true
		}
	}
	return false
}

func (h *HttpAPI) Events(ctx context.Context, topics []string, handler func(obj interface{})) error {
	for _, topic := range topics {
		if !isValidTopic(topic) {
			return fmt.Errorf("topic '%s' is not valid", topic)
		}
	}

	client := sse.NewClient(h.url + "/eth/v1/events?topics=" + strings.Join(topics, ","))
	if err := client.SubscribeRawWithContext(ctx, func(msg *sse.Event) {
		switch string(msg.Event) {
		case "head":
			var headEvent *HeadEvent
			if err := json.Unmarshal(msg.Data, &headEvent); err != nil {
				h.logger.Error("failed to decode head event", "err", err)
			} else {
				handler(err)
			}

		default:
			h.logger.Debug("event not tracked", "msg", string(msg.Event))
		}
	}); err != nil {
		return err
	}
	return nil
}

type AttesterDuty struct {
	PubKey                  string `json:"pubkey"`
	ValidatorIndex          uint   `json:"validator_index"`
	Slot                    uint64 `json:"slot"`
	CommitteeIndex          uint64 `json:"committee_index"`
	CommitteeLength         uint64 `json:"committee_length"`
	CommitteeAtSlot         uint64 `json:"committees_at_slot"`
	ValidatorCommitteeIndex uint64 `json:"validator_committee_index"`
}

func (h *HttpAPI) GetAttesterDuties(ctx context.Context, epoch uint64, indexes []string) ([]*AttesterDuty, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetAttesterDuties")
	defer span.End()

	var out []*AttesterDuty
	err := h.post(fmt.Sprintf("/eth/v1/validator/duties/attester/%d", epoch), indexes, &out)
	return out, err
}

type ProposerDuty struct {
	PubKey         string `json:"pubkey"`
	ValidatorIndex uint   `json:"validator_index"`
	Slot           uint64 `json:"slot"`
}

func (h *HttpAPI) GetProposerDuties(ctx context.Context, epoch uint64) ([]*ProposerDuty, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetProposerDuties")
	defer span.End()

	var out []*ProposerDuty
	err := h.get(fmt.Sprintf("/eth/v1/validator/duties/proposer/%d", epoch), &out)
	return out, err
}

type CommitteeSyncDuty struct {
	PubKey                        string   `json:"pubkey"`
	ValidatorIndex                uint     `json:"validator_index"`
	ValidatorSyncCommitteeIndices []string `json:"validator_sync_committee_indices"`
}

func (h *HttpAPI) GetCommitteeSyncDuties(ctx context.Context, epoch uint64, indexes []string) ([]*CommitteeSyncDuty, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetCommitteeSyncDuties")
	defer span.End()

	var out []*CommitteeSyncDuty
	err := h.post(fmt.Sprintf("/eth/v1/validator/duties/sync/%d", epoch), indexes, &out)
	return out, err
}

type SyncCommitteeMessage struct {
	Slot           uint64 `json:"slot"`
	BlockRoot      []byte `json:"beacon_block_root"`
	ValidatorIndex uint64 `json:"validator_index"`
	Signature      []byte `json:"signature"`
}

func (h *HttpAPI) SubmitCommitteeDuties(ctx context.Context, duties []*SyncCommitteeMessage) error {
	_, span := otel.Tracer("Validator").Start(ctx, "SubmitCommitteeDuties")
	defer span.End()

	err := h.post("/eth/v1/beacon/pool/sync_committees", duties, nil)
	return err
}

type Validator struct {
	Index     uint64 `json:"index"`
	Status    string `json:"status"`
	Validator struct {
		PubKey                     string `json:"pubkey"`
		Slashed                    bool   `json:"slashed"`
		ActivationElegibilityEpoch uint64 `json:"activation_eligibility_epoch"`
		ActivationEpoch            uint64 `json:"activation_epoch"`
		ExitEpoch                  uint64 `json:"exit_epoch"`
		WithdrawableEpoch          uint64 `json:"withdrawable_epoch"`
	} `json:"validator"`
}

func (h *HttpAPI) GetValidatorByPubKey(ctx context.Context, pub string) (*Validator, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetValidatorByPubKey")
	defer span.End()

	var out *Validator
	h.get("/eth/v1/beacon/states/head/validators/"+pub, &out)
	return out, nil
}

func (h *HttpAPI) GetBlock(ctx context.Context, slot uint64, randao []byte) (*structs.BeaconBlock, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetBlock")
	defer span.End()

	buf := "0x" + hex.EncodeToString(randao)

	var out *structs.BeaconBlock
	err := h.get(fmt.Sprintf("/eth/v1/validator/blocks/%d?randao_reveal=%s", slot, buf), &out)

	return out, err
}

func (h *HttpAPI) PublishSignedBlock(ctx context.Context, block *structs.SignedBeaconBlock) error {
	_, span := otel.Tracer("Validator").Start(ctx, "PublishSignedBlock")
	defer span.End()

	err := h.post("/eth/v1/beacon/blocks", block, nil)
	return err
}

func (h *HttpAPI) RequestAttestationData(ctx context.Context, slot uint64, committeeIndex uint64) (*structs.AttestationData, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "RequestAttestationData")
	defer span.End()

	var out *structs.AttestationData
	err := h.get(fmt.Sprintf("/eth/v1/validator/attestation_data?slot=%d&committee_index=%d", slot, committeeIndex), &out)
	return out, err
}

func (h *HttpAPI) PublishAttestations(ctx context.Context, data []*structs.Attestation) error {
	_, span := otel.Tracer("Validator").Start(ctx, "PublishAttestations")
	defer span.End()

	err := h.post("/eth/v1/beacon/pool/attestations", data, nil)
	return err
}

func (h *HttpAPI) AggregateAttestation(ctx context.Context, slot uint64, root [32]byte) (*structs.Attestation, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "AggregateAttestation")
	defer span.End()

	var out *structs.Attestation
	err := h.get(fmt.Sprintf("/eth/v1/validator/aggregate_attestation?slot=%d&attestation_data_root=0x%s", slot, hex.EncodeToString(root[:])), &out)
	return out, err
}

type SignedAggregateAndProof struct {
	Message   *structs.AggregateAndProof `json:"message"`
	Signature []byte                     `json:"signature" ssz-size:"96"`
}

func (h *HttpAPI) PublishAggregateAndProof(ctx context.Context, data []*SignedAggregateAndProof) error {
	_, span := otel.Tracer("Validator").Start(ctx, "PublishAggregateAndProof")
	defer span.End()

	err := h.post("/eth/v1/validator/aggregate_and_proofs", data, nil)
	return err
}

func (h *HttpAPI) GetHeadBlockRoot(ctx context.Context) ([]byte, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetHeadBlockRoot")
	defer span.End()

	var data struct {
		Root []byte
	}
	err := h.get("/eth/v1/beacon/blocks/head/root", &data)
	return data.Root, err
}

func (h *HttpAPI) ConfigSpec() (*ChainConfig, error) {
	var config *ChainConfig
	err := h.get("/eth/v1/config/spec", &config)
	return config, err
}

// produces a sync committee contribution
func (h *HttpAPI) SyncCommitteeContribution(ctx context.Context, slot uint64, subCommitteeIndex uint64, root []byte) (*structs.SyncCommitteeContribution, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "SyncCommitteeContribution")
	defer span.End()

	var out *structs.SyncCommitteeContribution
	err := h.get(fmt.Sprintf("/eth/v1/validator/sync_committee_contribution?slot=%d&subcommittee_index=%d&beacon_block_root=0x%s", slot, subCommitteeIndex, hex.EncodeToString(root[:])), &out)
	return out, err
}

func (h *HttpAPI) SubmitSignedContributionAndProof(ctx context.Context, signedContribution []*structs.SignedContributionAndProof) error {
	_, span := otel.Tracer("Validator").Start(ctx, "SignedContributionAndProof")
	defer span.End()

	err := h.post("/eth/v1/validator/contribution_and_proofs", signedContribution, nil)
	return err
}
