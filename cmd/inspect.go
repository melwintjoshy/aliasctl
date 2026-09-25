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
		fmt.Println()

		functionNames := make([]string, 0, len(env.Functions))

		for name := range env.Functions {
			functionNames = append(functionNames, name)
		}

		sort.Strings(functionNames)

		fmt.Println("Functions")
		fmt.Println("---------")

		for _, name := range functionNames {
			fmt.Println(name)

			body := strings.TrimRight(env.Functions[name], "\n")

			for _, line := range strings.Split(body, "\n") {
				fmt.Printf("  %s\n", line)
			}

			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
