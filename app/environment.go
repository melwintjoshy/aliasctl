package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/melwintjoshy/aliasctl/config"
	"github.com/melwintjoshy/aliasctl/resolver"
)

func LoadEnvironment(configPath string) (*resolver.Environment, string, error) {
	path, err := config.ResolvePath(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("could not find configuration: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("could not load configuration: %w", err)
	}

	env, err := environmentFromData(path, data)
	if err != nil {
		return nil, "", err
	}

	return env, path, nil
}

func environmentFromData(path string, data []byte) (*resolver.Environment, error) {
	cfg, err := config.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("could not load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	env, err := resolver.Resolve(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not resolve configuration: %w", err)
	}

	// tool checks run from here so relative check paths work from any subdirectory
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("could not resolve configuration path: %w", err)
	}

	env.Dir = filepath.Dir(absolutePath)

	return env, nil
}

var ErrNotAllowed = errors.New("configuration is not allowed")

// LoadTrustedEnvironment reads the config once and loads it only if those exact bytes are allowed,
// so an edit between the trust check and the load can never slip through.
func LoadTrustedEnvironment(configPath string) (*resolver.Environment, error) {
	absolute, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absolute)
	if err != nil {
		return nil, fmt.Errorf("could not load configuration: %w", err)
	}

	allowed, err := isAllowedContent(absolute, data)
	if err != nil {
		return nil, fmt.Errorf("could not check trust: %w", err)
	}

	if !allowed {
		return nil, ErrNotAllowed
	}

	return environmentFromData(absolute, data)
}
