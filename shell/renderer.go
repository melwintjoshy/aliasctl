package shell

import "github.com/melwintjoshy/aliasctl/resolver"

type Renderer interface {
	Render(env *resolver.Environment) (string, error)
}
