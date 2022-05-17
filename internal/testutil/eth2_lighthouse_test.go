package testutil

import (
	"testing"
)

func TestEth2_Lighthouse_SingleNode(t *testing.T) {
	t.Skip()

	testSingleNode(t, NewLighthouseBeacon, NewLighthouseValidator)
}
