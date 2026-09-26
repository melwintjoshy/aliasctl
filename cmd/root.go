package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/internal/buildinfo"
)

// ErrReported means the command already told the user what went wrong.
var ErrReported = errors.New("already reported")

var configPath string

var rootCmd = &cobra.Command{
	Use:           "aliasctl",
	Short:         "Declarative temporary command environments",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// cobra adds --version from this and prints "aliasctl version <string>"
	rootCmd.Version = buildinfo.String()

	rootCmd.PersistentFlags().StringVar(
		&configPath,
		"config",
		"",
		"Path to the AliasCtl configuration file",
	)
}
