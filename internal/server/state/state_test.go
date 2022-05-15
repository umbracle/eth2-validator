package state

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/stretchr/testify/assert"
	"github.com/umbracle/eth2-validator/internal/server/proto"
)

func newTestState(t *testing.T) *State {
	dir := fmt.Sprintf("/tmp/bridge-temp_%v", time.Now().Format(time.RFC3339))
	err := os.Mkdir(dir, 0777)
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
		PubKey: "pub1",
		Slot:   1,
		Job: &proto.Duty_BlockProposal{
			BlockProposal: &proto.BlockProposal{
				Root: "abc",
			},
		},
	}
	err := state.InsertDuty(duty1)
	assert.NoError(t, err)

	duty2 := &proto.Duty{
		PubKey: "pub1",
		Job: &proto.Duty_Attestation{
			Attestation: &proto.Attestation{
				Root: "def",
			},
		},
	}
	err = state.InsertDuty(duty2)
	assert.NoError(t, err)

	ws := memdb.NewWatchSet()
	iter, err := state.JobsList(ws)
	assert.NoError(t, err)

	// return two results
	assert.NotNil(t, iter.Next())
	assert.NotNil(t, iter.Next())
	assert.Nil(t, iter.Next())
}
