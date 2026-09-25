//go:build !unix

package tools

import "os/exec"

// only the direct child can be killed here
func isolateProcessGroup(cmd *exec.Cmd) {}
