package state

import (
	"bytes"
	"fmt"

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
	db, err := bolt.Open(path, 0600, nil) // TODO
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

func (s *State) UpsertValidator(validator *proto.Validator) error {
	txn := s.memdb.Txn(true)
	defer txn.Abort()

	if err := txn.Insert(validatorsTable, validator); err != nil {
		return err
	}

	txn.Commit()
	return nil
}

func (s *State) GetValidatorsActiveAt(epoch uint64) ([]*proto.Validator, error) {
	txn := s.memdb.Txn(false)
	defer txn.Abort()

	var it memdb.ResultIterator
	var err error

	if epoch == 0 {
		it, err = txn.Get(validatorsTable, "activationEpoch", uint64(0))
	} else {
		it, err = txn.ReverseLowerBound(validatorsTable, "activationEpoch", epoch+1)
	}
	if err != nil {
		return nil, err
	}

	result := []*proto.Validator{}
	for obj := it.Next(); obj != nil; obj = it.Next() {
		result = append(result, obj.(*proto.Validator))
	}
	return result, nil
}

func (s *State) GetValidatorByIndex(index uint64) (*proto.Validator, error) {
	txn := s.memdb.Txn(false)
	defer txn.Abort()

	validator, err := txn.First(validatorsTable, "index", index)
	if err != nil {
		return nil, err
	}
	if validator == nil {
		return nil, nil
	}
	return validator.(*proto.Validator), nil
}

func (s *State) ValidatorsList(ws memdb.WatchSet) (memdb.ResultIterator, error) {
	txn := s.memdb.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(validatorsTable, "id")
	if err != nil {
		return nil, err
	}

	ws.Add(iter.WatchCh())
	return iter, nil
}

type blockSlotProposal struct {
	duty *proto.Duty
}

func (b *blockSlotProposal) Slot() uint64 {
	return b.duty.Slot
}

func (b *blockSlotProposal) Equal(root []byte) bool {
	return bytes.Equal(b.duty.Result.BlockProposal.Root, root)
}

func (b *blockSlotProposal) Exists() bool {
	return b.duty != nil
}

func (s *State) SlashBlockCheck(valID uint64, slot uint64, root []byte) error {
	txn := s.memdb.Txn(false)
	defer txn.Abort()

	proposalAtSlotFn := func() (*blockSlotProposal, error) {
		result := &blockSlotProposal{}

		obj, err := txn.First(dutiesTable, "block_proposer", true, valID, slot)
		if err != nil {
			return nil, err
		}
		if obj != nil {
			result.duty = obj.(*proto.Duty)
		}
		return result, nil
	}

	lowestSignedProposalFn := func() (*blockSlotProposal, error) {
		result := &blockSlotProposal{}

		it, err := txn.LowerBound(dutiesTable, "block_proposer", true, valID, slot)
		if err != nil {
			return nil, err
		}
		obj := it.Next()
		if obj != nil {
			result.duty = obj.(*proto.Duty)
		}
		return result, nil
	}

	proposalAtSlot, err := proposalAtSlotFn()
	if err != nil {
		return err
	}
	lowestProposal, err := lowestSignedProposalFn()
	if err != nil {
		return err
	}

	if proposalAtSlot.Exists() {
		if !proposalAtSlot.Equal(root) {
			return fmt.Errorf("root at slot %d is repeated", slot)
		}
	}

	if lowestProposal.Exists() {
		if lowestProposal.Slot() >= slot {
			return fmt.Errorf("could not sign slot with lowest slot number")
		}
	}
	return nil
}
