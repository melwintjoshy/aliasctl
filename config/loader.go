package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ConfigFileName = "aliasctl.yaml"

type Loader struct{}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return Parse(data)
}

// Parse decodes config bytes; callers that check the bytes first (trust) parse exactly what they checked.
func Parse(data []byte) (*Config, error) {
	// checked before the strict decode, so a newer file says "upgrade" instead of "unknown field"
	if err := checkFormatVersion(data); err != nil {
		return nil, err
	}

	var cfg Config

	// reject unknown keys so a typo like "alias:" fails instead of being ignored
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err := decoder.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	return &cfg, nil
}

func FindConfig() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		path := filepath.Join(dir, ConfigFileName)

		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)

		// We've reached the filesystem root.
		if parent == dir {
			break
		}

		dir = parent
	}

	return "", fmt.Errorf("%s not found", ConfigFileName)
}

func (Loader) Load(path string) (*Config, error) {
	return Load(path)
}

func checkFormatVersion(data []byte) error {
	var header struct {
		Version any `yaml:"version"`
	}

	// a broken document is reported by the strict decode with its line number
	if err := yaml.Unmarshal(data, &header); err != nil || header.Version == nil {
		return nil
	}

	version, ok := header.Version.(int)
	if !ok || version < 1 {
		return fmt.Errorf("version must be a positive whole number, got %v", header.Version)
	}

	if version > FormatVersion {
		return fmt.Errorf(
			"aliasctl.yaml uses format version %d; this aliasctl supports %d, upgrade it",
			version,
			FormatVersion,
		)
	}

	return nil
}
