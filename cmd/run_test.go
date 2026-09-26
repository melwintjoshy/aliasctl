package cmd

import (
	"strings"
	"testing"
)

const completionConfig = `name: test
aliases:
  k: kubectl
  kgp: "kubectl get pods -n dev"
functions:
  deploy: |
    go build ./...
`

func TestRunCompletesAliasesAndFunctions(t *testing.T) {
	path := writeTestConfig(t, completionConfig)

	output, err := runRoot(t, "__complete", "--config", path, "run", "")
	if err != nil {
		t.Fatal(err)
	}

	// cobra prints the numeric directive to stdout and its name to stderr, and the helper captures both
	expected := "deploy\tfunction\nk\tkubectl\nkgp\tkubectl get pods -n dev\n:4\n"

	if !strings.HasPrefix(output, expected) {
		t.Fatalf("expected completions:\n%s\ngot:\n%s", expected, output)
	}

	// after the first word the command's own completion applies, so cobra falls back to files
	output, err = runRoot(t, "__complete", "--config", path, "run", "kgp", "")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(output, ":0\n") || strings.Contains(output, "deploy") {
		t.Fatalf("expected no name completions after the command, got:\n%s", output)
	}
}

func TestRunCompletesShellFlag(t *testing.T) {
	output, err := runRoot(t, "__complete", "run", "--shell", "")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(output, "bash\nzsh\nfish\n") {
		t.Fatalf("expected the supported shells, got:\n%s", output)
	}
}
