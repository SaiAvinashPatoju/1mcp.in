package sandbox

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/onemcp/central-mcp/internal/manifest"
	"github.com/onemcp/central-mcp/internal/upstream"
)

// Docker launches a child MCP inside a `docker run -i --rm` container.
//
// Stdio is piped through the docker CLI: stopping our process (kill) cascades
// to the container thanks to --rm. We pass --network=none unless the manifest
// explicitly grants network access.
//
// Env vars are forwarded with -e NAME, with values taken from the resolved
// env map (so secrets do not appear in the docker command line).
type Docker struct {
	// CLI overrides the docker binary path (test seam). Empty means "docker".
	CLI string
}

func (Docker) Name() string { return "docker" }

func (d Docker) Spec(id string, m *manifest.Manifest, _ string, _ []string, _ string, env map[string]string) (upstream.Spec, error) {
	if m.Entrypoint.Image == "" {
		return upstream.Spec{}, fmt.Errorf("docker driver: empty image for %s", id)
	}
	cli := d.CLI
	if cli == "" {
		cli = "docker"
	}
	args := []string{"run", "-i", "--rm", "--name", containerName(id)}

	if m.Permissions == nil || !m.Permissions.Network {
		args = append(args, "--network=none")
	}

	// Forward env names; docker pulls the values from our process env, which
	// upstream.Client.Start sets just before exec via cmd.Env.
	for name := range env {
		args = append(args, "-e", name)
	}
	for _, mnt := range m.Entrypoint.Mounts {
		flag := fmt.Sprintf("type=bind,source=%s,target=%s", mnt.Source, mnt.Target)
		if mnt.ReadOnly {
			flag += ",readonly"
		}
		args = append(args, "--mount", flag)
	}
	args = append(args, m.Entrypoint.Image)
	args = append(args, m.Entrypoint.Args...)

	return upstream.Spec{
		ID:      id,
		Command: cli,
		Args:    args,
		Env:     env,
	}, nil
}

func containerName(id string) string {
	// Docker container names must match [a-zA-Z0-9][a-zA-Z0-9_.-]+
	var b [4]byte
	_, _ = rand.Read(b[:])
	return "onemcp-" + id + "-" + hex.EncodeToString(b[:])
}
