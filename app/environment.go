package app

import (
	"fmt"

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

	return env, path, nil
}
