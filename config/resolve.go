package config

import "fmt"

func ResolvePath(explicitPath string) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}

	path, err := FindConfig()
	if err != nil {
		return "", fmt.Errorf("could not find configuration: %w", err)
	}

	return path, nil
}
