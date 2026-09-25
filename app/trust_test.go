package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllowTracksContent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path := filepath.Join(t.TempDir(), "aliasctl.yaml")

	if err := os.WriteFile(path, []byte("name: a\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if ok, err := IsAllowed(path); err != nil || ok {
		t.Fatalf("expected unknown config to be refused, ok=%v err=%v", ok, err)
	}

	if _, err := Allow(path); err != nil {
		t.Fatal(err)
	}

	if ok, err := IsAllowed(path); err != nil || !ok {
		t.Fatalf("expected allowed config, ok=%v err=%v", ok, err)
	}

	if err := os.WriteFile(path, []byte("name: changed\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if ok, err := IsAllowed(path); err != nil || ok {
		t.Fatalf("expected changed config to be refused, ok=%v err=%v", ok, err)
	}
}
