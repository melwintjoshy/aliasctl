package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/melwintjoshy/aliasctl/resolver"
)

const activeEnvironmentVariable = "ALIASCTL_ENV"

// Runner starts an interactive shell or runs one command inside the environment.
type Runner interface {
	Start(env *resolver.Environment) error
	Run(env *resolver.Environment, args []string) error
}

var Supported = []string{"bash", "zsh", "fish"}

func New(name string) (Runner, error) {
	switch name {
	case "bash":
		return bashRunner{}, nil
	case "zsh":
		return zshRunner{}, nil
	case "fish":
		return fishRunner{}, nil
	}

	return nil, fmt.Errorf(
		"unsupported shell %q (supported: %s)",
		name,
		strings.Join(Supported, ", "),
	)
}

// DefaultShell is the basename of $SHELL when supported, otherwise bash.
func DefaultShell() string {
	name := filepath.Base(os.Getenv("SHELL"))

	if slices.Contains(Supported, name) {
		return name
	}

	return "bash"
}

func checkNotNested() error {
	if active := os.Getenv(activeEnvironmentVariable); active != "" {
		return fmt.Errorf(
			"already inside aliasctl environment %q; exit it first",
			active,
		)
	}

	return nil
}

// reports whether run should go through the shell, and errors when only the other dialect defines it
func lookupCommand(env *resolver.Environment, name string, fish bool) (bool, error) {
	if _, ok := env.Alias(name); ok {
		return true, nil
	}

	own, other, otherKey := env.Functions, env.FunctionsFish, "functions_fish"

	if fish {
		own, other, otherKey = env.FunctionsFish, env.Functions, "functions"
	}

	if _, ok := own[name]; ok {
		return true, nil
	}

	if _, ok := other[name]; ok {
		return false, fmt.Errorf("function %q is only defined under %s", name, otherKey)
	}

	return false, nil
}

func writeTempFile(dir, pattern, content string) (string, error) {
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	path := file.Name()

	if _, err := file.WriteString(content); err != nil {
		file.Close()
		os.Remove(path)
		return "", fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("failed to close temporary file: %w", err)
	}

	return path, nil
}

func startInteractive(env *resolver.Environment, extraEnv []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// set before the user rc runs so it can see which environment is active
	cmd.Env = setEnvironmentVariable(os.Environ(), activeEnvironmentVariable, env.Name)
	cmd.Env = append(cmd.Env, extraEnv...)

	return cmd.Run()
}

// script files are read command by command, so aliases defined in them expand later on
func runScript(env *resolver.Environment, shellArgs []string, script string, args []string) error {
	path, err := writeTempFile("", "aliasctl-*.sh", script)
	if err != nil {
		return err
	}

	defer os.Remove(path)

	command := append(slices.Clone(shellArgs), path)

	return runPlain(env, append(command, args...))
}

func runPlain(env *resolver.Environment, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = buildEnvironment(env)

	return cmd.Run()
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

// prompts are re-parsed by the shell, so only plain characters of the name are shown
func promptSafeName(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', '0' <= r && r <= '9':
			return r
		case r == '.' || r == '_' || r == '-':
			return r
		}

		return '_'
	}, name)
}
