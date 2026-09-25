package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/app"
)

// puts fake tools first on PATH; sh scripts that print a version or fail
func fakeToolsOnPath(t *testing.T, tools map[string]string) {
	t.Helper()

	dir := t.TempDir()

	for name, body := range tools {
		script := "#!/bin/sh\n" + body + "\n"

		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

const toolsConfig = `name: test
aliases:
  k: kubectl
tools:
  fakego: "1.22"
  fakekube: "<1.30"
  fakejava: "*"
  fakemissing: ">=1.6"
`

func TestToolsCheckReportsProblems(t *testing.T) {
	fakeToolsOnPath(t, map[string]string{
		"fakego":   `echo "go version go1.22.4 darwin/arm64"`,
		"fakekube": `echo "Client Version: v1.36.4"`,
		"fakejava": `echo "Unable to locate a Java Runtime." >&2; exit 1`,
	})

	path := writeTestConfig(t, toolsConfig)

	output, err := runRoot(t, "--config", path, "tools", "check")

	if !errors.Is(err, ErrReported) {
		t.Fatalf("expected a reported failure, got %v", err)
	}

	for _, row := range []string{
		"fakego       1.22   1.22.4  ok",
		"fakejava     *      -       broken",
		"fakekube     <1.30  1.36.4  mismatch",
		"fakemissing  >=1.6  -       missing   -",
		"Unable to locate a Java Runtime.",
	} {
		if !strings.Contains(output, row) {
			t.Fatalf("expected output to contain %q, got:\n%s", row, output)
		}
	}
}

func TestToolsCheckPasses(t *testing.T) {
	fakeToolsOnPath(t, map[string]string{
		"fakego": `echo "go1.22.4"`,
	})

	path := writeTestConfig(t, `name: test
aliases:
  k: kubectl
tools:
  fakego: "1.22"
`)

	if _, err := runRoot(t, "--config", path, "tools", "check"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolsCheckWithoutTools(t *testing.T) {
	path := writeTestConfig(t, "name: test\naliases:\n  k: kubectl\n")

	output, err := runRoot(t, "--config", path, "tools", "check")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != "No tools declared.\n" {
		t.Fatalf("unexpected output %q", output)
	}
}

func TestWarnToolProblems(t *testing.T) {
	fakeToolsOnPath(t, map[string]string{
		"fakego":   `echo "go1.22.4"`,
		"fakekube": `echo "v1.36.4"`,
		"fakejava": `exit 1`,
	})

	env, err := app.LoadEnvironment(writeTestConfig(t, toolsConfig))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer

	warnToolProblems(&output, env)

	expected := `aliasctl: fakejava is installed but not working: exit status 1
aliasctl: fakekube 1.36.4 does not match "<1.30"
aliasctl: fakemissing is not installed (want ">=1.6")
aliasctl: run "aliasctl tools check" for details
`

	if output.String() != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, output.String())
	}
}

func TestWarnToolProblemsQuietWhenAllOK(t *testing.T) {
	fakeToolsOnPath(t, map[string]string{"fakego": `echo "go1.22.4"`})

	env, err := app.LoadEnvironment(writeTestConfig(t, "name: t\naliases:\n  k: kubectl\ntools:\n  fakego: \"1.22\"\n"))
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer

	warnToolProblems(&output, env)

	if output.Len() != 0 {
		t.Fatalf("expected no output, got %q", output.String())
	}
}
