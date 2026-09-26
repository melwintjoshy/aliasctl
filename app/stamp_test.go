package app

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewStampPrunesOldStamps(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)

	dir := filepath.Join(state, "aliasctl", "stamps")

	first, err := NewStamp(0)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(first, filepath.Join(dir, "loaded-0-")) {
		t.Fatalf("unexpected stamp path %s", first)
	}

	old := time.Now().Add(-stampMaxAge - time.Hour)

	if err := os.Chtimes(first, old, old); err != nil {
		t.Fatal(err)
	}

	unrelated := filepath.Join(dir, "keep-me")

	if err := os.WriteFile(unrelated, nil, 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chtimes(unrelated, old, old); err != nil {
		t.Fatal(err)
	}

	second, err := NewStamp(0)
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

func TestPruneStampsFollowsTheOwningShell(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)

	// this process stands in for a shell that is still running
	alive, err := NewStamp(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}

	old := time.Now().Add(-stampMaxAge - time.Hour)

	for _, path := range []string{alive, alive + SeenSuffix} {
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatal(err)
		}

		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}

	// no process can have this pid, so its stamp is stale however new it is
	dead, err := NewStamp(math.MaxInt32)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(dead+SeenSuffix, nil, 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := NewStamp(0); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{alive, alive + SeenSuffix} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to survive while its shell runs: %v", filepath.Base(path), err)
		}
	}

	for _, path := range []string{dead, dead + SeenSuffix} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be pruned once its shell is gone", filepath.Base(path))
		}
	}
}

func TestRemoveStampDropsSeenMarker(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	stamp, err := NewStamp(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(stamp+SeenSuffix, nil, 0600); err != nil {
		t.Fatal(err)
	}

	RemoveStamp(stamp)
	RemoveStamp(stamp)

	for _, path := range []string{stamp, stamp + SeenSuffix} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed", filepath.Base(path))
		}
	}
}
