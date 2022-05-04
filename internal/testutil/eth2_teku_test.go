package testutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/eth2-validator/internal/beacon"
)

func TestEth2_Teku_SingleNode(t *testing.T) {
	eth1 := NewEth1Server(t)
	account := NewAccount()

	spec := &Eth2Spec{
		DepositContract: eth1.deposit.String(),
	}

	err := eth1.MakeDeposit(account, spec.GetChainConfig())
	assert.NoError(t, err)

	b := NewTekuBeacon(t, eth1)
	v := NewTekuValidator(t, account, spec, b)

	api := beacon.NewHttpAPI(fmt.Sprintf("http://%s:5050", b.IP()))

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

	fmt.Println("-- b --")
	fmt.Println(b.node.GetLogs())

	fmt.Println("-- v --")
	fmt.Println(v.node.GetLogs())
}
