package cmd

import (
	"github.com/mitchellh/cli"
)

// E2ECommand is the command to show the version of the agent
type E2ECommand struct {
	UI cli.Ui
}

// Help implements the cli.Command interface
func (c *E2ECommand) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *E2ECommand) Synopsis() string {
	return ""
}

// Run implements the cli.Command interface
func (c *E2ECommand) Run(args []string) int {

	/*
		fr := &testutil.Framework{
			Config: &testutil.Eth2Config{},
		}

		fr.NewEth1Server()
		fr.NewBeaconNode()

		account := fr.NewAccount()
		fr.NewValidator(account)
	*/

	return 0
}
