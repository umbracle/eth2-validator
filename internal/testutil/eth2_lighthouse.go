package testutil

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bls"
)

// LighthouseBeacon is a prysm test server
type LighthouseBeacon struct {
	node   *node
	config *beacon.ChainConfig
}

// NewLighthouseBeacon creates a new prysm server
func NewLighthouseBeacon(t *testing.T, e *Eth1Server) *LighthouseBeacon {
	testConfig := &Eth2Spec{
		DepositContract: e.deposit.String(),
	}

	cmd := []string{
		"lighthouse", "beacon_node",
		"--http", "--http-address", "0.0.0.0",
		"--http-port", eth2ApiPort,
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

	node := newNode(t, opts...)
	srv := &LighthouseBeacon{
		node:   node,
		config: testConfig.GetChainConfig(),
	}
	return srv
}

func (b *LighthouseBeacon) IP() string {
	return b.node.IP()
}

func (b *LighthouseBeacon) Type() NodeClient {
	return Lighthouse
}

type LighthouseValidator struct {
	node *node
}

func NewLighthouseValidator(t *testing.T, account *Account, spec *Eth2Spec, beacon Node) *LighthouseValidator {
	pub := account.Bls.PubKey()
	pubStr := "0x" + hex.EncodeToString(pub[:])

	keystore, err := bls.ToKeystore(account.Bls, defWalletPassword)
	assert.NoError(t, err)

	cmd := []string{
		"lighthouse", "vc",
		"--debug-level", "debug",
		"--datadir", "/data/node",
		"--beacon-nodes", fmt.Sprintf("http://%s:%s", beacon.IP(), eth2ApiPort),
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

	node := newNode(t, opts...)

	srv := &LighthouseValidator{
		node: node,
	}
	return srv
}
