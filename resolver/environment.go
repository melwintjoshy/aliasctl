package resolver

type Environment struct {
	// directory holding aliasctl.yaml, "" when built without a file
	Dir string

	Name      string
	Variables map[string]string
	Aliases   map[string]Command
	Functions map[string]string

	FunctionsFish map[string]string

	// sorted by name
	Tools []ToolRequirement
}

func (e *Environment) Variable(name string) (string, bool) {
	value, ok := e.Variables[name]
	return value, ok
}

func (e *Environment) Alias(name string) (Command, bool) {
	command, ok := e.Aliases[name]
	return command, ok
}
