//go:build unix

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/melwintjoshy/aliasctl/resolver"
)

func TestCheckKillsGrandchildrenOnTimeout(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")

	// the wrapper backgrounds a sleeper and records its pid, like a launcher script would
	writeTool(t, dir, "wrapper", "/bin/sleep 60 &\necho $! > "+pidFile+"\nwait")

	// generous enough for sh to fork the sleeper before the group is killed
	results := Checker{Timeout: time.Second}.Check(
		context.Background(),
		[]resolver.ToolRequirement{requirement("wrapper", "*")},
		[]string{"PATH=" + dir},
	)

	if results[0].Status != StatusTimeout {
		t.Fatalf("expected a timeout, got %+v", results[0])
	}

	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	// the kill is asynchronous, so allow a moment for the sleeper to disappear
	deadline := time.Now().Add(2 * time.Second)

	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err == syscall.ESRCH {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("grandchild %d survived the timeout", pid)
}
