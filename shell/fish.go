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

// FishRenderer emits fish syntax; bash function bodies are never fed to it, only functions_fish.
type FishRenderer struct{}

func (FishRenderer) Render(env *resolver.Environment) (string, error) {
	var output strings.Builder

	for _, key := range slices.Sorted(maps.Keys(env.Variables)) {
		fmt.Fprintf(&output, "set -gx %s %s\n", key, fishQuote(env.Variables[key]))
	}

	for _, name := range slices.Sorted(maps.Keys(env.Aliases)) {
		command := env.Aliases[name]
		head, args := quoteFishToken(command.Name), formatFishArgs(command.Args)
		body := head + args + " $argv"

		// a self-named alias like "ls: ls -la" would recurse as a fish function
		if command.Name == name {
			body = fmt.Sprintf(
				"if contains -- %[1]s (builtin --names)\n"+
					"        builtin %[1]s%[2]s $argv\n"+
					"    else\n"+
					"        command %[1]s%[2]s $argv\n"+
					"    end",
				head,
				args,
			)
		}

		fmt.Fprintf(&output, "function %s\n    %s\nend\n", name, body)
	}

	for _, name := range slices.Sorted(maps.Keys(env.FunctionsFish)) {
		body := strings.TrimRight(env.FunctionsFish[name], "\n")
		fmt.Fprintf(&output, "function %s\n%s\nend\n", name, body)
	}

	return output.String(), nil
}

// fish expands %self, so % is left out of the bare set
var safeFishTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_./:=@+-]+$`)

func formatFishArgs(args []string) string {
	var output strings.Builder

	for _, arg := range args {
		output.WriteString(" " + quoteFishToken(arg))
	}

	return output.String()
}

func quoteFishToken(token string) string {
	if safeFishTokenPattern.MatchString(token) {
		return token
	}

	return fishQuote(token)
}

// inside fish single quotes only \ and ' are special
func fishQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)

	return "'" + value + "'"
}

// wraps the existing prompt; the name is printed with %s so it is never evaluated
const fishPromptHook = `if functions -q fish_prompt; and not functions -q __aliasctl_original_prompt
    functions -c fish_prompt __aliasctl_original_prompt
    function fish_prompt
        printf '(aliasctl:%s) ' $ALIASCTL_ENV
        __aliasctl_original_prompt
    end
end
`

type fishRunner struct{}

func (fishRunner) Start(env *resolver.Environment, configPath string) error {
	if err := checkNotNested(); err != nil {
		return err
	}

	warnBashOnlyFunctions(env)

	definitions, err := (FishRenderer{}).Render(env)
	if err != nil {
		return err
	}

	bannerScript := ""
	if configPath != "" {
		bannerScript = RenderBanner(env, configPath)
	}

	path, err := writeTempFile("", "aliasctl-*.fish", definitions+bannerScript+fishPromptHook)
	if err != nil {
		return err
	}

	defer os.Remove(path)

	// --init-command runs after the user's config.fish, so project definitions win
	return startInteractive(env, nil, "fish", "-i", "--init-command", "source "+fishQuote(path))
}

func (fishRunner) Run(env *resolver.Environment, args []string) error {
	if len(args) == 0 {
		return runPlain(env, args)
	}

	if found, err := lookupCommand(env, args[0], true); err != nil || !found {
		if err != nil {
			return err
		}

		return runPlain(env, args)
	}

	definitions, err := (FishRenderer{}).Render(env)
	if err != nil {
		return err
	}

	script := definitions + args[0] + " $argv\n"

	return runScript(env, []string{"fish", "--no-config"}, script, args[1:])
}

func warnBashOnlyFunctions(env *resolver.Environment) {
	for _, name := range slices.Sorted(maps.Keys(env.Functions)) {
		if _, ok := env.FunctionsFish[name]; !ok {
			fmt.Fprintf(
				os.Stderr,
				"aliasctl: function %q has no functions_fish body and is not available in fish\n",
				name,
			)
		}
	}
}
