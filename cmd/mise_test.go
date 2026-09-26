package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/internal/fakemise"
	"github.com/melwintjoshy/aliasctl/mise"
)

const miseToolsConfig = `name: test
aliases:
  k: kubectl
tools:
  fakego: "1.22"
`

func TestToolsCheckUsesMiseInstalledVersion(t *testing.T) {
	// the machine's own fakego is too new; mise has a matching one
	fakeToolsOnPath(t, map[string]string{"fakego": `echo "go1.27.1"`})

	fake := fakemise.Install(t)

	if err := (mise.CLI{}).Install("fakego@1.22.6", io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}

	path := writeTestConfig(t, miseToolsConfig)

	output, err := runRoot(t, "--config", path, "tools", "check")
	if err != nil {
		t.Fatalf("expected check to pass with the mise version, got %v\n%s", err, output)
	}

	misePath := filepath.Join(fake.Root, "installs", "fakego", "1.22.6", "bin", "fakego")

	if !strings.Contains(output, "1.22.6") || !strings.Contains(output, misePath) {
		t.Fatalf("expected the mise install to be used, got:\n%s", output)
	}
}

func TestRunPrintShowsMisePath(t *testing.T) {
	fake := fakemise.Install(t)

	if err := (mise.CLI{}).Install("fakego@1.22.6", io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}

	path := writeTestConfig(t, miseToolsConfig)

	output, err := runRoot(t, "--config", path, "run", "--print", "--shell", "bash", "fakego", "version")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output, "PATH="+filepath.Join(fake.Root, "installs", "fakego", "1.22.6", "bin")+":") {
		t.Fatalf("expected PATH to start with the mise dir, got:\n%s", output)
	}
}

func TestToolsCheckWithoutMiseKeepsWorking(t *testing.T) {
	fakeToolsOnPath(t, map[string]string{"fakego": `echo "go1.22.4"`})

	// no mise install matches, so the version on PATH decides
	path := writeTestConfig(t, miseToolsConfig)

	if output, err := runRoot(t, "--config", path, "tools", "check"); err != nil {
		t.Fatalf("expected check to pass from PATH alone, got %v\n%s", err, output)
	}
}

func TestRunUsesMiseInstalledVersion(t *testing.T) {
	// the only fakego is the one mise installed; run has to find it through the plan's PATH
	fake := fakemise.Install(t)

	if err := (mise.CLI{}).Install("fakego@1.22.6", io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}

	path := writeTestConfig(t, miseToolsConfig)

	stdout, err := os.CreateTemp(t.TempDir(), "stdout-*")
	if err != nil {
		t.Fatal(err)
	}

	defer stdout.Close()

	saved := os.Stdout
	os.Stdout = stdout

	output, runErr := runRoot(t, "--config", path, "run", "--shell", "bash", "fakego", "version")

	os.Stdout = saved

	if runErr != nil {
		t.Fatalf("run failed: %v\n%s", runErr, output)
	}

	data, err := os.ReadFile(stdout.Name())
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "fakego version 1.22.6\n" {
		t.Fatalf("expected the mise install of fakego to run, got %q (root %s)", data, fake.Root)
	}
}
