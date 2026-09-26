//go:build unix

package app

import (
	"errors"
	"syscall"
)

// signal 0 only checks that the pid exists; EPERM means it does but belongs to someone else
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)

	return err == nil || errors.Is(err, syscall.EPERM)
}
