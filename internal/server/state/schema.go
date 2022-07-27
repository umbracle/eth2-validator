package state

import (
	"github.com/hashicorp/go-memdb"
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
					Name:         "block_proposer",
					AllowMissing: true,
					Indexer:      &slashingBlockIndex{},
				},
				// index for EIP-3076 attest slash protection
				"attest_proposer": {
					Name:         "attest_proposer",
					AllowMissing: true,
					Indexer:      &slashingAttestIndex{},
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
