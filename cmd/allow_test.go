package cmd

import (
	"strings"
	"testing"
)

func TestAllowDenyRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path := writeTestConfig(t, "name: test\naliases:\n  k: kubectl\n")

	if output, err := runRoot(t, "allow", "--list"); err != nil || output != "Nothing is allowed yet.\n" {
		t.Fatalf("unexpected empty list: %q, %v", output, err)
	}

	if _, err := runRoot(t, "--config", path, "allow"); err != nil {
		t.Fatal(err)
	}

	output, err := runRoot(t, "allow", "--list")
	if err != nil || !strings.Contains(output, "ok      "+path) {
		t.Fatalf("expected the config listed as ok, got %q, %v", output, err)
	}

	output, err = runRoot(t, "--config", path, "deny")
	if err != nil || !strings.HasPrefix(output, "Denied "+path) {
		t.Fatalf("unexpected deny output %q, %v", output, err)
	}

	output, err = runRoot(t, "--config", path, "deny")
	if err != nil || output != path+" was not allowed\n" {
		t.Fatalf("unexpected second deny output %q, %v", output, err)
	}
}

func TestAllowRejectsListWithPrune(t *testing.T) {
	if _, err := runRoot(t, "allow", "--list", "--prune"); err == nil {
		t.Fatal("expected an error for --list with --prune")
	}
}
