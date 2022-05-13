package testutil

import (
	"encoding/hex"

	"github.com/umbracle/eth2-validator/internal/bls"
)

// LighthouseBeacon is a prysm test server
type LighthouseBeacon struct {
	*node
}

// NewLighthouseBeacon creates a new prysm server
func NewLighthouseBeacon(config *BeaconConfig) (*LighthouseBeacon, error) {
	cmd := []string{
		"lighthouse", "beacon_node",
		"--http", "--http-address", "0.0.0.0",
		"--http-port", `{{ Port "eth2.http" }}`,
		"--eth1-endpoints", config.Eth1.GetAddr(NodePortEth1Http),
		"--testnet-dir", "/data",
		"--http-allow-sync-stalled",
	}
	opts := []nodeOption{
		WithName("lighthouse-beacon"),
		WithContainer("sigp/lighthouse", "v2.2.1"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", config.Spec),
		WithFile("/data/deploy_block.txt", "0"),
	}

	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	srv := &LighthouseBeacon{
		node: node,
	}
	return srv, nil
}

type LighthouseValidator struct {
	*node
}

func NewLighthouseValidator(config *ValidatorConfig) (*LighthouseValidator, error) {
	cmd := []string{
		"lighthouse", "vc",
		"--debug-level", "debug",
		"--datadir", "/data/node",
		"--beacon-nodes", config.Beacon.GetAddr(NodePortHttp),
		"--testnet-dir", "/data",
		"--init-slashing-protection",
	}
	opts := []nodeOption{
		WithName("lighthouse-validator"),
		WithContainer("sigp/lighthouse", "v2.2.1"),
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

	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	srv := &LighthouseValidator{
		node: node,
	}
	return srv, nil
}
