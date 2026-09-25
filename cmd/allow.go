package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/melwintjoshy/aliasctl/config"
)

var allowCmd = &cobra.Command{
	Use:   "allow",
	Short: "Trust the configuration for auto-activation",

	RunE: func(cmd *cobra.Command, args []string) error {

		resolvedPath, err := config.ResolvePath(configPath)
		if err != nil {
			return err
		}

		// refuse to trust something that would not load anyway
		if _, _, err := app.LoadEnvironment(resolvedPath); err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		absolutePath, err := app.Allow(resolvedPath)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Allowed %s\n", absolutePath)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(allowCmd)
}
