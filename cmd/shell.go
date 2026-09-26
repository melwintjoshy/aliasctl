package cmd

import (
	"fmt"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/melwintjoshy/aliasctl/shell"
	"github.com/spf13/cobra"
)

var (
	shellName string
	printPlan bool
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start a temporary command environment",
	RunE: func(cmd *cobra.Command, args []string) error {

		runner, err := shell.New(shellName)
		if err != nil {
			return err
		}

		env, err := app.LoadEnvironment(configPath)
		if err != nil {
			return fmt.Errorf("failed to load environment: %w", err)
		}

		activateTools(cmd.ErrOrStderr(), env)

		if printPlan {
			return shell.DescribeStart(cmd.OutOrStdout(), runner, env)
		}

		warnToolProblems(cmd.ErrOrStderr(), env)

		if err := shell.Start(runner, env); err != nil {
			return fmt.Errorf("shell error: %w", err)
		}

		return nil
	},
}

func addShellFlag(command *cobra.Command) {
	command.Flags().StringVar(
		&shellName,
		"shell",
		shell.DefaultShell(),
		"Shell to use: bash, zsh or fish (default from $SHELL)",
	)
}

// prints what would run instead of running it, for debugging and bug reports
func addPrintFlag(command *cobra.Command) {
	command.Flags().BoolVar(
		&printPlan,
		"print",
		false,
		"Print the generated command and files instead of running them",
	)
}

func init() {
	addShellFlag(shellCmd)
	addPrintFlag(shellCmd)

	rootCmd.AddCommand(shellCmd)
}
