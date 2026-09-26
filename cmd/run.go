package cmd

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/melwintjoshy/aliasctl/resolver"
	"github.com/melwintjoshy/aliasctl/shell"
)

var runCmd = &cobra.Command{
	Use:   "run <command> [args...]",
	Short: "Run a command inside the AliasCtl environment",

	Args: cobra.MinimumNArgs(1),

	ValidArgsFunction: completeRunTarget,

	RunE: func(cmd *cobra.Command, args []string) error {

		runner, err := shell.New(shellName)
		if err != nil {
			return err
		}

		env, _, err := app.LoadEnvironment(configPath)
		if err != nil {
			return fmt.Errorf("failed to load environment: %w", err)
		}

		activateTools(cmd.ErrOrStderr(), env)

		if printPlan {
			return shell.DescribeRun(cmd.OutOrStdout(), runner, env, args)
		}

		if err := shell.Run(runner, env, args); err != nil {
			return fmt.Errorf("command failed: %w", err)
		}

		return nil
	},
}

// completes alias and function names for the first word; anything after belongs to the command
func completeRunTarget(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveDefault
	}

	env, _, err := app.LoadEnvironment(configPath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return commandNames(env, shellName == "fish"), cobra.ShellCompDirectiveNoFileComp
}

// sorted names with the alias command or "function" as the description zsh and fish show
func commandNames(env *resolver.Environment, fish bool) []string {
	names := make([]string, 0, len(env.Aliases)+len(env.Functions))

	for name, command := range env.Aliases {
		names = append(names, name+"\t"+shell.FormatCommand(command))
	}

	functions := env.Functions
	if fish {
		functions = env.FunctionsFish
	}

	for name := range functions {
		names = append(names, name+"\tfunction")
	}

	slices.Sort(names)

	return names
}

func init() {
	// flags after the command belong to it, e.g. "aliasctl run kgp --help"
	runCmd.Flags().SetInterspersed(false)
	addShellFlag(runCmd)
	addPrintFlag(runCmd)

	rootCmd.AddCommand(runCmd)
}
