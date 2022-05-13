package testutil

import (
	"encoding/json"
	"strings"

	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bls"
	"github.com/umbracle/ethgo/keystore"
)

// PrysmBeacon is a prysm test server
type PrysmBeacon struct {
	*node
	config *beacon.ChainConfig
}

// NewPrysmBeacon creates a new prysm server
func NewPrysmBeacon(e *Eth1Server) (*PrysmBeacon, error) {
	testConfig := &Eth2Spec{
		DepositContract: e.deposit.String(),
	}

	cmd := []string{
		"--verbosity", "debug",
		// eth1x
		"--http-web3provider", e.GetAddr(NodePortEth1Http),
		"--deposit-contract", e.deposit.String(),
		"--contract-deployment-block", "0",
		"--chain-id", "1337",
		"--network-id", "1337",
		// these sync fields have to be disabled for single node
		"--min-sync-peers", "0",
		"--disable-sync",
		// do not connect with any peers
		"--no-discovery",
		// grpc endpoint
		"--rpc-host", "0.0.0.0",
		"--rpc-port", `{{ Port "eth2.prysm.grpc" }}`,
		// http gateway for the grpc server and Open spec http server
		"--grpc-gateway-host", "0.0.0.0",
		"--grpc-gateway-port", `{{ Port "eth2.http" }}`,
		// config
		"--chain-config-file", "/data/config.yaml",
		// accept terms and conditions
		"--accept-terms-of-use",
		// use data dir
		"--datadir", "/data/eth2",
		"--e2e-config",
		"--force-clear-db",
	}
	opts := []nodeOption{
		WithName("prysm-beacon"),
		WithContainer("gcr.io/prysmaticlabs/prysm/beacon-chain", "v2.0.6"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", testConfig),
	}

	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	srv := &PrysmBeacon{
		node:   node,
		config: testConfig.GetChainConfig(),
	}
	return srv, nil
}

func (b *PrysmBeacon) Type() NodeClient {
	return Prysm
}

type PrysmValidator struct {
	*node
}

const defWalletPassword = "qwerty"

func NewPrysmValidator(account *Account, spec *Eth2Spec, beacon Node) (*PrysmValidator, error) {
	store := &accountStore{}
	store.AddKey(account.Bls)

	keystore, err := store.ToKeystore(defWalletPassword)
	if err != nil {
		return nil, err
	}

	cmd := []string{
		"--verbosity", "debug",
		// accept terms and conditions
		"--accept-terms-of-use",
		// wallet dir and password
		"--wallet-dir", "/data",
		"--wallet-password-file", "/data/wallet-password.txt",
		// beacon node reference of the GRPC endpoint
		"--beacon-rpc-provider", strings.TrimPrefix(beacon.GetAddr(NodePortPrysmGrpc), "http://"),
	}
	opts := []nodeOption{
		WithName("prysm-validator"),
		WithContainer("gcr.io/prysmaticlabs/prysm/validator", "v2.1.0"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/direct/accounts/all-accounts.keystore.json", keystore),
		WithFile("/data/wallet-password.txt", defWalletPassword),
	}

	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	return &PrysmValidator{node: node}, nil
}

// accountStore is the format used by all managers??
type accountStore struct {
	PrivateKeys [][]byte `json:"private_keys"`
	PublicKeys  [][]byte `json:"public_keys"`
}

func (a *accountStore) AddKey(k *bls.Key) error {
	if a.PrivateKeys == nil {
		a.PrivateKeys = [][]byte{}
	}
	if a.PublicKeys == nil {
		a.PublicKeys = [][]byte{}
	}

	priv, err := k.Prv.Marshal()
	if err != nil {
		return err
	}
	pub := k.Pub.Serialize()

	a.PrivateKeys = append(a.PrivateKeys, priv)
	a.PublicKeys = append(a.PublicKeys, pub)

	return nil
}

func (a *accountStore) ToKeystore(password string) ([]byte, error) {
	raw, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	keystore, err := keystore.EncryptV4(raw, password)
	if err != nil {
		return nil, err
	}
	return keystore, nil
}
