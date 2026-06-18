//go:build !windows

package gitops

import "os/exec"

// hideConsoleWindow is a no-op on non-Windows platforms.
func hideConsoleWindow(cmd *exec.Cmd) {}
