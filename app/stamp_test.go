package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewStampPrunesOldStamps(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)

	first, err := NewStamp()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(first, filepath.Join(state, "aliasctl", "stamps", "loaded-")) {
		t.Fatalf("unexpected stamp path %s", first)
	}

	old := time.Now().Add(-stampMaxAge - time.Hour)

	if err := os.Chtimes(first, old, old); err != nil {
		t.Fatal(err)
	}

	unrelated := filepath.Join(state, "aliasctl", "stamps", "keep-me")

	if err := os.WriteFile(unrelated, nil, 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chtimes(unrelated, old, old); err != nil {
		t.Fatal(err)
	}

	second, err := NewStamp()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatal("expected the old stamp to be pruned")
	}

	if _, err := os.Stat(second); err != nil {
		t.Fatalf("expected the new stamp to exist: %v", err)
	}

	if _, err := os.Stat(unrelated); err != nil {
		t.Fatal("pruning removed a file that is not a stamp")
	}
}
