package state

import (
	"path"
	"testing"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/uuid"
)

func newTestState(t *testing.T) *State {
	dir := t.TempDir()

	state, err := NewState(path.Join(dir, "my.db"))
	require.NoError(t, err)

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
	err := state.UpsertDuty(duty1)
	assert.NoError(t, err)

	duty2 := &proto.Duty{
		Id: "b",
		Job: &proto.Duty_Attestation_{
			Attestation: &proto.Duty_Attestation{
				CommitteeIndex: 1,
			},
		},
	}
	err = state.UpsertDuty(duty2)
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

func TestState_ReIndex(t *testing.T) {
	dir := t.TempDir()
	dbPath := path.Join(dir, "my.db")

	state, err := NewState(dbPath)
	require.NoError(t, err)

	// insert a duty
	duty := &proto.Duty{
		Id:   "a",
		Slot: 1,
		Job: &proto.Duty_BlockProposal_{
			BlockProposal: &proto.Duty_BlockProposal{},
		},
	}

	err = state.UpsertDuty(duty)
	require.NoError(t, err)

	// insert a validator
	val := &proto.Validator{
		PubKey: "a",
	}

	err = state.UpsertValidator(val)
	require.NoError(t, err)

	state.Close()

	state1, err := NewState(dbPath)
	require.NoError(t, err)

	duty1, err := state1.DutyByID("a")
	require.NoError(t, err)
	require.NotNil(t, duty1)

	val1, err := state1.GetValidatorsPending(memdb.NewWatchSet())
	require.NoError(t, err)
	require.Len(t, val1, 1)
}

func TestState_ValidatorsPending(t *testing.T) {
	state := newTestState(t)

	val := &proto.Validator{
		PubKey: "a",
	}
	require.NoError(t, state.UpsertValidator(val))

	vals, err := state.GetValidatorsPending(memdb.NewWatchSet())
	require.NoError(t, err)
	require.Len(t, vals, 1)

	val = val.Copy()
	val.Metadata = &proto.Validator_Metadata{}
	require.NoError(t, state.UpsertValidator(val))

	vals, err = state.GetValidatorsPending(memdb.NewWatchSet())
	require.NoError(t, err)
	require.Len(t, vals, 0)
}

func TestState_ValidatorByIndex(t *testing.T) {
	state := newTestState(t)

	val := &proto.Validator{
		PubKey: "a",
	}
	require.NoError(t, state.UpsertValidator(val))

	res, err := state.GetValidatorByIndex(10)
	require.Nil(t, err)
	require.Nil(t, res)

	val.Metadata = &proto.Validator_Metadata{
		Index: 10,
	}
	require.NoError(t, state.UpsertValidator(val))

	res, err = state.GetValidatorByIndex(10)
	require.Nil(t, err)
	require.Equal(t, res.Metadata.Index, uint64(10))
}

func TestState_ValidatorWorkflow(t *testing.T) {
	state := newTestState(t)

	metadata := []*proto.Validator_Metadata{
		{Index: 1, ActivationEpoch: 0},
		{Index: 2, ActivationEpoch: 0},
		{Index: 3, ActivationEpoch: 2},
		{Index: 4, ActivationEpoch: 5},
	}

	validators := []*proto.Validator{
		{PubKey: "a"},
		{PubKey: "b"},
		{PubKey: "c"},
		{PubKey: "d"},
	}
	assert.NoError(t, state.UpsertValidator(validators...))

	vals, err := state.GetValidatorsActiveAt(0)
	assert.NoError(t, err)
	assert.Len(t, vals, 0)

	// activate the validators
	for indx, val := range validators {
		val.Metadata = metadata[indx]
	}
	assert.NoError(t, state.UpsertValidator(validators...))

	vals, err = state.GetValidatorsActiveAt(0)
	assert.NoError(t, err)
	assert.Len(t, vals, 2)
	assert.Equal(t, vals[0].Metadata.Index, uint64(1))

	vals, err = state.GetValidatorsActiveAt(2)
	assert.NoError(t, err)
	assert.Len(t, vals, 3)
}

func TestState_SlashBlockCheck(t *testing.T) {
	state := newTestState(t)

	insertDuty := func(slot uint64, root []byte) {
		err := state.UpsertDuty(&proto.Duty{
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

	err := state.UpsertDuty(&proto.Duty{
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
