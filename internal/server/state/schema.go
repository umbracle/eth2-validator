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
				// pending is the index for validators that are not yet
				// active on the consensus
				"pending": {
					Name:         "pending",
					AllowMissing: false,
					Indexer: &memdb.FieldSetIndex{
						Field: "Metadata",
					},
				},
				// index is the indexer to query an "active" validator by
				// its validator index in the consensus.
				"index": {
					Name:         "index",
					AllowMissing: true,
					Indexer: &activeValidatorIndex{
						Indexer: &memdb.UintFieldIndex{
							Field: "Index",
						},
					},
				},
				// activationEpoch is the index to sort out "active" validators
				// by its activation epoch.
				"activationEpoch": {
					Name:         "activationEpoch",
					AllowMissing: true,
					Indexer: &activeValidatorIndex{
						Indexer: &memdb.UintFieldIndex{
							Field: "ActivationEpoch",
						},
					},
				},
			},
		},
	},
}
