package server

import (
	"github.com/umbracle/beacon/internal/beacon"
)

// Config is the parametrizable configuration of the
// validator
type Config struct {
	DepositAddress string
	Endpoint       string
	BeaconConfig   *beacon.ChainConfig
	GrpcAddr       string
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		GrpcAddr: "localhost:4002",
	}
}
