package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/resolver"
)

func TestHookIsInertInsideExplicitShell(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	project := t.TempDir()

	if err := os.WriteFile(filepath.Join(project, "aliasctl.yaml"), []byte("name: x\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// the binary path is bogus on purpose: the hook must return before calling it
	hook, err := RenderHook("bash", "/nonexistent/aliasctl")
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		"bash",
		"--noprofile",
		"--norc",
		"-c",
		hook+`cd "$1"; __aliasctl_hook; echo "${ALIASCTL_CONFIG:-none}"`,
		"bash",
		project,
	)

	cmd.Env = append(os.Environ(), "ALIASCTL_ENV=explicit", "ALIASCTL_HOOK=")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hook failed: %v\n%s", err, output)
	}

	if string(output) != "none\n" {
		t.Fatalf("expected the hook to do nothing, got %q", output)
	}
}

func TestRenderExportRejectsFish(t *testing.T) {
	if _, err := RenderExport(&resolver.Environment{Name: "x"}, "fish", "/x"); err == nil {
		t.Fatal("expected fish to be rejected for export")
	}

	if _, err := RenderHook("fish", "/x"); err == nil {
		t.Fatal("expected fish to be rejected for hook")
	}
}

func TestRenderExportUnloadRestoresPreviousState(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	env := &resolver.Environment{
		Name: "demo",
		Variables: map[string]string{
			"KEPT": "project",
			"NEW":  "project",
		},
		Aliases: map[string]resolver.Command{
			"hi": {Name: "echo", Args: []string{"project"}},
		},
	}

	script, err := RenderExport(env, "bash", "/p/aliasctl.yaml")
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		"bash",
		"--noprofile",
		"--norc",
		"-c",
		"shopt -s expand_aliases\n"+
			"export KEPT='it'\\''s user'\n"+
			"alias hi='echo user'\n"+
			script+
			"eval \"$ALIASCTL_UNLOAD\"\n"+
			"hi\n"+
			`echo "kept=$KEPT new=${NEW-unset}"`+"\n",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("export failed: %v\n%s", err, output)
	}

	expected := "user\nkept=it's user new=unset\n"

	if string(output) != expected {
		t.Fatalf("expected %q, got %q", expected, output)
	}

	if !strings.Contains(script, "export ALIASCTL_HOOK=1") {
		t.Fatal("expected export to mark the environment as hook-loaded")
	}
}
