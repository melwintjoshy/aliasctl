package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/app"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect the resolved environment",

	RunE: func(cmd *cobra.Command, args []string) error {

		env, err := app.LoadEnvironment(configPath)
		if err != nil {
			return fmt.Errorf("failed to load environment: %w", err)
		}

		fmt.Println("Environment")
		fmt.Println("-----------")
		fmt.Println("Name:", env.Name)

		fmt.Println()
		fmt.Println("Variables")
		fmt.Println("---------")

		variableNames := make([]string, 0, len(env.Variables))

		for name := range env.Variables {
			variableNames = append(variableNames, name)
		}

		sort.Strings(variableNames)

		for _, name := range variableNames {
			fmt.Printf("%s=%s\n", name, env.Variables[name])
		}

		fmt.Println()
		fmt.Println("Aliases")
		fmt.Println("-------")

		aliasNames := make([]string, 0, len(env.Aliases))

		for name := range env.Aliases {
			aliasNames = append(aliasNames, name)
		}

		sort.Strings(aliasNames)

		for _, name := range aliasNames {
			command := env.Aliases[name]

			value := command.Name

			if len(command.Args) > 0 {
				value += " " + strings.Join(command.Args, " ")
			}

			fmt.Printf("%s → %s\n", name, value)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
