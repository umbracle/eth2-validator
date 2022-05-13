package testutil

import (
	"testing"
)

func TestEth2_Teku_SingleNode(t *testing.T) {
	testSingleNode(t, NewTekuBeacon, NewTekuValidator)
}
