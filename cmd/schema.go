package cmd

import (
	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/schema"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Print the JSON schema for aliasctl.yaml",
	Long: `Print the JSON schema for aliasctl.yaml, for editor autocomplete and validation.

With the YAML language server (VS Code YAML extension and others), add this first line to aliasctl.yaml:

  # yaml-language-server: $schema=https://raw.githubusercontent.com/melwintjoshy/aliasctl/main/schema/aliasctl.schema.json`,

	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := cmd.OutOrStdout().Write(schema.JSON)
		return err
	},
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}
