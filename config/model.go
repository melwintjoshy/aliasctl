package config

type Config struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Variables   map[string]string `yaml:"variables"`
	Aliases     map[string]string `yaml:"aliases"`
	Functions   map[string]string `yaml:"functions"`
}
