package testutil

import (
	"encoding/hex"

	"github.com/umbracle/eth2-validator/internal/bls"
	"github.com/umbracle/eth2-validator/internal/testutil/proto"
)

// LighthouseBeacon is a prysm test server
type LighthouseBeacon struct {
	*node
}

// NewLighthouseBeacon creates a new prysm server
func NewLighthouseBeacon(config *BeaconConfig) ([]nodeOption, error) {
	cmd := []string{
		"lighthouse", "beacon_node",
		"--http", "--http-address", "0.0.0.0",
		"--http-port", `{{ Port "eth2.http" }}`,
		"--eth1-endpoints", config.Eth1,
		"--testnet-dir", "/data",
		"--http-allow-sync-stalled",
		"--debug-level", "trace",
		"--subscribe-all-subnets",
		"--staking",
		"--port", `{{ Port "eth2.p2p" }}`,
		"--enr-address", "127.0.0.1",
		"--enr-udp-port", `{{ Port "eth2.p2p" }}`,
		"--enr-tcp-port", `{{ Port "eth2.p2p" }}`,
		// required to allow discovery in private networks
		"--disable-packet-filter",
		"--enable-private-discovery",
	}
	opts := []nodeOption{
		WithNodeClient(proto.NodeClient_Lighthouse),
		WithNodeType(proto.NodeType_Beacon),
		WithContainer("sigp/lighthouse"),
		WithTag("v2.2.1"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", config.Spec),
		WithFile("/data/deploy_block.txt", "0"),
	}
	if config.Bootnode != "" {
		opts = append(opts, WithFile("/data/boot_enr.yaml", "- "+config.Bootnode+"\n"))
	}
	return opts, nil

	/*
		node, err := newNode(opts...)
		if err != nil {
			return nil, err
		}
		srv := &LighthouseBeacon{
			node: node,
		}
		return srv, nil
	*/
}

type LighthouseValidator struct {
	*node
}

func NewLighthouseValidator(config *ValidatorConfig) ([]nodeOption, error) {
	cmd := []string{
		"lighthouse", "vc",
		"--debug-level", "debug",
		"--datadir", "/data/node",
		"--beacon-nodes", config.Beacon.GetAddr(NodePortHttp),
		"--testnet-dir", "/data",
		"--init-slashing-protection",
	}
	opts := []nodeOption{
		WithNodeClient(proto.NodeClient_Lighthouse),
		WithNodeType(proto.NodeType_Validator),
		WithContainer("sigp/lighthouse"),
		WithTag("v2.2.1"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", config.Spec),
		WithFile("/data/deploy_block.txt", "0"),
	}

	// append validators
	for _, acct := range config.Accounts {
		pub := acct.Bls.PubKey()
		pubStr := "0x" + hex.EncodeToString(pub[:])

		keystore, err := bls.ToKeystore(acct.Bls, defWalletPassword)
		if err != nil {
			return nil, err
		}

		opts = append(opts, []nodeOption{
			WithFile("/data/node/validators/"+pubStr+"/voting-keystore.json", keystore),
			WithFile("/data/node/secrets/"+pubStr, defWalletPassword),
		}...)
	}
	return opts, nil

	/*
		node, err := newNode(opts...)
		if err != nil {
			return nil, err
		}
		srv := &LighthouseValidator{
			node: node,
		}
		return srv, nil
	*/
}
