package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// stamps from hooks that never said which shell owns them are only pruned by age
const stampMaxAge = 30 * 24 * time.Hour

const stampPrefix = "loaded-"

// SeenSuffix names the marker next to a stamp that records the last failed reload attempt.
const SeenSuffix = ".seen"

func stampDir() (string, error) {
	return xdgPath("XDG_STATE_HOME", []string{".local", "state"}, "stamps")
}

// NewStamp creates an empty file whose mtime marks when a config was loaded.
// the hook compares the config against it with the shell's -nt, so no process runs per prompt.
// shellPID is the shell that loaded it, or 0 when the hook did not say.
func NewStamp(shellPID int) (string, error) {
	dir, err := stampDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("could not create stamp directory: %w", err)
	}

	pruneStamps(dir, time.Now().Add(-stampMaxAge))

	file, err := os.CreateTemp(dir, stampPrefix+strconv.Itoa(shellPID)+"-*")
	if err != nil {
		return "", fmt.Errorf("could not create stamp: %w", err)
	}

	return file.Name(), file.Close()
}

// RemoveStamp drops a stamp and its seen marker; a missing file is fine.
func RemoveStamp(path string) {
	os.Remove(path)
	os.Remove(path + SeenSuffix)
}

// shells that were killed never ran their unload, so their stamps are cleaned up here; a stamp whose
// shell still runs is kept whatever its age, since removing it would make that shell reload on its next prompt
func pruneStamps(dir string, before time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), stampPrefix) {
			continue
		}

		if pid, ok := stampOwner(entry.Name()); ok {
			if !processAlive(pid) {
				os.Remove(filepath.Join(dir, entry.Name()))
			}

			continue
		}

		if info, err := entry.Info(); err == nil && info.ModTime().Before(before) {
			os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

// the name is loaded-<pid>-<random>, optionally followed by the seen suffix; ok is false for older stamps without a pid
func stampOwner(name string) (int, bool) {
	rest := strings.TrimPrefix(name, stampPrefix)

	owner, _, found := strings.Cut(rest, "-")
	if !found {
		return 0, false
	}

	pid, err := strconv.Atoi(owner)
	if err != nil || pid <= 0 {
		return 0, false
	}

	return pid, true
}
