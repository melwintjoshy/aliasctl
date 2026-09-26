package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/resolver"
)

func TestFishRendererOutput(t *testing.T) {
	env := &resolver.Environment{
		Variables: map[string]string{
			"VALUE": `it's \ here`,
		},
		Aliases: map[string]resolver.Command{
			"kgp": {Name: "kubectl", Args: []string{"get", "pods", "%self", "a b"}},
			"ls":  {Name: "ls", Args: []string{"-la"}},
		},
		Functions: map[string]string{
			"bashonly": "echo bash",
		},
		FunctionsFish: map[string]string{
			"deploy": "echo fish $argv",
		},
	}

	got, err := (FishRenderer{}).Render(env)
	if err != nil {
		t.Fatal(err)
	}

	expected := "set -gx VALUE 'it\\'s \\\\ here'\n" +
		"function kgp\n    kubectl get pods '%self' 'a b' $argv\nend\n" +
		"function ls\n" +
		"    if contains -- ls (builtin --names)\n" +
		"        builtin ls -la $argv\n" +
		"    else\n" +
		"        command ls -la $argv\n" +
		"    end\n" +
		"end\n" +
		"function deploy\necho fish $argv\nend\n"

	if got != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, got)
	}

	if strings.Contains(got, "bashonly") {
		t.Fatal("bash function body was rendered for fish")
	}
}

func TestFishRunKeepsShellCharactersLiteral(t *testing.T) {
	env := &resolver.Environment{
		Aliases: map[string]resolver.Command{
			"danger": {Name: "write", Args: []string{"hello; touch pwned"}},
		},
		FunctionsFish: map[string]string{
			"write": `printf '%s' $argv[1..-2] > $argv[-1]`,
		},
	}

	dir := t.TempDir()
	t.Chdir(dir)

	got := runToFileWith(t, fishRunner{}, "fish", env, "danger")

	if got != "hello; touch pwned" {
		t.Fatalf("unexpected output %q", got)
	}

	if _, err := os.Stat(filepath.Join(dir, "pwned")); err == nil {
		t.Fatal("alias argument was interpreted as shell code")
	}
}

func TestFishRunFunctionUsesAlias(t *testing.T) {
	env := &resolver.Environment{
		Variables: map[string]string{
			"GREETING": "hello",
		},
		Aliases: map[string]resolver.Command{
			"say": {Name: "echo"},
		},
		FunctionsFish: map[string]string{
			"greet": `say "$GREETING $argv[1]" > $argv[2]`,
		},
	}

	got := runToFileWith(t, fishRunner{}, "fish", env, "greet", "big world")

	if got != "hello big world\n" {
		t.Fatalf("expected %q, got %q", "hello big world\n", got)
	}
}

func TestFishRunRejectsBashOnlyFunction(t *testing.T) {
	env := &resolver.Environment{
		Functions: map[string]string{
			"deploy": "echo bash",
		},
	}

	err := (fishRunner{}).Run(env, []string{"deploy"})
	if err == nil || !strings.Contains(err.Error(), "functions") {
		t.Fatalf("expected bash-only function error, got %v", err)
	}
}

func TestNewRejectsUnknownShell(t *testing.T) {
	if _, err := New("tcsh"); err == nil {
		t.Fatal("expected unsupported shell error")
	}

	for _, name := range Supported {
		if _, err := New(name); err != nil {
			t.Fatalf("unexpected error for %s: %v", name, err)
		}
	}
}

func TestDefaultShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")

	if got := DefaultShell(); got != "zsh" {
		t.Fatalf("expected zsh, got %s", got)
	}

	t.Setenv("SHELL", "/bin/tcsh")

	if got := DefaultShell(); got != "bash" {
		t.Fatalf("expected bash fallback, got %s", got)
	}
}
