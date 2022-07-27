package state

import (
	"encoding/binary"
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

	typ := elem.Type()
	return true, []byte(typ), nil
}

func (idx *IndexJob) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}

	typ := args[0].(proto.DutyType)
	return []byte(typ), nil
}

type slashingBlockIndex struct {
}

func (s *slashingBlockIndex) FromObject(raw interface{}) (bool, []byte, error) {
	duty, ok := raw.(*proto.Duty)
	if !ok {
		return false, nil, fmt.Errorf("obj is not duty")
	}
	if duty.Type() != proto.DutyBlockProposal {
		return false, nil, nil
	}

	bb := &bytesWritter{}
	bb.uint64(duty.ValidatorIndex)
	bb.uint64(duty.Slot)

	return true, bb.out, nil
}

func (s *slashingBlockIndex) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("2 args expected but %d found", len(args))
	}
	valId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("index 0 is not uin64 val id")
	}
	slotId, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf("index 1 is not uint64 slot id")
	}

	bb := &bytesWritter{}
	bb.uint64(valId)
	bb.uint64(slotId)

	return bb.out, nil
}

func (s *slashingBlockIndex) PrefixFromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("1 args expected but %d found", len(args))
	}
	valId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("index 0 is not uin64 val id")
	}

	bb := &bytesWritter{}
	bb.uint64(valId)

	return bb.out, nil
}

type slashingAttestIndex struct {
}

func (s *slashingAttestIndex) FromObject(raw interface{}) (bool, [][]byte, error) {
	duty, ok := raw.(*proto.Duty)
	if !ok {
		return false, nil, fmt.Errorf("obj is not duty")
	}
	if duty.Type() != proto.DutyAttestation {
		return false, nil, nil
	}

	addCheckpoint := func(name string, c *proto.Duty_AttestationResult_Checkpoint) []byte {
		bb := &bytesWritter{}
		bb.uint64(duty.ValidatorIndex)
		bb.string(name)
		bb.uint64(c.Epoch)

		return bb.out
	}

	buf := [][]byte{
		addCheckpoint("source", duty.Result.Attestation.Source),
		addCheckpoint("target", duty.Result.Attestation.Target),
	}
	return true, buf, nil
}

func (s *slashingAttestIndex) FromArgs(args ...interface{}) ([]byte, error) {
	panic("TODO")
}

func (s *slashingAttestIndex) PrefixFromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("2 args expected but %d found", len(args))
	}
	valId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("index 0 is not uin64 val id")
	}
	typ, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("index 0 is not string")
	}
	if typ != "source" && typ != "target" {
		return nil, fmt.Errorf("typ expects 'source' or 'target'")
	}

	bb := &bytesWritter{}
	bb.uint64(valId)
	bb.string(typ)

	return bb.out, nil
}

type bytesWritter struct {
	out []byte
}

func (b *bytesWritter) write(buf []byte) {
	if b.out == nil {
		b.out = []byte{}
	}
	b.out = append(b.out, buf...)
}

func (b *bytesWritter) uint64(i uint64) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, i)
	b.write(buf)
}

func (b *bytesWritter) string(s string) {
	b.write([]byte(s + "\x00"))
}
