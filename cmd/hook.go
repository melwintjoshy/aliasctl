package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/shell"
)

var hookCmd = &cobra.Command{
	Use:   "hook <bash|zsh>",
	Short: "Print a snippet that auto-activates environments on cd",
	Long: `Print a snippet that loads aliasctl.yaml when you enter a project and unloads it when you leave.

Add it to your shell rc:

  eval "$(aliasctl hook bash)"   # ~/.bashrc
  eval "$(aliasctl hook zsh)"    # ~/.zshrc`,

	Args: cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {

		// the absolute path avoids depending on PATH at every prompt
		binary, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not locate aliasctl binary: %w", err)
		}

		snippet, err := shell.RenderHook(args[0], binary)
		if err != nil {
			return err
		}

		fmt.Fprint(cmd.OutOrStdout(), snippet)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(hookCmd)
}
