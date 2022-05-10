package testutil

import (
	"encoding/hex"

	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bls"
)

// LighthouseBeacon is a prysm test server
type LighthouseBeacon struct {
	*node
	config *beacon.ChainConfig
}

// NewLighthouseBeacon creates a new prysm server
func NewLighthouseBeacon(e *Eth1Server) (*LighthouseBeacon, error) {
	testConfig := &Eth2Spec{
		DepositContract: e.deposit.String(),
	}

	cmd := []string{
		"lighthouse", "beacon_node",
		"--http", "--http-address", "0.0.0.0",
		"--http-port", `{{ Port "eth2.http" }}`,
		"--eth1-endpoints", e.http(),
		"--testnet-dir", "/data",
		"--http-allow-sync-stalled",
	}
	opts := []nodeOption{
		WithName("lighthouse-beacon"),
		WithContainer("sigp/lighthouse", "v2.2.1"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", testConfig),
		WithFile("/data/deploy_block.txt", "0"),
	}

	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	srv := &LighthouseBeacon{
		node:   node,
		config: testConfig.GetChainConfig(),
	}
	return srv, nil
}

func (b *LighthouseBeacon) Type() NodeClient {
	return Lighthouse
}

type LighthouseValidator struct {
	node *node
}

func NewLighthouseValidator(account *Account, spec *Eth2Spec, beacon Node) (*LighthouseValidator, error) {
	pub := account.Bls.PubKey()
	pubStr := "0x" + hex.EncodeToString(pub[:])

	keystore, err := bls.ToKeystore(account.Bls, defWalletPassword)
	if err != nil {
		return nil, err
	}

	cmd := []string{
		"lighthouse", "vc",
		"--debug-level", "debug",
		"--datadir", "/data/node",
		"--beacon-nodes", beacon.GetAddr(NodePortHttp),
		"--testnet-dir", "/data",
		"--init-slashing-protection",
	}
	opts := []nodeOption{
		WithName("lighthouse-validator"),
		WithContainer("sigp/lighthouse", "v2.2.1"),
		WithCmd(cmd),
		WithMount("/data"),
		WithFile("/data/config.yaml", spec),
		WithFile("/data/deploy_block.txt", "0"),
		WithFile("/data/node/validators/"+pubStr+"/voting-keystore.json", keystore),
		WithFile("/data/node/secrets/"+pubStr, defWalletPassword),
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
