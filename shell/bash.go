package shell

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/melwintjoshy/aliasctl/resolver"
)

type BashRenderer struct{}

func (r BashRenderer) Render(env *resolver.Environment) (string, error) {
	var output strings.Builder

	for _, key := range slices.Sorted(maps.Keys(env.Variables)) {
		fmt.Fprintf(
			&output,
			"export %s=%s\n",
			key,
			shellQuote(env.Variables[key]),
		)
	}

	for _, name := range slices.Sorted(maps.Keys(env.Aliases)) {
		fmt.Fprintf(
			&output,
			"alias %s=%s\n",
			name,
			shellQuote(aliasValue(env.Aliases[name])),
		)
	}

	for _, name := range slices.Sorted(maps.Keys(env.Functions)) {
		body := strings.TrimRight(env.Functions[name], "\n")
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
