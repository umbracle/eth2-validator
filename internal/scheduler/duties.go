package scheduler

import (
	"context"
	"fmt"

	ssz "github.com/ferranbt/fastssz"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bitlist"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	consensus "github.com/umbracle/go-eth-consensus"
)

type Duty struct {
	client beacon.Api
	state  State
	ctx    context.Context
}

func NewDuty(ctx context.Context, client beacon.Api, state State) *Duty {
	return &Duty{ctx: ctx, client: client, state: state}
}

func (d *Duty) Sign(ctx context.Context, domain proto.DomainType, epoch uint64, accountIndex uint64, root [32]byte) ([96]byte, error) {
	return d.state.Sign(ctx, domain, epoch, accountIndex, root)
}

func (d *Duty) Handle(duty *proto.Duty) (*proto.Duty_Result, error) {
	var err error
	var res *proto.Duty_Result

	switch duty.Job.(type) {
	case *proto.Duty_BlockProposal_:
		res, err = d.runBlockProposal(d.ctx, duty)
	case *proto.Duty_Attestation_:
		res, err = d.runSingleAttestation(d.ctx, duty)
	case *proto.Duty_AttestationAggregate_:
		res, err = d.runAttestationAggregate(d.ctx, duty)
	case *proto.Duty_SyncCommittee_:
		res, err = d.runSyncCommittee(d.ctx, duty)
	case *proto.Duty_SyncCommitteeAggregate_:
		res, err = d.runSyncCommitteeAggregate(d.ctx, duty)
	}
	if err != nil {
		panic(fmt.Errorf("failed to handle %s: %v", duty.Type(), err))
	}

	return res, err
}

func (d *Duty) runSyncCommitteeAggregate(ctx context.Context, duty *proto.Duty) (*proto.Duty_Result, error) {
	job := duty.GetSyncCommitteeAggregate()

	latestRoot, err := d.client.GetHeadBlockRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get head block root: %v", err)
	}

	contribution, err := d.client.SyncCommitteeContribution(ctx, duty.Slot, job.SubCommitteeIndex, latestRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync committee contribution: %v", err)
	}

	contributionAggregate := &consensus.ContributionAndProof{
		AggregatorIndex: duty.ValidatorIndex,
		Contribution:    contribution,
		SelectionProof:  consensus.ToBytes96(job.SelectionProof),
	}
	root, err := contributionAggregate.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	signature, err := d.Sign(ctx, proto.DomainContributionAndProof, duty.Epoch, duty.ValidatorIndex, root)
	if err != nil {
		return nil, err
	}

	msg := &consensus.SignedContributionAndProof{
		Message:   contributionAggregate,
		Signature: signature,
	}
	if err := d.client.SubmitSignedContributionAndProof(ctx, []*consensus.SignedContributionAndProof{msg}); err != nil {
		return nil, fmt.Errorf("failed to submit signed committee aggregate proof: %v", err)
	}

	result := &proto.Duty_Result{
		SyncCommitteeAggregate: &proto.Duty_SyncCommitteeAggregateResult{},
	}
	return result, nil
}

func (d *Duty) runSyncCommittee(ctx context.Context, duty *proto.Duty) (*proto.Duty_Result, error) {
	// get root
	latestRoot, err := d.client.GetHeadBlockRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get head block root: %v", err)
	}

	signature, err := d.Sign(ctx, proto.DomainSyncCommitteeType, duty.Epoch, duty.ValidatorIndex, proto.RootSSZ(latestRoot))
	if err != nil {
		return nil, err
	}

	committeeDuty := []*consensus.SyncCommitteeMessage{
		{
			Slot:           duty.Slot,
			BlockRoot:      latestRoot,
			ValidatorIndex: duty.ValidatorIndex,
			Signature:      signature,
		},
	}

	if err := d.client.SubmitCommitteeDuties(ctx, committeeDuty); err != nil {
		return nil, fmt.Errorf("failed to submit committee duties: %v", err)
	}

	// store the attestation in the state
	result := &proto.Duty_Result{
		SyncCommittee: &proto.Duty_SyncCommitteeResult{},
	}
	return result, nil
}

func (d *Duty) runSingleAttestation(ctx context.Context, duty *proto.Duty) (*proto.Duty_Result, error) {
	job := duty.GetAttestation()

	attestationData, err := d.client.RequestAttestationData(ctx, duty.Slot, job.CommitteeIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to reuest attestation data: %v", err)
	}
	attestationRoot, err := attestationData.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	attestedSignature, err := d.Sign(ctx, proto.DomainBeaconAttesterType, duty.Epoch, duty.ValidatorIndex, attestationRoot)
	if err != nil {
		return nil, err
	}

	bitlist := bitlist.NewBitlist(job.CommitteeLength)
	bitlist.SetBitAt(job.CommitteeIndex, true)

	attestation := &consensus.Attestation{
		Data:            attestationData,
		AggregationBits: bitlist,
		Signature:       attestedSignature,
	}
	if err := d.client.PublishAttestations(ctx, []*consensus.Attestation{attestation}); err != nil {
		return nil, fmt.Errorf("failed to publish attestations: %v", err)
	}

	// store the attestation in the state
	result := &proto.Duty_Result{
		Attestation: &proto.Duty_AttestationResult{
			Root: attestationRoot[:],
			Source: &proto.Duty_AttestationResult_Checkpoint{
				Root:  attestationData.Source.Root[:],
				Epoch: attestationData.Source.Epoch,
			},
			Target: &proto.Duty_AttestationResult_Checkpoint{
				Root:  attestationData.Target.Root[:],
				Epoch: attestationData.Target.Epoch,
			},
		},
	}
	return result, nil
}

func (d *Duty) runAttestationAggregate(ctx context.Context, duty *proto.Duty) (*proto.Duty_Result, error) {
	job := duty.GetAttestationAggregate()

	attestation, err := d.state.DutyByID(duty.BlockedBy[0])
	if err != nil {
		return nil, err
	}

	aggregateAttestation, err := d.client.AggregateAttestation(ctx, duty.Slot, consensus.ToBytes32(attestation.Result.Attestation.Root))
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate attestation: %v", err)
	}

	// Sign the aggregate attestation.
	aggregateAndProof := &consensus.AggregateAndProof{
		Index:          duty.ValidatorIndex,
		Aggregate:      aggregateAttestation,
		SelectionProof: consensus.ToBytes96(job.SelectionProof),
	}
	aggregateAndProofRoot, err := aggregateAndProof.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	aggregateAndProofRootSignature, err := d.Sign(ctx, proto.DomainAggregateAndProofType, duty.Epoch, duty.ValidatorIndex, proto.RootSSZ(aggregateAndProofRoot))
	if err != nil {
		return nil, err
	}

	req := []*consensus.SignedAggregateAndProof{
		{
			Message:   aggregateAndProof,
			Signature: aggregateAndProofRootSignature,
		},
	}
	if err := d.client.PublishAggregateAndProof(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to publish aggregate and proof: %v", err)
	}

	result := &proto.Duty_Result{
		AttestationAggregate: &proto.Duty_AttestationAggregateResult{},
	}
	return result, nil
}

type block interface {
	consensus.BeaconBlock
	ssz.HashRoot
}

type signedBlock interface {
	consensus.SignedBeaconBlock
}

func (d *Duty) runBlockProposal(ctx context.Context, duty *proto.Duty) (*proto.Duty_Result, error) {
	// create the randao
	randaoReveal, err := d.Sign(ctx, proto.DomainRandaomType, duty.Epoch, duty.ValidatorIndex, proto.Uint64SSZ(duty.Epoch))
	if err != nil {
		return nil, err
	}

	var block block
	switch duty.Fork {
	case proto.Duty_Phase0:
		block = &consensus.BeaconBlockPhase0{}
	default:
		block = &consensus.BeaconBlockAltair{}
	}

	if err := d.client.GetBlock(ctx, block, duty.Slot, randaoReveal); err != nil {
		return nil, fmt.Errorf("failed to get block: %v", err)
	}

	blockRoot, err := block.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	blockSignature, err := d.Sign(ctx, proto.DomainBeaconProposerType, duty.Epoch, duty.ValidatorIndex, blockRoot)
	if err != nil {
		return nil, err
	}

	var signedBlock signedBlock
	var stateRoot [32]byte

	// TODO: Signing of messages could be simplified with a generic
	// signing container.
	switch duty.Fork {
	case proto.Duty_Phase0:
		b := block.(*consensus.BeaconBlockPhase0)
		stateRoot = b.StateRoot

		signedBlock = &consensus.SignedBeaconBlockPhase0{
			Block:     b,
			Signature: blockSignature,
		}
	default:
		b := block.(*consensus.BeaconBlockAltair)
		stateRoot = b.StateRoot

		signedBlock = &consensus.SignedBeaconBlockAltair{
			Block:     b,
			Signature: blockSignature,
		}
	}

	if err := d.client.PublishSignedBlock(ctx, signedBlock); err != nil {
		return nil, fmt.Errorf("failed to publish block: %v", err)
	}

	result := &proto.Duty_Result{
		BlockProposal: &proto.Duty_BlockProposalResult{
			Root:      stateRoot[:],
			Signature: blockSignature[:],
		},
	}
	return result, nil
}
