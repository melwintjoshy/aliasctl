package shell

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/melwintjoshy/aliasctl/resolver"
)

func TestBashRenderer(t *testing.T) {
	env := &resolver.Environment{
		Name: "test",

		Variables: map[string]string{
			"NAMESPACE": "app-prod",
		},

		Aliases: map[string]resolver.Command{
			"kgp": {
				Name: "kubectl",
				Args: []string{"get", "pods", "-n", "app-prod"},
			},
		},
	}

	renderer := BashRenderer{}

	output, err := renderer.Render(env)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedVariable := "export NAMESPACE='app-prod'"
	expectedAlias := "alias kgp=" + shellQuote(
		"'kubectl' 'get' 'pods' '-n' 'app-prod'",
	)

	if !strings.Contains(output, expectedVariable) {
		t.Fatalf(
			"expected output to contain %q\nGot:\n%s",
			expectedVariable,
			output,
		)
	}

	if !strings.Contains(output, expectedAlias) {
		t.Fatalf(
			"expected output to contain %q\nGot:\n%s",
			expectedAlias,
			output,
		)
	}
}

func TestBashRendererQuotesVariableValue(t *testing.T) {
	env := &resolver.Environment{
		Variables: map[string]string{
			"VALUE": `hello'; echo MALICIOUS; echo '`,
		},
	}

	renderer := BashRenderer{}

	output, err := renderer.Render(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `export VALUE='hello'\''; echo MALICIOUS; echo '\''`

	if !strings.Contains(output, expected) {
		t.Fatalf(
			"expected output to contain %q\nGot:\n%s",
			expected,
			output,
		)
	}
}

func TestBashRendererQuotesAliasCommand(t *testing.T) {
	env := &resolver.Environment{
		Aliases: map[string]resolver.Command{
			"danger": {
				Name: "echo",
				Args: []string{
					`hello'; echo MALICIOUS; echo '`,
				},
			},
		},
	}

	renderer := BashRenderer{}

	output, err := renderer.Render(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "alias danger=" + shellQuote(
		"'echo' "+shellQuote(`hello'; echo MALICIOUS; echo '`),
	)

	if !strings.Contains(output, expected) {
		t.Fatalf(
			"expected output to contain %q\nGot:\n%s",
			expected,
			output,
		)
	}
}

func TestBashRendererDoesNotExecuteVariableValue(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	marker, err := os.CreateTemp("", "aliasctl-security-*")
	if err != nil {
		t.Fatal(err)
	}

	markerPath := marker.Name()
	marker.Close()
	os.Remove(markerPath)

	env := &resolver.Environment{
		Variables: map[string]string{
			"VALUE": "hello; touch " + markerPath,
		},
	}

	renderer := BashRenderer{}

	script, err := renderer.Render(env)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		"bash",
		"-c",
		script,
	)

	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(markerPath); err == nil {
		t.Fatal("variable value was interpreted as shell code")
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected error checking marker: %v", err)
	}
}

func TestBashRendererDoesNotExecuteAliasArgument(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	marker, err := os.CreateTemp("", "aliasctl-security-*")
	if err != nil {
		t.Fatal(err)
	}

	markerPath := marker.Name()
	marker.Close()
	os.Remove(markerPath)

	env := &resolver.Environment{
		Aliases: map[string]resolver.Command{
			"danger": {
				Name: "echo",
				Args: []string{
					"hello; touch " + markerPath,
				},
			},
		},
	}

	renderer := BashRenderer{}

	script, err := renderer.Render(env)
	if err != nil {
		t.Fatal(err)
	}

	// defining the alias is not enough, the injection only fires when it expands
	cmd := exec.Command(
		"bash",
		"-c",
		"shopt -s expand_aliases\n"+script+"danger\n",
	)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "hello; touch " + markerPath + "\n"

	if string(output) != expected {
		t.Fatalf("expected %q, got %q", expected, string(output))
	}

	if _, err := os.Stat(markerPath); err == nil {
		t.Fatal("alias argument was interpreted as shell code")
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected error checking marker: %v", err)
	}
}

func TestBashRendererAliasKeepsArgumentBoundaries(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	env := &resolver.Environment{
		Aliases: map[string]resolver.Command{
			"show": {
				Name: "printf",
				Args: []string{`[%s]\n`, "-m", "initial commit"},
			},
		},
	}

	script, err := (BashRenderer{}).Render(env)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		"bash",
		"-c",
		"shopt -s expand_aliases\n"+script+"show\n",
	)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("alias execution failed: %v", err)
	}

	expected := "[-m]\n[initial commit]\n"

	if string(output) != expected {
		t.Fatalf("expected %q, got %q", expected, string(output))
	}
}

func TestBashRendererRendersFunctions(t *testing.T) {
	renderer := BashRenderer{}

	env := &resolver.Environment{
		Functions: map[string]string{
			"deploy": `echo "Building..."
go build ./...
`,
		},
	}

	output, err := renderer.Render(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `deploy() {
echo "Building..."
go build ./...
}
`

	if output != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, output)
	}
}

func TestBashRendererFunctionExecutes(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}

	renderer := BashRenderer{}

	env := &resolver.Environment{
		Functions: map[string]string{
			"hello": `echo "hello from function"`,
		},
	}

	script, err := renderer.Render(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := exec.Command(
		"bash",
		"-c",
		script+`hello`,
	)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("function execution failed: %v", err)
	}

	expected := "hello from function\n"

	if string(output) != expected {
		t.Fatalf("expected %q, got %q", expected, string(output))
	}
}
