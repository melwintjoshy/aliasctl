package resolver

type Environment struct {
	Name      string
	Variables map[string]string
	Aliases   map[string]Command
	Functions map[string]string
}

func (e *Environment) Variable(name string) (string, bool) {
	value, ok := e.Variables[name]
	return value, ok
}

func (e *Environment) Alias(name string) (Command, bool) {
	command, ok := e.Aliases[name]
	return command, ok
}
