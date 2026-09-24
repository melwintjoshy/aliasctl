package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/config"
)

const defaultConfig = `name: my-project

variables:
  ENVIRONMENT: development

aliases:
  hello: "echo hello"
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new AliasCtl configuration",

	RunE: func(cmd *cobra.Command, args []string) error {

		_, err := os.Stat(config.ConfigFileName)

		if err == nil {
			return fmt.Errorf(
				"%s already exists",
				config.ConfigFileName,
			)
		}

		if !os.IsNotExist(err) {
			return fmt.Errorf(
				"could not check configuration: %w",
				err,
			)
		}

		if err := os.WriteFile(
			config.ConfigFileName,
			[]byte(defaultConfig),
			0644,
		); err != nil {
			return fmt.Errorf(
				"could not create configuration: %w",
				err,
			)
		}

		fmt.Printf(
			"Created %s\n",
			config.ConfigFileName,
		)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
