package config

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
)

func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	if len(c.Aliases) == 0 {
		return fmt.Errorf("at least one alias is required")
	}

	for _, name := range slices.Sorted(maps.Keys(c.Aliases)) {
		if !isValidAliasName(name) {
			return fmt.Errorf("invalid alias name: %q", name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(c.Functions)) {
		if !isValidIdentifier(name) {
			return fmt.Errorf("invalid function name: %q", name)
		}
	}

	for _, name := range slices.Sorted(maps.Keys(c.Variables)) {
		if !isValidIdentifier(name) {
			return fmt.Errorf("invalid variable name: %q", name)
		}
	}

	return nil
}

func isValidIdentifier(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}

func isValidAliasName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_-]*$`, name)
	return matched
}
