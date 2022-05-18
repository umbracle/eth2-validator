package testutil

import (
	"testing"
)

func TestEth2_Prysm_SingleNode(t *testing.T) {
	t.Skip("not ready")

	testSingleNode(t, NewPrysmBeacon, NewPrysmValidator)
}
