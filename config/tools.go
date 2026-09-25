package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ToolSpec is a required tool: a version rule, an optional command that prints its version
// and an optional time limit for it.
type ToolSpec struct {
	Version string `yaml:"version"`
	Check   string `yaml:"check"`
	Timeout string `yaml:"timeout"`
}

// UnmarshalYAML accepts the short form `go: "1.22"` as well as a mapping.
func (s *ToolSpec) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		return node.Decode(&s.Version)
	}

	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: a tool must be a version string or a mapping", node.Line)
	}

	// node.Decode does not inherit the loader's KnownFields, so unknown keys are checked here
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]

		switch key.Value {
		case "version", "check", "timeout":
		default:
			return fmt.Errorf("line %d: field %s not found in tool", key.Line, key.Value)
		}
	}

	type plain ToolSpec

	var decoded plain

	if err := node.Decode(&decoded); err != nil {
		return err
	}

	*s = ToolSpec(decoded)

	return nil
}
