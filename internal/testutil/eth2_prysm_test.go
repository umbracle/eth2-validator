package testutil

import (
	"testing"
)

func TestEth2_Prysm_SingleNode(t *testing.T) {
	t.Skip()

	testSingleNode(t, NewPrysmBeacon, NewPrysmValidator)
}
