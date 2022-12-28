package state

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func NewInmemState(t *testing.T) *State {
	tmpDir, err := ioutil.TempDir("/tmp", "eth2-validator-")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	s, err := NewState(filepath.Join(tmpDir, "my.db"))
	require.NoError(t, err)

	return s
}
