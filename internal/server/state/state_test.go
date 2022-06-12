package state

import (
	"os"
	"path"
	"testing"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/assert"
	"github.com/umbracle/eth2-validator/internal/server/proto"
)

func newTestState(t *testing.T) *State {
	dir, err := os.MkdirTemp("/tmp", "eth2-state-")
	if err != nil {
		t.Fatal(err)
	}

	state, err := NewState(path.Join(dir, "my.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	})
	return state
}

func TestState_InsertDuty(t *testing.T) {
	state := newTestState(t)

	duty1 := &proto.Duty{
		Id:   "a",
		Slot: 1,
		Job: &proto.Duty_BlockProposal{
			BlockProposal: &proto.BlockProposal{
				Root: "abc",
			},
		},
	}
	err := state.InsertDuty(duty1)
	assert.NoError(t, err)

	duty2 := &proto.Duty{
		Id: "b",
		Job: &proto.Duty_Attestation{
			Attestation: &proto.Attestation{
				Root: "def",
			},
		},
	}
	err = state.InsertDuty(duty2)
	assert.NoError(t, err)

	ws := memdb.NewWatchSet()
	iter, err := state.DutiesList(ws)
	assert.NoError(t, err)

	// return two results
	assert.NotNil(t, iter.Next())
	assert.NotNil(t, iter.Next())
	assert.Nil(t, iter.Next())

	found, err := state.DutyByID("b")
	assert.Nil(t, err)
	assert.Equal(t, found.Id, "b")
}

func TestState_ValidatorWorkflow(t *testing.T) {
	state := newTestState(t)

	val := &proto.Validator{
		PubKey:          "a",
		Index:           1,
		ActivationEpoch: 5,
	}
	assert.NoError(t, state.UpsertValidator(val))

	val = &proto.Validator{
		PubKey:          "b",
		Index:           2,
		ActivationEpoch: 2,
	}
	assert.NoError(t, state.UpsertValidator(val))

	vals, err := state.GetValidatorsActiveAt(4)
	assert.NoError(t, err)
	assert.Len(t, vals, 1)
	assert.Equal(t, vals[0].Index, uint64(2))
}
