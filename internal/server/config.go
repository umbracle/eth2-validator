package server

// Config is the parametrizable configuration of the
// validator
type Config struct {
	DepositAddress        string
	Endpoint              string
	GrpcAddr              string
	PrivKey               []string
	TelemetryOLTPExporter string
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		GrpcAddr: "localhost:4002",
	}
}
