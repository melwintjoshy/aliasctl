package cmd

import (
	"fmt"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/melwintjoshy/aliasctl/shell"
	"github.com/spf13/cobra"
)

var shellName string

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start a temporary command environment",
	RunE: func(cmd *cobra.Command, args []string) error {

		runner, err := shell.New(shellName)
		if err != nil {
			return err
		}

		env, resolvedConfigPath, err := app.LoadEnvironment(configPath)
		if err != nil {
			return fmt.Errorf("failed to load environment: %w", err)
		}

		if err := runner.Start(env, resolvedConfigPath); err != nil {
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

func init() {
	addShellFlag(shellCmd)

	rootCmd.AddCommand(shellCmd)
}
