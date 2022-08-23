package cmd

import (
	"context"
	"io/ioutil"
	"strconv"

	flag "github.com/spf13/pflag"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/slashing"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ValidatorDutiesCommand is the command to show the version of the agent
type ValidatorDutiesCommand struct {
	*Meta

	export bool
}

// Help implements the cli.Command interface
func (c *ValidatorDutiesCommand) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *ValidatorDutiesCommand) Synopsis() string {
	return ""
}

func (c *ValidatorDutiesCommand) Flags() *flag.FlagSet {
	flags := c.FlagSet("validator duties")

	flags.BoolVar(&c.export, "export", false, "")

	return flags
}

// Run implements the cli.Command interface
func (c *ValidatorDutiesCommand) Run(args []string) int {
	flags := c.Flags()
	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	args = flags.Args()
	if len(args) != 1 {
		c.UI.Error("validator not found in args")
		return 1
	}

	validatorId, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		c.UI.Error("cannot convert arg to int")
		return 1
	}

	conn, err := c.Conn()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	resp, err := conn.ListDuties(context.Background(), &proto.ListDutiesRequest{ValidatorId: validatorId})
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	if c.export {
		genesis, err := conn.GetGenesis(context.Background(), &emptypb.Empty{})
		if err != nil {
			c.UI.Error(err.Error())
			return 1
		}
		input := &slashing.Input{
			Duties:               resp.Duties,
			GenesisValidatorRoot: genesis.Root,
		}
		res, err := slashing.Format(input)
		if err != nil {
			c.UI.Error(err.Error())
			return 1
		}
		if err := ioutil.WriteFile("output.json", res, 0755); err != nil {
			c.UI.Error(err.Error())
			return 1
		}
		c.UI.Output("Slashing info exported to output.json...")
	} else {
		c.UI.Output(formatDuties(resp.Duties))
	}
	return 0
}
