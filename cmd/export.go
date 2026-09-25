package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/melwintjoshy/aliasctl/config"
	"github.com/melwintjoshy/aliasctl/shell"
)

var exportQuiet bool

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Print the environment as shell code for eval",

	RunE: func(cmd *cobra.Command, args []string) error {

		resolvedPath, err := config.ResolvePath(configPath)
		if err != nil {
			return err
		}

		absolutePath, err := filepath.Abs(resolvedPath)
		if err != nil {
			return err
		}

		allowed, err := app.IsAllowed(absolutePath)
		if err != nil {
			return fmt.Errorf("could not check trust: %w", err)
		}

		// without this, cloning a repo would run its config on the next prompt
		if !allowed {
			if !exportQuiet {
				fmt.Fprintf(
					os.Stderr,
					"aliasctl: %s is not allowed; review it and run \"aliasctl allow\"\n",
					absolutePath,
				)
			}

			return ErrReported
		}

		env, err := app.LoadEnvironment(absolutePath)
		if err != nil {
			return fmt.Errorf("failed to load environment: %w", err)
		}

		script, err := shell.RenderExport(env, shellName, absolutePath)
		if err != nil {
			return err
		}

		fmt.Fprint(cmd.OutOrStdout(), script)

		return nil
	},
}

func init() {
	addShellFlag(exportCmd)

	exportCmd.Flags().BoolVar(
		&exportQuiet,
		"quiet",
		false,
		"Do not print the hint for an untrusted configuration",
	)

	rootCmd.AddCommand(exportCmd)
}
