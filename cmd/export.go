package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/melwintjoshy/aliasctl/config"
	"github.com/melwintjoshy/aliasctl/shell"
)

var (
	exportQuiet    bool
	exportShellPID int
)

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

		// stamped before reading, so an edit made while loading still counts as newer
		stamp, err := app.NewStamp(exportShellPID)
		if err != nil {
			return err
		}

		env, err := app.LoadTrustedEnvironment(absolutePath)

		// without this, cloning a repo would run its config on the next prompt
		if errors.Is(err, app.ErrNotAllowed) {
			app.RemoveStamp(stamp)

			if !exportQuiet {
				fmt.Fprintf(
					os.Stderr,
					"aliasctl: %s is not allowed; review it and run \"aliasctl allow\"\n",
					absolutePath,
				)
			}

			return ErrReported
		}

		if err != nil {
			app.RemoveStamp(stamp)
			return fmt.Errorf("failed to load environment: %w", err)
		}

		activateTools(os.Stderr, env)

		allowedPath, err := app.AllowedFile()
		if err != nil {
			app.RemoveStamp(stamp)
			return err
		}

		script, err := shell.RenderExport(env, shellName, absolutePath, stamp, allowedPath)
		if err != nil {
			app.RemoveStamp(stamp)
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

	// the hook passes $$ so stale stamps can be told apart from those of shells still running
	exportCmd.Flags().IntVar(
		&exportShellPID,
		"shell-pid",
		0,
		"Process id of the shell that will eval the output",
	)

	rootCmd.AddCommand(exportCmd)
}
