package cmd

import (
	"context"
	"strconv"

	"github.com/umbracle/eth2-validator/internal/server/proto"
)

// ValidatorDutiesCommand is the command to show the version of the agent
type ValidatorDutiesCommand struct {
	*Meta
}

// Help implements the cli.Command interface
func (c *ValidatorDutiesCommand) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *ValidatorDutiesCommand) Synopsis() string {
	return ""
}

// Run implements the cli.Command interface
func (c *ValidatorDutiesCommand) Run(args []string) int {
	flags := c.FlagSet("validator duties")
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

	c.UI.Output(formatDuties(resp.Duties))
	return 0
}
