package cmd

import (
	"github.com/mitchellh/cli"
)

// Validators is the command to show the version of the agent
type Validators struct {
	UI cli.Ui
}

// Help implements the cli.Command interface
func (c *Validators) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *Validators) Synopsis() string {
	return ""
}

// Run implements the cli.Command interface
func (c *Validators) Run(args []string) int {
	return 0
}
