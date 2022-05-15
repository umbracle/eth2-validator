package cmd

import (
	"context"
	"fmt"

	"github.com/umbracle/eth2-validator/internal/server/proto"
)

// DutyListCommand is the command to show the version of the agent
type DutyListCommand struct {
	*Meta
}

// Help implements the cli.Command interface
func (c *DutyListCommand) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *DutyListCommand) Synopsis() string {
	return ""
}

// Run implements the cli.Command interface
func (c *DutyListCommand) Run(args []string) int {
	flags := c.FlagSet("duty list")
	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	conn, err := c.Conn()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	resp, err := conn.ListDuties(context.Background(), &proto.ListDutiesRequest{})
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	for _, duty := range resp.Duties {
		fmt.Println("-- elem --")
		fmt.Println(duty)
	}

	return 0
}
