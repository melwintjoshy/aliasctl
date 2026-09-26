package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/melwintjoshy/aliasctl/mise"
	"github.com/melwintjoshy/aliasctl/resolver"
	"github.com/melwintjoshy/aliasctl/shell"
	"github.com/melwintjoshy/aliasctl/tools"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Work with the tools the environment requires",
}

var toolsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check installed tools against the configured versions",

	RunE: func(cmd *cobra.Command, args []string) error {

		env, err := app.LoadEnvironment(configPath)
		if err != nil {
			return fmt.Errorf("failed to load environment: %w", err)
		}

		activateTools(cmd.ErrOrStderr(), env)

		output := cmd.OutOrStdout()

		if len(env.Tools) == 0 {
			fmt.Fprintln(output, "No tools declared.")
			return nil
		}

		failed := printToolTable(output, checkTools(env))

		// non-zero so it works as a ci gate; the table already says why
		if failed {
			return ErrReported
		}

		return nil
	},
}

// printToolTable reports whether any tool is not ok.
func printToolTable(w io.Writer, results []tools.Result) bool {
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	fmt.Fprintln(table, "TOOL\tWANT\tFOUND\tSTATUS\tPATH\tNOTE")

	failed := false

	for _, result := range results {
		found := "-"
		if result.Found != nil {
			found = result.Found.String()
		}

		path := result.Path
		if path == "" {
			path = "-"
		}

		fmt.Fprintf(
			table,
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			result.Name,
			result.Rule,
			found,
			result.Status,
			path,
			result.Detail,
		)

		failed = failed || result.Status != tools.StatusOK
	}

	table.Flush()

	return failed
}

func checkTools(env *resolver.Environment) []tools.Result {
	// checks run in their own process groups, so ctrl-c has to reach them through the context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	checker := tools.Checker{Timeout: tools.DefaultTimeout, Dir: env.Dir}

	return checker.Check(ctx, env.Tools, shell.BuildEnvironment(env))
}

// warns without blocking, a wrong kubectl shouldn't stop someone opening the shell
func warnToolProblems(w io.Writer, env *resolver.Environment) {
	if len(env.Tools) == 0 {
		return
	}

	problems := 0

	for _, result := range checkTools(env) {
		if problem := result.Problem(); problem != "" {
			fmt.Fprintf(w, "aliasctl: %s\n", problem)
			problems++
		}
	}

	if problems > 0 {
		fmt.Fprintln(w, `aliasctl: run "aliasctl tools check" for details`)
	}
}

func init() {
	toolsCmd.AddCommand(toolsCheckCmd)
	toolsCmd.AddCommand(toolsInstallCmd)
	rootCmd.AddCommand(toolsCmd)
}

// swapped in tests; the real one runs the mise binary
var miseBackend mise.Backend = mise.CLI{}

// uses mise-installed tool versions when there are any; a mise problem is reported, never fatal
func activateTools(w io.Writer, env *resolver.Environment) {
	if err := app.ActivateTools(env, miseBackend); err != nil {
		fmt.Fprintf(w, "aliasctl: could not use mise tool versions: %v\n", err)
	}
}
