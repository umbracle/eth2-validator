package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEth2_Teku_Single(t *testing.T) {
	eth1 := NewEth1Server(t)
	account := NewAccount()

	spec := &Eth2Spec{
		DepositContract: eth1.deposit.String(),
	}

	err := eth1.MakeDeposit(account, spec.GetChainConfig())
	assert.NoError(t, err)

	beacon := NewTekuBeacon(t, eth1)
	NewTekuValidator(t, account, spec, beacon)
}
