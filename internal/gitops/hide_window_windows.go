//go:build windows

package gitops

import (
	"os/exec"
	"syscall"
)

// hideConsoleWindow prevents a console window from flashing on screen each time
// a git subprocess is launched on Windows.
func hideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
