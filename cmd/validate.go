package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/config"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the AliasCtl configuration",

	RunE: func(cmd *cobra.Command, args []string) error {

		resolvedPath, err := config.ResolvePath(configPath)
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		cfg, err := config.Load(resolvedPath)
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		fmt.Println("Configuration is valid.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
