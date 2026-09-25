package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/melwintjoshy/aliasctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		if code, ok := childExitCode(err); ok {
			os.Exit(code)
		}

		if errors.Is(err, cmd.ErrReported) {
			os.Exit(1)
		}

		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// the child already reported its own failure, so pass its code through quietly
func childExitCode(err error) (int, bool) {
	var exitErr *exec.ExitError

	if errors.As(err, &exitErr) && exitErr.ExitCode() > 0 {
		return exitErr.ExitCode(), true
	}

	return 0, false
}
