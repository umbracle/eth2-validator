package testutil

import (
	"fmt"

	"github.com/umbracle/eth2-validator/internal/bls"
)

// TekuBeacon is a teku test server
type TekuBeacon struct {
	*node
}

// NewTekuBeacon creates a new teku server
func NewTekuBeacon(config *BeaconConfig) (Node, error) {
	cmd := []string{
		// eth1x
		"--eth1-endpoint", config.Eth1.GetAddr(NodePortEth1Http),
		// eth1x deposit contract
		"--eth1-deposit-contract-address", config.Spec.DepositContract,
		// run only beacon node
		"--rest-api-enabled",
		// allow requests from anyone
		"--rest-api-host-allowlist", "*",
		// config
		"--network", "/data/config.yaml",
		// port
		"--rest-api-port", `{{ Port "eth2.http" }}`,
		// debug log
		"--logging", "debug",
	}
	opts := []nodeOption{
		WithName("teku-beacon"),
		WithNodeType(Teku),
		WithContainer("consensys/teku", "22.4.0"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", config.Spec),
		WithUser("0:0"),
	}

	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	srv := &TekuBeacon{
		node: node,
	}
	return srv, nil
}

type TekuValidator struct {
	*node
}

func NewTekuValidator(config *ValidatorConfig) (Node, error) {
	cmd := []string{
		"vc",
		// beacon api
		"--beacon-node-api-endpoint", config.Beacon.GetAddr(NodePortHttp),
		// data
		"--data-path", "/data",
		// eth1x deposit contract (required for custom networks)
		"--eth1-deposit-contract-address", config.Spec.DepositContract,
		// config
		"--network", "/data/config.yaml",
		// keys
		"--validator-keys", "/data/keys:/data/pass",
	}
	opts := []nodeOption{
		WithName("teku-validator"),
		WithNodeType(Teku),
		WithContainer("consensys/teku", "22.4.0"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", config.Spec),
	}

	for indx, acct := range config.Accounts {
		keystore, err := bls.ToKeystore(acct.Bls, defWalletPassword)
		if err != nil {
			return nil, err
		}

		name := fmt.Sprintf("account_%d", indx)
		opts = append(opts, []nodeOption{
			WithFile("/data/keys/"+name+".json", keystore),
			WithFile("/data/pass/"+name+".txt", defWalletPassword),
		}...)
	}

	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	return &TekuValidator{node: node}, nil
}
