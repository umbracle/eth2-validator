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
