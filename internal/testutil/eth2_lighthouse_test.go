package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umbracle/eth2-validator/internal/beacon"
)

func TestEth2_Lighthouse_SingleNode(t *testing.T) {
	eth1, err := NewEth1Server()
	assert.NoError(t, err)

	account := NewAccount()

	spec := &Eth2Spec{
		DepositContract: eth1.deposit.String(),
	}

	err = eth1.MakeDeposit(account, spec.GetChainConfig())
	assert.NoError(t, err)

	b, err := NewLighthouseBeacon(eth1)
	assert.NoError(t, err)

	NewLighthouseValidator(account, spec, b)

	api := beacon.NewHttpAPI(b.GetAddr(NodePortHttp))

	require.Eventually(t, func() bool {
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
