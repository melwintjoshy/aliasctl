package resolver

import "testing"

func TestEnvironmentVariable(t *testing.T) {
	env := &Environment{
		Variables: map[string]string{
			"NAMESPACE": "app-prod",
		},
	}

	value, ok := env.Variable("NAMESPACE")

	if !ok {
		t.Fatal("expected variable to exist")
	}

	if value != "app-prod" {
		t.Fatalf("expected %q, got %q", "app-prod", value)
	}
}

func TestEnvironmentAlias(t *testing.T) {
	env := &Environment{
		Aliases: map[string]Command{
			"kgp": {
				Name: "kubectl",
				Args: []string{
					"get",
					"pods",
				},
			},
		},
	}

	command, ok := env.Alias("kgp")

	if !ok {
		t.Fatal("expected alias to exist")
	}

	if command.Name != "kubectl" {
		t.Fatalf(
			"expected %q, got %q",
			"kubectl",
			command.Name,
		)
	}
}
