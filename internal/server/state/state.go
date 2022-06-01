package state

import (
	"github.com/boltdb/bolt"
	"github.com/hashicorp/go-memdb"
	"github.com/umbracle/eth2-validator/internal/server/proto"
)

// State is the entity that stores the state of the validator
type State struct {
	db    *bolt.DB
	memdb *memdb.MemDB
}

func NewState(path string) (*State, error) {
	db, err := bolt.Open("/tmp/bolt.db", 0600, nil) // TODO
	if err != nil {
		return nil, err
	}

	memdb, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}

	state := &State{
		db:    db,
		memdb: memdb,
	}
	return state, nil
}

func (s *State) Close() error {
	return s.db.Close()
}

func (s *State) InsertDuty(duty *proto.Duty) error {
	txn := s.memdb.Txn(true)
	defer txn.Abort()

	if err := txn.Insert(dutiesTable, duty); err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *State) DutyByID(dutyID string) (*proto.Duty, error) {
	txn := s.memdb.Txn(false)
	defer txn.Abort()

	duty, err := txn.First(dutiesTable, "id", dutyID)
	if err != nil {
		return nil, err
	}
	if duty == nil {
		return nil, nil
	}
	return duty.(*proto.Duty), nil
}

func (s *State) DutiesList(ws memdb.WatchSet) (memdb.ResultIterator, error) {
	txn := s.memdb.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(dutiesTable, "id")
	if err != nil {
		return nil, err
	}

	ws.Add(iter.WatchCh())
	return iter, nil
}
