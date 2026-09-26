package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/melwintjoshy/aliasctl/resolver"
)

// zsh shares bash's alias, export and function syntax, so only startup and prompt differ
type ZshRenderer struct {
	BashRenderer
}

type zshRunner struct{}

func (zshRunner) StartPlan(env *resolver.Environment, dir string) (Plan, error) {
	zshenv, zshrc, err := renderZshStartup(env, dir, userZdotdir())
	if err != nil {
		return Plan{}, err
	}

	return Plan{
		Argv: []string{"zsh", "-i"},
		Env:  setEnvironmentVariable(startEnvironment(env), "ZDOTDIR", dir),
		Files: []File{
			{Name: ".zshenv", Content: zshenv},
			{Name: ".zshrc", Content: zshrc},
		},
	}, nil
}

func (zshRunner) RunPlan(env *resolver.Environment, args []string, dir string) (Plan, error) {
	found, err := lookupCommand(env, args[0], false)
	if err != nil {
		return Plan{}, err
	}

	if !found {
		return plainPlan(env, args), nil
	}

	definitions, err := (ZshRenderer{}).Render(env)
	if err != nil {
		return Plan{}, err
	}

	script := definitions + args[0] + ` "$@"` + "\n"

	return scriptPlan(env, dir, "run.zsh", script, []string{"zsh", "-f"}, args[1:]), nil
}

func userZdotdir() string {
	if dir := os.Getenv("ZDOTDIR"); dir != "" {
		return dir
	}

	return os.Getenv("HOME")
}

// zsh reads .zshenv and .zshrc from ZDOTDIR, so both are proxied and ZDOTDIR handed back to the user
func renderZshStartup(env *resolver.Environment, dir, userDir string) (string, string, error) {
	definitions, err := (ZshRenderer{}).Render(env)
	if err != nil {
		return "", "", err
	}

	var zshenv strings.Builder

	fmt.Fprintf(&zshenv, "ZDOTDIR=%s\n", shellQuote(userDir))
	zshenv.WriteString(`if [ -f "$ZDOTDIR/.zshenv" ]; then . "$ZDOTDIR/.zshenv"; fi` + "\n")

	// a user .zshenv may move ZDOTDIR, so remember where it ended up before pointing back here
	zshenv.WriteString(`__aliasctl_user_zdotdir="$ZDOTDIR"` + "\n")
	fmt.Fprintf(&zshenv, "ZDOTDIR=%s\n", shellQuote(dir))

	prefix := shellQuote("(aliasctl:" + promptSafeName(env.Name) + ") ")

	var zshrc strings.Builder

	fmt.Fprintf(&zshrc, "export %s=%s\n", activeEnvironmentVariable, shellQuote(env.Name))
	zshrc.WriteString(`ZDOTDIR="$__aliasctl_user_zdotdir"` + "\n")
	zshrc.WriteString("unset __aliasctl_user_zdotdir\n")
	zshrc.WriteString(`if [ -f "$ZDOTDIR/.zshrc" ]; then . "$ZDOTDIR/.zshrc"; fi` + "\n")
	zshrc.WriteString(definitions)

	// precmd re-adds the prefix for prompts that rebuild PROMPT each time
	fmt.Fprintf(
		&zshrc,
		"__aliasctl_prompt() {\n  [[ \"$PROMPT\" == %s* ]] || PROMPT=%s\"$PROMPT\"\n}\n",
		prefix,
		prefix,
	)
	zshrc.WriteString("__aliasctl_prompt\n")
	zshrc.WriteString("precmd_functions+=(__aliasctl_prompt)\n")

	return zshenv.String(), zshrc.String(), nil
}
