package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mitchellh/cli"
	"github.com/umbracle/eth2-validator/internal/testutil"
)

// E2EServerCommand is the command to deploy an e2e network
type E2EServerCommand struct {
	UI cli.Ui

	server       *testutil.Server
	name         string
	nodeLogLevel string

	genesisValidatorCount uint64
	genesisTime           string
}

// Help implements the cli.Command interface
func (c *E2EServerCommand) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *E2EServerCommand) Synopsis() string {
	return ""
}

// Run implements the cli.Command interface
func (c *E2EServerCommand) Run(args []string) int {

	flags := flag.NewFlagSet("e2e server", flag.ContinueOnError)
	flags.StringVar(&c.name, "name", "random", "")
	flags.StringVar(&c.nodeLogLevel, "node-log-level", "info", "")
	flags.Uint64Var(&c.genesisValidatorCount, "genesis-validator-count", 10, "")
	flags.StringVar(&c.genesisTime, "genesis-time", "", "")
	flags.Parse(args)

	config := testutil.DefaultConfig()
	config.Name = c.name
	config.LogLevel = c.nodeLogLevel
	config.Spec.GenesisValidatorCount = int(c.genesisValidatorCount)

	config.Spec.MinGenesisTime = int(time.Now().Unix())
	if c.genesisTime != "" {
		duration, err := time.ParseDuration(c.genesisTime)
		if err != nil {
			c.UI.Error(err.Error())
			return 1
		}
		config.Spec.MinGenesisTime += int(duration.Seconds())
	}

	fmt.Println(config.Spec.MinGenesisTime)

	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "beacon",
		Level: hclog.LevelFromString("info"),
	})
	srv, err := testutil.NewServer(logger, config)
	if err != nil {
		panic(err)
	}

	c.server = srv
	return c.handleSignals()
}

func (c *E2EServerCommand) handleSignals() int {
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	sig := <-signalCh

	c.UI.Output(fmt.Sprintf("Caught signal: %v", sig))
	c.UI.Output("Gracefully shutting down agent...")

	gracefulCh := make(chan struct{})
	go func() {
		c.server.Stop()
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
