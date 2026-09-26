package resolver

import (
	"reflect"
	"testing"
	"time"

	"github.com/melwintjoshy/aliasctl/config"
)

func TestResolveTools(t *testing.T) {
	cfg := &config.Config{
		Name: "test",

		Variables: map[string]string{
			"CTX": "prod",
		},

		Aliases: map[string]string{"k": "kubectl"},

		Tools: map[string]config.ToolSpec{
			"terraform": {Version: ">=1.6"},
			"go":        {Version: "1.22"},
			"mytool":    {Version: "2", Check: `mytool version --context ${CTX} "long name"`},
			"swift":     {Version: "*", Timeout: "20s"},
		},
	}

	env, err := Resolve(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []ToolRequirement{
		{Name: "go", Rule: "1.22", Check: Command{Name: "go", Args: []string{"version"}}},
		{
			Name:  "mytool",
			Rule:  "2",
			Check: Command{Name: "mytool", Args: []string{"version", "--context", "prod", "long name"}},
		},
		{Name: "swift", Rule: "*", Check: Command{Name: "swift", Args: []string{"--version"}}, Timeout: 20 * time.Second},
		{Name: "terraform", Rule: ">=1.6", Check: Command{Name: "terraform", Args: []string{"--version"}}},
	}

	if !reflect.DeepEqual(env.Tools, expected) {
		t.Fatalf("expected %+v, got %+v", expected, env.Tools)
	}
}

func TestResolveToolsRejectsShellSyntaxInCheck(t *testing.T) {
	cfg := &config.Config{
		Name:    "test",
		Aliases: map[string]string{"k": "kubectl"},
		Tools: map[string]config.ToolSpec{
			"mytool": {Version: "1", Check: "mytool --version | head -1"},
		},
	}

	_, err := Resolve(cfg)
	if err == nil {
		t.Fatal("expected shell syntax error")
	}

	expected := `invalid tool "mytool": shell syntax "|" is not supported; quote it, checks run without a shell`

	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestDefaultCheckIsNotShared(t *testing.T) {
	first := defaultCheck("kubectl")
	first.Args[0] = "mutated"

	if defaultCheck("kubectl").Args[0] != "version" {
		t.Fatal("builtin check args were mutated through a returned command")
	}
}
