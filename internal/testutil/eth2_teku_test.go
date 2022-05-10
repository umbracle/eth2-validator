package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/eth2-validator/internal/beacon"
)

func TestEth2_Teku_SingleNode(t *testing.T) {
	t.Skip()

	eth1, err := NewEth1Server()
	assert.NoError(t, err)

	account := NewAccount()

	spec := &Eth2Spec{
		DepositContract: eth1.deposit.String(),
	}

	err = eth1.MakeDeposit(account, spec.GetChainConfig())
	assert.NoError(t, err)

	b, err := NewTekuBeacon(eth1)
	assert.NoError(t, err)

	NewTekuValidator(account, spec, b)

	api := beacon.NewHttpAPI(b.GetAddr(NodePortHttp))

	assert.Eventually(t, func() bool {
		syncing, err := api.Syncing()
		if err != nil {
			return false
		}
		if syncing.IsSyncing {
			return false
		}
		if syncing.HeadSlot < 2 {
			return false
		}
		return true
	}, 2*time.Minute, 10*time.Second)
}
