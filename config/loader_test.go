package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ConfigFileName)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	path := writeConfig(t, `name: test
alias:
  k: kubectl
`)

	if _, err := Load(path); err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestLoadParsesConfig(t *testing.T) {
	path := writeConfig(t, `name: test
variables:
  NAMESPACE: app-prod
aliases:
  k: kubectl
functions:
  hello: echo hello
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Name != "test" || cfg.Aliases["k"] != "kubectl" ||
		cfg.Variables["NAMESPACE"] != "app-prod" ||
		cfg.Functions["hello"] != "echo hello" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadEmptyFile(t *testing.T) {
	path := writeConfig(t, "")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected empty config to fail validation")
	}
}

func TestLoadToolForms(t *testing.T) {
	path := writeConfig(t, `name: test
aliases:
  k: kubectl
tools:
  go: 1.20
  kubectl: ">=1.29"
  mytool:
    version: "2.1"
    check: "mytool version --short"
    timeout: 20s
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]ToolSpec{
		"go":      {Version: "1.20"},
		"kubectl": {Version: ">=1.29"},
		"mytool":  {Version: "2.1", Check: "mytool version --short", Timeout: "20s"},
	}

	if !reflect.DeepEqual(cfg.Tools, expected) {
		t.Fatalf("expected %+v, got %+v", expected, cfg.Tools)
	}
}

func TestLoadRejectsUnknownToolField(t *testing.T) {
	path := writeConfig(t, `name: test
aliases:
  k: kubectl
tools:
  go:
    versoin: "1.22"
`)

	if _, err := Load(path); err == nil {
		t.Fatal("expected error for unknown tool field")
	}
}

func TestLoadRejectsToolList(t *testing.T) {
	path := writeConfig(t, `name: test
aliases:
  k: kubectl
tools:
  go: ["1.22"]
`)

	if _, err := Load(path); err == nil {
		t.Fatal("expected error for a list-valued tool")
	}
}
