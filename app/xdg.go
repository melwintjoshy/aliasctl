package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// xdgPath is $envVar/aliasctl/<parts...>, or the xdg default under the home directory when the variable is unset.
func xdgPath(envVar string, fallback []string, parts ...string) (string, error) {
	base := os.Getenv(envVar)

	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not find home directory: %w", err)
		}

		base = filepath.Join(append([]string{home}, fallback...)...)
	}

	return filepath.Join(append([]string{base, "aliasctl"}, parts...)...), nil
}
