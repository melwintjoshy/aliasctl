package resolver

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/melwintjoshy/aliasctl/config"
)

func Resolve(cfg *config.Config) (*Environment, error) {
	env := &Environment{
		Name:      cfg.Name,
		Variables: make(map[string]string),
		Aliases:   make(map[string]Command),
		Functions: make(map[string]string),

		FunctionsFish: make(map[string]string),
	}

	for key, value := range cfg.Variables {
		env.Variables[key] = value
	}

	for name, body := range cfg.Functions {
		env.Functions[name] = body
	}

	for name, body := range cfg.FunctionsFish {
		env.FunctionsFish[name] = body
	}

	for name, command := range cfg.Aliases {
		if err := checkShellSyntax(command); err != nil {
			return nil, fmt.Errorf(
				"invalid alias %q: %w",
				name,
				err,
			)
		}

		resolved, err := resolveVariables(command, cfg.Variables)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid alias %q: %w",
				name,
				err,
			)
		}

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

var placeholderPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// single pass so substituted values are never re-expanded; config wins over the process env
func resolveVariables(value string, variables map[string]string) (string, error) {
	var missing []string

	resolved := placeholderPattern.ReplaceAllStringFunc(value, func(match string) string {
		name := placeholderPattern.FindStringSubmatch(match)[1]

		if variableValue, ok := variables[name]; ok {
			return variableValue
		}

		if variableValue, ok := os.LookupEnv(name); ok {
			return variableValue
		}

		if !slices.Contains(missing, name) {
			missing = append(missing, name)
		}

		return match
	})

	if len(missing) == 1 {
		return "", fmt.Errorf("undefined variable %q", missing[0])
	}

	if len(missing) > 1 {
		quoted := make([]string, len(missing))

		for i, name := range missing {
			quoted[i] = strconv.Quote(name)
		}

		return "", fmt.Errorf("undefined variables %s", strings.Join(quoted, ", "))
	}

	return resolved, nil
}
