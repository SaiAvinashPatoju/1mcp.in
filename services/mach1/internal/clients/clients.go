// Package clients writes (and removes) the 1mcp.in entry in the config files
// of supported MCP clients: VS Code, Cursor, Claude Desktop.
//
// We intentionally read-modify-write the existing JSON instead of regenerating
// it from scratch so we never clobber the user's other servers.
package clients

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Kind is a supported client.
type Kind string

const (
	VSCode Kind = "vscode"
	Cursor Kind = "cursor"
	Claude Kind = "claude"
)

// All returns the kinds we know how to configure.
func All() []Kind { return []Kind{VSCode, Cursor, Claude} }

// EntryName is the key under which 1mcp.in registers itself in client configs.
const EntryName = "mach1"

// ServerEntry is the per-server JSON shape every supported client uses.
type ServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// Connect writes (or replaces) the 1mcp.in entry in client kind's config file.
// It creates the file and parent dirs if absent. The kind's existing entries
// are preserved.
func Connect(kind Kind, entry ServerEntry) (path string, err error) {
	path, key, err := configPath(kind)
	if err != nil {
		return "", err
	}
	root, err := readJSONObject(path)
	if err != nil {
		return "", err
	}
	servers, _ := root[key].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	b, _ := json.Marshal(entry)
	var asAny any
	_ = json.Unmarshal(b, &asAny)
	servers[EntryName] = asAny
	root[key] = servers
	if err := writeJSONObject(path, root); err != nil {
		return "", err
	}
	return path, nil
}

// Disconnect removes the 1mcp.in entry from the client config (if present).
// The file itself is left in place so other entries survive.
func Disconnect(kind Kind) (path string, removed bool, err error) {
	path, key, err := configPath(kind)
	if err != nil {
		return "", false, err
	}
	if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
		return path, false, nil
	}
	root, err := readJSONObject(path)
	if err != nil {
		return path, false, err
	}
	servers, _ := root[key].(map[string]any)
	if servers == nil {
		return path, false, nil
	}
	if _, ok := servers[EntryName]; !ok {
		return path, false, nil
	}
	delete(servers, EntryName)
	root[key] = servers
	if err := writeJSONObject(path, root); err != nil {
		return path, false, err
	}
	return path, true, nil
}

// configPath returns the absolute path and the top-level key (e.g. "servers"
// for VS Code, "mcpServers" for Cursor/Claude) for the given client.
func configPath(kind Kind) (string, string, error) {
	switch kind {
	case VSCode:
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", "", err
			}
			// macOS path; Linux uses ~/.config/Code/User which can be added later.
			return filepath.Join(home, "Library", "Application Support", "Code", "User", "mcp.json"), "servers", nil
		}
		return filepath.Join(appdata, "Code", "User", "mcp.json"), "servers", nil
	case Cursor:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		return filepath.Join(home, ".cursor", "mcp.json"), "mcpServers", nil
	case Claude:
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", "", err
			}
			return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), "mcpServers", nil
		}
		return filepath.Join(appdata, "Claude", "claude_desktop_config.json"), "mcpServers", nil
	}
	return "", "", fmt.Errorf("unsupported client kind: %s", kind)
}

func readJSONObject(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(b) == 0 {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func writeJSONObject(path string, m map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
