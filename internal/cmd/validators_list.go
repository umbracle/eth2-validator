package cmd

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
	return 0
}
