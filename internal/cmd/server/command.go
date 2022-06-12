package server

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/google/gops/agent"
	"github.com/hashicorp/go-hclog"
	"github.com/mitchellh/cli"
	"github.com/umbracle/eth2-validator/internal/server"
	"github.com/umbracle/eth2-validator/internal/server/structs"
	"github.com/umbracle/eth2-validator/internal/testutil"
)

// Command is the command that starts the agent
type Command struct {
	UI     cli.Ui
	client *server.Server
}

// Help implements the cli.Command interface
func (c *Command) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *Command) Synopsis() string {
	return ""
}

// Run implements the cli.Command interface
func (c *Command) Run(args []string) int {
	config, err := c.readConfig(args)
	if err != nil {
		c.UI.Output(fmt.Sprintf("failed to read config: %v", err))
		return 1
	}

	if config.Debug {
		if err := agent.Listen(agent.Options{}); err != nil {
			c.UI.Output(fmt.Sprintf("failed to start gops debugger: %v", err))
			return 1
		}
	}

	clientConfig, err := buildValidatorConfig(config)
	if err != nil {
		c.UI.Output(fmt.Sprintf("failed to build validator config: %v", err))
		return 1
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "beacon",
		Level: hclog.LevelFromString(config.LogLevel),
	})
	client, err := server.NewServer(logger, clientConfig)
	if err != nil {
		c.UI.Output(fmt.Sprintf("failed to start validator: %v", err))
		return 1
	}
	c.client = client

	return c.handleSignals()
}

func (c *Command) handleSignals() int {
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	sig := <-signalCh

	c.UI.Output(fmt.Sprintf("Caught signal: %v", sig))
	c.UI.Output("Gracefully shutting down agent...")

	gracefulCh := make(chan struct{})
	go func() {
		c.client.Stop()
		close(gracefulCh)
	}()

	select {
	case <-signalCh:
		return 1
	case <-time.After(10 * time.Second):
		return 1
	case <-gracefulCh:
		return 0
	}
}

func buildValidatorConfig(c *Config) (*server.Config, error) {
	testConfig := &testutil.Eth2Spec{
		Forks: testutil.Forks{
			Altair: testutil.ForkSpec{
				Epoch:   1,
				Version: structs.Domain{0x2, 0x0, 0x0, 0x0},
			},
		},
	}

	cc := server.DefaultConfig()
	cc.BeaconConfig = testConfig.GetChainConfig()
	cc.Endpoint = c.Endpoint
	cc.PrivKey = c.PrivKey

	return cc, nil
}

func (c *Command) readConfig(args []string) (*Config, error) {
	var configFilePath string

	cliConfig := &Config{}

	flags := flag.NewFlagSet("agent", flag.ContinueOnError)
	flags.Usage = func() { c.UI.Error(c.Help()) }

	flags.StringVar(&configFilePath, "config", "", "")
	flags.StringVar(&cliConfig.LogLevel, "log-level", "", "")
	flags.StringVar(&cliConfig.DataDir, "data-dir", "", "")
	flags.BoolVar(&cliConfig.Debug, "debug", false, "")
	flags.StringVar(&cliConfig.BeaconChain, "beacon-chain", "", "")
	flags.StringVar(&cliConfig.Endpoint, "endpoint", "", "")
	flags.StringArrayVar(&cliConfig.PrivKey, "priv-key", []string{}, "")

	if err := flags.Parse(args); err != nil {
		return nil, err
	}

	config := DefaultConfig()
	if configFilePath != "" {
		configFile, err := loadConfig(configFilePath)
		if err != nil {
			return nil, err
		}
		if err := config.Merge(configFile); err != nil {
			return nil, err
		}
	}
	if err := config.Merge(cliConfig); err != nil {
		return nil, err
	}
	return config, nil
}
