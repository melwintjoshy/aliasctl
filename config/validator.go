package config

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"time"

	"github.com/melwintjoshy/aliasctl/versions"
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

	for _, name := range slices.Sorted(maps.Keys(c.Tools)) {
		if !toolNamePattern.MatchString(name) {
			return fmt.Errorf("invalid tool name: %q", name)
		}

		if _, err := versions.ParseConstraint(c.Tools[name].Version); err != nil {
			return fmt.Errorf("invalid tool %q: %w", name, err)
		}

		if err := validateTimeout(c.Tools[name].Timeout); err != nil {
			return fmt.Errorf("invalid tool %q: %w", name, err)
		}
	}

	for _, name := range slices.Sorted(maps.Keys(c.Variables)) {
		if !isValidIdentifier(name) {
			return fmt.Errorf("invalid variable name: %q", name)
		}
	}

	return nil
}

func validateTimeout(text string) error {
	if text == "" {
		return nil
	}

	timeout, err := time.ParseDuration(text)
	if err != nil {
		return fmt.Errorf("invalid timeout %q, use a duration like 10s", text)
	}

	if timeout <= 0 {
		return fmt.Errorf("timeout %q must be positive", text)
	}

	return nil
}

var (
	identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	aliasNamePattern  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

	// binary names like python3 and docker-compose
	toolNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

func isValidIdentifier(name string) bool {
	return identifierPattern.MatchString(name)
}

func isValidAliasName(name string) bool {
	return aliasNamePattern.MatchString(name)
}
