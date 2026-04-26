// Package sandbox abstracts how a child MCP is launched. The supervisor calls
// Driver.Spec(...) to get an exec-ready upstream.Spec; from that point on the
// upstream package handles JSON-RPC plumbing.
//
// Drivers:
//   - process: direct exec of the manifest command. Used for node/python/binary
//     runtimes. No isolation beyond OS user perms.
//   - docker:  `docker run -i --rm` with --network=none unless the manifest
//     declares network. Mounts come from manifest entrypoint.mounts.
//
// Phase 5.1+ will add wasm and deno drivers behind this same interface.
package sandbox

import (
	"fmt"

	"github.com/onemcp/central-mcp/internal/manifest"
	"github.com/onemcp/central-mcp/internal/upstream"
)

// Driver builds an upstream.Spec for a single managed MCP.
type Driver interface {
	// Name returns a short identifier for logging (e.g. "process", "docker").
	Name() string
	// Spec produces the launch spec. resolvedEnv is the merged env (manifest
	// defaults + registry env + secrets), already template-expanded by the
	// supervisor; the driver should not perform further expansion.
	Spec(id string, m *manifest.Manifest, command string, args []string, cwd string, resolvedEnv map[string]string) (upstream.Spec, error)
}

// Pick selects the appropriate driver for a manifest's runtime.
func Pick(m *manifest.Manifest) (Driver, error) {
	switch m.Runtime {
	case "node", "python", "binary":
		return &Process{}, nil
	case "docker":
		return &Docker{}, nil
	}
	return nil, fmt.Errorf("sandbox: unsupported runtime %q", m.Runtime)
}
