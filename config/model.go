package config

type Config struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Variables   map[string]string `yaml:"variables"`
	Aliases     map[string]string `yaml:"aliases"`
	Functions   map[string]string `yaml:"functions"`

	// fish can't run bash bodies, so fish gets its own
	FunctionsFish map[string]string `yaml:"functions_fish"`
}
