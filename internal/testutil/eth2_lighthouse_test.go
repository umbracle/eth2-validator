package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEth2_Lighthouse_Simple(t *testing.T) {
	eth1 := NewEth1Server(t)
	account := NewAccount()

	spec := &Eth2Spec{
		DepositContract: eth1.deposit.String(),
	}

	err := eth1.MakeDeposit(account, spec.GetChainConfig())
	assert.NoError(t, err)

	beacon := NewLighthouseBeacon(t, eth1)
	NewLighthouseValidator(t, account, spec, beacon)
}
