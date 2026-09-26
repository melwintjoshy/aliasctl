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

	// the deadline is tripped by hand once the sleeper exists, so a slow sh can never lose the race
	ctx := &manualDeadline{Context: context.Background(), done: make(chan struct{})}

	results := make(chan []Result, 1)

	go func() {
		results <- Checker{Timeout: testTimeout}.Check(
			ctx,
			[]resolver.ToolRequirement{requirement("wrapper", "*")},
			[]string{"PATH=" + dir},
		)
	}()

	pid := waitForPid(t, pidFile)

	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	ctx.expire()

	if result := (<-results)[0]; result.Status != StatusTimeout {
		t.Fatalf("expected a timeout, got %+v", result)
	}

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

// polls until the wrapper has written a whole line, which under a loaded suite can take a while
func waitForPid(t *testing.T, path string) int {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil && strings.HasSuffix(string(data), "\n") {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				return pid
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("wrapper never recorded the sleeper's pid")

	return 0
}

// a context whose deadline passes when the test says so; WithTimeout inherits it as DeadlineExceeded
type manualDeadline struct {
	context.Context
	done chan struct{}
}

func (m *manualDeadline) Done() <-chan struct{} { return m.done }

func (m *manualDeadline) Err() error {
	select {
	case <-m.done:
		return context.DeadlineExceeded
	default:
		return nil
	}
}

func (m *manualDeadline) expire() { close(m.done) }
