package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const stampMaxAge = 30 * 24 * time.Hour

func stampDir() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")

	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not find home directory: %w", err)
		}

		base = filepath.Join(home, ".local", "state")
	}

	return filepath.Join(base, "aliasctl", "stamps"), nil
}

// NewStamp creates an empty file whose mtime marks when a config was loaded.
// the hook compares the config against it with the shell's -nt, so no process runs per prompt.
func NewStamp() (string, error) {
	dir, err := stampDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("could not create stamp directory: %w", err)
	}

	pruneStamps(dir, time.Now().Add(-stampMaxAge))

	file, err := os.CreateTemp(dir, "loaded-*")
	if err != nil {
		return "", fmt.Errorf("could not create stamp: %w", err)
	}

	return file.Name(), file.Close()
}

// shells that were killed never ran their unload, so their stamps are cleaned up here
func pruneStamps(dir string, before time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "loaded-") {
			continue
		}

		if info, err := entry.Info(); err == nil && info.ModTime().Before(before) {
			os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}
