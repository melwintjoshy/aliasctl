package resolver

import (
	"fmt"
	"strings"

	"github.com/melwintjoshy/aliasctl/config"
)

func Resolve(cfg *config.Config) (*Environment, error) {
	env := &Environment{
		Name:      cfg.Name,
		Variables: make(map[string]string),
		Aliases:   make(map[string]Command),
	}

	for key, value := range cfg.Variables {
		env.Variables[key] = value
	}

	for name, command := range cfg.Aliases {
		resolved := resolveVariables(command, cfg.Variables)

		parsed, err := parseCommand(resolved)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid alias %q: %w",
				name,
				err,
			)
		}

		env.Aliases[name] = parsed
	}

	return env, nil
}

func resolveVariables(value string, variables map[string]string) string {
	for key, variableValue := range variables {
		placeholder := "${" + key + "}"
		value = strings.ReplaceAll(value, placeholder, variableValue)
	}

	return value
}
