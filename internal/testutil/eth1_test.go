package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/ethgo"
)

func TestEth1_Deposit(t *testing.T) {
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

func TestEth1_Multiple(t *testing.T) {
	// test that multiple eth1 nodes are deployed and
	// get assigned a different port
	srv1, err := NewEth1Server()
	assert.NoError(t, err)

	srv2, err := NewEth1Server()
	assert.NoError(t, err)

	addr1 := srv1.node.GetAddr(NodePortEth1Http)
	addr2 := srv2.node.GetAddr(NodePortEth1Http)
	assert.NotEqual(t, addr1, addr2)
}
