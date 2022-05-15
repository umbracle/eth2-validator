package testutil

import (
	"testing"
)

func TestEth2_Teku_SingleNode(t *testing.T) {
	t.Skip()

	testSingleNode(t, NewTekuBeacon, NewTekuValidator)
}
