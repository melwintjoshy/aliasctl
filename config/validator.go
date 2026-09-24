package config

import (
	"fmt"
	"regexp"
)

func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	if len(c.Aliases) == 0 {
		return fmt.Errorf("at least one alias is required")
	}

	for name := range c.Aliases {
		if !isValidAliasName(name) {
			return fmt.Errorf("invalid alias name: %q", name)
		}
	}

	return nil
}

func isValidAliasName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_-]*$`, name)
	return matched
}
