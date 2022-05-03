package deposit

import (
	"fmt"
	"math/big"

	web3 "github.com/umbracle/go-web3"
	"github.com/umbracle/go-web3/contract"
	"github.com/umbracle/go-web3/jsonrpc"
)

var (
	_ = big.NewInt
)

// Deposit is a solidity contract
type Deposit struct {
	c *contract.Contract
}

// DeployDeposit deploys a new Deposit contract
func DeployDeposit(provider *jsonrpc.Client, from web3.Address, args ...interface{}) *contract.Txn {
	return contract.DeployContract(provider, from, abiDeposit, binDeposit, args...)
}

// NewDeposit creates a new instance of the contract at a specific address
func NewDeposit(addr web3.Address, provider *jsonrpc.Client) *Deposit {
	return &Deposit{c: contract.NewContract(addr, abiDeposit, provider)}
}

// Contract returns the contract object
func (a *Deposit) Contract() *contract.Contract {
	return a.c
}

// calls

// GetDepositCount calls the get_deposit_count method in the solidity contract
func (a *Deposit) GetDepositCount(block ...web3.BlockNumber) (val0 []byte, err error) {
	var out map[string]interface{}
	var ok bool

	out, err = a.c.Call("get_deposit_count", web3.EncodeBlock(block...))
	if err != nil {
		return
	}

	// decode outputs
	val0, ok = out["0"].([]byte)
	if !ok {
		err = fmt.Errorf("failed to encode output at index 0")
		return
	}
	
	return
}

// GetDepositRoot calls the get_deposit_root method in the solidity contract
func (a *Deposit) GetDepositRoot(block ...web3.BlockNumber) (val0 [32]byte, err error) {
	var out map[string]interface{}
	var ok bool

	out, err = a.c.Call("get_deposit_root", web3.EncodeBlock(block...))
	if err != nil {
		return
	}

	// decode outputs
	val0, ok = out["0"].([32]byte)
	if !ok {
		err = fmt.Errorf("failed to encode output at index 0")
		return
	}
	
	return
}

// SupportsInterface calls the supportsInterface method in the solidity contract
func (a *Deposit) SupportsInterface(interfaceId [4]byte, block ...web3.BlockNumber) (val0 bool, err error) {
	var out map[string]interface{}
	var ok bool

	out, err = a.c.Call("supportsInterface", web3.EncodeBlock(block...), interfaceId)
	if err != nil {
		return
	}

	// decode outputs
	val0, ok = out["0"].(bool)
	if !ok {
		err = fmt.Errorf("failed to encode output at index 0")
		return
	}
	
	return
}


// txns

// Deposit sends a deposit transaction in the solidity contract
func (a *Deposit) Deposit(pubkey []byte, withdrawalCredentials []byte, signature []byte, depositDataRoot [32]byte) *contract.Txn {
	return a.c.Txn("deposit", pubkey, withdrawalCredentials, signature, depositDataRoot)
}
