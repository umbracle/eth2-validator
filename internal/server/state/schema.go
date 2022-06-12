package state

import "github.com/hashicorp/go-memdb"

const (
	dutiesTable     = "duties"
	validatorsTable = "validators"
)

var schema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		dutiesTable: {
			Name: dutiesTable,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.StringFieldIndex{Field: "Id"},
				},
				"type": {
					Name:    "type",
					Unique:  false,
					Indexer: &IndexJob{},
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
					Unique:  false,
					Indexer: &memdb.UintFieldIndex{Field: "Index"},
				},
				"activationEpoch": {
					Name:    "activationEpoch",
					Unique:  false,
					Indexer: &memdb.UintFieldIndex{Field: "ActivationEpoch"},
				},
			},
		},
	},
}
