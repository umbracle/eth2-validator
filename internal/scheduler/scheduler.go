package scheduler

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/uuid"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type State interface {
	Sign(ctx context.Context, domain proto.DomainType, epoch uint64, account uint64, root []byte) ([]byte, error)
}

type Scheduler struct {
	logger hclog.Logger
	ctx    context.Context
	state  State
	eval   *proto.Evaluation
	duties []*proto.Duty
	config *beacon.ChainConfig
}

func NewScheduler(logger hclog.Logger, ctx context.Context, state State, config *beacon.ChainConfig) *Scheduler {
	return &Scheduler{
		logger: logger,
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
		attestationDelay            = slotDuration / 3
		attestationAggregationDelay = slotDuration * 2 / 3
		syncCommitteeDelay          = slotDuration / 3
	)

	var duties []*proto.Duty
	for _, proposal := range eval.Proposer {
		timeToStart := s.atSlot(proposal.Slot)

		proposalDuty := &proto.Duty{
			Id:             uuid.Generate(),
			Slot:           proposal.Slot,
			Epoch:          eval.Epoch,
			ActiveTime:     timestamppb.New(timeToStart),
			Job:            &proto.Duty_BlockProposal{},
			ValidatorIndex: uint64(proposal.ValidatorIndex),
		}
		duties = append(duties, proposalDuty)
	}

	for _, attestation := range eval.Attestation {
		raw, err := json.Marshal(attestation)
		if err != nil {
			return nil, err
		}
		attestationDuty := &proto.Duty{
			Id:             uuid.Generate(),
			Slot:           attestation.Slot,
			Epoch:          eval.Epoch,
			ActiveTime:     timestamppb.New(s.atSlot(attestation.Slot).Add(attestationDelay)),
			ValidatorIndex: uint64(attestation.ValidatorIndex),
			Input: &anypb.Any{
				Value: raw,
			},
			Job: &proto.Duty_Attestation{
				Attestation: &proto.Attestation{},
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
				Job: &proto.Duty_AttestationAggregate{
					AttestationAggregate: &proto.AttestationAggregate{
						SelectionProof: hex.EncodeToString(selectionProof),
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
				Job:            &proto.Duty_SyncCommittee{},
				ValidatorIndex: uint64(committee.ValidatorIndex),
			}
			duties = append(duties, committeDuty)
		}
	}

	s.duties = duties
	s.cleanDuties()

	plan := &proto.Plan{
		Duties: s.duties,
	}
	return plan, nil
}

func (s *Scheduler) isAttestatorAggregate(committeeSize uint64, slot uint64, account uint64) (bool, []byte, error) {
	modulo := committeeSize / s.config.TargetAggregatorsPerCommittee
	if modulo == 0 {
		modulo = 1
	}

	slotRoot := proto.Uint64SSZ(slot)

	signature, err := s.state.Sign(s.ctx, proto.DomainSelectionProofType, s.eval.Epoch, account, slotRoot)
	if err != nil {
		return false, nil, fmt.Errorf("failed to sign attestation aggregate: %v", err)
	}

	hash := sha256.Sum256(signature)
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
