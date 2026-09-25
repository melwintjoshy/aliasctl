package shell

import (
	"os"
	"os/exec"
	"testing"

	"github.com/melwintjoshy/aliasctl/resolver"
)

func TestRunCommandPassesVariables(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is not available")
	}

	env := &resolver.Environment{
		Name: "test",
		Variables: map[string]string{
			"ALIASCTL_TEST": "hello",
		},
	}

	outputFile, err := os.CreateTemp("", "aliasctl-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outputFile.Name())
	outputFile.Close()

	args := []string{
		"sh",
		"-c",
		"echo $ALIASCTL_TEST > " + outputFile.Name(),
	}

	if err := RunCommand(env, args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outputFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "hello\n" {
		t.Fatalf("expected %q, got %q", "hello\n", string(data))
	}
}
func TestRunCommandPreservesEnvironment(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is not available")
	}

	env := &resolver.Environment{
		Name: "test",
		Variables: map[string]string{
			"ALIASCTL_TEST": "hello",
		},
	}

	cmd := exec.Command("sh", "-c", "test -n \"$PATH\"")

	cmd.Env = buildEnvironment(env)

	if err := cmd.Run(); err != nil {
		t.Fatalf("PATH was not preserved: %v", err)
	}
}

func TestRunCommandArguments(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is not available")
	}

	env := &resolver.Environment{
		Name: "test",
	}

	cmd := exec.Command(
		"sh",
		"-c",
		"test \"$1\" = \"hello world\"",
		"shell",
		"hello world",
	)

	cmd.Env = buildEnvironment(env)

	if err := cmd.Run(); err != nil {
		t.Fatalf("argument was not passed correctly: %v", err)
	}
}

func TestRunCommandMissingCommand(t *testing.T) {
	env := &resolver.Environment{
		Name: "test",
	}

	err := RunCommand(
		env,
		[]string{"this-command-definitely-does-not-exist"},
	)

	if err == nil {
		t.Fatal("expected an error")
	}
}
func TestBuildEnvironmentAddsAliasCtlVariables(t *testing.T) {
	env := &resolver.Environment{
		Variables: map[string]string{
			"ALIASCTL_TEST": "hello",
		},
	}

	environment := buildEnvironment(env)

	found := false

	for _, entry := range environment {
		if entry == "ALIASCTL_TEST=hello" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("expected ALIASCTL_TEST to be present")
	}
}

func TestRunShellExecutesFunctions(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	env := &resolver.Environment{
		Functions: map[string]string{
			"hello": `echo "hello from function"`,
		},
	}

	script, err := (BashRenderer{}).Render(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := exec.Command("bash", "-c", script+`hello`)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("function execution failed: %v", err)
	}

	expected := "hello from function\n"
	if string(output) != expected {
		t.Fatalf("expected %q, got %q", expected, string(output))
	}
}
