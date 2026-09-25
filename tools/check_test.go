package tools

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/melwintjoshy/aliasctl/resolver"
	"github.com/melwintjoshy/aliasctl/versions"
)

// writes a fake tool as a tiny sh script into dir
func writeTool(t *testing.T, dir, name, body string) {
	t.Helper()

	script := "#!/bin/sh\n" + body + "\n"

	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
}

func requirement(name, rule string, args ...string) resolver.ToolRequirement {
	return resolver.ToolRequirement{
		Name:  name,
		Rule:  rule,
		Check: resolver.Command{Name: name, Args: args},
	}
}

func TestCheckStatuses(t *testing.T) {
	dir := t.TempDir()

	writeTool(t, dir, "good", `echo "good version v1.22.4"`)
	writeTool(t, dir, "old", `echo "old 1.28.1"`)
	writeTool(t, dir, "failing", `echo "Unable to locate a Java Runtime." >&2; exit 1`)
	writeTool(t, dir, "silent", `echo "no digits here"`)
	writeTool(t, dir, "stderr", `echo "openjdk version 21.0.2" >&2`)

	// not executable, so it counts as missing
	if err := os.WriteFile(filepath.Join(dir, "noexec"), []byte("#!/bin/sh\necho 1.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	requirements := []resolver.ToolRequirement{
		requirement("old", ">=1.29"),
		requirement("good", "1.22", "--version"),
		requirement("absent", "*"),
		requirement("failing", "*"),
		requirement("silent", "*"),
		requirement("stderr", "21"),
		requirement("noexec", "*"),
	}

	results := Check(context.Background(), requirements, []string{"PATH=" + dir})

	got := make(map[string]Status)

	for _, result := range results {
		got[result.Name] = result.Status
	}

	expected := map[string]Status{
		"absent":  StatusMissing,
		"failing": StatusBroken,
		"good":    StatusOK,
		"noexec":  StatusMissing,
		"old":     StatusMismatch,
		"silent":  StatusBroken,
		"stderr":  StatusOK,
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	for i := 1; i < len(results); i++ {
		if results[i-1].Name > results[i].Name {
			t.Fatalf("results are not sorted: %s before %s", results[i-1].Name, results[i].Name)
		}
	}
}

func TestCheckReportsDetails(t *testing.T) {
	dir := t.TempDir()

	writeTool(t, dir, "good", `echo "v1.22.4"`)
	writeTool(t, dir, "failing", `echo "Unable to locate a Java Runtime." >&2; exit 1`)

	results := Check(
		context.Background(),
		[]resolver.ToolRequirement{requirement("failing", "*"), requirement("good", "1.22")},
		[]string{"PATH=" + dir},
	)

	if results[0].Detail != "Unable to locate a Java Runtime." {
		t.Fatalf("unexpected detail %q", results[0].Detail)
	}

	if results[1].Path != filepath.Join(dir, "good") {
		t.Fatalf("unexpected path %q", results[1].Path)
	}

	if !reflect.DeepEqual(results[1].Found, versions.Version{1, 22, 4}) {
		t.Fatalf("unexpected version %v", results[1].Found)
	}
}

func TestCheckTimesOut(t *testing.T) {
	dir := t.TempDir()

	// absolute path since the fake PATH holds only the temp dir
	writeTool(t, dir, "slow", "/bin/sleep 5; echo 1.0")

	start := time.Now()

	results := Checker{Timeout: 200 * time.Millisecond}.Check(
		context.Background(),
		[]resolver.ToolRequirement{requirement("slow", "*")},
		[]string{"PATH=" + dir},
	)

	if results[0].Status != StatusTimeout || results[0].Detail != "no answer within 200ms" {
		t.Fatalf("expected timeout, got %+v", results[0])
	}

	if time.Since(start) > 3*time.Second {
		t.Fatalf("timeout was not enforced, took %s", time.Since(start))
	}
}

func TestCheckUsesProjectEnvironment(t *testing.T) {
	dir := t.TempDir()

	// the probe must see project variables, e.g. a KUBECONFIG the tool reads
	writeTool(t, dir, "ctx", `echo "ctx $TOOL_VERSION"`)

	results := Check(
		context.Background(),
		[]resolver.ToolRequirement{requirement("ctx", "3.1")},
		[]string{"PATH=" + dir, "TOOL_VERSION=3.1.0"},
	)

	if results[0].Status != StatusOK {
		t.Fatalf("expected project variable to reach the probe, got %+v", results[0])
	}
}

func TestResultProblem(t *testing.T) {
	tests := []struct {
		result   Result
		expected string
	}{
		{Result{Name: "go", Status: StatusOK}, ""},
		{Result{Name: "tf", Rule: ">=1.6", Status: StatusMissing}, `tf is not installed (want ">=1.6")`},
		{
			Result{Name: "kubectl", Rule: "<1.30", Found: versions.Version{1, 36, 4}, Status: StatusMismatch},
			`kubectl 1.36.4 does not match "<1.30"`,
		},
		{
			Result{Name: "java", Status: StatusBroken, Detail: "Unable to locate a Java Runtime."},
			"java is installed but not working: Unable to locate a Java Runtime.",
		},
		{
			Result{Name: "swift", Status: StatusTimeout, Detail: "no answer within 5s"},
			"swift gave no answer within 5s; set timeout: in the config if it is just slow to start",
		},
	}

	for _, tt := range tests {
		if got := tt.result.Problem(); got != tt.expected {
			t.Fatalf("expected %q, got %q", tt.expected, got)
		}
	}
}

func TestCheckResolvesRelativeCheckAgainstDir(t *testing.T) {
	project := t.TempDir()

	if err := os.Mkdir(filepath.Join(project, "bin"), 0755); err != nil {
		t.Fatal(err)
	}

	writeTool(t, filepath.Join(project, "bin"), "mytool", `echo "mytool 2.1.0"`)

	requirements := []resolver.ToolRequirement{{
		Name:  "mytool",
		Rule:  "2.1",
		Check: resolver.Command{Name: "./bin/mytool", Args: []string{"--version"}},
	}}

	// the working directory is somewhere else, as when running from a subdirectory
	results := Checker{Timeout: DefaultTimeout, Dir: project}.Check(
		context.Background(),
		requirements,
		[]string{"PATH=" + t.TempDir()},
	)

	if results[0].Status != StatusOK || results[0].Path != filepath.Join(project, "bin", "mytool") {
		t.Fatalf("expected the check to resolve against the config directory, got %+v", results[0])
	}

	if results := Check(context.Background(), requirements, nil); results[0].Status != StatusMissing {
		t.Fatalf("expected a miss from an unrelated working directory, got %+v", results[0])
	}
}

func TestCheckRunsFromDir(t *testing.T) {
	project := t.TempDir()
	dir := t.TempDir()

	writeTool(t, dir, "here", `echo "here 1.0"; pwd`)

	results := Checker{Timeout: DefaultTimeout, Dir: project}.Check(
		context.Background(),
		[]resolver.ToolRequirement{requirement("here", "*")},
		[]string{"PATH=" + dir},
	)

	if results[0].Status != StatusOK {
		t.Fatalf("expected ok, got %+v", results[0])
	}
}

func TestFindExecutableKeepsDotEntriesRelative(t *testing.T) {
	checker := Checker{}

	// with an empty PATH entry a bare name would be looked up on PATH again by exec
	for _, path := range []string{"tool", "./tool", "bin/tool"} {
		if got := checker.resolve(path); !strings.Contains(got, "/") {
			t.Fatalf("resolve(%q) = %q, expected an explicit path", path, got)
		}
	}

	if got := checker.resolve("/usr/bin/tool"); got != "/usr/bin/tool" {
		t.Fatalf("absolute path changed to %q", got)
	}
}

func TestCheckHonoursRequirementTimeout(t *testing.T) {
	dir := t.TempDir()

	writeTool(t, dir, "slowstart", "/bin/sleep 0.5; echo 1.0")

	slow := requirement("slowstart", "*")
	slow.Timeout = 5 * time.Second

	results := Checker{Timeout: 100 * time.Millisecond}.Check(
		context.Background(),
		[]resolver.ToolRequirement{slow},
		[]string{"PATH=" + dir},
	)

	if results[0].Status != StatusOK {
		t.Fatalf("expected the per-tool timeout to apply, got %+v", results[0])
	}
}
