package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/mitchellh/cli"
	"github.com/umbracle/eth2-validator/internal/testutil/proto"
	"google.golang.org/grpc"
)

// E2ENodeCommand is the command to deploy an e2e network
type E2ENodeCommand struct {
	UI cli.Ui

	nodeType      string
	numValidators uint64

	validator  bool
	beaconNode bool
}

// Help implements the cli.Command interface
func (c *E2ENodeCommand) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *E2ENodeCommand) Synopsis() string {
	return ""
}

// Run implements the cli.Command interface
func (c *E2ENodeCommand) Run(args []string) int {
	flags := flag.NewFlagSet("e2e node", flag.ContinueOnError)

	flags.StringVar(&c.nodeType, "node-type", "", "")
	flags.Uint64Var(&c.numValidators, "num-validators", 0, "")
	flags.BoolVar(&c.validator, "validator", false, "")
	flags.BoolVar(&c.beaconNode, "beacon-node", false, "")

	flags.Parse(args)

	conn, err := grpc.Dial("localhost:5555", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	clt := proto.NewE2EServiceClient(conn)

	if c.beaconNode {
		fmt.Println(clt.DeployNode(context.Background(), &proto.DeployNodeRequest{NodeType: c.nodeType}))
	} else if c.validator {
		fmt.Println(clt.DeployValidator(context.Background(), &proto.DeployValidatorRequest{NumValidators: c.numValidators}))
	} else {
		c.UI.Output("this means nothing")
	}
	return 0
}
