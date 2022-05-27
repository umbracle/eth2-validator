package testutil

import (
	"testing"
)

func TestEth2_Prysm_SingleNode(t *testing.T) {
	testSingleNode(t, NewPrysmBeacon, NewPrysmValidator)
}
