package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestAllowTracksContent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path := filepath.Join(t.TempDir(), "aliasctl.yaml")

	if err := os.WriteFile(path, []byte("name: a\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if ok, err := isAllowed(path); err != nil || ok {
		t.Fatalf("expected unknown config to be refused, ok=%v err=%v", ok, err)
	}

	if _, err := Allow(path); err != nil {
		t.Fatal(err)
	}

	if ok, err := isAllowed(path); err != nil || !ok {
		t.Fatalf("expected allowed config, ok=%v err=%v", ok, err)
	}

	if err := os.WriteFile(path, []byte("name: changed\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if ok, err := isAllowed(path); err != nil || ok {
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
		if ok, err := isAllowed(path); err != nil || !ok {
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

	if ok, err := isAllowed(path); err != nil || !ok {
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

func TestDenyListAndPrune(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	dir := t.TempDir()

	kept := writeConfigFile(t, dir, "kept.yaml", "name: kept\n")
	edited := writeConfigFile(t, dir, "edited.yaml", "name: edited\n")
	deleted := writeConfigFile(t, dir, "deleted.yaml", "name: deleted\n")
	denied := writeConfigFile(t, dir, "denied.yaml", "name: denied\n")

	for _, path := range []string{kept, edited, deleted, denied} {
		if _, err := Allow(path); err != nil {
			t.Fatal(err)
		}
	}

	if _, removed, err := Deny(denied); err != nil || !removed {
		t.Fatalf("expected deny to remove the entry, removed=%v err=%v", removed, err)
	}

	if _, removed, err := Deny(denied); err != nil || removed {
		t.Fatalf("expected a second deny to be a no-op, removed=%v err=%v", removed, err)
	}

	writeConfigFile(t, dir, "edited.yaml", "name: edited again\n")

	if err := os.Remove(deleted); err != nil {
		t.Fatal(err)
	}

	entries, err := ListAllowed()
	if err != nil {
		t.Fatal(err)
	}

	expected := []AllowedEntry{
		{Path: deleted, Status: TrustMissing},
		{Path: edited, Status: TrustChanged},
		{Path: kept, Status: TrustOK},
	}

	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("expected %+v, got %+v", expected, entries)
	}

	pruned, err := Prune()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(pruned, []string{deleted}) {
		t.Fatalf("expected only the deleted config to be pruned, got %v", pruned)
	}

	entries, _ = ListAllowed()

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries after prune, got %+v", entries)
	}
}

func TestLoadTrustedEnvironment(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path := writeConfigFile(t, t.TempDir(), "aliasctl.yaml", "name: a\naliases:\n  hi: echo trusted\n")

	if _, err := LoadTrustedEnvironment(path); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("expected an unknown config to be refused, got %v", err)
	}

	if _, err := Allow(path); err != nil {
		t.Fatal(err)
	}

	env, err := LoadTrustedEnvironment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if env.Aliases["hi"].Args[0] != "trusted" {
		t.Fatalf("unexpected alias %+v", env.Aliases["hi"])
	}

	writeConfigFile(t, filepath.Dir(path), "aliasctl.yaml", "name: a\naliases:\n  hi: echo edited\n")

	if _, err := LoadTrustedEnvironment(path); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("expected an edited config to be refused, got %v", err)
	}
}

// reads the file the way the hook does, then asks whether those bytes are allowed
func isAllowed(path string) (bool, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}

	data, err := os.ReadFile(absolute)
	if err != nil {
		return false, err
	}

	return isAllowedContent(absolute, data)
}

func TestTrustCheckNeverWritesToConfigHome(t *testing.T) {
	// the hook checks trust on every cd, so a config home nobody can write to must mean "not allowed", not an error
	home := filepath.Join(t.TempDir(), "readonly")

	if err := os.Mkdir(home, 0500); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chmod(home, 0700) })

	t.Setenv("XDG_CONFIG_HOME", home)

	path := filepath.Join(t.TempDir(), "aliasctl.yaml")

	if err := os.WriteFile(path, []byte("name: demo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if ok, err := isAllowed(path); err != nil || ok {
		t.Fatalf("expected not allowed without an error, got %v, %v", ok, err)
	}

	if _, err := LoadTrustedEnvironment(path); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("expected ErrNotAllowed, got %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, "aliasctl")); !os.IsNotExist(err) {
		t.Fatal("a read-only trust check created the allow list directory")
	}
}
