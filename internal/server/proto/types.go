package proto

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
