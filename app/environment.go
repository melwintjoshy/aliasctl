package app

import (
	"fmt"
	"path/filepath"

	"github.com/melwintjoshy/aliasctl/config"
	"github.com/melwintjoshy/aliasctl/resolver"
)

func LoadEnvironment(configPath string) (*resolver.Environment, string, error) {
	path, err := config.ResolvePath(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("could not find configuration: %w", err)
	}

	loader := config.Loader{}
	cfg, err := loader.Load(path)
	if err != nil {
		return nil, "", fmt.Errorf("could not load configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, "", fmt.Errorf("invalid configuration: %w", err)
	}

	env, err := resolver.Resolve(cfg)
	if err != nil {
		return nil, "", fmt.Errorf("could not resolve configuration: %w", err)
	}

	// tool checks run from here so relative check paths work from any subdirectory
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, "", fmt.Errorf("could not resolve configuration path: %w", err)
	}

	env.Dir = filepath.Dir(absolutePath)

	return env, path, nil
}
