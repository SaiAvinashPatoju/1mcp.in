// Package paths centralizes filesystem layout decisions so mach1 and
// mach1ctl agree on where local state lives.
//
// Layout (Windows):
//   %APPDATA%\\Mach1\\
//     registry.db
//     secrets.json
//     mcps\\<id>\\
//
// Layout (POSIX):
//   $XDG_DATA_HOME/mach1 or ~/.mach1
package paths

import (
	"errors"
	"os"
	"path/filepath"
)

// Root returns the 1mcp.in data dir, honoring MACH1_HOME for tests.
func Root() (string, error) {
	if v := os.Getenv("MACH1_HOME"); v != "" {
		return v, nil
	}
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		return filepath.Join(appdata, "Mach1"), nil
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "mach1"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("paths: cannot determine home dir")
	}
	return filepath.Join(home, ".mach1"), nil
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
