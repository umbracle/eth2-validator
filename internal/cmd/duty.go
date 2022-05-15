package cmd

import (
	"github.com/mitchellh/cli"
)

// DutyCommand is the command to show the version of the agent
type DutyCommand struct {
	UI cli.Ui
}

// Help implements the cli.Command interface
func (c *DutyCommand) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *DutyCommand) Synopsis() string {
	return ""
}

// Run implements the cli.Command interface
func (c *DutyCommand) Run(args []string) int {
	return 0
}
