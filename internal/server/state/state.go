package state

import (
	"github.com/boltdb/bolt"
)

// State is the entity that stores the state of the validator
type State struct {
	db *bolt.DB
}

func NewState() *State {
	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		panic(err)
	}
	state := &State{
		db: db,
	}
	return state
}

func (s *State) Close() {
	s.db.Close()
}
