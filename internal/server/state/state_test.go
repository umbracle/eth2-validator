package state

import (
	"os"
	"path"
	"testing"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/assert"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/uuid"
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
		Job: &proto.Duty_BlockProposal_{
			BlockProposal: &proto.Duty_BlockProposal{},
		},
	}
	err := state.InsertDuty(duty1)
	assert.NoError(t, err)

	duty2 := &proto.Duty{
		Id: "b",
		Job: &proto.Duty_Attestation_{
			Attestation: &proto.Duty_Attestation{
				CommitteeIndex: 1,
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

	validators := []*proto.Validator{
		{PubKey: "a", Index: 1, ActivationEpoch: 0},
		{PubKey: "b", Index: 2, ActivationEpoch: 0},
		{PubKey: "c", Index: 3, ActivationEpoch: 2},
		{PubKey: "d", Index: 4, ActivationEpoch: 5},
	}
	for _, val := range validators {
		assert.NoError(t, state.UpsertValidator(val))
	}

	vals, err := state.GetValidatorsActiveAt(0)
	assert.NoError(t, err)
	assert.Len(t, vals, 2)
	assert.Equal(t, vals[0].Index, uint64(1))

	vals, err = state.GetValidatorsActiveAt(2)
	assert.NoError(t, err)
	assert.Len(t, vals, 3)
}

func TestState_SlashBlockCheck(t *testing.T) {
	state := newTestState(t)

	insertDuty := func(slot uint64, root []byte) {
		err := state.InsertDuty(&proto.Duty{
			Id:             uuid.Generate(),
			ValidatorIndex: 1,
			Slot:           slot,
			Job: &proto.Duty_BlockProposal_{
				BlockProposal: &proto.Duty_BlockProposal{},
			},
			Result: &proto.Duty_Result{
				BlockProposal: &proto.Duty_BlockProposalResult{
					Root: root,
				},
			},
		})
		assert.NoError(t, err)
	}

	root1 := []byte{0x1}
	root2 := []byte{0x2}

	insertDuty(5, root1)

	// future block, no slash
	assert.NoError(t, state.SlashBlockCheck(1, 6, root1))

	// last block, slash
	assert.Error(t, state.SlashBlockCheck(1, 5, root2))

	// past block
	assert.Error(t, state.SlashBlockCheck(1, 4, root2))
}

func TestState_SlashAttestCheck(t *testing.T) {
	state := newTestState(t)

	err := state.InsertDuty(&proto.Duty{
		Id:             uuid.Generate(),
		ValidatorIndex: 1,
		Slot:           5,
		Job:            &proto.Duty_Attestation_{},
		Result: &proto.Duty_Result{
			Attestation: &proto.Duty_AttestationResult{
				Source: &proto.Duty_AttestationResult_Checkpoint{Epoch: 10},
				Target: &proto.Duty_AttestationResult_Checkpoint{Epoch: 15},
			},
		},
	})
	assert.NoError(t, err)
	assert.Error(t, state.SlashAttestCheck(1, 10, 15))
}
