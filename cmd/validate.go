package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/app"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the AliasCtl configuration",

	RunE: func(cmd *cobra.Command, args []string) error {

		// resolving too, so undefined ${VAR} fails here instead of at shell or run time
		if _, err := app.LoadEnvironment(configPath); err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Configuration is valid.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
