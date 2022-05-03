package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/go-web3"
)

func TestEth1(t *testing.T) {
	eth1 := NewEth1Server(t)
	account := NewAccount()

	chainConfig := (&Eth2Spec{}).GetChainConfig()
	err := eth1.MakeDeposit(account, chainConfig)
	assert.NoError(t, err)

	contract := eth1.GetDepositContract()
	count, err := contract.GetDepositCount(web3.Latest)
	assert.NoError(t, err)
	assert.Equal(t, int(count[0]), 1)
}
