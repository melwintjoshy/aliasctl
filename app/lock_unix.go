//go:build unix

package app

import (
	"os"
	"syscall"
)

// flock, so concurrent allow runs from several shells never lose each other's entries
func lockFile(file *os.File, exclusive bool) error {
	mode := syscall.LOCK_SH
	if exclusive {
		mode = syscall.LOCK_EX
	}

	return syscall.Flock(int(file.Fd()), mode)
}

func unlockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
