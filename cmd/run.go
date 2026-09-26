package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/melwintjoshy/aliasctl/shell"
)

var runCmd = &cobra.Command{
	Use:   "run <command> [args...]",
	Short: "Run a command inside the AliasCtl environment",

	Args: cobra.MinimumNArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		runner, err := shell.New(shellName)
		if err != nil {
			return err
		}

		env, err := app.LoadEnvironment(configPath)
		if err != nil {
			return fmt.Errorf("failed to load environment: %w", err)
		}

		if printPlan {
			return shell.DescribeRun(cmd.OutOrStdout(), runner, env, args)
		}

		if err := shell.Run(runner, env, args); err != nil {
			return fmt.Errorf("command failed: %w", err)
		}

		return nil
	},
}

func init() {
	// flags after the command belong to it, e.g. "aliasctl run kgp --help"
	runCmd.Flags().SetInterspersed(false)
	addShellFlag(runCmd)
	addPrintFlag(runCmd)

	rootCmd.AddCommand(runCmd)
}
