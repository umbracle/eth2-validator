package state

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
	"github.com/umbracle/eth2-validator/internal/server/proto"
)

const (
	dutiesTable     = "duties"
	validatorsTable = "validators"
)

var schema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		dutiesTable: {
			Name: dutiesTable,
			Indexes: map[string]*memdb.IndexSchema{
				// index by the id of the duty
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "Id"},
				},
				// index by type of duty
				"type": {
					Name:    "type",
					Indexer: &IndexJob{},
				},
				// index for EIP-3076 block proposer slash protection
				"block_proposer": {
					Name: "block_proposer",
					Indexer: &memdb.CompoundIndex{
						Indexes: []memdb.Indexer{
							&memdb.ConditionalIndex{
								Conditional: isJobConditional(proto.DutyBlockProposal, true),
							},
							&memdb.UintFieldIndex{
								Field: "ValidatorIndex",
							},
							&memdb.UintFieldIndex{
								Field: "Slot",
							},
						},
					},
				},
			},
		},
		validatorsTable: {
			Name: validatorsTable,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "PubKey"},
				},
				"index": {
					Name:    "index",
					Indexer: &memdb.UintFieldIndex{Field: "Index"},
				},
				"activationEpoch": {
					Name:    "activationEpoch",
					Indexer: &memdb.UintFieldIndex{Field: "ActivationEpoch"},
				},
			},
		},
	},
}

func isJobConditional(typ proto.DutyType, withResult bool) func(obj interface{}) (bool, error) {
	return func(obj interface{}) (bool, error) {
		duty, ok := obj.(*proto.Duty)
		if !ok {
			return false, fmt.Errorf("obj is not duty")
		}
		if duty.Type() != typ {
			return false, nil
		}
		if !withResult {
			return true, nil
		}
		return duty.Result != nil, nil
	}
}
