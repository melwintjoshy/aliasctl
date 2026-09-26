package resolver

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/melwintjoshy/aliasctl/config"
)

// ToolRequirement is a tool the environment needs and how to ask it for its version.
type ToolRequirement struct {
	Name  string
	Rule  string
	Check Command

	// zero means the checker's default
	Timeout time.Duration

	// name mise knows the tool by
	Mise string
}

// tools whose --version flag doesn't work or prints the wrong thing
var builtinChecks = map[string]Command{
	"go":      {Name: "go", Args: []string{"version"}},
	"kubectl": {Name: "kubectl", Args: []string{"version", "--client"}},
	"helm":    {Name: "helm", Args: []string{"version", "--short"}},
	"java":    {Name: "java", Args: []string{"-version"}},
}

func defaultCheck(name string) Command {
	if command, ok := builtinChecks[name]; ok {
		return Command{Name: command.Name, Args: slices.Clone(command.Args)}
	}

	return Command{Name: name, Args: []string{"--version"}}
}

// custom checks follow alias rules: no shell syntax, ${VAR} allowed
func resolveTools(cfg *config.Config) ([]ToolRequirement, error) {
	requirements := make([]ToolRequirement, 0, len(cfg.Tools))

	for _, name := range slices.Sorted(maps.Keys(cfg.Tools)) {
		spec := cfg.Tools[name]
		check := defaultCheck(name)

		if spec.Check != "" {
			parsed, err := resolveCommand(spec.Check, cfg.Variables, "quote it, checks run without a shell")
			if err != nil {
				return nil, fmt.Errorf("invalid tool %q: %w", name, err)
			}

			check = parsed
		}

		// validation already rejected anything that does not parse
		timeout, _ := time.ParseDuration(spec.Timeout)

		requirements = append(requirements, ToolRequirement{
			Name:    name,
			Rule:    spec.Version,
			Check:   check,
			Timeout: timeout,
			Mise:    miseName(name, spec.Mise),
		})
	}

	return requirements, nil
}

func miseName(name, override string) string {
	if override != "" {
		return override
	}

	return name
}
