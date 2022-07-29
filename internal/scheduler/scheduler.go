package scheduler

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strconv"
	"time"

	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/uuid"
	consensus "github.com/umbracle/go-eth-consensus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type State interface {
	Sign(ctx context.Context, domain proto.DomainType, epoch uint64, account uint64, root [32]byte) ([96]byte, error)
	DutyByID(dutyID string) (*proto.Duty, error)
}

type Scheduler struct {
	ctx    context.Context
	state  State
	eval   *proto.Evaluation
	duties []*proto.Duty
	config *consensus.Spec
}

func NewScheduler(ctx context.Context, state State, config *consensus.Spec) *Scheduler {
	return &Scheduler{
		ctx:    ctx,
		state:  state,
		config: config,
	}
}

func (s *Scheduler) atSlot(slot uint64) time.Time {
	return s.eval.GenesisTime.Add(time.Duration(slot*s.config.SecondsPerSlot) * time.Second)
}

func (s *Scheduler) Process(eval *proto.Evaluation) (*proto.Plan, error) {
	s.eval = eval

	slotDuration := time.Duration(s.config.SecondsPerSlot) * time.Second

	// pre-compute the delays
	var (
		attestationDelay              = slotDuration / 3
		attestationAggregationDelay   = slotDuration * 2 / 3
		syncCommitteeDelay            = slotDuration / 3
		syncCommitteeAggregationDelay = slotDuration * 2 / 3
	)

	var duties []*proto.Duty
	for _, proposal := range eval.Proposer {
		timeToStart := s.atSlot(proposal.Slot)

		proposalDuty := &proto.Duty{
			Id:             uuid.Generate(),
			Slot:           proposal.Slot,
			Epoch:          eval.Epoch,
			ActiveTime:     timestamppb.New(timeToStart),
			Job:            &proto.Duty_BlockProposal_{},
			ValidatorIndex: uint64(proposal.ValidatorIndex),
		}
		duties = append(duties, proposalDuty)
	}

	for _, attestation := range eval.Attestation {
		attestationDuty := &proto.Duty{
			Id:             uuid.Generate(),
			Slot:           attestation.Slot,
			Epoch:          eval.Epoch,
			ActiveTime:     timestamppb.New(s.atSlot(attestation.Slot).Add(attestationDelay)),
			ValidatorIndex: uint64(attestation.ValidatorIndex),
			Job: &proto.Duty_Attestation_{
				Attestation: &proto.Duty_Attestation{
					CommitteeIndex:  attestation.CommitteeIndex,
					CommitteeLength: attestation.CommitteeLength,
				},
			},
		}
		duties = append(duties, attestationDuty)

		isAggregate, selectionProof, err := s.isAttestatorAggregate(attestation.CommitteeLength, attestation.Slot, uint64(attestation.ValidatorIndex))
		if err != nil {
			return nil, err
		}
		if isAggregate {
			aggregationDuty := &proto.Duty{
				Id:             uuid.Generate(),
				Slot:           attestation.Slot,
				Epoch:          eval.Epoch,
				ActiveTime:     timestamppb.New(s.atSlot(attestation.Slot).Add(attestationAggregationDelay)),
				ValidatorIndex: uint64(attestation.ValidatorIndex),
				Job: &proto.Duty_AttestationAggregate_{
					AttestationAggregate: &proto.Duty_AttestationAggregate{
						SelectionProof: selectionProof[:],
					},
				},
				BlockedBy: []string{
					attestationDuty.Id,
				},
			}
			duties = append(duties, aggregationDuty)
		}
	}

	FirstSlot := eval.Epoch * s.config.SlotsPerEpoch
	LastSlot := eval.Epoch*s.config.SlotsPerEpoch + s.config.SlotsPerEpoch

	for _, committee := range eval.Committee {
		for slot := FirstSlot; slot < LastSlot; slot++ {
			committeDuty := &proto.Duty{
				Id:             uuid.Generate(),
				Slot:           slot,
				Epoch:          eval.Epoch,
				ActiveTime:     timestamppb.New(s.atSlot(slot).Add(syncCommitteeDelay)),
				Job:            &proto.Duty_SyncCommittee_{},
				ValidatorIndex: uint64(committee.ValidatorIndex),
			}
			duties = append(duties, committeDuty)

			indices, err := convertToIntIndices(committee.ValidatorSyncCommitteeIndices)
			if err != nil {
				return nil, err
			}

			// aggregate indices in subCommitteIndex
			committeeIndices := map[uint64]struct{}{}
			for _, index := range indices {
				subcommittee := uint64(index) / (s.config.SyncCommitteeSize / syncCommitteeSubnetCount)
				committeeIndices[subcommittee] = struct{}{}
			}

			for index := range committeeIndices {
				isAggregate, selectionProof, err := s.isSyncCommitteeAggregator(index, slot, uint64(committee.ValidatorIndex))
				if err != nil {
					return nil, err
				}
				if isAggregate {
					aggregateDuty := &proto.Duty{
						Id:         uuid.Generate(),
						Slot:       slot,
						Epoch:      eval.Epoch,
						ActiveTime: timestamppb.New(s.atSlot(slot).Add(syncCommitteeAggregationDelay)),
						Job: &proto.Duty_SyncCommitteeAggregate_{
							SyncCommitteeAggregate: &proto.Duty_SyncCommitteeAggregate{
								SelectionProof:    selectionProof[:],
								SubCommitteeIndex: index,
							},
						},
						ValidatorIndex: uint64(committee.ValidatorIndex),
					}
					duties = append(duties, aggregateDuty)
				}
			}
		}
	}

	s.duties = duties
	s.cleanDuties()

	plan := &proto.Plan{
		Duties: s.duties,
	}
	return plan, nil
}

func (s *Scheduler) isAttestatorAggregate(committeeSize uint64, slot uint64, account uint64) (bool, [96]byte, error) {
	modulo := committeeSize / s.config.TargetAggregatorsPerCommittee
	if modulo == 0 {
		modulo = 1
	}

	slotRoot := proto.Uint64SSZ(slot)

	signature, err := s.state.Sign(s.ctx, proto.DomainSelectionProofType, s.eval.Epoch, account, slotRoot)
	if err != nil {
		return false, [96]byte{}, fmt.Errorf("failed to sign attestation aggregate: %v", err)
	}

	hash := sha256.Sum256(signature[:])
	return binary.LittleEndian.Uint64(hash[:8])%modulo == 0, signature, nil
}

const (
	syncCommitteeSubnetCount              = 4
	targetAggregattorsPerSyncSubcommittee = 16
)

func (s *Scheduler) isSyncCommitteeAggregator(subCommitteeIndex uint64, slot uint64, account uint64) (bool, [96]byte, error) {
	modulo := s.config.SyncCommitteeSize / syncCommitteeSubnetCount / targetAggregattorsPerSyncSubcommittee
	if modulo < 1 {
		modulo = 1
	}

	signObj := &consensus.SyncAggregatorSelectionData{
		Slot:              slot,
		SubCommitteeIndex: subCommitteeIndex,
	}
	root, err := signObj.HashTreeRoot()
	if err != nil {
		return false, [96]byte{}, err
	}

	signature, err := s.state.Sign(s.ctx, proto.DomainSyncCommitteeSelectionProof, s.eval.Epoch, account, root)
	if err != nil {
		return false, [96]byte{}, err
	}

	hash := sha256.Sum256(signature[:])
	return binary.LittleEndian.Uint64(hash[:8])%modulo == 0, signature, nil
}

func (s *Scheduler) cleanDuties() {
	var cleanDuties []*proto.Duty

	now := time.Now()
	for _, d := range s.duties {
		if d.ActiveTime.AsTime().After(now) {
			cleanDuties = append(cleanDuties, d)
		}
	}
	s.duties = cleanDuties
}

func convertToIntIndices(s []string) ([]uint64, error) {
	res := []uint64{}
	for _, str := range s {
		u64, err := strconv.ParseUint(str, 10, 32)
		if err != nil {
			return nil, err
		}
		res = append(res, u64)
	}
	return res, nil
}
