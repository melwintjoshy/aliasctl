package cmd

import "github.com/spf13/cobra"

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
	rootCmd.PersistentFlags().StringVar(
		&configPath,
		"config",
		"",
		"Path to the AliasCtl configuration file",
	)
}
