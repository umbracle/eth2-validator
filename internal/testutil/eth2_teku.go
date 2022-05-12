package testutil

import (
	"fmt"

	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bls"
)

// TekuBeacon is a teku test server
type TekuBeacon struct {
	node   *node
	config *beacon.ChainConfig
}

// NewTekuBeacon creates a new teku server
func NewTekuBeacon(e *Eth1Server) (*TekuBeacon, error) {
	testConfig := &Eth2Spec{
		DepositContract: e.deposit.String(),
	}

	cmd := []string{
		// eth1x
		"--eth1-endpoint", e.http(),
		// eth1x deposit contract
		"--eth1-deposit-contract-address", e.deposit.String(),
		// run only beacon node
		"--rest-api-enabled",
		// allow requests from anyone
		"--rest-api-host-allowlist", "*",
		// config
		"--network", "/data/config.yaml",
		// port
		"--rest-api-port", eth2ApiPort,
		// debug log
		"--logging", "debug",
	}
	opts := []nodeOption{
		WithName("teku-beacon"),
		WithContainer("consensys/teku", "22.4.0"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", testConfig),
	}

	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	srv := &TekuBeacon{
		node:   node,
		config: testConfig.GetChainConfig(),
	}
	return srv, nil
}

func (b *TekuBeacon) IP() string {
	return b.node.IP()
}

func (b *TekuBeacon) Stop() {

}

func (b *TekuBeacon) Type() NodeClient {
	return Teku
}

type TekuValidator struct {
	node *node
}

func NewTekuValidator(account *Account, spec *Eth2Spec, beacon Node) (*TekuValidator, error) {
	keystore, err := bls.ToKeystore(account.Bls, defWalletPassword)
	if err != nil {
		return nil, err
	}

	cmd := []string{
		"vc",
		// beacon api
		"--beacon-node-api-endpoint", fmt.Sprintf("http://%s:%s", beacon.IP(), eth2ApiPort),
		// data
		"--data-path", "/data",
		// eth1x deposit contract (required for custom networks)
		"--eth1-deposit-contract-address", spec.DepositContract,
		// config
		"--network", "/data/config.yaml",
		// keys
		"--validator-keys", "/data/wallet/wallet.json:/data/wallet/wallet.txt",
	}
	opts := []nodeOption{
		WithName("teku-validator"),
		WithContainer("consensys/teku", "22.4.0"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", spec),
		WithFile("/data/wallet/wallet.json", keystore),
		WithFile("/data/wallet/wallet.txt", defWalletPassword),
	}

	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	return &TekuValidator{node: node}, nil
}

func (v *TekuValidator) IP() string {
	return v.node.IP()
}
