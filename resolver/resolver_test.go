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

func TestResolveVariableWithShellCharacters(t *testing.T) {
	cfg := &config.Config{
		Name: "test",

		Variables: map[string]string{
			"VALUE": "hello; touch /tmp/marker",
		},

		Aliases: map[string]string{
			"test": "echo ${VALUE}",
		},
	}

	env, err := Resolve(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	command, ok := env.Alias("test")
	if !ok {
		t.Fatal("expected test alias to exist")
	}

	expected := Command{
		Name: "echo",
		Args: []string{
			"hello;",
			"touch",
			"/tmp/marker",
		},
	}

	if !reflect.DeepEqual(command, expected) {
		t.Fatalf(
			"expected %+v, got %+v",
			expected,
			command,
		)
	}
}

func TestResolveFunctions(t *testing.T) {
	cfg := &config.Config{
		Name: "test",
		Functions: map[string]string{
			"deploy": `echo "Building..."
go build ./...
`,
		},
	}

	env, err := Resolve(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, ok := env.Functions["deploy"]
	if !ok {
		t.Fatal("expected deploy function")
	}

	expected := `echo "Building..."
go build ./...
`

	if body != expected {
		t.Fatalf("expected %q, got %q", expected, body)
	}
}

func TestResolveUndefinedVariable(t *testing.T) {
	cfg := &config.Config{
		Name: "test",

		Aliases: map[string]string{
			"kgp": "kubectl get pods -n ${ALIASCTL_TEST_UNDEFINED}",
		},
	}

	_, err := Resolve(cfg)

	if err == nil {
		t.Fatal("expected error for undefined variable")
	}

	expected := `invalid alias "kgp": undefined variable "ALIASCTL_TEST_UNDEFINED"`

	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestResolveFallsBackToProcessEnvironment(t *testing.T) {
	t.Setenv("ALIASCTL_TEST_HOME", "/home/test")

	cfg := &config.Config{
		Name: "test",

		Aliases: map[string]string{
			"home": "ls ${ALIASCTL_TEST_HOME}",
		},
	}

	env, err := Resolve(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := Command{
		Name: "ls",
		Args: []string{"/home/test"},
	}

	if !reflect.DeepEqual(env.Aliases["home"], expected) {
		t.Fatalf("expected %+v, got %+v", expected, env.Aliases["home"])
	}
}

func TestResolveConfigVariableOverridesProcessEnvironment(t *testing.T) {
	t.Setenv("NAMESPACE", "from-env")

	cfg := &config.Config{
		Name: "test",

		Variables: map[string]string{
			"NAMESPACE": "from-config",
		},

		Aliases: map[string]string{
			"ns": "echo ${NAMESPACE}",
		},
	}

	env, err := Resolve(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := env.Aliases["ns"].Args[0]; got != "from-config" {
		t.Fatalf("expected %q, got %q", "from-config", got)
	}
}

func TestResolveDoesNotReexpandSubstitutedValues(t *testing.T) {
	cfg := &config.Config{
		Name: "test",

		Variables: map[string]string{
			"A": "${B}",
			"B": "expanded",
		},

		Aliases: map[string]string{
			"show": "echo ${A}",
		},
	}

	for range 20 {
		env, err := Resolve(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := env.Aliases["show"].Args[0]; got != "${B}" {
			t.Fatalf("expected %q, got %q", "${B}", got)
		}
	}
}

func TestResolveReportsAllUndefinedVariables(t *testing.T) {
	cfg := &config.Config{
		Name: "test",

		Aliases: map[string]string{
			"x": "echo ${ALIASCTL_TEST_A} ${ALIASCTL_TEST_B} ${ALIASCTL_TEST_A}",
		},
	}

	_, err := Resolve(cfg)
	if err == nil {
		t.Fatal("expected error")
	}

	expected := `invalid alias "x": undefined variables "ALIASCTL_TEST_A", "ALIASCTL_TEST_B"`

	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestResolveRejectsShellSyntaxInAlias(t *testing.T) {
	cfg := &config.Config{
		Name: "test",

		Aliases: map[string]string{
			"gofiles": "ls *.go",
		},
	}

	_, err := Resolve(cfg)
	if err == nil {
		t.Fatal("expected shell syntax error")
	}

	expected := `invalid alias "gofiles": shell syntax "*" is not supported; quote it or use a function`

	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}
