// Package install materializes a catalog manifest into the local registry.
//
// "Install" in the OneMCP MVP is intentionally lightweight:
//   - node/python: nothing is fetched at install time. Runtime deps are
//     resolved by `npx -y` / `uvx` on first use, which already cache.
//   - docker: `docker pull <image>` happens here, best-effort. If the daemon
//     is unreachable the install still succeeds with a warning so the user
//     can install Docker later.
//   - binary: not yet supported (Phase 5).
//
// The trade-off is fast installs; the cost is first-run latency. Phase 5's
// supervisor offsets this with a "warming" notification in the hub UI.
package install

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/onemcp/central-mcp/internal/manifest"
	"github.com/onemcp/central-mcp/internal/registry"
)

// Result reports what happened during an install (for CLI/UI display).
type Result struct {
	ID       string
	Already  bool   // already installed; this was a re-install
	Warning  string // non-fatal message (e.g. docker daemon unreachable)
	Duration time.Duration
}

// Installer is the unit-of-work for install/uninstall flows. It deliberately
// does not own the DB handle's lifecycle; callers pass an already-open *DB.
type Installer struct {
	DB     *registry.DB
	Logger *slog.Logger
}

// Install validates m and writes it to the registry. Idempotent: a second
// install of the same id replaces the row but preserves user-set env (caller
// is responsible for not clobbering env when re-installing the same version).
func (i *Installer) Install(ctx context.Context, m *manifest.Manifest) (*Result, error) {
	start := time.Now()
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	// Preserve any existing env so re-install of an already-configured MCP
	// doesn't drop the user's settings.
	var preservedEnv map[string]string
	already := false
	if existing, _, err := i.DB.Get(ctx, m.ID); err == nil {
		preservedEnv = existing.Env
		already = true
	}

	res := &Result{ID: m.ID, Already: already}

	// Docker pre-pull, best-effort.
	if m.Runtime == "docker" {
		if err := dockerPull(ctx, m.Entrypoint.Image, i.Logger); err != nil {
			res.Warning = fmt.Sprintf("docker pull failed (will retry on first use): %v", err)
			i.Logger.Warn("docker pull failed", "id", m.ID, "image", m.Entrypoint.Image, "err", err)
		}
	}

	// Build the registry Entry. Command/args are stored as authored in the
	// manifest; ${VAR} expansion happens at supervisor launch time so users
	// can re-bind values without re-installing.
	entry := registry.Entry{
		ID:      m.ID,
		Name:    m.Name,
		Version: m.Version,
		Enabled: true,
		Runtime: m.Runtime,
		Command: pickCommand(m),
		Args:    m.Entrypoint.Args,
		Env:     preservedEnv,
		Cwd:     m.Entrypoint.Cwd,
	}

	manifestJSON, err := manifestToJSON(m)
	if err != nil {
		return nil, err
	}
	if err := i.DB.Upsert(ctx, entry, manifestJSON, time.Now().Unix()); err != nil {
		return nil, fmt.Errorf("registry upsert: %w", err)
	}
	res.Duration = time.Since(start)
	return res, nil
}

// Uninstall removes the registry row. Caller is responsible for purging
// secrets (see secrets.Store.DeleteAll); this package does not depend on the
// secrets package to keep the dependency graph acyclic.
func (i *Installer) Uninstall(ctx context.Context, id string) error {
	return i.DB.Delete(ctx, id)
}

func pickCommand(m *manifest.Manifest) string {
	if m.Runtime == "docker" {
		// Supervisor's docker driver builds the actual `docker run` invocation;
		// we record the image as the command for visibility in `list` output.
		return m.Entrypoint.Image
	}
	return m.Entrypoint.Command
}

func dockerPull(ctx context.Context, image string, logger *slog.Logger) error {
	if image == "" {
		return errors.New("empty image")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker CLI not found in PATH: %w", err)
	}
	pullCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(pullCtx, "docker", "pull", image)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, truncate(string(out), 256))
	}
	if logger != nil {
		logger.Debug("docker pulled", "image", image)
	}
	return nil
}

func manifestToJSON(m *manifest.Manifest) ([]byte, error) {
	return jsonMarshal(m)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
