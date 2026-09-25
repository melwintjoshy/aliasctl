package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/melwintjoshy/aliasctl/resolver"
)

func RunBash(env *resolver.Environment) error {
	renderer := BashRenderer{}

	script, err := renderer.Render(env)
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

func RunCommand(env *resolver.Environment, args []string) error {
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
	environment := os.Environ()

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
