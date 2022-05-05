package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/ethgo"
)

func TestEth1(t *testing.T) {
	eth1, err := NewEth1Server()
	assert.NoError(t, err)

	code, err := eth1.Provider().Eth().GetCode(eth1.deposit, ethgo.Latest)
	assert.NoError(t, err)
	assert.NotEqual(t, code, "0x")

	account := NewAccount()

	chainConfig := (&Eth2Spec{}).GetChainConfig()
	err = eth1.MakeDeposit(account, chainConfig)
	assert.NoError(t, err)

	contract := eth1.GetDepositContract()
	count, err := contract.GetDepositCount(ethgo.Latest)
	assert.NoError(t, err)
	assert.Equal(t, int(count[0]), 1)
}
