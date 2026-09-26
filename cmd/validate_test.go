package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()

	var output bytes.Buffer

	resetFlags(rootCmd)

	rootCmd.SetOut(&output)
	rootCmd.SetArgs(args)

	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
		configPath = ""
	})

	err := rootCmd.Execute()

	return output.String(), err
}

// cobra keeps flag values between Execute calls, so each run starts from the defaults
func resetFlags(command *cobra.Command) {
	reset := func(flag *pflag.Flag) {
		flag.Value.Set(flag.DefValue)
		flag.Changed = false
	}

	command.Flags().VisitAll(reset)
	command.PersistentFlags().VisitAll(reset)

	for _, child := range command.Commands() {
		resetFlags(child)
	}
}

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "aliasctl.yaml")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestValidateRejectsUndefinedVariable(t *testing.T) {
	path := writeTestConfig(t, `name: test
aliases:
  k: "kubectl -n ${ALIASCTL_TEST_UNSET}"
`)

	_, err := runRoot(t, "--config", path, "validate")
	if err == nil {
		t.Fatal("expected undefined variable error")
	}

	if !strings.Contains(err.Error(), `undefined variable "ALIASCTL_TEST_UNSET"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	path := writeTestConfig(t, `name: test
variables:
  NAMESPACE: app-prod
aliases:
  k: "kubectl -n ${NAMESPACE}"
`)

	output, err := runRoot(t, "--config", path, "validate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != "Configuration is valid.\n" {
		t.Fatalf("unexpected output: %q", output)
	}
}
