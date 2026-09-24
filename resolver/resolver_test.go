package resolver

import (
	"reflect"
	"testing"

	"github.com/melwintjoshy/aliasctl/config"
)

func TestResolveVariables(t *testing.T) {
	cfg := &config.Config{
		Name: "test",

		Variables: map[string]string{
			"NAMESPACE": "app-prod",
		},

		Aliases: map[string]string{
			"kgp": "kubectl get pods -n ${NAMESPACE}",
		},
	}

	env, err := Resolve(cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	command, ok := env.Aliases["kgp"]

	if !ok {
		t.Fatal("expected kgp alias to exist")
	}

	if command.Name != "kubectl" {
		t.Fatalf(
			"expected command name %q, got %q",
			"kubectl",
			command.Name,
		)
	}

	expectedArgs := []string{
		"get",
		"pods",
		"-n",
		"app-prod",
	}

	if !reflect.DeepEqual(command.Args, expectedArgs) {
		t.Fatalf(
			"expected args %v, got %v",
			expectedArgs,
			command.Args,
		)
	}
}

func TestResolveWithoutVariables(t *testing.T) {
	cfg := &config.Config{
		Name: "test",

		Aliases: map[string]string{
			"k": "kubectl",
		},
	}

	env, err := Resolve(cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	command, ok := env.Aliases["k"]

	if !ok {
		t.Fatal("expected k alias to exist")
	}

	if command.Name != "kubectl" {
		t.Fatalf(
			"expected command name %q, got %q",
			"kubectl",
			command.Name,
		)
	}

	if len(command.Args) != 0 {
		t.Fatalf(
			"expected no arguments, got %v",
			command.Args,
		)
	}
}

func TestResolveInvalidCommand(t *testing.T) {
	cfg := &config.Config{
		Name: "test",

		Aliases: map[string]string{
			"broken": `echo "hello`,
		},
	}

	_, err := Resolve(cfg)

	if err == nil {
		t.Fatal("expected error for invalid command")
	}
}
