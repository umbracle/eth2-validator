package beacon

import (
	"context"

	consensus "github.com/umbracle/go-eth-consensus"
	"github.com/umbracle/go-eth-consensus/http"
)

var _ Api = &NullAPI{}

type NullAPI struct {
}

func (n *NullAPI) Syncing() (*http.Syncing, error) {
	panic("unimplemented")
}

func (n *NullAPI) Genesis(ctx context.Context) (*http.GenesisInfo, error) {
	panic("unimplemented")
}

func (n *NullAPI) Events(ctx context.Context, topics []string, handler func(obj interface{})) error {
	panic("unimplemented")
}

func (n *NullAPI) GetAttesterDuties(ctx context.Context, epoch uint64, indexes []string) ([]*http.AttesterDuty, error) {
	panic("unimplemented")
}

func (n *NullAPI) GetProposerDuties(ctx context.Context, epoch uint64) ([]*http.ProposerDuty, error) {
	panic("unimplemented")
}

func (n *NullAPI) GetCommitteeSyncDuties(ctx context.Context, epoch uint64, indexes []string) ([]*http.CommitteeSyncDuty, error) {
	panic("unimplemented")
}

func (n *NullAPI) SubmitCommitteeDuties(ctx context.Context, duties []*consensus.SyncCommitteeMessage) error {
	panic("unimplemented")
}

func (n *NullAPI) GetValidatorByPubKey(ctx context.Context, pub string) (*http.Validator, error) {
	panic("unimplemented")
}

func (n *NullAPI) GetBlock(ctx context.Context, obj consensus.BeaconBlock, slot uint64, randao [96]byte) error {
	panic("unimplemented")
}

func (n *NullAPI) PublishSignedBlock(ctx context.Context, block consensus.SignedBeaconBlock) error {
	panic("unimplemented")
}

func (n *NullAPI) RequestAttestationData(ctx context.Context, slot uint64, committeeIndex uint64) (*consensus.AttestationData, error) {
	panic("unimplemented")
}

func (n *NullAPI) PublishAttestations(ctx context.Context, data []*consensus.Attestation) error {
	panic("unimplemented")
}

func (n *NullAPI) AggregateAttestation(ctx context.Context, slot uint64, root [32]byte) (*consensus.Attestation, error) {
	panic("unimplemented")
}

func (n *NullAPI) PublishAggregateAndProof(ctx context.Context, data []*consensus.SignedAggregateAndProof) error {
	panic("unimplemented")
}

func (n *NullAPI) GetHeadBlockRoot(ctx context.Context) ([32]byte, error) {
	panic("unimplemented")
}

func (n *NullAPI) ConfigSpec() (*consensus.Spec, error) {
	panic("unimplemented")
}

func (n *NullAPI) SyncCommitteeContribution(ctx context.Context, slot uint64, subCommitteeIndex uint64, root [32]byte) (*consensus.SyncCommitteeContribution, error) {
	panic("unimplemented")
}

func (n *NullAPI) SubmitSignedContributionAndProof(ctx context.Context, signedContribution []*consensus.SignedContributionAndProof) error {
	panic("unimplemented")
}

func (n *NullAPI) SyncCommitteeSubscriptions(ctx context.Context, subs []*http.SyncCommitteeSubscription) error {
	panic("unimplemented")
}

func (n *NullAPI) BeaconCommitteeSubscriptions(ctx context.Context, subs []*http.BeaconCommitteeSubscription) error {
	panic("unimplemented")
}
