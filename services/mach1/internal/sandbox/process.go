package sandbox

import (
	"fmt"
	"os/exec"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/manifest"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/upstream"
)

// Process is the direct-exec driver: it just hands the resolved command and
// args to upstream.Client. No isolation. Suitable for node/python/binary
// MCPs the user has chosen to trust.
type Process struct{}

func (Process) Name() string { return "process" }

// Spec validates that the command exists in PATH before returning the spec.
// This gives users a clear "npx not found" error at warmup instead of an
// opaque "child exited" at call time.
func (Process) Spec(id string, m *manifest.Manifest, command string, args []string, cwd string, env map[string]string) (upstream.Spec, error) {
	if _, err := exec.LookPath(command); err != nil {
		hint := runtimeHint(m.Runtime, command)
		return upstream.Spec{}, fmt.Errorf("command %q not found in PATH (runtime=%q): %w. %s",
			command, m.Runtime, err, hint)
	}
	return upstream.Spec{
		ID:      id,
		Command: command,
		Args:    args,
		Env:     env,
		Cwd:     cwd,
	}, nil
}

// runtimeHint returns a human-readable installation hint for a given runtime/command.
func runtimeHint(runtime, command string) string {
	switch runtime {
	case "node":
		return "Install Node.js (includes npx) from https://nodejs.org"
	case "python":
		return "Install uv (includes uvx): pip install uv  or  https://docs.astral.sh/uv/"
	case "binary":
		return fmt.Sprintf("Ensure %s is installed and in PATH", command)
	default:
		return ""
	}
}
