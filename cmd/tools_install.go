package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/melwintjoshy/aliasctl/mise"
	"github.com/melwintjoshy/aliasctl/resolver"
	"github.com/melwintjoshy/aliasctl/tools"
	"github.com/melwintjoshy/aliasctl/versions"
)

var (
	installYes    bool
	installDryRun bool
)

type installStep struct {
	tool string
	spec string
}

var toolsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install missing or mismatched tools with mise",
	Long: `Install every declared tool that is missing or the wrong version, using mise.

The highest mise release matching each tool's rule is chosen, the plan is shown,
and nothing is installed until you confirm (or pass --yes). mise's own config
files are not changed; aliasctl puts the installed versions on PATH itself.`,

	RunE: func(cmd *cobra.Command, args []string) error {

		output := cmd.OutOrStdout()

		env, _, err := app.LoadEnvironment(configPath)
		if err != nil {
			return fmt.Errorf("failed to load environment: %w", err)
		}

		if len(env.Tools) == 0 {
			fmt.Fprintln(output, "No tools declared.")
			return nil
		}

		if !miseBackend.Available() {
			return fmt.Errorf("mise is not installed; install it (brew install mise, or see https://mise.jdx.dev) and run this again")
		}

		activateTools(cmd.ErrOrStderr(), env)

		steps, problems := planInstall(env, checkTools(env), miseBackend)

		if len(steps) == 0 && len(problems) == 0 {
			fmt.Fprintln(output, "Every tool is already ok.")
			return nil
		}

		printInstallPlan(output, steps, problems)

		if installDryRun || len(steps) == 0 {
			if len(problems) > 0 {
				return ErrReported
			}

			return nil
		}

		// a cloned repo decides what mise downloads, so installing needs a yes from a person
		if !installYes {
			confirmed, err := confirm(cmd.InOrStdin(), output, "Install? [y/N] ")
			if err != nil {
				return err
			}

			if !confirmed {
				fmt.Fprintln(output, "Nothing installed.")
				return ErrReported
			}
		}

		for _, step := range steps {
			fmt.Fprintf(output, "\n==> mise install %s\n", step.spec)

			if err := miseBackend.Install(step.spec, output, cmd.ErrOrStderr()); err != nil {
				problems = append(problems, err.Error())
			}
		}

		// re-resolved so the table shows the versions just installed
		env.PathPrepend = nil
		activateTools(cmd.ErrOrStderr(), env)

		fmt.Fprintln(output)

		failed := printToolTable(output, checkTools(env))

		if failed || len(problems) > 0 {
			for _, problem := range problems {
				fmt.Fprintf(cmd.ErrOrStderr(), "aliasctl: %s\n", problem)
			}

			return ErrReported
		}

		return nil
	},
}

// timeouts are left alone: the tool is installed, just slow, and reinstalling won't help
func planInstall(env *resolver.Environment, results []tools.Result, backend mise.Backend) ([]installStep, []string) {
	requirements := make(map[string]resolver.ToolRequirement, len(env.Tools))

	for _, requirement := range env.Tools {
		requirements[requirement.Name] = requirement
	}

	var steps []installStep
	var problems []string

	for _, result := range results {
		if result.Status == tools.StatusOK {
			continue
		}

		if result.Status == tools.StatusTimeout {
			problems = append(problems, fmt.Sprintf("%s is installed but slow to answer; not reinstalling it", result.Name))
			continue
		}

		requirement := requirements[result.Name]

		constraint, err := versions.ParseConstraint(requirement.Rule)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", result.Name, err))
			continue
		}

		releases, err := backend.Remote(requirement.Mise)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", result.Name, err))
			continue
		}

		best, ok := mise.Best(releases, constraint)
		if !ok {
			problems = append(problems, fmt.Sprintf(
				"%s: no mise release of %s matches %q",
				result.Name,
				requirement.Mise,
				requirement.Rule,
			))

			continue
		}

		steps = append(steps, installStep{tool: result.Name, spec: requirement.Mise + "@" + best.Raw})
	}

	return steps, problems
}

func printInstallPlan(w io.Writer, steps []installStep, problems []string) {
	if len(steps) > 0 {
		fmt.Fprintln(w, "Will install with mise:")

		table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

		for _, step := range steps {
			fmt.Fprintf(table, "  %s\t%s\n", step.tool, step.spec)
		}

		table.Flush()
	}

	if len(problems) > 0 {
		fmt.Fprintln(w, "Can't install:")

		for _, problem := range problems {
			fmt.Fprintf(w, "  %s\n", problem)
		}
	}
}

// refuses rather than guessing when nobody is there to answer
func confirm(input io.Reader, output io.Writer, prompt string) (bool, error) {
	if file, ok := input.(*os.File); ok {
		info, err := file.Stat()
		if err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return false, fmt.Errorf("refusing to install without confirmation; run with --yes to install non-interactively")
		}
	}

	fmt.Fprint(output, prompt)

	answer, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && answer == "" {
		return false, nil
	}

	answer = strings.ToLower(strings.TrimSpace(answer))

	return answer == "y" || answer == "yes", nil
}

func init() {
	toolsInstallCmd.Flags().BoolVarP(&installYes, "yes", "y", false, "Install without asking for confirmation")
	toolsInstallCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "Show what would be installed and stop")
}
