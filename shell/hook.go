package shell

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/melwintjoshy/aliasctl/resolver"
)

const hookEnvironmentVariable = "ALIASCTL_HOOK"

var hookShells = []string{"bash", "zsh"}

func checkHookShell(name string) error {
	if !slices.Contains(hookShells, name) {
		return fmt.Errorf("auto-activation supports %s, not %q", strings.Join(hookShells, " and "), name)
	}

	return nil
}

// RenderExport prints definitions for eval, plus ALIASCTL_UNLOAD to undo them and restore what they shadowed.
func RenderExport(env *resolver.Environment, shellName, configPath string) (string, error) {
	if err := checkHookShell(shellName); err != nil {
		return "", err
	}

	definitions, err := (BashRenderer{}).Render(env)
	if err != nil {
		return "", err
	}

	aliasNames := slices.Sorted(maps.Keys(env.Aliases))
	functionNames := slices.Sorted(maps.Keys(env.Functions))
	variableNames := slices.Sorted(maps.Keys(env.Variables))

	// bash and zsh print reusable definitions with different flags
	saveAlias, saveFunction := "alias %s", "declare -f %s"

	if shellName == "zsh" {
		saveAlias, saveFunction = "alias -L %s", "functions %s"
	}

	var output strings.Builder

	// removal first, then restores are appended as each shadowed definition is saved
	output.WriteString("ALIASCTL_UNLOAD=''\n")

	for _, name := range aliasNames {
		fmt.Fprintf(&output, "ALIASCTL_UNLOAD+=$'unalias %s 2>/dev/null\\n'\n", name)
	}

	for _, name := range functionNames {
		fmt.Fprintf(&output, "ALIASCTL_UNLOAD+=$'unset -f %s 2>/dev/null\\n'\n", name)
	}

	for _, name := range slices.Concat(aliasNames, functionNames) {
		fmt.Fprintf(
			&output,
			"if __aliasctl_saved=\"$("+saveAlias+" 2>/dev/null)\"; then ALIASCTL_UNLOAD+=\"$__aliasctl_saved\"$'\\n'; fi\n",
			name,
		)
	}

	for _, name := range functionNames {
		fmt.Fprintf(
			&output,
			"if __aliasctl_saved=\"$("+saveFunction+" 2>/dev/null)\"; then ALIASCTL_UNLOAD+=\"$__aliasctl_saved\"$'\\n'; fi\n",
			name,
		)
	}

	for _, name := range variableNames {
		fmt.Fprintf(
			&output,
			"if [ -n \"${%[1]s+x}\" ]; then ALIASCTL_UNLOAD+=\"export %[1]s=$(printf %%q \"$%[1]s\")\"$'\\n'; "+
				"else ALIASCTL_UNLOAD+=$'unset %[1]s\\n'; fi\n",
			name,
		)
	}

	output.WriteString("unset __aliasctl_saved\n")
	output.WriteString(definitions)

	fmt.Fprintf(&output, "export %s=%s\n", activeEnvironmentVariable, shellQuote(env.Name))
	fmt.Fprintf(&output, "export %s=1\n", hookEnvironmentVariable)
	fmt.Fprintf(&output, "ALIASCTL_CONFIG=%s\n", shellQuote(configPath))
	fmt.Fprintf(&output, "__aliasctl_prefix=%s\n", shellQuote("(aliasctl:"+promptSafeName(env.Name)+") "))

	return output.String(), nil
}

// the walk mirrors config.FindConfig so nothing is spawned on prompts outside a project
const hookFunction = `__aliasctl_hook() {
  local __aliasctl_status=$?

  # an explicit aliasctl shell owns this environment
  if [ -n "${ALIASCTL_ENV:-}" ] && [ -z "${ALIASCTL_HOOK:-}" ]; then
    return $__aliasctl_status
  fi

  local __aliasctl_dir="$PWD" __aliasctl_found=""

  while [ -n "$__aliasctl_dir" ]; do
    if [ -f "$__aliasctl_dir/aliasctl.yaml" ]; then
      __aliasctl_found="$__aliasctl_dir/aliasctl.yaml"
      break
    fi

    [ "$__aliasctl_dir" = / ] && break
    __aliasctl_dir="${__aliasctl_dir%/*}"
    __aliasctl_dir="${__aliasctl_dir:-/}"
  done

  if [ "$__aliasctl_found" != "${ALIASCTL_CONFIG:-}" ]; then
    if [ -n "${ALIASCTL_CONFIG:-}" ]; then
      eval "$ALIASCTL_UNLOAD"
      __PROMPT__="${__PROMPT__#"$__aliasctl_prefix"}"
    fi

    unset ALIASCTL_ENV ALIASCTL_HOOK ALIASCTL_CONFIG ALIASCTL_UNLOAD __aliasctl_prefix

    if [ -n "$__aliasctl_found" ]; then
      local __aliasctl_quiet=""
      [ "$__aliasctl_found" = "${__aliasctl_denied:-}" ] && __aliasctl_quiet="--quiet"

      local __aliasctl_export
      if __aliasctl_export="$(__BINARY__ --config "$__aliasctl_found" export --shell __SHELL__ $__aliasctl_quiet)"; then
        eval "$__aliasctl_export"
        __aliasctl_denied=""
      else
        __aliasctl_denied="$__aliasctl_found"
      fi
    fi
  fi

  if [ -n "${ALIASCTL_CONFIG:-}" ]; then
    case "$__PROMPT__" in
      "$__aliasctl_prefix"*) ;;
      *) __PROMPT__="$__aliasctl_prefix$__PROMPT__" ;;
    esac
  fi

  return $__aliasctl_status
}
`

// RenderHook prints the snippet users eval from their rc to auto-activate on cd.
func RenderHook(shellName, binary string) (string, error) {
	if err := checkHookShell(shellName); err != nil {
		return "", err
	}

	// runs after existing prompt commands so a rebuilt PS1 still gets the prefix
	prompt, register := "PS1", `PROMPT_COMMAND="${PROMPT_COMMAND}"$'\n''__aliasctl_hook'`

	if shellName == "zsh" {
		prompt, register = "PROMPT", "precmd_functions+=(__aliasctl_hook)"
	}

	hook := strings.NewReplacer(
		"__PROMPT__", prompt,
		"__BINARY__", shellQuote(binary),
		"__SHELL__", shellName,
	).Replace(hookFunction)

	return hook + register + "\n", nil
}
