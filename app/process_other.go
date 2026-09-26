//go:build !unix

package app

import "os"

// FindProcess opens a handle here, so it fails for a pid that is gone
func processAlive(pid int) bool {
	_, err := os.FindProcess(pid)

	return err == nil
}
