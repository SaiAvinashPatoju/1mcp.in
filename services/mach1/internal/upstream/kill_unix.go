//go:build !windows

package upstream

import "os/exec"

// killProcessTree kills the process and all its descendants.
// On Unix we rely on the process group (set by SysProcAttr.Setpgid)
// being killed when we signal the negative PID.  For now we fall back
// to the standard Kill() because the supervisor does not yet set
// process groups.
func killProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
