package sandbox

import (
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/manifest"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/upstream"
)

// Process is the direct-exec driver: it just hands the resolved command and
// args to upstream.Client. No isolation. Suitable for node/python/binary
// MCPs the user has chosen to trust.
type Process struct{}

func (Process) Name() string { return "process" }

func (Process) Spec(id string, _ *manifest.Manifest, command string, args []string, cwd string, env map[string]string) (upstream.Spec, error) {
	return upstream.Spec{
		ID:      id,
		Command: command,
		Args:    args,
		Env:     env,
		Cwd:     cwd,
	}, nil
}
