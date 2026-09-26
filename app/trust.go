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

func readAllowed() (map[string]string, error) {
	path, err := allowedFile()
	if err != nil {
		return nil, err
	}

	allowed := make(map[string]string)

	file, err := os.Open(path)
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

// Allow trusts the config's current content; any later edit needs a new allow.
func Allow(path string) (string, error) {
	absolute, hash, err := hashConfig(path)
	if err != nil {
		return "", fmt.Errorf("could not read configuration: %w", err)
	}

	allowed, err := readAllowed()
	if err != nil {
		return "", fmt.Errorf("could not read allow list: %w", err)
	}

	allowed[absolute] = hash

	listPath, err := allowedFile()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(listPath), 0700); err != nil {
		return "", fmt.Errorf("could not create allow list: %w", err)
	}

	var content strings.Builder

	for _, configPath := range slices.Sorted(maps.Keys(allowed)) {
		fmt.Fprintf(&content, "%s %s\n", allowed[configPath], configPath)
	}

	if err := os.WriteFile(listPath, []byte(content.String()), 0600); err != nil {
		return "", fmt.Errorf("could not write allow list: %w", err)
	}

	return absolute, nil
}

func IsAllowed(path string) (bool, error) {
	absolute, hash, err := hashConfig(path)
	if err != nil {
		return false, err
	}

	allowed, err := readAllowed()
	if err != nil {
		return false, err
	}

	return allowed[absolute] == hash, nil
}
