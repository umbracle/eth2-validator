package proto

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	proto "github.com/golang/protobuf/proto"
	"github.com/umbracle/go-eth-consensus/bls"
	"github.com/umbracle/go-eth-consensus/http"
)

type DutyType string

const (
	DutyBlockProposal          = "block-proposal"
	DutyAttestation            = "attestation"
	DutyAttestationAggregate   = "attesation-aggregate"
	DutySyncCommittee          = "sync-committee"
	DutySyncCommitteeAggregate = "sync-committee-aggregate"
)

func (d DutyType) String() string {
	return string(d)
}

func (d *Duty) Type() DutyType {
	switch d.Job.(type) {
	case *Duty_BlockProposal_:
		return DutyBlockProposal
	case *Duty_Attestation_:
		return DutyAttestation
	case *Duty_AttestationAggregate_:
		return DutyAttestationAggregate
	case *Duty_SyncCommittee_:
		return DutySyncCommittee
	case *Duty_SyncCommitteeAggregate_:
		return DutySyncCommitteeAggregate
	default:
		panic("BUG")
	}
}

type DutyJob interface {
	isDuty_Job
}

type DomainType string

const (
	DomainBeaconProposerType          DomainType = "beacon-proposer"
	DomainRandaomType                 DomainType = "randao"
	DomainBeaconAttesterType          DomainType = "beacon-attester"
	DomainDepositType                 DomainType = "deposit"
	DomainVoluntaryExitType           DomainType = "voluntary-exit"
	DomainSelectionProofType          DomainType = "selection-proof"
	DomainAggregateAndProofType       DomainType = "aggregate-and-proof"
	DomainSyncCommitteeType           DomainType = "sync-committee"
	DomainSyncCommitteeSelectionProof DomainType = "sync-committee-selection-proof"
	DomainContributionAndProof        DomainType = "contribution-and-proof"
)

type Evaluation struct {
	Epoch       uint64
	Attestation []*http.AttesterDuty
	Proposer    []*http.ProposerDuty
	Committee   []*http.CommitteeSyncDuty
	GenesisTime time.Time
}

type Plan struct {
	Duties []*Duty
}

func (p *Plan) GoPrint() string {
	counts := map[DutyType]uint64{}
	for _, duty := range p.Duties {
		counts[duty.Type()]++
	}

	grp := []string{}
	for n, count := range counts {
		grp = append(grp, fmt.Sprintf("(%s: %d)", n, count))
	}

	var res string
	if len(grp) == 0 {
		res = "no duties found"
	} else {
		res = strings.Join(grp, ", ")
	}

	str := fmt.Sprintf("Duties: %s", res)
	return str
}

func Uint64SSZ(num uint64) [32]byte {
	buf := [32]byte{}
	binary.LittleEndian.PutUint64(buf[:], num)
	return buf
}

func RootSSZ(root [32]byte) [32]byte {
	return root
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

func (d *Duty) Copy() *Duty {
	return proto.Clone(d).(*Duty)
}

func NewDuty() *Duty {
	return &Duty{}
}

func (d *Duty) WithEpoch(epoch uint64) *Duty {
	d.Epoch = epoch
	return d
}

func (d *Duty) WithID(id string) *Duty {
	d.Id = id
	return d
}

func (d *Duty) WithValIndex(indx uint64) *Duty {
	d.ValidatorIndex = indx
	return d
}

func (d *Duty) WithAttestation(job *Duty_Attestation) *Duty {
	d.Job = &Duty_Attestation_{
		Attestation: job,
	}
	return d
}

func (d *Duty) WithSyncCommitteeAggregate(job *Duty_SyncCommitteeAggregate) *Duty {
	d.Job = &Duty_SyncCommitteeAggregate_{
		SyncCommitteeAggregate: job,
	}
	return d
}

func (v *Validator) Copy() *Validator {
	return proto.Clone(v).(*Validator)
}
