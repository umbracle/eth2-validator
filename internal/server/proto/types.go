package proto

import (
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bls"
)

type DutyType string

const (
	DutyBlockProposal        = "block-proposal"
	DutyAttestation          = "attestation"
	DutyAttestationAggregate = "attesation-aggregate"
	DutySyncCommittee        = "sync-committee"
)

func (d *Duty) Type() string {
	switch d.Job.(type) {
	case *Duty_BlockProposal:
		return DutyBlockProposal
	case *Duty_Attestation:
		return DutyAttestation
	case *Duty_AttestationAggregate:
		return DutyAttestationAggregate
	case *Duty_SyncCommittee:
		return DutySyncCommittee
	default:
		panic("BUG")
	}
}

type DutyJob interface {
	isDuty_Job
}

type DomainType string

const (
	DomainBeaconProposerType    DomainType = "beacon-proposer"
	DomainRandaomType           DomainType = "randao"
	DomainBeaconAttesterType    DomainType = "beacon-attester"
	DomainDepositType           DomainType = "deposit"
	DomainVoluntaryExitType     DomainType = "voluntary-exit"
	DomainSelectionProofType    DomainType = "selection-proof"
	DomainAggregateAndProofType DomainType = "aggregate-and-proof"
	DomainSyncCommitteeType     DomainType = "sync-committee"
)

type Evaluation struct {
	Epoch       uint64
	Attestation []*beacon.AttesterDuty
	Proposer    []*beacon.ProposerDuty
	Committee   []*beacon.CommitteeSyncDuty
	GenesisTime time.Time
}

type Plan struct {
	Duties []*Duty
}

func Uint64Root(num uint64) []byte {
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, num)
	return buf
}

func (v *Validator) Key() (*bls.Key, error) {
	buf, err := hex.DecodeString(v.PrivKey)
	if err != nil {
		return nil, err
	}
	key, err := bls.NewKeyFromPriv(buf)
	if err != nil {
		return nil, err
	}
	return key, nil
}
