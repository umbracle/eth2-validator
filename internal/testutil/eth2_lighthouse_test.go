package testutil

import (
	"testing"
)

func TestEth2_Lighthouse_SingleNode(t *testing.T) {
	testSingleNode(t, NewLighthouseBeacon, NewLighthouseValidator)
}
