package cmd

import (
	"fmt"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the aliases and variables in the environment",

	RunE: func(cmd *cobra.Command, args []string) error {

		env, err := app.LoadEnvironment(configPath)
		if err != nil {
			return fmt.Errorf("failed to load environment: %w", err)
		}

		fmt.Println("Environment:", env.Name)
		fmt.Println()

		fmt.Println("Variables:")
		for key, value := range env.Variables {
			fmt.Printf("  %s=%s\n", key, value)
		}

		fmt.Println()

		fmt.Println("Aliases:")
		for name, command := range env.Aliases {
			fmt.Printf("  %-6s → %s\n", name, command)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
