package testutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bls"
	"github.com/umbracle/eth2-validator/internal/server/structs"
	"github.com/umbracle/ethgo/wallet"
)

type ValidatorConfig struct {
	Spec     *Eth2Spec
	Accounts []*Account
	Beacon   Node
}

type BeaconConfig struct {
	Spec *Eth2Spec
	Eth1 Node
}

type Account struct {
	Bls   *bls.Key
	Ecdsa *wallet.Key
}

func NewAccounts(num int) []*Account {
	accts := []*Account{}
	for i := 0; i < num; i++ {
		accts = append(accts, NewAccount())
	}
	return accts
}

func NewAccount() *Account {
	key, err := wallet.GenerateKey()
	if err != nil {
		panic(fmt.Errorf("BUG: failed to generate key %v", err))
	}
	account := &Account{
		Bls:   bls.NewRandomKey(),
		Ecdsa: key,
	}
	return account
}

// CreateBeacon is a factory method to create beacon nodes
type CreateBeacon func(cfg *BeaconConfig) (Node, error)

// CreateValidator is a factory method to create validator nodes
type CreateValidator func(cfg *ValidatorConfig) (Node, error)

func testSingleNode(t *testing.T, beaconFn CreateBeacon, validatorFn CreateValidator) {
	eth1, err := NewEth1Server()
	require.NoError(t, err)

	spec := &Eth2Spec{
		DepositContract: eth1.deposit.String(),
		Forks: Forks{
			Altair: ForkSpec{
				Epoch:   2,
				Version: structs.Domain{0x2, 0x0, 0x0, 0x0},
			},
		},
	}

	accounts := NewAccounts(1)

	err = eth1.MakeDeposits(accounts, spec.GetChainConfig())
	require.NoError(t, err)

	bCfg := &BeaconConfig{
		Spec: spec,
		Eth1: eth1.node,
	}
	b, err := beaconFn(bCfg)
	require.NoError(t, err)

	vCfg := &ValidatorConfig{
		Accounts: accounts,
		Spec:     spec,
		Beacon:   b,
	}
	validatorFn(vCfg)

	api := beacon.NewHttpAPI(b.GetAddr(NodePortHttp))

	require.Eventually(t, func() bool {
		syncing, err := api.Syncing()
		if err != nil {
			return false
		}
		if syncing.IsSyncing {
			return false
		}
		if syncing.HeadSlot <= uint64(spec.SlotsPerEpoch) {
			// wait at least for one epoch
			return false
		}
		return true
	}, 2*time.Minute, 10*time.Second)
}
