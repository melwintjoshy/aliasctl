package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/internal/fakemise"
	"github.com/melwintjoshy/aliasctl/mise"
)

const installConfig = `name: test
aliases:
  k: kubectl
tools:
  fakego: "1.22"
  faketf:
    version: ">=1.6 <2"
    mise: "aqua:hashicorp/faketf"
  fakeok: "*"
`

func setupInstall(t *testing.T) (*fakemise.Fake, string) {
	t.Helper()

	// fakego is on PATH but too new, faketf is missing, fakeok is fine
	fakeToolsOnPath(t, map[string]string{
		"fakego": `echo "go1.27.1"`,
		"fakeok": `echo "fakeok 3.0"`,
	})

	fake := fakemise.Install(t)

	fake.SetRemote(t, "fakego", "1.21.13", "1.22.6", "1.22.10", "1.22rc1", "1.23.2")
	fake.SetRemote(t, "aqua:hashicorp/faketf", "1.5.7", "1.9.8", "2.0.0")

	return fake, writeTestConfig(t, installConfig)
}

func TestToolsInstallDryRun(t *testing.T) {
	fake, path := setupInstall(t)

	output, err := runRoot(t, "--config", path, "tools", "install", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, output)
	}

	for _, want := range []string{"Will install with mise:", "fakego  fakego@1.22.10", "faketf  aqua:hashicorp/faketf@1.9.8"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in:\n%s", want, output)
		}
	}

	for _, call := range fake.Calls(t) {
		if strings.HasPrefix(call, "install") {
			t.Fatalf("dry run installed something: %v", fake.Calls(t))
		}
	}
}

func TestToolsInstallWithYes(t *testing.T) {
	fake, path := setupInstall(t)

	output, err := runRoot(t, "--config", path, "tools", "install", "--yes")
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, output)
	}

	for _, want := range []string{
		"installed fakego@1.22.10",
		"installed aqua:hashicorp/faketf@1.9.8",
		filepath.Join(fake.Root, "installs", "fakego", "1.22.10", "bin", "fakego"),
		filepath.Join(fake.Root, "installs", "aqua_hashicorp_faketf", "1.9.8", "bin", "faketf"),
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in:\n%s", want, output)
		}
	}

	// a second run finds everything ok and installs nothing
	output, err = runRoot(t, "--config", path, "tools", "install", "--yes")
	if err != nil || !strings.Contains(output, "Every tool is already ok.") {
		t.Fatalf("expected nothing left to install, got %v\n%s", err, output)
	}
}

func TestToolsInstallAsksFirst(t *testing.T) {
	fake, path := setupInstall(t)

	output, err := runRootWithInput(t, strings.NewReader("n\n"), "--config", path, "tools", "install")
	if !errors.Is(err, ErrReported) || !strings.Contains(output, "Install? [y/N] Nothing installed.") {
		t.Fatalf("expected a refusal, got %v\n%s", err, output)
	}

	for _, call := range fake.Calls(t) {
		if strings.HasPrefix(call, "install") {
			t.Fatalf("installed after answering no: %v", fake.Calls(t))
		}
	}

	output, err = runRootWithInput(t, strings.NewReader("y\n"), "--config", path, "tools", "install")
	if err != nil || !strings.Contains(output, "installed fakego@1.22.10") {
		t.Fatalf("expected install after yes, got %v\n%s", err, output)
	}
}

func TestToolsInstallRefusesWithoutTerminal(t *testing.T) {
	_, path := setupInstall(t)

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	writer.Close()

	defer reader.Close()

	_, err = runRootWithInput(t, reader, "--config", path, "tools", "install")

	if err == nil || !strings.Contains(err.Error(), "run with --yes") {
		t.Fatalf("expected a refusal without a terminal, got %v", err)
	}
}

func TestToolsInstallReportsUnresolvable(t *testing.T) {
	fake, path := setupInstall(t)

	fake.SetRemote(t, "aqua:hashicorp/faketf", "1.5.7", "2.0.0")

	output, err := runRoot(t, "--config", path, "tools", "install", "--dry-run")

	if !errors.Is(err, ErrReported) {
		t.Fatalf("expected a reported failure, got %v", err)
	}

	if !strings.Contains(output, `faketf: no mise release of aqua:hashicorp/faketf matches ">=1.6 <2"`) {
		t.Fatalf("expected the unresolvable rule to be named, got:\n%s", output)
	}
}

func TestToolsInstallReportsFailedInstall(t *testing.T) {
	fake, path := setupInstall(t)

	fake.FailInstall(t, "aqua:hashicorp/faketf")

	output, err := runRoot(t, "--config", path, "tools", "install", "--yes")

	if !errors.Is(err, ErrReported) {
		t.Fatalf("expected a reported failure, got %v", err)
	}

	for _, want := range []string{"installed fakego@1.22.10", "mise install aqua:hashicorp/faketf@1.9.8: exit status 1"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in:\n%s", want, output)
		}
	}
}

func TestToolsInstallWithoutMise(t *testing.T) {
	path := writeTestConfig(t, installConfig)

	t.Setenv("PATH", t.TempDir())

	_, err := runRoot(t, "--config", path, "tools", "install", "--yes")

	if err == nil || !strings.Contains(err.Error(), "mise is not installed") {
		t.Fatalf("expected a missing-mise error, got %v", err)
	}
}

func TestToolsInstallUsesExistingMiseInstall(t *testing.T) {
	_, path := setupInstall(t)

	// already installed in mise: activation makes it ok, so only faketf is planned
	if err := (mise.CLI{}).Install("fakego@1.22.6", io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}

	output, err := runRoot(t, "--config", path, "tools", "install", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(output, "fakego@") || !strings.Contains(output, "faketf  aqua:hashicorp/faketf@1.9.8") {
		t.Fatalf("expected only faketf to be planned, got:\n%s", output)
	}
}
