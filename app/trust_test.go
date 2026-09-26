package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
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

func writeConfigFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestAllowConcurrentWritersKeepEveryEntry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	dir := t.TempDir()

	var paths []string

	for i := range 20 {
		paths = append(paths, writeConfigFile(t, dir, fmt.Sprintf("c%d.yaml", i), fmt.Sprintf("name: c%d\n", i)))
	}

	var wg sync.WaitGroup

	errs := make(chan error, len(paths))

	for _, path := range paths {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if _, err := Allow(path); err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("allow failed: %v", err)
	}

	for _, path := range paths {
		if ok, err := IsAllowed(path); err != nil || !ok {
			t.Fatalf("lost entry for %s (ok=%v err=%v)", path, ok, err)
		}
	}
}

func TestAllowIgnoresLeftoverTempFile(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	path := writeConfigFile(t, t.TempDir(), "aliasctl.yaml", "name: a\n")

	if _, err := Allow(path); err != nil {
		t.Fatal(err)
	}

	// an interrupted write leaves a temp file behind; it must not be read as the list
	leftover := filepath.Join(configHome, "aliasctl", ".allowed-crashed")

	if err := os.WriteFile(leftover, []byte("garbage"), 0600); err != nil {
		t.Fatal(err)
	}

	if ok, err := IsAllowed(path); err != nil || !ok {
		t.Fatalf("expected entry to survive, ok=%v err=%v", ok, err)
	}
}

func TestAllowListIsPrivate(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if _, err := Allow(writeConfigFile(t, t.TempDir(), "aliasctl.yaml", "name: a\n")); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(configHome, "aliasctl", "allowed"))
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}
}
