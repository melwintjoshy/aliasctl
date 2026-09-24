package resolver

type Environment struct {
	Name      string
	Variables map[string]string
	Aliases   map[string]Command
}
