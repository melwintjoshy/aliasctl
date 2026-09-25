package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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

func TestRunBashRefusesNestedEnvironment(t *testing.T) {
	t.Setenv("ALIASCTL_ENV", "outer")

	err := RunBash(&resolver.Environment{Name: "inner"})
	if err == nil {
		t.Fatal("expected nested environment error")
	}

	expected := `already inside aliasctl environment "outer"; exit it first`

	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestInteractiveRCLayersOnUserRC(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	home := t.TempDir()

	userRC := "PS1='base> '\n" +
		"alias mine='echo mine'\n" +
		"alias shared='echo user'\n"

	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte(userRC), 0644); err != nil {
		t.Fatal(err)
	}

	env := &resolver.Environment{
		Name: "demo $(touch pwned)",
		Aliases: map[string]resolver.Command{
			"shared": {Name: "echo", Args: []string{"project"}},
		},
	}

	rc, err := renderInteractiveRC(env)
	if err != nil {
		t.Fatal(err)
	}

	rcPath := filepath.Join(t.TempDir(), "rc")

	if err := os.WriteFile(rcPath, []byte(rc), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		"bash",
		"--noprofile",
		"--rcfile", rcPath,
		"-i",
		"-c", `echo "$PS1"; mine; shared; echo "$ALIASCTL_ENV"`,
	)

	cmd.Dir = home
	cmd.Env = append(os.Environ(), "HOME="+home)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("interactive shell failed: %v", err)
	}

	// the prompt holds a reference to the name, never the name itself
	expected := "(aliasctl:${ALIASCTL_ENV}) base> \n" +
		"mine\n" +
		"project\n" +
		"demo $(touch pwned)\n"

	if string(output) != expected {
		t.Fatalf("expected %q, got %q", expected, string(output))
	}

	if _, err := os.Stat(filepath.Join(home, "pwned")); err == nil {
		t.Fatal("environment name was executed")
	}
}

func TestRunCommandExpandsAlias(t *testing.T) {
	env := &resolver.Environment{
		Aliases: map[string]resolver.Command{
			"greet": {Name: "echo", Args: []string{"hello world"}},
		},
	}

	got, err := expandCommand(env, []string{"greet", "--loud"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"echo", "hello world", "--loud"}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestRunCommandLeavesPlainCommand(t *testing.T) {
	env := &resolver.Environment{
		Aliases: map[string]resolver.Command{
			"greet": {Name: "echo"},
		},
	}

	got, err := expandCommand(env, []string{"ls", "-la"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(got, []string{"ls", "-la"}) {
		t.Fatalf("expected command unchanged, got %v", got)
	}
}

func TestRunCommandRunsFunctionWithArguments(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	outputPath := filepath.Join(t.TempDir(), "out")

	env := &resolver.Environment{
		Variables: map[string]string{
			"GREETING": "hello",
		},
		Aliases: map[string]resolver.Command{
			"say": {Name: "echo"},
		},
		Functions: map[string]string{
			"greet": `say "$GREETING $1" > "$2"`,
		},
	}

	if err := RunCommand(env, []string{"greet", "big world", outputPath}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "hello big world\n" {
		t.Fatalf("expected %q, got %q", "hello big world\n", string(data))
	}
}
