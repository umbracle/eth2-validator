package scheduler

import (
	"crypto/sha256"
	"encoding/binary"
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
	Sign(domain proto.DomainType, account uint64, root []byte) ([]byte, error)
}

type Scheduler struct {
	logger hclog.Logger
	state  State
	eval   *proto.Evaluation
	duties []*proto.Duty
	config *beacon.ChainConfig
}

func NewScheduler(logger hclog.Logger, state State, config *beacon.ChainConfig) *Scheduler {
	return &Scheduler{
		logger: logger,
		state:  state,
		config: config,
	}
}

func (s *Scheduler) atSlot(slot uint64) time.Time {
	return s.eval.GenesisTime.Add(time.Duration(slot*s.config.SecondsPerSlot) * time.Second)
}

func (s *Scheduler) Process(eval *proto.Evaluation) (*proto.Plan, error) {
	s.eval = eval

	proposalDutiesBySlot := map[uint64]*proto.Duty{}

	var duties []*proto.Duty
	for _, proposal := range eval.Proposer {
		timeToStart := s.atSlot(proposal.Slot)

		proposalDuty := &proto.Duty{
			Id:         uuid.Generate(),
			Slot:       proposal.Slot,
			Epoch:      eval.Epoch,
			ActiveTime: timestamppb.New(timeToStart),
			Job:        &proto.Duty_BlockProposal{},
		}
		proposalDutiesBySlot[proposal.Slot] = proposalDuty
		duties = append(duties, proposalDuty)
	}

	for _, attestation := range eval.Attestation {
		timeToStart := s.atSlot(attestation.Slot)

		raw, err := json.Marshal(attestation)
		if err != nil {
			return nil, err
		}
		attestationDuty := &proto.Duty{
			Id:         uuid.Generate(),
			Slot:       attestation.Slot,
			Epoch:      eval.Epoch,
			ActiveTime: timestamppb.New(timeToStart),
			Input: &anypb.Any{
				Value: raw,
			},
			Job: &proto.Duty_Attestation{
				Attestation: &proto.Attestation{},
			},
		}
		duties = append(duties, attestationDuty)

		isAggregate, err := s.isAttestatorAggregate(attestation.CommitteeLength, attestation.Slot, 0)
		if err != nil {
			return nil, err
		}
		if isAggregate {
			aggregationDuty := &proto.Duty{
				Id:         uuid.Generate(),
				Slot:       attestation.Slot,
				Epoch:      eval.Epoch,
				ActiveTime: timestamppb.New(timeToStart),
				Job: &proto.Duty_AttestationAggregate{
					AttestationAggregate: &proto.AttestationAggregate{},
				},
				BlockedBy: []string{
					attestationDuty.Id,
					proposalDutiesBySlot[attestation.Slot].Id,
				},
			}
			duties = append(duties, aggregationDuty)
		}

	}

	FirstSlot := eval.Epoch * s.config.SlotsPerEpoch
	LastSlot := eval.Epoch*s.config.SlotsPerEpoch + s.config.SlotsPerEpoch

	for range eval.Committee {
		for slot := FirstSlot; slot < LastSlot; slot++ {
			timeToStart := s.atSlot(slot)

			committeDuty := &proto.Duty{
				Id:         uuid.Generate(),
				Slot:       slot,
				Epoch:      eval.Epoch,
				ActiveTime: timestamppb.New(timeToStart),
				Job:        &proto.Duty_SyncCommittee{},
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

func (s *Scheduler) isAttestatorAggregate(committeeSize uint64, slot uint64, account uint64) (bool, error) {
	modulo := committeeSize / s.config.TargetAggregatorsPerCommittee
	if modulo == 0 {
		modulo = 1
	}

	slotRoot := proto.Uint64Root(slot)

	signature, err := s.state.Sign(proto.DomainSelectionProofType, account, slotRoot)
	if err != nil {
		return false, fmt.Errorf("failed to sign attestation aggregate: %v", err)
	}

	hash := sha256.Sum256(signature)
	return binary.LittleEndian.Uint64(hash[:8])%modulo == 0, nil
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
