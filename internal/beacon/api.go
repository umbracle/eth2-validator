package beacon

import (
	"context"

	consensus "github.com/umbracle/go-eth-consensus"
	"github.com/umbracle/go-eth-consensus/http"
)

// Api is the api definition required by the validator
type Api interface {
	Syncing() (*http.Syncing, error)
	Genesis(ctx context.Context) (*http.Genesis, error)
	Events(ctx context.Context, topics []string, handler func(obj interface{})) error
	GetAttesterDuties(ctx context.Context, epoch uint64, indexes []string) ([]*http.AttesterDuty, error)
	GetProposerDuties(ctx context.Context, epoch uint64) ([]*http.ProposerDuty, error)
	GetCommitteeSyncDuties(ctx context.Context, epoch uint64, indexes []string) ([]*http.CommitteeSyncDuty, error)
	SubmitCommitteeDuties(ctx context.Context, duties []*consensus.SyncCommitteeMessage) error
	GetValidatorByPubKey(ctx context.Context, pub string) (*http.Validator, error)
	GetBlock(ctx context.Context, obj consensus.BeaconBlock, slot uint64, randao [96]byte) error
	PublishSignedBlock(ctx context.Context, block consensus.SignedBeaconBlock) error
	RequestAttestationData(ctx context.Context, slot uint64, committeeIndex uint64) (*consensus.AttestationData, error)
	PublishAttestations(ctx context.Context, data []*consensus.Attestation) error
	AggregateAttestation(ctx context.Context, slot uint64, root [32]byte) (*consensus.Attestation, error)
	PublishAggregateAndProof(ctx context.Context, data []*consensus.SignedAggregateAndProof) error
	GetHeadBlockRoot(ctx context.Context) ([32]byte, error)
	ConfigSpec() (*consensus.Spec, error)
	SyncCommitteeContribution(ctx context.Context, slot uint64, subCommitteeIndex uint64, root [32]byte) (*consensus.SyncCommitteeContribution, error)
	SubmitSignedContributionAndProof(ctx context.Context, signedContribution []*consensus.SignedContributionAndProof) error
}
