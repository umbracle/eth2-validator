package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mitchellh/cli"
	"github.com/umbracle/eth2-validator/internal/testutil"
	"github.com/umbracle/eth2-validator/internal/testutil/proto"
	"google.golang.org/grpc"
)

// E2EServerCommand is the command to deploy an e2e network
type E2EServerCommand struct {
	UI cli.Ui

	server *testutil.Server
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
	if len(args) != 0 {
		conn, err := grpc.Dial("localhost:5555", grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		clt := proto.NewE2EServiceClient(conn)

		if args[0] == "beacon" {
			fmt.Println(clt.DeployNode(context.Background(), &proto.DeployNodeRequest{}))
		} else if args[0] == "validator" {
			fmt.Println(clt.DeployValidator(context.Background(), &proto.DeployValidatorRequest{}))
		} else {
			c.UI.Output("this means nothing")
		}

		return 0
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "beacon",
		Level: hclog.LevelFromString("info"),
	})
	srv, err := testutil.NewServer(logger)
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
