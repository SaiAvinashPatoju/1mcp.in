//go:build windows

package upstream

import (
	"os/exec"
	"strconv"
	"syscall"
)

// killProcessTree kills the process and all its descendants on Windows.
// go's cmd.Process.Kill() only kills the immediate process; child processes
// (e.g. node.exe spawned by npx) survive and leak.  We use taskkill /T
// which sends the signal to the entire tree.
func killProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	// /T = terminate tree, /F = force, /IM = image name (fallback if PID fails)
	kill := exec.Command("taskkill", "/T", "/F", "/PID", pid)
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = kill.Run()
	// Fallback to direct Kill() in case taskkill isn't available.
	return cmd.Process.Kill()
}
