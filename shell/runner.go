package shell

import (
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/melwintjoshy/aliasctl/resolver"
)

const activeEnvironmentVariable = "ALIASCTL_ENV"

// Runner builds the plan for an interactive shell or for one command; nothing runs until it is executed.
type Runner interface {
	StartPlan(env *resolver.Environment, configPath, dir string) (Plan, error)
	RunPlan(env *resolver.Environment, args []string, dir string) (Plan, error)
}

// Plan is the command to run, its full environment and the files it expects in dir.
type Plan struct {
	Argv  []string
	Env   []string
	Files []File
}

type File struct {
	// relative to the plan's dir
	Name    string
	Content string
}

// Start opens the interactive shell described by the runner's plan; configPath is shown in the banner.
func Start(runner Runner, env *resolver.Environment, configPath string) error {
	if err := checkNotNested(); err != nil {
		return err
	}

	return execute(func(dir string) (Plan, error) {
		return runner.StartPlan(env, configPath, dir)
	})
}

// Run runs one command, going through the shell only when it names an alias or function.
func Run(runner Runner, env *resolver.Environment, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	return execute(func(dir string) (Plan, error) {
		return runner.RunPlan(env, args, dir)
	})
}

// the private temp dir holds every generated file and is removed when the command exits
func execute(build func(dir string) (Plan, error)) error {
	dir, err := os.MkdirTemp("", "aliasctl-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary dir: %w", err)
	}

	defer os.RemoveAll(dir)

	plan, err := build(dir)
	if err != nil {
		return err
	}

	for _, file := range plan.Files {
		if err := os.WriteFile(filepath.Join(dir, file.Name), []byte(file.Content), 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", file.Name, err)
		}
	}

	// exec.Command would search aliasctl's own PATH, which lacks the tool dirs the plan prepends
	path, err := lookPath(plan.Argv[0], environmentMap(plan.Env)["PATH"])
	if err != nil {
		return err
	}

	cmd := exec.Command(path, plan.Argv[1:]...)
	cmd.Args = slices.Clone(plan.Argv)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = plan.Env

	return cmd.Run()
}

// like exec.LookPath, but searching the given PATH; a name with a separator is used as is
func lookPath(name, path string) (string, error) {
	if strings.Contains(name, string(os.PathSeparator)) {
		return name, nil
	}

	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			continue
		}

		candidate := filepath.Join(dir, name)

		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
			return candidate, nil
		}
	}

	return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
}

// stands in for the temp dir when a plan is printed instead of run
const printDir = "<tmp>"

// DescribeStart prints what Start would run and write, without running it.
func DescribeStart(w io.Writer, runner Runner, env *resolver.Environment, configPath string) error {
	plan, err := runner.StartPlan(env, configPath, printDir)
	if err != nil {
		return err
	}

	describe(w, plan)

	return nil
}

// DescribeRun prints what Run would run and write, without running it.
func DescribeRun(w io.Writer, runner Runner, env *resolver.Environment, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	plan, err := runner.RunPlan(env, args, printDir)
	if err != nil {
		return err
	}

	describe(w, plan)

	return nil
}

func describe(w io.Writer, plan Plan) {
	tokens := make([]string, len(plan.Argv))

	for i, token := range plan.Argv {
		tokens[i] = displayToken(token)
	}

	fmt.Fprintf(w, "command: %s\n", strings.Join(tokens, " "))

	// the full environment is mostly the caller's own, so only the differences are shown
	for i, change := range environmentChanges(os.Environ(), plan.Env) {
		label := "env:    "
		if i > 0 {
			label = "        "
		}

		fmt.Fprintf(w, "%s %s\n", label, change)
	}

	for _, file := range plan.Files {
		fmt.Fprintf(w, "\n--- %s\n%s", filepath.Join(printDir, file.Name), file.Content)

		if !strings.HasSuffix(file.Content, "\n") {
			fmt.Fprintln(w)
		}
	}
}

// like quoteToken, but <tmp> stays readable
func displayToken(token string) string {
	if token != "" && !strings.ContainsAny(token, " \t\n'\"\\$`;&|*?") {
		return token
	}

	return shellQuote(token)
}

// changes are "KEY=value" for set or changed keys and "-KEY" for removed ones, sorted by key
func environmentChanges(base, next []string) []string {
	before := environmentMap(base)
	after := environmentMap(next)

	var changes []string

	for _, key := range slices.Sorted(maps.Keys(after)) {
		if value, ok := before[key]; !ok || value != after[key] {
			changes = append(changes, key+"="+after[key])
		}
	}

	for _, key := range slices.Sorted(maps.Keys(before)) {
		if _, ok := after[key]; !ok {
			changes = append(changes, "-"+key)
		}
	}

	return changes
}

func environmentMap(environ []string) map[string]string {
	values := make(map[string]string, len(environ))

	// later entries win, matching how exec treats duplicate keys
	for _, entry := range environ {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}

	return values
}

// set before the user rc runs so it can see which environment is active
func startEnvironment(env *resolver.Environment) []string {
	return setEnvironmentVariable(
		unsetEnvironmentVariable(os.Environ(), hookEnvironmentVariable),
		activeEnvironmentVariable,
		env.Name,
	)
}

func plainPlan(env *resolver.Environment, args []string) Plan {
	return Plan{Argv: slices.Clone(args), Env: BuildEnvironment(env)}
}

// script files are read command by command, so aliases defined in them expand later on
func scriptPlan(env *resolver.Environment, dir, name, script string, shellArgs, args []string) Plan {
	argv := append(slices.Clone(shellArgs), filepath.Join(dir, name))

	return Plan{
		Argv:  append(argv, args...),
		Env:   BuildEnvironment(env),
		Files: []File{{Name: name, Content: script}},
	}
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

// hook-loaded state is fine to start a shell from; only an explicit aliasctl shell is refused
func checkNotNested() error {
	if os.Getenv(hookEnvironmentVariable) != "" {
		return nil
	}

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

// BuildEnvironment is the process environment with the project's variables applied.
func BuildEnvironment(env *resolver.Environment) []string {
	environment := setEnvironmentVariable(
		unsetEnvironmentVariable(os.Environ(), hookEnvironmentVariable),
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

	if len(env.PathPrepend) > 0 {
		path := strings.Join(env.PathPrepend, string(os.PathListSeparator))

		if current := environmentMap(environment)["PATH"]; current != "" {
			path += string(os.PathListSeparator) + current
		}

		environment = setEnvironmentVariable(environment, "PATH", path)
	}

	return environment
}

// written after the user rc, since an rc that rebuilds PATH would otherwise drop the tool dirs
func posixPathExport(env *resolver.Environment) string {
	if len(env.PathPrepend) == 0 {
		return ""
	}

	quoted := make([]string, len(env.PathPrepend))

	for i, dir := range env.PathPrepend {
		quoted[i] = shellQuote(dir)
	}

	return "export PATH=" + strings.Join(quoted, ":") + `:"$PATH"` + "\n"
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

func unsetEnvironmentVariable(environment []string, key string) []string {
	return slices.DeleteFunc(environment, func(entry string) bool {
		return strings.HasPrefix(entry, key+"=")
	})
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
