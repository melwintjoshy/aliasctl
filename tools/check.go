package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/melwintjoshy/aliasctl/resolver"
	"github.com/melwintjoshy/aliasctl/versions"
)

type Status string

const (
	StatusOK       Status = "ok"
	StatusMissing  Status = "missing"
	StatusBroken   Status = "broken"
	StatusMismatch Status = "mismatch"
	StatusTimeout  Status = "timeout"
)

// Result is what one tool probe found; Found is nil when no version could be read.
type Result struct {
	Name   string
	Rule   string
	Path   string
	Found  versions.Version
	Status Status
	Detail string
}

const DefaultTimeout = 5 * time.Second

type Checker struct {
	Timeout time.Duration

	// checks run here and relative check paths resolve against it; "" means the working directory
	Dir string
}

// Check probes every tool in parallel with the default timeout, from the working directory.
func Check(ctx context.Context, requirements []resolver.ToolRequirement, environ []string) []Result {
	return Checker{Timeout: DefaultTimeout}.Check(ctx, requirements, environ)
}

func (c Checker) Check(ctx context.Context, requirements []resolver.ToolRequirement, environ []string) []Result {
	results := make([]Result, len(requirements))

	var wg sync.WaitGroup

	for i, requirement := range requirements {
		wg.Add(1)

		go func() {
			defer wg.Done()
			results[i] = c.checkOne(ctx, requirement, environ)
		}()
	}

	wg.Wait()

	slices.SortFunc(results, func(a, b Result) int {
		return strings.Compare(a.Name, b.Name)
	})

	return results
}

func (c Checker) checkOne(ctx context.Context, requirement resolver.ToolRequirement, environ []string) Result {
	result := Result{Name: requirement.Name, Rule: requirement.Rule}

	constraint, err := versions.ParseConstraint(requirement.Rule)
	if err != nil {
		result.Status, result.Detail = StatusMismatch, err.Error()
		return result
	}

	path, ok := c.findExecutable(requirement.Check.Name, environ)
	if !ok {
		result.Status = StatusMissing
		return result
	}

	result.Path = path

	timeout := c.Timeout
	if requirement.Timeout > 0 {
		timeout = requirement.Timeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, requirement.Check.Args...)
	cmd.Env = environ
	cmd.Dir = c.Dir

	isolateProcessGroup(cmd)

	// stops a child that keeps the output pipe open from outliving the timeout
	cmd.WaitDelay = time.Second

	// java -version prints to stderr, so both streams are read
	output, err := cmd.CombinedOutput()

	switch {
	// a slow first start is not a broken tool, so it gets its own status and hint
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		result.Status, result.Detail = StatusTimeout, fmt.Sprintf("no answer within %s", timeout)
		return result

	case err != nil:
		result.Status, result.Detail = StatusBroken, firstLine(output, err.Error())
		return result
	}

	found, ok := versions.Parse(string(output))
	if !ok {
		result.Status, result.Detail = StatusBroken, "no version in output: "+firstLine(output, "(empty)")
		return result
	}

	result.Found = found
	result.Status = StatusOK

	if !constraint.Match(found) {
		result.Status = StatusMismatch
	}

	return result
}

// uses the PATH from environ, not aliasctl's own, so project variables apply
func (c Checker) findExecutable(name string, environ []string) (string, bool) {
	if strings.Contains(name, "/") {
		path := c.resolve(name)
		return path, isExecutable(path)
	}

	for _, dir := range filepath.SplitList(environValue(environ, "PATH")) {
		if dir == "" {
			dir = "."
		}

		path := c.resolve(filepath.Join(dir, name))

		if isExecutable(path) {
			return path, true
		}
	}

	return "", false
}

// a bare name would make exec search PATH again, so every result is explicitly relative or absolute
func (c Checker) resolve(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	if c.Dir != "" {
		return filepath.Join(c.Dir, path)
	}

	if !strings.Contains(path, "/") {
		return "./" + path
	}

	return path
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0
}

func environValue(environ []string, key string) string {
	for i := len(environ) - 1; i >= 0; i-- {
		if value, ok := strings.CutPrefix(environ[i], key+"="); ok {
			return value
		}
	}

	return ""
}

func firstLine(output []byte, fallback string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(string(output)), "\n")

	if line == "" {
		return fallback
	}

	return line
}

// Problem is a one-line description of what is wrong, or "" when the tool is ok.
func (r Result) Problem() string {
	switch r.Status {
	case StatusMissing:
		return fmt.Sprintf("%s is not installed (want %q)", r.Name, r.Rule)
	case StatusMismatch:
		if r.Found == nil {
			return fmt.Sprintf("%s: %s", r.Name, r.Detail)
		}

		return fmt.Sprintf("%s %s does not match %q", r.Name, r.Found, r.Rule)
	case StatusBroken:
		return fmt.Sprintf("%s is installed but not working: %s", r.Name, r.Detail)
	case StatusTimeout:
		return fmt.Sprintf("%s gave %s; set timeout: in the config if it is just slow to start", r.Name, r.Detail)
	}

	return ""
}
