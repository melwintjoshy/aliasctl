package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/resolver"
)

func TestBuildEnvironmentPrependsToolDirs(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")

	environ := BuildEnvironment(&resolver.Environment{
		Name:        "demo",
		PathPrepend: []string{"/mise/go/1.22.6/bin", "/mise/kubectl/1.30.2/bin"},
	})

	if got := environmentMap(environ)["PATH"]; got != "/mise/go/1.22.6/bin:/mise/kubectl/1.30.2/bin:/usr/bin:/bin" {
		t.Fatalf("unexpected PATH %q", got)
	}
}

func TestInteractiveRCPutsToolDirsFirstAfterUserRC(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	home := t.TempDir()
	toolDir := filepath.Join(t.TempDir(), "go", "1.22.6", "bin")

	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatal(err)
	}

	// an rc that rebuilds PATH from scratch must not drop the tool dir
	writeHomeFile(t, home, ".bashrc", "export PATH=/usr/bin:/bin\n")

	got := runInteractive(
		t,
		home,
		&resolver.Environment{Name: "demo", PathPrepend: []string{toolDir}},
		`echo "${PATH%%:*}"`,
	)

	if got != toolDir+"\n" {
		t.Fatalf("expected %s first on PATH, got %q", toolDir, got)
	}
}

func TestRenderExportRestoresPath(t *testing.T) {
	// a dir the user adds while the project is loaded must survive leaving it,
	// and a copy of the tool dir that was already on PATH is not taken away
	script := "PATH=/usr/bin:/bin:/mise/go/1.22.6/bin\n" +
		"%s" +
		`echo "in=$PATH"` + "\n" +
		"PATH=/opt/venv/bin:$PATH\n" +
		`eval "$ALIASCTL_UNLOAD"; echo "out=$PATH"` + "\n" +
		`type __aliasctl_path_remove >/dev/null 2>&1 && echo "leaked"; true` + "\n"

	expected := "in=/mise/go/1.22.6/bin:/usr/bin:/bin:/mise/go/1.22.6/bin\n" +
		"out=/opt/venv/bin:/usr/bin:/bin:/mise/go/1.22.6/bin\n"

	for _, shellName := range hookShells {
		t.Run(shellName, func(t *testing.T) {
			if _, err := exec.LookPath(shellName); err != nil {
				t.Skipf("%s is not available", shellName)
			}

			export, err := RenderExport(
				&resolver.Environment{Name: "demo", PathPrepend: []string{"/mise/go/1.22.6/bin"}},
				shellName,
				"/p/aliasctl.yaml",
				"",
				"",
			)
			if err != nil {
				t.Fatal(err)
			}

			args := []string{"-c", fmt.Sprintf(script, export)}
			if shellName == "bash" {
				args = append([]string{"--noprofile", "--norc"}, args...)
			} else {
				args = append([]string{"-f"}, args...)
			}

			output, err := exec.Command(shellName, args...).CombinedOutput()
			if err != nil {
				t.Fatalf("export failed: %v\n%s", err, output)
			}

			if string(output) != expected {
				t.Fatalf("expected %q, got %q", expected, output)
			}
		})
	}
}

func TestPathRemoveKeepsEmptyEntries(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	// empty entries mean the current dir and are kept where they are
	output, err := exec.Command(
		"bash", "--noprofile", "--norc", "-c",
		pathRemoveFunction+"PATH=':/a:/b::/a:'; __aliasctl_path_remove /a; echo \"$PATH\"",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("bash failed: %v\n%s", err, output)
	}

	if string(output) != ":/b::/a:\n" {
		t.Fatalf("unexpected PATH %q", output)
	}
}

func TestFishStartPlanSetsPath(t *testing.T) {
	plan, err := (fishRunner{}).StartPlan(
		&resolver.Environment{Name: "demo", PathPrepend: []string{"/mise/go/1.22.6/bin"}},
		"",
		printDir,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(plan.Files[0].Content, "set -gx PATH '/mise/go/1.22.6/bin' $PATH\n") {
		t.Fatalf("expected fish to set PATH, got:\n%s", plan.Files[0].Content)
	}
}
