package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/melwintjoshy/aliasctl/app"
	"github.com/melwintjoshy/aliasctl/config"
)

var (
	allowList  bool
	allowPrune bool
)

var allowCmd = &cobra.Command{
	Use:   "allow",
	Short: "Trust the configuration for auto-activation",

	RunE: func(cmd *cobra.Command, args []string) error {

		output := cmd.OutOrStdout()

		if allowList && allowPrune {
			return fmt.Errorf("--list and --prune can't be used together")
		}

		if allowList {
			entries, err := app.ListAllowed()
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Fprintln(output, "Nothing is allowed yet.")
				return nil
			}

			table := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)

			fmt.Fprintln(table, "STATUS\tCONFIG")

			for _, entry := range entries {
				fmt.Fprintf(table, "%s\t%s\n", entry.Status, entry.Path)
			}

			return table.Flush()
		}

		if allowPrune {
			pruned, err := app.Prune()
			if err != nil {
				return err
			}

			for _, path := range pruned {
				fmt.Fprintf(output, "Removed %s (file no longer exists)\n", path)
			}

			if len(pruned) == 0 {
				fmt.Fprintln(output, "Nothing to prune.")
			}

			return nil
		}

		resolvedPath, err := config.ResolvePath(configPath)
		if err != nil {
			return err
		}

		// refuse to trust something that would not load anyway
		if _, _, err := app.LoadEnvironment(resolvedPath); err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		absolutePath, err := app.Allow(resolvedPath)
		if err != nil {
			return err
		}

		fmt.Fprintf(output, "Allowed %s\n", absolutePath)

		return nil
	},
}

var denyCmd = &cobra.Command{
	Use:   "deny",
	Short: "Stop trusting the configuration for auto-activation",

	RunE: func(cmd *cobra.Command, args []string) error {

		resolvedPath, err := config.ResolvePath(configPath)
		if err != nil {
			return err
		}

		absolutePath, removed, err := app.Deny(resolvedPath)
		if err != nil {
			return err
		}

		if !removed {
			fmt.Fprintf(cmd.OutOrStdout(), "%s was not allowed\n", absolutePath)
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Denied %s; it unloads on the next directory change\n", absolutePath)

		return nil
	},
}

func init() {
	allowCmd.Flags().BoolVar(&allowList, "list", false, "List trusted configurations and whether they changed")
	allowCmd.Flags().BoolVar(&allowPrune, "prune", false, "Forget trusted configurations whose file no longer exists")

	rootCmd.AddCommand(allowCmd)
	rootCmd.AddCommand(denyCmd)
}
