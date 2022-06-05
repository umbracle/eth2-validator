package testutil

import (
	"fmt"

	"github.com/umbracle/eth2-validator/internal/bls"
	"github.com/umbracle/eth2-validator/internal/testutil/proto"
)

// TekuBeacon is a teku test server
type TekuBeacon struct {
	*node
}

// NewTekuBeacon creates a new teku server
func NewTekuBeacon(config *BeaconConfig) ([]nodeOption, error) {
	cmd := []string{
		// debug log
		"--logging", "debug",
		// eth1x
		"--eth1-endpoint", config.Eth1,
		// run only beacon node
		"--rest-api-enabled",
		// config
		"--network", "/data/config.yaml",
		// port
		"--rest-api-port", `{{ Port "eth2.http" }}`,
		// logs
		"--log-file", "/data/logs.txt",
		"--p2p-advertised-ip", "0.0.0.0",
		"--p2p-port", `{{ Port "eth2.p2p" }}`,
	}
	if config.Bootnode != "" {
		cmd = append(cmd, "--p2p-discovery-bootnodes", config.Bootnode)
	}

	opts := []nodeOption{
		WithNodeClient(proto.NodeClient_Teku),
		WithNodeType(proto.NodeType_Beacon),
		WithContainer("consensys/teku"),
		WithTag("22.4.0"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", config.Spec),
		WithUser("0:0"),
	}
	return opts, nil

	/*
		node, err := newNode(opts...)
		if err != nil {
			return nil, err
		}
		srv := &TekuBeacon{
			node: node,
		}
		return srv, nil
	*/
}

type TekuValidator struct {
	*node
}

func NewTekuValidator(config *ValidatorConfig) ([]nodeOption, error) {
	cmd := []string{
		"vc",
		// beacon api
		"--beacon-node-api-endpoint", config.Beacon.GetAddr(NodePortHttp),
		// data
		"--data-path", "/data",
		// config
		"--network", "/data/config.yaml",
		// keys
		"--validator-keys", "/data/keys:/data/pass",
	}
	opts := []nodeOption{
		WithNodeClient(proto.NodeClient_Teku),
		WithNodeType(proto.NodeType_Validator),
		WithContainer("consensys/teku"),
		WithTag("22.4.0"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", config.Spec),
		WithUser("0:0"),
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
	return opts, nil

	/*
		node, err := newNode(opts...)
		if err != nil {
			return nil, err
		}
		return &TekuValidator{node: node}, nil
	*/
}
