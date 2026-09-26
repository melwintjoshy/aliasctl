package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// builds the real binary since the hook shells out to it on every directory change
func buildBinary(t *testing.T) string {
	t.Helper()

	binary := filepath.Join(t.TempDir(), "aliasctl")

	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, output)
	}

	return binary
}

const hookProjectConfig = `name: demo
variables:
  NAMESPACE: project
aliases:
  hi: "echo project-hi"
functions:
  greet: |
    echo "project greet"
`

// cd in and out, allowing in between; eval because zsh -c parses aliases before the hook runs
const hookScript = `
cd "$PROJECT/sub"; __aliasctl_hook
eval hi; echo "ns=$NAMESPACE env=${ALIASCTL_ENV:-none}"
"$ALIASCTL_BIN" --config "$PROJECT/aliasctl.yaml" allow >/dev/null
__aliasctl_hook
eval hi; greet; echo "ns=$NAMESPACE env=${ALIASCTL_ENV:-none} hook=${ALIASCTL_HOOK:-none}"
echo "prompt=$PROMPT_VALUE"
cd "$HOME"; __aliasctl_hook
eval hi; greet; echo "ns=$NAMESPACE env=${ALIASCTL_ENV:-none}"
echo "prompt=$PROMPT_VALUE"
cd "$PROJECT"; __aliasctl_hook
echo "project changed" >> "$PROJECT/aliasctl.yaml"
cd "$HOME"; __aliasctl_hook; cd "$PROJECT"; __aliasctl_hook
echo "after edit env=${ALIASCTL_ENV:-none}"
`

const hookExpected = `user-hi
ns=user env=none
project-hi
project greet
ns=project env=demo hook=1
prompt=(aliasctl:demo) base> 
user-hi
user greet
ns=user env=none
prompt=base> 
after edit env=none
`

// starts an interactive shell whose rc evals the hook, in a temp HOME with one project
func runHookShell(t *testing.T, shellName, projectConfig, script string) (string, string, string) {
	t.Helper()

	if _, err := exec.LookPath(shellName); err != nil {
		t.Skipf("%s is not available", shellName)
	}

	binary := buildBinary(t)

	home := t.TempDir()
	project := filepath.Join(home, "project")

	if err := os.MkdirAll(filepath.Join(project, "sub"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(project, "aliasctl.yaml"), []byte(projectConfig), 0644); err != nil {
		t.Fatal(err)
	}

	promptVariable := map[string]string{"bash": "PS1", "zsh": "PROMPT"}[shellName]
	rcName := map[string]string{"bash": ".bashrc", "zsh": ".zshrc"}[shellName]

	userRC := "export NAMESPACE=user\n" +
		"alias hi='echo user-hi'\n" +
		"greet() { echo 'user greet'; }\n" +
		promptVariable + "='base> '\n" +
		`eval "$("$ALIASCTL_BIN" hook ` + shellName + `)"` + "\n"

	if err := os.WriteFile(filepath.Join(home, rcName), []byte(userRC), 0644); err != nil {
		t.Fatal(err)
	}

	script = strings.ReplaceAll(script, "$PROMPT_VALUE", "$"+promptVariable)

	args := []string{"-i", "-c", script}
	if shellName == "bash" {
		args = append([]string{"--noprofile", "--rcfile", filepath.Join(home, rcName)}, args...)
	}

	cmd := exec.Command(shellName, args...)

	cmd.Dir = home
	cmd.Env = append(
		os.Environ(),
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_STATE_HOME="+filepath.Join(home, ".local", "state"),
		"ALIASCTL_BIN="+binary,
		"PROJECT="+project,
		"ALIASCTL_ENV=",
		"ALIASCTL_HOOK=",
		"PROMPT_COMMAND=",
		"ZDOTDIR="+home,
	)

	var stderr strings.Builder

	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s%s", shellName, err, output, stderr.String())
	}

	return string(output), stderr.String(), project
}

func runHookScenario(t *testing.T, shellName string) {
	t.Helper()

	output, stderr, project := runHookShell(t, shellName, hookProjectConfig, hookScript)

	// the untrusted hint is printed for the first visit and again after the edit, never on quiet retries
	hint := "aliasctl: " + filepath.Join(project, "aliasctl.yaml") + " is not allowed"

	if strings.Count(stderr, hint) != 2 {
		t.Fatalf("expected the hint twice, stderr:\n%s", stderr)
	}

	if output != hookExpected {
		t.Fatalf("expected:\n%s\ngot:\n%s\nstderr:\n%s", hookExpected, output, stderr)
	}
}

// sleep 1 because bash 3.2 compares mtimes in whole seconds
const hookEditScript = `
"$ALIASCTL_BIN" --config "$PROJECT/aliasctl.yaml" allow >/dev/null
cd "$PROJECT"; __aliasctl_hook
eval hi
sleep 1
printf 'name: demo\naliases:\n  hi: "echo edited-hi"\n' > "$PROJECT/aliasctl.yaml"
__aliasctl_hook; eval hi
__aliasctl_hook; eval hi
"$ALIASCTL_BIN" --config "$PROJECT/aliasctl.yaml" allow >/dev/null
__aliasctl_hook; eval hi
echo "prompt=$PROMPT_VALUE"
cd "$HOME"; __aliasctl_hook; eval hi
echo "stamps=$(ls "$HOME/.local/state/aliasctl/stamps" | wc -l | tr -d ' ')"
`

const hookEditExpected = `project-hi
project-hi
project-hi
edited-hi
prompt=(aliasctl:demo) base> 
user-hi
stamps=0
`

func runHookEditScenario(t *testing.T, shellName string) {
	t.Helper()

	output, stderr, project := runHookShell(t, shellName, hookProjectConfig, hookEditScript)

	hint := "aliasctl: " + filepath.Join(project, "aliasctl.yaml") + " changed; keeping the previous version"

	if strings.Count(stderr, hint) != 1 {
		t.Fatalf("expected the edit hint exactly once, stderr:\n%s", stderr)
	}

	if output != hookEditExpected {
		t.Fatalf("expected:\n%s\ngot:\n%s\nstderr:\n%s", hookEditExpected, output, stderr)
	}
}

func TestHookBash(t *testing.T) {
	runHookScenario(t, "bash")
}

func TestHookZsh(t *testing.T) {
	runHookScenario(t, "zsh")
}

func TestHookEditBash(t *testing.T) {
	runHookEditScenario(t, "bash")
}

func TestHookEditZsh(t *testing.T) {
	runHookEditScenario(t, "zsh")
}
