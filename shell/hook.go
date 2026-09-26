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
// stampPath, when set, lets the hook notice edits to the config while it is loaded.
func RenderExport(env *resolver.Environment, shellName, configPath, stampPath string) (string, error) {
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

	// PATH is saved like any variable so leaving the project restores it
	if len(env.PathPrepend) > 0 && !slices.Contains(variableNames, "PATH") {
		variableNames = append(variableNames, "PATH")
	}

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
	output.WriteString(posixPathExport(env))

	fmt.Fprintf(&output, "export %s=%s\n", activeEnvironmentVariable, shellQuote(env.Name))
	fmt.Fprintf(&output, "export %s=1\n", hookEnvironmentVariable)
	fmt.Fprintf(&output, "ALIASCTL_CONFIG=%s\n", shellQuote(configPath))
	fmt.Fprintf(&output, "__aliasctl_prefix=%s\n", shellQuote("(aliasctl:"+promptSafeName(env.Name)+") "))

	if stampPath != "" {
		fmt.Fprintf(&output, "__aliasctl_stamp=%s\n", shellQuote(stampPath))
		fmt.Fprintf(&output, "ALIASCTL_UNLOAD+=%s$'\\n'\n", shellQuote("rm -f -- "+shellQuote(stampPath)))
	}

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

    unset ALIASCTL_ENV ALIASCTL_HOOK ALIASCTL_CONFIG ALIASCTL_UNLOAD __aliasctl_prefix __aliasctl_stamp __aliasctl_edit_hinted

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

  # edited in place: reload once trusted again, keep the old version until then
  # bash 3.2 compares whole seconds, so an edit in the same second as loading can be missed
  elif [ -n "${__aliasctl_stamp:-}" ] && [ "$ALIASCTL_CONFIG" -nt "$__aliasctl_stamp" ]; then
    local __aliasctl_reload
    if __aliasctl_reload="$(__BINARY__ --config "$ALIASCTL_CONFIG" export --shell __SHELL__ --quiet)"; then
      eval "$ALIASCTL_UNLOAD"
      __PROMPT__="${__PROMPT__#"$__aliasctl_prefix"}"
      eval "$__aliasctl_reload"
      unset __aliasctl_edit_hinted
    elif [ "${__aliasctl_edit_hinted:-}" != "$ALIASCTL_CONFIG" ]; then
      printf 'aliasctl: %s changed; keeping the previous version, run "aliasctl allow" to load it\n' "$ALIASCTL_CONFIG" >&2
      __aliasctl_edit_hinted="$ALIASCTL_CONFIG"
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
