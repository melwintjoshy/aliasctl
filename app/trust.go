package app

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

// allowedFile lists trusted configs as "<sha256> <absolute path>", one per line.
func allowedFile() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")

	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not find home directory: %w", err)
		}

		base = filepath.Join(home, ".config")
	}

	return filepath.Join(base, "aliasctl", "allowed"), nil
}

func hashConfig(path string) (string, string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}

	data, err := os.ReadFile(absolute)
	if err != nil {
		return "", "", err
	}

	sum := sha256.Sum256(data)

	return absolute, hex.EncodeToString(sum[:]), nil
}

// withAllowList runs fn on the list under a lock and saves it when fn reports a change.
func withAllowList(exclusive bool, fn func(allowed map[string]string) (bool, error)) error {
	listPath, err := allowedFile()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(listPath), 0700); err != nil {
		return fmt.Errorf("could not create allow list directory: %w", err)
	}

	// a separate lock file, since the list itself is replaced by rename on every write
	lock, err := os.OpenFile(listPath+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("could not open allow list lock: %w", err)
	}

	defer lock.Close()

	mode := syscall.LOCK_SH
	if exclusive {
		mode = syscall.LOCK_EX
	}

	if err := syscall.Flock(int(lock.Fd()), mode); err != nil {
		return fmt.Errorf("could not lock allow list: %w", err)
	}

	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	allowed, err := readAllowed(listPath)
	if err != nil {
		return fmt.Errorf("could not read allow list: %w", err)
	}

	changed, err := fn(allowed)
	if err != nil || !changed {
		return err
	}

	if !exclusive {
		return fmt.Errorf("allow list changed under a shared lock")
	}

	if err := writeAllowed(listPath, allowed); err != nil {
		return fmt.Errorf("could not write allow list: %w", err)
	}

	return nil
}

func readAllowed(listPath string) (map[string]string, error) {
	allowed := make(map[string]string)

	file, err := os.Open(listPath)
	if os.IsNotExist(err) {
		return allowed, nil
	}

	if err != nil {
		return nil, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		hash, configPath, ok := strings.Cut(scanner.Text(), " ")
		if ok {
			allowed[configPath] = hash
		}
	}

	return allowed, scanner.Err()
}

// temp file plus rename, so a crash mid-write leaves the old list intact
func writeAllowed(listPath string, allowed map[string]string) error {
	var content strings.Builder

	for _, configPath := range slices.Sorted(maps.Keys(allowed)) {
		fmt.Fprintf(&content, "%s %s\n", allowed[configPath], configPath)
	}

	temp, err := os.CreateTemp(filepath.Dir(listPath), ".allowed-*")
	if err != nil {
		return err
	}

	tempPath := temp.Name()

	defer os.Remove(tempPath)

	if _, err := temp.WriteString(content.String()); err != nil {
		temp.Close()
		return err
	}

	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}

	if err := temp.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, listPath)
}

// Allow trusts the config's current content; any later edit needs a new allow.
func Allow(path string) (string, error) {
	absolute, hash, err := hashConfig(path)
	if err != nil {
		return "", fmt.Errorf("could not read configuration: %w", err)
	}

	err = withAllowList(true, func(allowed map[string]string) (bool, error) {
		allowed[absolute] = hash
		return true, nil
	})

	return absolute, err
}

func IsAllowed(path string) (bool, error) {
	absolute, hash, err := hashConfig(path)
	if err != nil {
		return false, err
	}

	var ok bool

	err = withAllowList(false, func(allowed map[string]string) (bool, error) {
		ok = allowed[absolute] == hash
		return false, nil
	})

	return ok, err
}

// Deny forgets the config; it reports false when it was not allowed.
func Deny(path string) (string, bool, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", false, err
	}

	var removed bool

	err = withAllowList(true, func(allowed map[string]string) (bool, error) {
		_, removed = allowed[absolute]
		delete(allowed, absolute)

		return removed, nil
	})

	return absolute, removed, err
}

type TrustStatus string

const (
	TrustOK      TrustStatus = "ok"
	TrustChanged TrustStatus = "changed"
	TrustMissing TrustStatus = "missing"
)

type AllowedEntry struct {
	Path   string
	Status TrustStatus
}

// ListAllowed reports every trusted config and whether it still matches what was allowed.
func ListAllowed() ([]AllowedEntry, error) {
	var entries []AllowedEntry

	err := withAllowList(false, func(allowed map[string]string) (bool, error) {
		for _, configPath := range slices.Sorted(maps.Keys(allowed)) {
			entries = append(entries, AllowedEntry{
				Path:   configPath,
				Status: trustStatus(configPath, allowed[configPath]),
			})
		}

		return false, nil
	})

	return entries, err
}

func trustStatus(configPath, hash string) TrustStatus {
	_, current, err := hashConfig(configPath)

	switch {
	case os.IsNotExist(err):
		return TrustMissing
	case err != nil || current != hash:
		return TrustChanged
	}

	return TrustOK
}

// Prune drops entries whose config file no longer exists and returns their paths.
func Prune() ([]string, error) {
	var pruned []string

	err := withAllowList(true, func(allowed map[string]string) (bool, error) {
		for _, configPath := range slices.Sorted(maps.Keys(allowed)) {
			if trustStatus(configPath, allowed[configPath]) == TrustMissing {
				pruned = append(pruned, configPath)
				delete(allowed, configPath)
			}
		}

		return len(pruned) > 0, nil
	})

	return pruned, err
}
