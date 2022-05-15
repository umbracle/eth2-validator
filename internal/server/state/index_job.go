package state

import (
	"fmt"

	"github.com/umbracle/eth2-validator/internal/server/proto"
)

type IndexJob struct {
}

func (idx *IndexJob) FromObject(obj interface{}) (bool, []byte, error) {
	elem, ok := obj.(*proto.Duty)
	if !ok {
		return false, nil, fmt.Errorf("bad")
	}
	if elem.Job == nil {
		return false, nil, fmt.Errorf("bad")
	}

	var typ proto.DutyType
	switch elem.Job.(type) {
	case *proto.Duty_Attestation:
		typ = proto.DutyAttestation

	case *proto.Duty_BlockProposal:
		typ = proto.DutyBlockProposal

	default:
		return false, nil, fmt.Errorf("not expected")
	}

	return true, []byte(typ), nil
}

func (idx *IndexJob) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}

	typ := args[0].(proto.DutyType)
	return []byte(typ), nil
}
