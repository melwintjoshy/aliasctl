package app

import (
	"io"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/melwintjoshy/aliasctl/internal/fakemise"
	"github.com/melwintjoshy/aliasctl/mise"
	"github.com/melwintjoshy/aliasctl/resolver"
)

func TestActivateToolsPicksHighestMatchingInstall(t *testing.T) {
	fake := fakemise.Install(t)

	for _, spec := range []string{"go@1.21.13", "go@1.22.6", "go@1.22.10", "go@1.23.2", "aqua:hashicorp/terraform@1.9.8"} {
		if err := (mise.CLI{}).Install(spec, io.Discard, io.Discard); err != nil {
			t.Fatal(err)
		}
	}

	env := &resolver.Environment{
		Tools: []resolver.ToolRequirement{
			{Name: "go", Rule: "1.22", Mise: "go"},
			{Name: "kubectl", Rule: "*", Mise: "kubectl"},
			{Name: "terraform", Rule: ">=1.6", Mise: "aqua:hashicorp/terraform"},
		},
	}

	if err := ActivateTools(env, mise.CLI{}); err != nil {
		t.Fatal(err)
	}

	// kubectl has no mise install, so it is left to PATH
	expected := []string{
		filepath.Join(fake.Root, "installs", "go", "1.22.10", "bin"),
		filepath.Join(fake.Root, "installs", "aqua_hashicorp_terraform", "1.9.8", "bin"),
	}

	if !reflect.DeepEqual(env.PathPrepend, expected) {
		t.Fatalf("expected %v, got %v", expected, env.PathPrepend)
	}
}

func TestActivateToolsIsANoOpWithoutMiseOrTools(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	env := &resolver.Environment{
		Tools: []resolver.ToolRequirement{{Name: "go", Rule: "*", Mise: "go"}},
	}

	if err := ActivateTools(env, mise.CLI{}); err != nil || env.PathPrepend != nil {
		t.Fatalf("expected nothing without mise, got %v, %v", env.PathPrepend, err)
	}

	fake := fakemise.Install(t)

	if err := ActivateTools(&resolver.Environment{}, mise.CLI{}); err != nil {
		t.Fatal(err)
	}

	if calls := fake.Calls(t); calls != nil {
		t.Fatalf("expected no mise calls without tools, got %v", calls)
	}
}
