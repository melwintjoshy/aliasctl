package shell

import (
	"fmt"
	"strings"

	"github.com/melwintjoshy/aliasctl/resolver"
)

type BashRenderer struct{}

func (r BashRenderer) Render(env *resolver.Environment) (string, error) {
	var output strings.Builder

	for key, value := range env.Variables {
		fmt.Fprintf(
			&output,
			"export %s=%s\n",
			key,
			shellQuote(value),
		)
	}

	for name, command := range env.Aliases {
		value := command.Name

		if len(command.Args) > 0 {
			value += " " + strings.Join(command.Args, " ")
		}

		fmt.Fprintf(
			&output,
			"alias %s=%s\n",
			name,
			shellQuote(value),
		)
	}

	return output.String(), nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
