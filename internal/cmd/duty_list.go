package cmd

import (
	"context"
	"fmt"
	"sort"

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

	resp, err := conn.ListDuties(context.Background(), &proto.ListDutiesRequest{ValidatorId: -1})
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	c.UI.Output(formatDuties(resp.Duties))
	return 0
}

func formatDuties(duties []*proto.Duty) string {
	sort.Slice(duties, func(i, j int) bool {
		d0, d1 := duties[i], duties[j]
		if d0.PubKey < d1.PubKey {
			return true
		}
		if d0.Slot < d1.Slot {
			return true
		}
		return false
	})

	if len(duties) == 0 {
		return "No duties found"
	}

	rows := make([]string, len(duties)+1)
	rows[0] = "ID|Validator|Type|Slot|State"
	for i, d := range duties {
		rows[i+1] = fmt.Sprintf("%s|%d|%s|%d|%s",
			d.Id[:8],
			d.ValidatorIndex,
			d.Type(),
			d.Slot,
			d.State)
	}
	return formatList(rows)
}
