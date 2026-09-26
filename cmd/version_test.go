package cmd

import (
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	// only the prefix is fixed; the rest depends on how the test binary was stamped
	output, err := runRoot(t, "--version")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(output, "aliasctl version ") {
		t.Fatalf("unexpected version output %q", output)
	}
}
