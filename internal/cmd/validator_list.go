package cmd

import (
	"context"
	"fmt"

	"github.com/umbracle/eth2-validator/internal/server/proto"
)

// ValidatorsList is the command to show the version of the agent
type ValidatorsList struct {
	*Meta
}

// Help implements the cli.Command interface
func (c *ValidatorsList) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *ValidatorsList) Synopsis() string {
	return ""
}

// Run implements the cli.Command interface
func (c *ValidatorsList) Run(args []string) int {
	flags := c.FlagSet("validator list")
	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	conn, err := c.Conn()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	resp, err := conn.ValidatorList(context.Background(), &proto.ValidatorListRequest{})
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output(formatValidators(resp.Validators))
	return 0
}

func formatValidators(vals []*proto.Validator) string {
	if len(vals) == 0 {
		return "No validators found"
	}

	// filter only by active validators
	activeVals := []*proto.Validator{}
	for _, val := range vals {
		if val.Metadata != nil {
			activeVals = append(activeVals, val)
		}
	}

	rows := make([]string, len(vals)+1)
	rows[0] = "Validator id|Activation epoch"
	for i, d := range activeVals {
		rows[i+1] = fmt.Sprintf("%d|%d",
			d.Metadata.Index,
			d.Metadata.ActivationEpoch)
	}
	return formatList(rows)
}
