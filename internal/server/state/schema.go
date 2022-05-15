package state

import "github.com/hashicorp/go-memdb"

const (
	dutiesTable = "duties"
)

var schema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		dutiesTable: {
			Name: dutiesTable,
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &memdb.UintFieldIndex{Field: "Slot"},
				},
				"type": {
					Name:    "type",
					Unique:  false,
					Indexer: &IndexJob{},
				},
				"pubkey": {
					Name:    "pubkey",
					Unique:  false,
					Indexer: &memdb.StringFieldIndex{Field: "PubKey"},
				},
			},
		},
	},
}
