// Package install materializes a catalog manifest into the local registry.
//
// "Install" in the 1mcp.in MVP is intentionally lightweight:
//   - node/python: nothing is fetched at install time. Runtime deps are
//     resolved by `npx -y` / `uvx` on first use, which already cache.
//   - docker: `docker pull <image>` happens here, best-effort. If the daemon
//     is unreachable the install still succeeds with a warning so the user
//     can install Docker later.
//   - binary: not yet supported (Phase 5).
//
// The trade-off is fast installs; the cost is first-run latency. Phase 5's
// supervisor offsets this with a "warming" notification in the hub UI.
//
// Unlike the original design, install.go now also checks runtime prerequisites
// (npx for node, uvx for python) and attempts to auto-install missing runners
// so users never see a opaque "child exited" error at startup.
package install

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/manifest"
	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/registry"
)

// runnerForRuntime maps MCP runtimes to the CLI tool that resolves deps.
func runnerForRuntime(runtime string) string {
	switch runtime {
	case "node":
		return "npx"
	case "python":
		return "uvx"
	case "docker":
		return "docker"
	default:
		return ""
	}
}

// runnerInstallHint returns a human-readable install command for a runner.
func runnerInstallHint(runner string) string {
	switch runner {
	case "npx":
		return "Install Node.js (includes npm/npx) from https://nodejs.org"
	case "uvx":
		return "Install uv (includes uvx): pip install uv  (or  curl -LsSf https://astral.sh/uv/install.sh | sh)"
	case "docker":
		return "Install Docker from https://docker.com"
	default:
		return ""
	}
}

// EnsureRuntimeRunner checks that the runner for a given runtime exists in
// PATH. For `python` runtime it attempts to auto-install uv if uvx is missing.
// Returns a clear actionable error if the runner cannot be resolved.
func EnsureRuntimeRunner(ctx context.Context, runtime string, logger *slog.Logger) error {
	runner := runnerForRuntime(runtime)
	if runner == "" {
		return nil // unknown runtime, skip check
	}
	_, err := exec.LookPath(runner)
	if err == nil {
		return nil
	}

	// Python: try to auto-install uv via pip.
	if runtime == "python" {
		if logger != nil {
			logger.Info("uvx not found; attempting pip install uv")
		}
		pipPath, pipErr := exec.LookPath("pip")
		if pipErr == nil {
			installCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			cmd := exec.CommandContext(installCtx, pipPath, "install", "uv")
			if out, pErr := cmd.CombinedOutput(); pErr == nil {
				// Verify uvx is now available.
				if _, lpErr := exec.LookPath("uvx"); lpErr == nil {
					if logger != nil {
						logger.Info("uv installed via pip")
					}
					return nil
				}
				if logger != nil {
					logger.Warn("pip install uv succeeded but uvx still not in PATH", "output", string(out))
				}
			} else {
				if logger != nil {
					logger.Warn("pip install uv failed", "err", pErr, "output", truncate(string(out), 256))
				}
			}
		} else {
			if logger != nil {
				logger.Warn("pip not found either; cannot auto-install uv", "err", pipErr)
			}
		}
	}

	return fmt.Errorf("%s not found in PATH. %s", runner, runnerInstallHint(runner))
}

// Result reports what happened during an install (for CLI/UI display).
type Result struct {
	ID           string
	Already      bool   // already installed; this was a re-install
	Verification string // marketplace trust class verified before install
	Warning      string // non-fatal message (e.g. docker daemon unreachable)
	Duration     time.Duration
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
	if err := manifest.VerifyCatalogDigest(m); err != nil {
		return nil, fmt.Errorf("supply-chain verification failed: %w", err)
	}

	// Preserve any existing env so re-install of an already-configured MCP
	// doesn't drop the user's settings.
	var preservedEnv map[string]string
	already := false
	if existing, _, err := i.DB.Get(ctx, m.ID); err == nil {
		preservedEnv = existing.Env
		already = true
	}

	res := &Result{ID: m.ID, Already: already, Verification: m.Verification}

	// Ensure runtime runner exists. For python, auto-install uv if missing.
	if err := EnsureRuntimeRunner(ctx, m.Runtime, i.Logger); err != nil {
		return nil, fmt.Errorf("runtime prerequisite: %w", err)
	}

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
