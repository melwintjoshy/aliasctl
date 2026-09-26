package config

// FormatVersion is the newest aliasctl.yaml format this build understands.
const FormatVersion = 1

type Config struct {
	// optional; a missing value means 1
	Version int `yaml:"version"`

	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Variables   map[string]string `yaml:"variables"`
	Aliases     map[string]string `yaml:"aliases"`
	Functions   map[string]string `yaml:"functions"`

	// fish can't run bash bodies, so fish gets its own
	FunctionsFish map[string]string `yaml:"functions_fish"`

	Tools map[string]ToolSpec `yaml:"tools"`
}
