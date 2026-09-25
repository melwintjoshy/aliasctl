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
		fmt.Fprintf(
			&output,
			"alias %s=%s\n",
			name,
			shellQuote(aliasValue(command)),
		)
	}

	for name, body := range env.Functions {
		body = strings.TrimRight(body, "\n")
		fmt.Fprintf(&output, "%s() {\n%s\n}\n", name, body)
	}

	return output.String(), nil
}

// bash re-parses alias values on use, so each token is quoted on its own
func aliasValue(command resolver.Command) string {
	tokens := make([]string, 0, len(command.Args)+1)

	tokens = append(tokens, shellQuote(command.Name))

	for _, arg := range command.Args {
		tokens = append(tokens, shellQuote(arg))
	}

	return strings.Join(tokens, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
