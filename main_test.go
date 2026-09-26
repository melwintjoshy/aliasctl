package main

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"
)

func TestChildExitCodeUnwrapsExitError(t *testing.T) {
	err := exec.Command("sh", "-c", "exit 7").Run()

	code, ok := childExitCode(fmt.Errorf("command failed: %w", err))

	if !ok || code != 7 {
		t.Fatalf("expected code 7, got %d (ok=%v)", code, ok)
	}
}

func TestChildExitCodeIgnoresOtherErrors(t *testing.T) {
	if _, ok := childExitCode(errors.New("configuration error")); ok {
		t.Fatal("expected non-exit error to be reported normally")
	}
}
