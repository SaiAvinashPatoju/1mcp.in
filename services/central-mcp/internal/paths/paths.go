// Package paths centralizes filesystem layout decisions so centralmcpd and
// onemcpctl never disagree on where the registry, secrets, or per-MCP data
// directories live.
//
// Layout (Windows):
//   %APPDATA%\OneMcp\
//     ├── registry.db
//     ├── secrets.json   (mode 0600)
//     └── mcps\<id>\     (per-MCP scratch dir, used as cwd)
//
// Layout (POSIX): $XDG_DATA_HOME or ~/.onemcp.
package paths

import (
	"errors"
	"os"
	"path/filepath"
)

// Root returns the OneMCP data dir, honoring ONEMCP_HOME for tests.
func Root() (string, error) {
	if v := os.Getenv("ONEMCP_HOME"); v != "" {
		return v, nil
	}
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		return filepath.Join(appdata, "OneMcp"), nil
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "onemcp"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("paths: cannot determine home dir")
	}
	return filepath.Join(home, ".onemcp"), nil
}

func RegistryDB() (string, error) {
	r, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, "registry.db"), nil
}

func SecretsFile() (string, error) {
	r, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(r, "secrets.json"), nil
}

// MCPDataDir returns (and ensures) the scratch directory for a given MCP id.
func MCPDataDir(id string) (string, error) {
	r, err := Root()
	if err != nil {
		return "", err
	}
	d := filepath.Join(r, "mcps", id)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return d, nil
}
