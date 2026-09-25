package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/melwintjoshy/aliasctl/resolver"
)

const activeEnvironmentVariable = "ALIASCTL_ENV"

func RunBash(env *resolver.Environment) error {
	if active := os.Getenv(activeEnvironmentVariable); active != "" {
		return fmt.Errorf(
			"already inside aliasctl environment %q; exit it first",
			active,
		)
	}

	script, err := renderInteractiveRC(env)
	if err != nil {
		return err
	}

	// Create temporary Bash rc file.
	file, err := os.CreateTemp("", "aliasctl-*.bashrc")
	if err != nil {
		return fmt.Errorf("failed to create temporary rc file: %w", err)
	}

	rcPath := file.Name()

	defer func() {
		file.Close()
		os.Remove(rcPath)
	}()

	// Write our generated environment into the rc file.
	if _, err := file.WriteString(script); err != nil {
		return fmt.Errorf("failed to write rc file: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close rc file: %w", err)
	}

	// Start interactive Bash using our temporary rc file.
	cmd := exec.Command(
		"bash",
		"--noprofile",
		"--rcfile",
		rcPath,
		"-i",
	)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// user rc first so project definitions win, prompt last so the rc can't overwrite it
func renderInteractiveRC(env *resolver.Environment) (string, error) {
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

	rc.WriteString(`if [ -f "$HOME/.bashrc" ]; then . "$HOME/.bashrc"; fi` + "\n")
	rc.WriteString(definitions)

	// referencing the variable keeps the name out of prompt expansion
	fmt.Fprintf(
		&rc,
		"PS1='(aliasctl:${%s}) '\"$PS1\"\n",
		activeEnvironmentVariable,
	)

	return rc.String(), nil
}

func RunCommand(env *resolver.Environment, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	args, err := expandCommand(env, args)
	if err != nil {
		return err
	}

	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = buildEnvironment(env)

	return cmd.Run()
}

// aliases and functions both go through bash so chaining and builtins behave like the shell
func expandCommand(env *resolver.Environment, args []string) ([]string, error) {
	_, isAlias := env.Alias(args[0])
	_, isFunction := env.Functions[args[0]]

	if !isAlias && !isFunction {
		return args, nil
	}

	definitions, err := (BashRenderer{}).Render(env)
	if err != nil {
		return nil, err
	}

	// the name is written literally since bash never alias-expands "$1"; names are validated
	script := "shopt -s expand_aliases\n" + definitions + args[0] + ` "$@"` + "\n"

	return append(
		[]string{"bash", "--noprofile", "--norc", "-c", script, "aliasctl"},
		args[1:]...,
	), nil
}

func buildEnvironment(env *resolver.Environment) []string {
	environment := setEnvironmentVariable(
		os.Environ(),
		activeEnvironmentVariable,
		env.Name,
	)

	for key, value := range env.Variables {
		environment = setEnvironmentVariable(
			environment,
			key,
			value,
		)
	}

	return environment
}

func setEnvironmentVariable(
	environment []string,
	key string,
	value string,
) []string {

	prefix := key + "="

	for i, entry := range environment {
		if strings.HasPrefix(entry, prefix) {
			environment[i] = key + "=" + value
			return environment
		}
	}

	return append(environment, key+"="+value)
}
