package shell

import (
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/resolver"
)

func TestRenderBanner(t *testing.T) {
	env := &resolver.Environment{
		Name: "k8s",
	}

	output := RenderBanner(env, "./aliasctl.yaml")

	if !strings.Contains(output, "Environment: k8s") {
		t.Fatal("expected environment name in banner")
	}

	if !strings.Contains(output, "Config:      ./aliasctl.yaml") {
		t.Fatal("expected config path in banner")
	}

	if !strings.Contains(output, "█████████") {
		t.Fatal("expected AliasCtl banner")
	}
}
