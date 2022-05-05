package testutil

import (
	"fmt"
	"net/http"
	"time"

	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bls"
	"github.com/umbracle/eth2-validator/internal/deposit"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/umbracle/ethgo/wallet"
)

type NodeClient string

const (
	Teku       NodeClient = "teku"
	Prysm      NodeClient = "prysm"
	Lighthouse NodeClient = "lighthouse"
)

/*
type Config struct {
	Spec *Eth2Spec
}

type Framework struct {
	t      *testing.T
	Config *Eth2Spec
	eth1   *Eth1Server
}

func (f *Framework) NewEth1Server() *Eth1Server {
	srv := NewEth1Server(f.t)

	f.Config.DepositContract = srv.deposit.String()

	f.eth1 = srv
	return srv
}

func (f *Framework) NewBeaconNode(typ NodeClient) Node {
	if typ == Teku {
		return NewTekuBeacon(f.t, f.eth1)
	} else if typ == Prysm {
		return NewPrysmBeacon(f.t, f.eth1)
	} else if typ == Lighthouse {
		return NewLighthouseBeacon(f.t, f.eth1)
	}
	panic("bad")
}

func (f *Framework) NewValidator(n Node, account *Account) {
	typ := n.Type()
	if typ == Teku {
		NewTekuValidator(f.t, account.Bls, n.IP(), f.eth1)
	} else if typ == Prysm {
		NewPrysmValidator(f.t, account.Bls, n.IP())
	} else if typ == Lighthouse {
		NewLighthouseValidator(f.t, account.Bls, n.IP(), f.eth1)
	}
}
*/

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

// Eth1Server is an eth1x testutil server using go-ethereum
type Eth1Server struct {
	node    *node
	deposit ethgo.Address
	client  *jsonrpc.Client
}

// NewEth1Server creates a new eth1 server with go-ethereum
func NewEth1Server() (*Eth1Server, error) {
	cmd := []string{
		"--dev",
		"--dev.period", "1",
		"--http", "--http.addr", "0.0.0.0",
		"--ws", "--ws.addr", "0.0.0.0",
	}
	opts := []nodeOption{
		WithName("eth1"),
		WithContainer("ethereum/client-go", "v1.9.25"),
		WithCmd(cmd),
		WithRetry(func(n *node) error {
			return testHTTPEndpoint(fmt.Sprintf("http://%s:8545", n.IP()))
		}),
	}

	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	server := &Eth1Server{
		node: node,
	}

	provider, err := jsonrpc.NewClient(server.http())
	if err != nil {
		return nil, err
	}
	server.client = provider

	if err := server.deployDeposit(); err != nil {
		return nil, err
	}
	return server, nil
}

func (e *Eth1Server) http() string {
	return fmt.Sprintf("http://%s:8545", e.node.IP())
}

const (
	defaultGasPrice = 1879048192 // 0x70000000
	defaultGasLimit = 5242880    // 0x500000
)

func (e *Eth1Server) fund(addr ethgo.Address) error {
	nonce, err := e.client.Eth().GetNonce(addr, ethgo.Latest)
	if err != nil {
		return err
	}

	txn := &ethgo.Transaction{
		From:     e.Owner(),
		To:       &addr,
		Nonce:    nonce,
		GasPrice: defaultGasPrice,
		Gas:      defaultGasLimit,
		// fund the account with enoung balance to validate and send the transaction
		Value: ethgo.Ether(deposit.MinGweiAmount + 1),
	}
	hash, err := e.client.Eth().SendTransaction(txn)
	if err != nil {
		return err
	}
	if _, err := e.waitForReceipt(hash); err != nil {
		return err
	}
	return nil
}

func (e *Eth1Server) waitForReceipt(hash ethgo.Hash) (*ethgo.Receipt, error) {
	var count uint64
	for {
		receipt, err := e.client.Eth().GetTransactionReceipt(hash)
		if err != nil {
			if err.Error() != "not found" {
				return nil, err
			}
		}
		if receipt != nil {
			return receipt, nil
		}
		if count > 120 {
			break
		}
		time.Sleep(1 * time.Second)
		count++
	}
	return nil, fmt.Errorf("timeout")
}

// Provider returns the jsonrpc provider
func (e *Eth1Server) Provider() *jsonrpc.Client {
	return e.client
}

// Owner returns the account with balance on go-ethereum
func (e *Eth1Server) Owner() ethgo.Address {
	owner, _ := e.Provider().Eth().Accounts()
	return owner[0]
}

// DeployDeposit deploys the eth2.0 deposit contract
func (e *Eth1Server) deployDeposit() error {
	provider := e.Provider()

	owner, err := provider.Eth().Accounts()
	if err != nil {
		return err
	}

	deployTxn := &ethgo.Transaction{
		Input: deposit.DepositBin(),
		From:  owner[0],
	}
	hash, err := provider.Eth().SendTransaction(deployTxn)
	if err != nil {
		return err
	}
	receipt, err := e.waitForReceipt(hash)
	if err != nil {
		return err
	}
	e.deposit = receipt.ContractAddress
	return nil
}

func (e *Eth1Server) Deposit() ethgo.Address {
	return e.deposit
}

func (e *Eth1Server) GetDepositContract() *deposit.Deposit {
	return deposit.NewDeposit(e.deposit, contract.WithJsonRPC(e.Provider().Eth()))
}

// MakeDeposit deposits the minimum required value to become a validator
func (e *Eth1Server) MakeDeposit(account *Account, config *beacon.ChainConfig) error {
	depositAmount := deposit.MinGweiAmount

	// fund the owner address
	if err := e.fund(account.Ecdsa.Address()); err != nil {
		return err
	}

	data, err := deposit.Input(account.Bls, nil, ethgo.Gwei(depositAmount).Uint64(), config)
	if err != nil {
		return err
	}

	depositC := deposit.NewDeposit(e.deposit, contract.WithSender(account.Ecdsa), contract.WithJsonRPC(e.Provider().Eth()))

	txn, err := depositC.Deposit(data.Pubkey, data.WithdrawalCredentials, data.Signature, data.Root)
	if err != nil {
		return err
	}
	txn.WithOpts(&contract.TxnOpts{Value: ethgo.Ether(depositAmount)})

	if err := txn.Do(); err != nil {
		return err
	}
	if _, err := txn.Wait(); err != nil {
		return err
	}
	return nil
}

func testHTTPEndpoint(endpoint string) error {
	resp, err := http.Post(endpoint, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type Node interface {
	IP() string
	Type() NodeClient
}
