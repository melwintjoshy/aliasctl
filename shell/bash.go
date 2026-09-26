package shell

import (
	"fmt"
	"maps"
	"os"
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

type bashRunner struct{}

func (bashRunner) Start(env *resolver.Environment, configPath string) error {
	if err := checkNotNested(); err != nil {
		return err
	}

	script, err := renderInteractiveRC(env, configPath)
	if err != nil {
		return err
	}

	rcPath, err := writeTempFile("", "aliasctl-*.bashrc", script)
	if err != nil {
		return err
	}

	defer os.Remove(rcPath)

	return startInteractive(env, nil, "bash", "--noprofile", "--rcfile", rcPath, "-i")
}

func (bashRunner) Run(env *resolver.Environment, args []string) error {
	if len(args) == 0 {
		return runPlain(env, args)
	}

	if found, err := lookupCommand(env, args[0], false); err != nil || !found {
		if err != nil {
			return err
		}

		return runPlain(env, args)
	}

	definitions, err := (BashRenderer{}).Render(env)
	if err != nil {
		return err
	}

	// the name is written literally since bash never alias-expands "$1"; names are validated
	script := "shopt -s expand_aliases\n" + definitions + args[0] + ` "$@"` + "\n"

	return runScript(env, []string{"bash", "--noprofile", "--norc"}, script, args[1:])
}

// re-applied from PROMPT_COMMAND since prompts like starship rebuild PS1 before every prompt
const bashPromptHook = `__aliasctl_prompt() {
  case "$PS1" in
    '(aliasctl:${ALIASCTL_ENV}) '*) ;;
    *) PS1='(aliasctl:${ALIASCTL_ENV}) '"$PS1" ;;
  esac
}
__aliasctl_prompt
PROMPT_COMMAND="${PROMPT_COMMAND}"$'\n''__aliasctl_prompt'
`

// user rc first so project definitions win, prompt last so the rc can't overwrite it
func renderInteractiveRC(env *resolver.Environment, configPath string) (string, error) {
	definitions, err := (BashRenderer{}).Render(env)
	if err != nil {
		return "", err
	}

	var rc strings.Builder

	fmt.Fprintf(
		&rc,
		"export %s=%s\n",
		activeEnvironmentVariable,
		shellQuote(env.Name),
	)

	// --noprofile skips .bash_profile, which is where macOS login-shell users keep things
	rc.WriteString(`if [ -f "$HOME/.bashrc" ]; then . "$HOME/.bashrc"; ` +
		`elif [ -f "$HOME/.bash_profile" ]; then . "$HOME/.bash_profile"; fi` + "\n")
	rc.WriteString(definitions)

	if configPath != "" {
		rc.WriteString(RenderBanner(env, configPath))
	}

	// referencing the variable keeps the name out of prompt expansion
	rc.WriteString(bashPromptHook)

	return rc.String(), nil
}
