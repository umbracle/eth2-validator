package testutil

import (
	"fmt"

	"github.com/umbracle/eth2-validator/internal/bls"
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
