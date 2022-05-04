package cmd

import (
	"github.com/mitchellh/cli"
	"github.com/umbracle/eth2-validator/internal/testutil"
)

// E2EDeployCommand is the command to deploy an e2e network
type E2EDeployCommand struct {
	UI cli.Ui
}

// Help implements the cli.Command interface
func (c *E2EDeployCommand) Help() string {
	return ""
}

// Synopsis implements the cli.Command interface
func (c *E2EDeployCommand) Synopsis() string {
	return ""
}

// Run implements the cli.Command interface
func (c *E2EDeployCommand) Run(args []string) int {
	c.UI.Output("=> Provision Eth1 server")
	eth1, err := testutil.NewEth1Server()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	account := testutil.NewAccount()
	spec := &testutil.Eth2Spec{
		DepositContract: eth1.Deposit().String(),
	}

	c.UI.Output("=> Deploy deposit")
	if err = eth1.MakeDeposit(account, spec.GetChainConfig()); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output("=> Provision beacon node")
	b, err := testutil.NewLighthouseBeacon(eth1)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output("=> Provision validator")
	testutil.NewLighthouseValidator(account, spec, b)

	c.UI.Output("E2E setup done")
	return 0
}
