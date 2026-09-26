package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/resolver"
)

func TestZshRunKeepsShellCharactersLiteral(t *testing.T) {
	env := &resolver.Environment{
		Aliases: map[string]resolver.Command{
			"danger": {Name: "write", Args: []string{"hello; touch pwned"}},
		},
		Functions: map[string]string{
			"write": `print -rn -- "${@[1,-2]}" > "${@[-1]}"`,
		},
	}

	dir := t.TempDir()
	t.Chdir(dir)

	got := runToFileWith(t, zshRunner{}, "zsh", env, "danger")

	if got != "hello; touch pwned" {
		t.Fatalf("unexpected output %q", got)
	}

	if _, err := os.Stat(filepath.Join(dir, "pwned")); err == nil {
		t.Fatal("alias argument was interpreted as shell code")
	}
}

func TestZshRunFunctionUsesAlias(t *testing.T) {
	env := &resolver.Environment{
		Variables: map[string]string{
			"GREETING": "hello",
		},
		Aliases: map[string]resolver.Command{
			"say": {Name: "echo"},
		},
		Functions: map[string]string{
			"greet": `say "$GREETING $1" > "$2"`,
		},
	}

	got := runToFileWith(t, zshRunner{}, "zsh", env, "greet", "big world")

	if got != "hello big world\n" {
		t.Fatalf("expected %q, got %q", "hello big world\n", got)
	}
}

func startZshForTest(t *testing.T, env *resolver.Environment, userDir, script string) string {
	t.Helper()

	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh is not available")
	}

	dir := t.TempDir()

	zshenv, zshrc, err := renderZshStartup(env, "", dir, userDir)
	if err != nil {
		t.Fatal(err)
	}

	writeHomeFile(t, dir, ".zshenv", zshenv)
	writeHomeFile(t, dir, ".zshrc", zshrc)

	cmd := exec.Command("zsh", "-i", "-c", script)

	cmd.Dir = userDir
	cmd.Env = append(os.Environ(), "HOME="+userDir, "ZDOTDIR="+dir, "ALIASCTL_ENV="+env.Name)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("interactive zsh failed: %v", err)
	}

	return string(output)
}

func TestZshStartupLayersOnUserRC(t *testing.T) {
	home := t.TempDir()

	writeHomeFile(t, home, ".zshrc", "PROMPT='base> '\n"+
		"alias mine='echo mine'\n"+
		"alias shared='echo user'\n"+
		"alias deploy='echo user deploy'\n")

	env := &resolver.Environment{
		Name: "demo $(touch pwned)",
		Aliases: map[string]resolver.Command{
			"shared": {Name: "echo", Args: []string{"project"}},
		},
		Functions: map[string]string{
			"deploy": "echo project deploy",
		},
	}

	got := startZshForTest(
		t,
		env,
		home,
		`print -r -- "$PROMPT"; mine; shared; deploy; print -r -- "$ZDOTDIR"`,
	)

	expected := "(aliasctl:demo___touch_pwned_) base> \n" +
		"mine\n" +
		"project\n" +
		"project deploy\n" +
		home + "\n"

	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}

	if _, err := os.Stat(filepath.Join(home, "pwned")); err == nil {
		t.Fatal("environment name was executed")
	}
}

func TestZshStartupFollowsZdotdirMovedByZshenv(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "zsh")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeHomeFile(t, home, ".zshenv", `ZDOTDIR="$HOME/.config/zsh"`+"\n")
	writeHomeFile(t, configDir, ".zshrc", "alias moved='echo moved'\n")

	got := startZshForTest(t, &resolver.Environment{Name: "demo"}, home, "moved")

	if got != "moved\n" {
		t.Fatalf("expected %q, got %q", "moved\n", got)
	}
}

func TestZshPromptHookReappliesPrefix(t *testing.T) {
	home := t.TempDir()

	// mimics prompts that rebuild PROMPT in precmd
	writeHomeFile(t, home, ".zshrc", `precmd() { PROMPT="dyn> " }`+"\n")

	got := startZshForTest(
		t,
		&resolver.Environment{Name: "demo"},
		home,
		`precmd; for f in $precmd_functions; do $f; done; print -r -- "$PROMPT"`,
	)

	if got != "(aliasctl:demo) dyn> \n" {
		t.Fatalf("unexpected prompt %q", got)
	}
}

func TestRenderZshStartupIncludesBanner(t *testing.T) {
	env := &resolver.Environment{
		Name: "dev",
	}

	_, zshrc, err := renderZshStartup(env, "/path/to/aliasctl.yaml", "/tmp/zsh", "/home/user")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(zshrc, "Environment: dev") {
		t.Fatal("expected environment name in rendered zshrc banner")
	}

	if !strings.Contains(zshrc, "Config:      /path/to/aliasctl.yaml") {
		t.Fatal("expected config path in rendered zshrc banner")
	}
}
