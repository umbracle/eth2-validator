package server

import (
	"io/ioutil"

	"github.com/hashicorp/hcl"
	"github.com/imdario/mergo"
)

// Config is the agent configuration
type Config struct {
	// DepositAddress is the address of the deposit contract in eth1
	DepositAddress string `hcl:"deposit_addr"`

	// Endpoint is the endpoint for eth1
	Endpoint string `hcl:"endpoint"`

	// LogLevel is used to set the log level.
	LogLevel string `hcl:"log_level"`

	// Debug enables debug mode
	Debug bool `hcl:"debug"`

	// DataDir is the store data dir
	DataDir string `hcl:"data_dir"`

	// BeaconChainConfig is the reference to the beacon chain
	BeaconChain string `hcl:"beacon_chain"`

	// Keymanager is the configuration for the keymanager
	Keymanager map[string]map[string]interface{} `hcl:"keymanager"`
}

// DefaultConfig returns the default configuration for the agent
func DefaultConfig() *Config {
	return &Config{
		LogLevel: "INFO",
	}
}

func loadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	if err := hcl.Decode(c, string(data)); err != nil {
		return nil, err
	}
	return c, nil
}

// Merge merges two configurations
func (c *Config) Merge(c1 ...*Config) error {
	for _, i := range c1 {
		if err := mergo.Merge(c, *i, mergo.WithOverride); err != nil {
			return err
		}
	}
	return nil
}
