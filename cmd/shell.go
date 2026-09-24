package cmd

import (
	"fmt"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/melwintjoshy/aliasctl/shell"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start a temporary command environment",
	RunE: func(cmd *cobra.Command, args []string) error {

		env, err := app.LoadEnvironment(configPath)
		if err != nil {
			return fmt.Errorf("failed to load environment: %w", err)
		}

		if err := shell.RunBash(env); err != nil {
			return fmt.Errorf("shell error: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
}
