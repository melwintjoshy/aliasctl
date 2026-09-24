package shell

import (
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/resolver"
)

func TestBashRenderer(t *testing.T) {
	env := &resolver.Environment{
		Name: "test",

		Variables: map[string]string{
			"NAMESPACE": "app-prod",
		},

		Aliases: map[string]resolver.Command{
			"kgp": {
				Name: "kubectl",
				Args: []string{"get", "pods", "-n", "app-prod"},
			},
		},
	}

	renderer := BashRenderer{}

	output, err := renderer.Render(env)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedVariable := "export NAMESPACE='app-prod'"
	expectedAlias := "alias kgp='kubectl get pods -n app-prod'"

	if !strings.Contains(output, expectedVariable) {
		t.Fatalf(
			"expected output to contain %q\nGot:\n%s",
			expectedVariable,
			output,
		)
	}

	if !strings.Contains(output, expectedAlias) {
		t.Fatalf(
			"expected output to contain %q\nGot:\n%s",
			expectedAlias,
			output,
		)
	}
}
