package beacon

import (
	"context"
	"encoding/json"
	"fmt"

	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/r3labs/sse/v2"
	consensus "github.com/umbracle/go-eth-consensus"
	"github.com/umbracle/go-eth-consensus/http"
	"go.opentelemetry.io/otel"
)

type HttpAPI struct {
	client *http.Client
	url    string
	logger hclog.Logger
}

func NewHttpAPI(url string) *HttpAPI {
	return &HttpAPI{client: http.New(url), url: url, logger: hclog.L()}
}

func (h *HttpAPI) SetLogger(logger hclog.Logger) {
	h.client.SetLogger(logger.StandardLogger(&hclog.StandardLoggerOptions{InferLevels: true}))
}

func (h *HttpAPI) Syncing() (*http.Syncing, error) {
	return h.client.Node().Syncing()
}

func (h *HttpAPI) Genesis(ctx context.Context) (*http.Genesis, error) {
	return h.client.Beacon().Genesis()
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

func (h *HttpAPI) GetAttesterDuties(ctx context.Context, epoch uint64, indexes []string) ([]*http.AttesterDuty, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetAttesterDuties")
	defer span.End()

	return h.client.Validator().GetAttesterDuties(epoch, indexes)
}

func (h *HttpAPI) GetProposerDuties(ctx context.Context, epoch uint64) ([]*http.ProposerDuty, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetProposerDuties")
	defer span.End()

	return h.client.Validator().GetProposerDuties(epoch)
}

func (h *HttpAPI) GetCommitteeSyncDuties(ctx context.Context, epoch uint64, indexes []string) ([]*http.CommitteeSyncDuty, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetCommitteeSyncDuties")
	defer span.End()

	return h.client.Validator().GetCommitteeSyncDuties(epoch, indexes)
}

func (h *HttpAPI) SubmitCommitteeDuties(ctx context.Context, duties []*consensus.SyncCommitteeMessage) error {
	_, span := otel.Tracer("Validator").Start(ctx, "SubmitCommitteeDuties")
	defer span.End()

	return h.client.Beacon().SubmitCommitteeDuties(duties)
}

func (h *HttpAPI) GetValidatorByPubKey(ctx context.Context, pub string) (*http.Validator, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetValidatorByPubKey")
	defer span.End()

	return h.client.Beacon().GetValidatorByPubKey(pub)
}

func (h *HttpAPI) GetBlock(ctx context.Context, slot uint64, randao []byte) (*consensus.BeaconBlock, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetBlock")
	defer span.End()

	panic("TODO")
}

func (h *HttpAPI) PublishSignedBlock(ctx context.Context, block *consensus.SignedBeaconBlock) error {
	_, span := otel.Tracer("Validator").Start(ctx, "PublishSignedBlock")
	defer span.End()

	return h.client.Beacon().PublishSignedBlock(block)
}

func (h *HttpAPI) RequestAttestationData(ctx context.Context, slot uint64, committeeIndex uint64) (*consensus.AttestationData, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "RequestAttestationData")
	defer span.End()

	return h.client.Validator().RequestAttestationData(slot, committeeIndex)
}

func (h *HttpAPI) PublishAttestations(ctx context.Context, data []*consensus.Attestation) error {
	_, span := otel.Tracer("Validator").Start(ctx, "PublishAttestations")
	defer span.End()

	return h.client.Beacon().PublishAttestations(data)
}

func (h *HttpAPI) AggregateAttestation(ctx context.Context, slot uint64, root []byte) (*consensus.Attestation, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "AggregateAttestation")
	defer span.End()

	return h.client.Validator().AggregateAttestation(slot, root)
}

func (h *HttpAPI) PublishAggregateAndProof(ctx context.Context, data []*consensus.SignedAggregateAndProof) error {
	_, span := otel.Tracer("Validator").Start(ctx, "PublishAggregateAndProof")
	defer span.End()

	return h.client.Validator().PublishAggregateAndProof(data)
}

func (h *HttpAPI) GetHeadBlockRoot(ctx context.Context) ([32]byte, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "GetHeadBlockRoot")
	defer span.End()

	return h.client.Beacon().GetHeadBlockRoot()
}

func (h *HttpAPI) ConfigSpec() (*consensus.Spec, error) {
	return h.client.Config().Spec()
}

func (h *HttpAPI) SyncCommitteeContribution(ctx context.Context, slot uint64, subCommitteeIndex uint64, root []byte) (*consensus.SyncCommitteeContribution, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "SyncCommitteeContribution")
	defer span.End()

	return h.client.Validator().SyncCommitteeContribution(slot, subCommitteeIndex, root)
}

func (h *HttpAPI) SubmitSignedContributionAndProof(ctx context.Context, signedContribution []*consensus.SignedContributionAndProof) error {
	_, span := otel.Tracer("Validator").Start(ctx, "SignedContributionAndProof")
	defer span.End()

	return h.client.Validator().SubmitSignedContributionAndProof(signedContribution)
}
