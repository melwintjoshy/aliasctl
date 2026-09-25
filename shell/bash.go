package shell

import (
	"fmt"
	"maps"
	"regexp"
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
			shellQuote(FormatCommand(env.Aliases[name])),
		)
	}

	for _, name := range slices.Sorted(maps.Keys(env.Functions)) {
		body := strings.TrimRight(env.Functions[name], "\n")

		// a user alias with this name would expand "name() {" into a syntax error
		fmt.Fprintf(&output, "unalias %s 2>/dev/null\n", name)
		fmt.Fprintf(&output, "%s() {\n%s\n}\n", name, body)
	}

	return output.String(), nil
}

var safeTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_./:=@%+-]+$`)

// bash re-parses alias values on use, so each token is quoted on its own
func FormatCommand(command resolver.Command) string {
	tokens := make([]string, 0, len(command.Args)+1)

	tokens = append(tokens, quoteToken(command.Name))

	for _, arg := range command.Args {
		tokens = append(tokens, quoteToken(arg))
	}

	return strings.Join(tokens, " ")
}

// plain words stay bare so a leading alias name still chains
func quoteToken(token string) string {
	if safeTokenPattern.MatchString(token) {
		return token
	}

	return shellQuote(token)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
