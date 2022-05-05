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
	"github.com/umbracle/ethgo"
)

type HttpAPI struct {
	url    string
	logger hclog.Logger
}

func NewHttpAPI(url string) *HttpAPI {
	return &HttpAPI{url: url, logger: hclog.L()}
}

func (h *HttpAPI) SetLogger(logger hclog.Logger) {
	h.logger = logger
}

func (h *HttpAPI) post(path string, input interface{}, out interface{}) error {
	postBody, err := Marshal(input)
	if err != nil {
		return err
	}
	responseBody := bytes.NewBuffer(postBody)

	resp, err := http.Post(h.url+path, "application/json", responseBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

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

	var output struct {
		Data json.RawMessage
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
	Time uint64     `json:"genesis_time"`
	Root ethgo.Hash `json:"genesis_validators_root"`
	Fork string     `json:"genesis_fork_version"`
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
	PubKey          string `json:"pubkey"`
	ValidatorIndex  uint   `json:"validator_index"`
	Slot            uint64 `json:"slot"`
	CommitteeIndex  uint64 `json:"committee_index"`
	CommitteeLength uint64 `json:"committee_length"`
	CommitteeAtSlot uint64 `json:"committees_at_slot"`
}

func (h *HttpAPI) GetAttesterDuties(epoch uint64, indexes []string) ([]*AttesterDuty, error) {
	var out []*AttesterDuty
	err := h.post(fmt.Sprintf("/eth/v1/validator/duties/attester/%d", epoch), indexes, &out)
	return out, err
}

type ProposerDuty struct {
	PubKey         string `json:"pubkey"`
	ValidatorIndex uint   `json:"validator_index"`
	Slot           uint64 `json:"slot"`
}

func (h *HttpAPI) GetProposerDuties(epoch uint64) ([]*ProposerDuty, error) {
	var out []*ProposerDuty
	err := h.get(fmt.Sprintf("/eth/v1/validator/duties/proposer/%d", epoch), &out)
	return out, err
}

type SyncDuty struct {
	PubKey                        string   `json:"pubkey"`
	ValidatorIndex                uint     `json:"validator_index"`
	ValidatorSyncCommitteeIndices []string `json:"validator_sync_committee_indices"`
}

func (h *HttpAPI) GetCommitteeSyncDuties(epoch uint64, indexes []string) ([]*SyncDuty, error) {
	var out []*SyncDuty
	err := h.post(fmt.Sprintf("/eth/v1/validator/duties/sync/%d", epoch), indexes, &out)
	return out, err
}

type Validator struct {
	Index  uint64 `json:"index"`
	Status string `json:"status"`
}

func (h *HttpAPI) GetValidatorByPubKey(pub string) (*Validator, error) {
	var out *Validator
	h.get("/eth/v1/beacon/states/head/validators/"+pub, &out)
	return out, nil
}

func (h *HttpAPI) GetBlock(slot uint64, randao []byte) (*structs.BeaconBlock, error) {
	buf := "0x" + hex.EncodeToString(randao)

	var out *structs.BeaconBlock
	err := h.get(fmt.Sprintf("/eth/v1/validator/blocks/%d?randao_reveal=%s", slot, buf), &out)

	return out, err
}

func (h *HttpAPI) PublishSignedBlock(block *structs.SignedBeaconBlock) error {
	var out interface{}
	err := h.post("/eth/v1/beacon/blocks", block, &out)
	return err
}
