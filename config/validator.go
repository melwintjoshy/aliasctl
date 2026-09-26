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

	if len(c.Aliases) == 0 && len(c.Functions) == 0 && len(c.FunctionsFish) == 0 {
		return fmt.Errorf("at least one alias or function is required")
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

		if _, ok := c.Aliases[name]; ok {
			return fmt.Errorf("%q is defined as both an alias and a function", name)
		}
	}

	for _, name := range slices.Sorted(maps.Keys(c.FunctionsFish)) {
		if !isValidIdentifier(name) {
			return fmt.Errorf("invalid fish function name: %q", name)
		}

		if _, ok := c.Aliases[name]; ok {
			return fmt.Errorf("%q is defined as both an alias and a fish function", name)
		}
	}

	for _, name := range slices.Sorted(maps.Keys(c.Variables)) {
		if !isValidIdentifier(name) {
			return fmt.Errorf("invalid variable name: %q", name)
		}
	}

	return nil
}

var (
	identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	aliasNamePattern  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)
)

func isValidIdentifier(name string) bool {
	return identifierPattern.MatchString(name)
}

func isValidAliasName(name string) bool {
	return aliasNamePattern.MatchString(name)
}
