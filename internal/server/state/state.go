package state

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/boltdb/bolt"
	"github.com/hashicorp/go-memdb"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	gproto "google.golang.org/protobuf/proto"
)

// list of persistent buckets
var (
	dutiesBucket     = []byte("duties")
	validatorsBucket = []byte("validators")
)

// State is the entity that stores the state of the validator
type State struct {
	db    *bolt.DB
	memdb *memdb.MemDB
}

func NewState(path string) (*State, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bkts := [][]byte{
			dutiesBucket,
			validatorsBucket,
		}

		for _, b := range bkts {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})
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
	if err := state.reIndex(); err != nil {
		return nil, err
	}

	return state, nil
}

func (s *State) reIndex() error {
	memTxn := s.memdb.Txn(true)
	defer memTxn.Abort()

	err := s.db.View(func(tx *bolt.Tx) error {
		// index duties
		if err := indexObject(tx.Bucket(dutiesBucket), memTxn, "duties", proto.Duty{}); err != nil {
			return err
		}
		// index validators
		if err := indexObject(tx.Bucket(validatorsBucket), memTxn, "validators", proto.Validator{}); err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		memTxn.Commit()
	}

	return err
}

func indexObject(bkt *bolt.Bucket, memTxn *memdb.Txn, table string, obj interface{}) error {
	err := bkt.ForEach(func(k, v []byte) error {
		elem := reflect.New(reflect.TypeOf(obj)).Interface().(gproto.Message)

		if err := dbGet(bkt, k, elem); err != nil {
			return err
		}
		if err := memTxn.Insert(table, elem); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *State) Close() error {
	return s.db.Close()
}

func (s *State) UpsertDuties(duties []*proto.Duty) error {
	memTxn := s.memdb.Txn(true)
	defer memTxn.Abort()

	err := s.db.Update(func(dbTxn *bolt.Tx) error {
		bkt := dbTxn.Bucket(dutiesBucket)

		for _, duty := range duties {
			if err := memTxn.Insert(dutiesTable, duty); err != nil {
				return err
			}
			if err := dbPut(bkt, []byte(duty.Id), duty); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		memTxn.Commit()
	}

	return err
}

func (s *State) UpsertDuty(duty *proto.Duty) error {
	return s.UpsertDuties([]*proto.Duty{duty})
}

func (s *State) DutyByID(dutyID string) (*proto.Duty, error) {
	memTxn := s.memdb.Txn(false)
	defer memTxn.Abort()

	duty, err := memTxn.First(dutiesTable, "id", dutyID)
	if err != nil {
		return nil, err
	}
	if duty == nil {
		return nil, nil
	}
	return duty.(*proto.Duty), nil
}

func (s *State) DutiesList(ws memdb.WatchSet) (memdb.ResultIterator, error) {
	memTxn := s.memdb.Txn(false)
	defer memTxn.Abort()

	iter, err := memTxn.Get(dutiesTable, "id")
	if err != nil {
		return nil, err
	}

	ws.Add(iter.WatchCh())
	return iter, nil
}

func (s *State) UpsertValidator(validators ...*proto.Validator) error {
	memTxn := s.memdb.Txn(true)
	defer memTxn.Abort()

	err := s.db.Update(func(dbTxn *bolt.Tx) error {
		bkt := dbTxn.Bucket(validatorsBucket)

		for _, validator := range validators {
			if err := memTxn.Insert(validatorsTable, validator); err != nil {
				return err
			}
			if err := dbPut(bkt, []byte(validator.PubKey), validator); err != nil {
				return err
			}
		}

		return nil
	})
	if err == nil {
		memTxn.Commit()
	}

	return err
}

func (s *State) GetValidatorsPending(ws memdb.WatchSet) ([]*proto.Validator, error) {
	memTxn := s.memdb.Txn(false)
	defer memTxn.Abort()

	iter, err := memTxn.Get(validatorsTable, "pending", false)
	if err != nil {
		return nil, err
	}

	ws.Add(iter.WatchCh())

	result := []*proto.Validator{}
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		result = append(result, obj.(*proto.Validator))
	}
	return result, nil
}

func (s *State) GetValidatorsActiveAt(epoch uint64) ([]*proto.Validator, error) {
	memTxn := s.memdb.Txn(false)
	defer memTxn.Abort()

	var it memdb.ResultIterator
	var err error

	if epoch == 0 {
		it, err = memTxn.Get(validatorsTable, "activationEpoch", uint64(0))
	} else {
		it, err = memTxn.ReverseLowerBound(validatorsTable, "activationEpoch", epoch+1)
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
	memTxn := s.memdb.Txn(false)
	defer memTxn.Abort()

	validator, err := memTxn.First(validatorsTable, "index", index)
	if err != nil {
		return nil, err
	}
	if validator == nil {
		return nil, nil
	}
	return validator.(*proto.Validator), nil
}

func (s *State) ValidatorsList(ws memdb.WatchSet) (memdb.ResultIterator, error) {
	memTxn := s.memdb.Txn(false)
	defer memTxn.Abort()

	iter, err := memTxn.Get(validatorsTable, "id")
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
	memTxn := s.memdb.Txn(false)
	defer memTxn.Abort()

	proposalAtSlotFn := func() (*blockSlotProposal, error) {
		result := &blockSlotProposal{}

		obj, err := memTxn.First(dutiesTable, "block_proposer", valID, slot)
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

		it, err := memTxn.LowerBound(dutiesTable, "block_proposer_prefix", valID)
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

type attestSlashProposal struct {
	typ  string
	duty *proto.Duty
}

func (a *attestSlashProposal) Epoch() uint64 {
	if a.typ == "source" {
		return a.duty.Result.Attestation.Source.Epoch
	}
	return a.duty.Result.Attestation.Target.Epoch
}

func (a *attestSlashProposal) Exists() bool {
	return a.duty != nil
}

func (s *State) SlashAttestCheck(valID uint64, source, target uint64) error {
	memTxn := s.memdb.Txn(false)
	defer memTxn.Abort()

	lowestSignedEpochFn := func(typ string) (*attestSlashProposal, error) {
		result := &attestSlashProposal{typ: typ}

		it, err := memTxn.LowerBound(dutiesTable, "attest_proposer_prefix", valID, typ)
		if err != nil {
			return nil, err
		}
		obj := it.Next()
		if obj != nil {
			result.duty = obj.(*proto.Duty)
		}
		return result, nil
	}

	lowestSourceEpoch, err := lowestSignedEpochFn("source")
	if err != nil {
		return err
	}
	lowestTargetEpoch, err := lowestSignedEpochFn("target")
	if err != nil {
		return err
	}

	if lowestSourceEpoch.Exists() {
		if source < lowestSourceEpoch.Epoch() {
			return fmt.Errorf("bad")
		}
	}
	if lowestTargetEpoch.Exists() {
		if target <= lowestTargetEpoch.Epoch() {
			return fmt.Errorf("bad2")
		}
	}
	return nil
}

func dbPut(b *bolt.Bucket, id []byte, msg gproto.Message) error {
	enc, err := gproto.Marshal(msg)
	if err != nil {
		return err
	}

	if err := b.Put(id, enc); err != nil {
		return err
	}
	return nil
}

func dbGet(b *bolt.Bucket, id []byte, msg gproto.Message) error {
	raw := b.Get(id)
	if raw == nil {
		return fmt.Errorf("record not found")
	}

	if err := gproto.Unmarshal(raw, msg); err != nil {
		return fmt.Errorf("failed to decode data: %v", err)
	}
	return nil
}
