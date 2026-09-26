package shell

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/resolver"
)

func planEnvironment() *resolver.Environment {
	return &resolver.Environment{
		Name: "demo",
		Variables: map[string]string{
			"NAMESPACE": "app-prod",
		},
		Aliases: map[string]resolver.Command{
			"kgp": {Name: "kubectl", Args: []string{"get", "pods"}},
		},
		Functions: map[string]string{
			"hello": "echo hello",
		},
		FunctionsFish: map[string]string{
			"hello": "echo hello",
		},
	}
}

func TestDescribeStart(t *testing.T) {
	t.Setenv("ALIASCTL_ENV", "")
	t.Setenv("ALIASCTL_HOOK", "1")

	tests := []struct {
		runner   Runner
		contains []string
	}{
		{
			runner: bashRunner{},
			contains: []string{
				"command: bash --noprofile --rcfile <tmp>/bashrc -i\n",
				"ALIASCTL_ENV=demo\n",
				"-ALIASCTL_HOOK\n",
				"\n--- <tmp>/bashrc\n",
				"alias kgp='kubectl get pods'\n",
				"  Config:      /p/aliasctl.yaml",
			},
		},
		{
			runner: zshRunner{},
			contains: []string{
				"command: zsh -i\n",
				"ZDOTDIR=<tmp>\n",
				"\n--- <tmp>/.zshenv\n",
				"\n--- <tmp>/.zshrc\n",
				"  Config:      /p/aliasctl.yaml",
			},
		},
		{
			runner: fishRunner{},
			contains: []string{
				"command: fish -i --init-command 'source '\\''<tmp>/init.fish'\\'''\n",
				"\n--- <tmp>/init.fish\n",
				"function kgp\n",
				"  Config:      /p/aliasctl.yaml",
			},
		},
	}

	for _, tt := range tests {
		var output bytes.Buffer

		if err := DescribeStart(&output, tt.runner, planEnvironment(), "/p/aliasctl.yaml"); err != nil {
			t.Fatal(err)
		}

		for _, want := range tt.contains {
			if !strings.Contains(output.String(), want) {
				t.Fatalf("%T: expected output to contain %q, got:\n%s", tt.runner, want, output.String())
			}
		}
	}
}

func TestDescribeRun(t *testing.T) {
	var output bytes.Buffer

	if err := DescribeRun(&output, bashRunner{}, planEnvironment(), []string{"kgp", "-o", "wide"}); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"command: bash --noprofile --norc <tmp>/run.sh -o wide\n",
		"NAMESPACE=app-prod\n",
		"\n--- <tmp>/run.sh\n",
		`kgp "$@"` + "\n",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output.String())
		}
	}

	output.Reset()

	// a plain command runs directly, so there is nothing to write
	if err := DescribeRun(&output, bashRunner{}, planEnvironment(), []string{"ls", "-la"}); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(output.String(), "---") || !strings.HasPrefix(output.String(), "command: ls -la\n") {
		t.Fatalf("unexpected plain command plan:\n%s", output.String())
	}
}

func TestEnvironmentChanges(t *testing.T) {
	got := environmentChanges(
		[]string{"A=1", "B=2", "C=3"},
		[]string{"A=1", "B=changed", "D=new", "D=newer"},
	)

	expected := []string{"B=changed", "D=newer", "-C"}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}
