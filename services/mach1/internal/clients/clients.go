// Package clients writes (and removes) the 1mcp.in entry in the config files
// of supported MCP clients: VS Code, Cursor, Claude Desktop.
//
// ConnectTakeover REPLACES the entire client MCP config with a single mach1
// entry. Existing entries are backed up to a sidecar file so DisconnectRestore
// can bring them back. This ensures the AI client has no choice but to route
// every tool call through mach1.
package clients

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pelletier/go-toml/v2"
)

// Kind is a supported client.
type Kind string

const (
	VSCode      Kind = "vscode"
	Cursor      Kind = "cursor"
	Claude      Kind = "claude"
	ClaudeCode  Kind = "claudecode"
	Windsurf    Kind = "windsurf"
	Codex       Kind = "codex"
	OpenCode    Kind = "opencode"
	Antigravity Kind = "antigravity"
)

// All returns the kinds we know how to configure.
func All() []Kind {
	return []Kind{VSCode, Cursor, Claude, ClaudeCode, Windsurf, Codex, OpenCode, Antigravity}
}

// EntryName is the key under which 1mcp.in registers itself in client configs.
const EntryName = "mach1"

// ServerEntry is the per-server JSON shape every supported client uses.
type ServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Type    string            `json:"type,omitempty"`
}

// HTTPEntry is the per-server JSON shape for HTTP transport clients.
type HTTPEntry struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Type    string            `json:"type,omitempty"`
}

// BackupResult describes what happened during a takeover backup.
type BackupResult struct {
	BackedUpCount int
	BackupPath    string
}

// ---------------------------------------------------------------------------
// Takeover (connect) – replace everything with mach1
// ---------------------------------------------------------------------------

// ConnectTakeover reads the client's existing MCP config, backs up every
// non-mach1 entry to a sidecar file, then writes a config that contains ONLY
// the mach1 entry. Returns the config path and backup metadata.
func ConnectTakeover(kind Kind, entry ServerEntry) (path string, backup BackupResult, err error) {
	path, key, err := configPath(kind)
	if err != nil {
		return "", backup, err
	}
	root, err := readJSONObject(path)
	if err != nil {
		return "", backup, err
	}
	servers, _ := root[key].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	// 1. Build backup of everything except mach1
	backupEntries := map[string]any{}
	for name, val := range servers {
		if name == EntryName {
			continue
		}
		backupEntries[name] = val
	}
	backup.BackedUpCount = len(backupEntries)
	backup.BackupPath = backupPath(path)

	if backup.BackedUpCount > 0 {
		if err := writeJSONObject(backup.BackupPath, backupEntries); err != nil {
			return "", backup, fmt.Errorf("write backup: %w", err)
		}
	} else {
		// No prior entries – ensure no stale backup file exists
		_ = os.Remove(backup.BackupPath)
	}

	// 2. Replace the entire servers section with ONLY mach1
	b, _ := json.Marshal(entry)
	var asAny any
	_ = json.Unmarshal(b, &asAny)
	root[key] = map[string]any{EntryName: asAny}

	if err := writeJSONObject(path, root); err != nil {
		return "", backup, err
	}
	return path, backup, nil
}

// ConnectTakeoverOpenCode is the OpenCode-specific variant of ConnectTakeover.
func ConnectTakeoverOpenCode(entry ServerEntry) (path string, backup BackupResult, err error) {
	path, key, err := configPath(OpenCode)
	if err != nil {
		return "", backup, err
	}
	root, err := readJSONCObject(path)
	if err != nil {
		return "", backup, err
	}
	servers, _ := root[key].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	backupEntries := map[string]any{}
	for name, val := range servers {
		if name == EntryName {
			continue
		}
		backupEntries[name] = val
	}
	backup.BackedUpCount = len(backupEntries)
	backup.BackupPath = backupPath(path)

	if backup.BackedUpCount > 0 {
		if err := writeJSONObject(backup.BackupPath, backupEntries); err != nil {
			return "", backup, fmt.Errorf("write backup: %w", err)
		}
	} else {
		_ = os.Remove(backup.BackupPath)
	}

	var entryAny any
	if entry.Type == "http" || entry.Type == "remote" {
		entryAny = map[string]any{
			"type":    "remote",
			"url":     entry.Command,
			"enabled": true,
		}
	} else {
		cmd := append([]string{entry.Command}, entry.Args...)
		entryAny = map[string]any{
			"type":    "local",
			"command": cmd,
			"enabled": true,
		}
	}
	root[key] = map[string]any{EntryName: entryAny}

	if err := writeJSONObject(path, root); err != nil {
		return "", backup, err
	}
	return path, backup, nil
}

// ConnectTakeoverCodex is the Codex-specific (TOML) variant.
func ConnectTakeoverCodex(entry ServerEntry) (path string, backup BackupResult, err error) {
	path, _, err = configPath(Codex)
	if err != nil {
		return "", backup, err
	}

	var doc map[string]any
	if b, err := os.ReadFile(path); err == nil {
		if err := toml.Unmarshal(b, &doc); err != nil {
			return "", backup, fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", backup, fmt.Errorf("read %s: %w", path, err)
	}
	if doc == nil {
		doc = map[string]any{}
	}

	servers, _ := doc["mcp_servers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	backupEntries := map[string]any{}
	for name, val := range servers {
		if name == EntryName {
			continue
		}
		backupEntries[name] = val
	}
	backup.BackedUpCount = len(backupEntries)
	backup.BackupPath = backupPath(path)

	if backup.BackedUpCount > 0 {
		if err := writeTOMLBackup(backup.BackupPath, backupEntries); err != nil {
			return "", backup, fmt.Errorf("write backup: %w", err)
		}
	} else {
		_ = os.Remove(backup.BackupPath)
	}

	doc["mcp_servers"] = map[string]any{
		EntryName: map[string]any{
			"command": entry.Command,
			"args":    entry.Args,
		},
	}

	b, err := toml.Marshal(doc)
	if err != nil {
		return "", backup, fmt.Errorf("serialize %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", backup, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return "", backup, err
	}
	return path, backup, os.Rename(tmp, path)
}

// ConnectTakeoverHTTP is the HTTP variant for JSON-based clients.
func ConnectTakeoverHTTP(kind Kind, entry HTTPEntry) (path string, backup BackupResult, err error) {
	path, key, err := configPath(kind)
	if err != nil {
		return "", backup, err
	}
	root, err := readJSONObject(path)
	if err != nil {
		return "", backup, err
	}
	servers, _ := root[key].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	backupEntries := map[string]any{}
	for name, val := range servers {
		if name == EntryName {
			continue
		}
		backupEntries[name] = val
	}
	backup.BackedUpCount = len(backupEntries)
	backup.BackupPath = backupPath(path)

	if backup.BackedUpCount > 0 {
		if err := writeJSONObject(backup.BackupPath, backupEntries); err != nil {
			return "", backup, fmt.Errorf("write backup: %w", err)
		}
	} else {
		_ = os.Remove(backup.BackupPath)
	}

	obj := map[string]any{
		"type": "http",
		"url":  entry.URL,
	}
	if len(entry.Headers) > 0 {
		obj["headers"] = entry.Headers
	}
	root[key] = map[string]any{EntryName: obj}

	if err := writeJSONObject(path, root); err != nil {
		return "", backup, err
	}
	return path, backup, nil
}

// ConnectTakeoverHTTPCodex is the HTTP variant for Codex (TOML).
func ConnectTakeoverHTTPCodex(url string) (path string, backup BackupResult, err error) {
	path, _, err = configPath(Codex)
	if err != nil {
		return "", backup, err
	}

	var doc map[string]any
	if b, err := os.ReadFile(path); err == nil {
		if err := toml.Unmarshal(b, &doc); err != nil {
			return "", backup, fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", backup, fmt.Errorf("read %s: %w", path, err)
	}
	if doc == nil {
		doc = map[string]any{}
	}

	servers, _ := doc["mcp_servers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	backupEntries := map[string]any{}
	for name, val := range servers {
		if name == EntryName {
			continue
		}
		backupEntries[name] = val
	}
	backup.BackedUpCount = len(backupEntries)
	backup.BackupPath = backupPath(path)

	if backup.BackedUpCount > 0 {
		if err := writeTOMLBackup(backup.BackupPath, backupEntries); err != nil {
			return "", backup, fmt.Errorf("write backup: %w", err)
		}
	} else {
		_ = os.Remove(backup.BackupPath)
	}

	doc["mcp_servers"] = map[string]any{
		EntryName: map[string]any{
			"url":  url,
			"type": "http",
		},
	}

	b, err := toml.Marshal(doc)
	if err != nil {
		return "", backup, fmt.Errorf("serialize %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", backup, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return "", backup, err
	}
	return path, backup, os.Rename(tmp, path)
}

// ---------------------------------------------------------------------------
// Disconnect (restore) – bring back the backed-up entries
// ---------------------------------------------------------------------------

// DisconnectRestore removes the mach1 entry and restores every MCP that was
// backed up during ConnectTakeover. If no backup exists it simply removes
// mach1 and leaves everything else untouched.
func DisconnectRestore(kind Kind) (path string, restored int, err error) {
	path, key, err := configPath(kind)
	if err != nil {
		return "", 0, err
	}
	if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
		return path, 0, nil
	}

	root, err := readJSONObject(path)
	if err != nil {
		return path, 0, err
	}
	servers, _ := root[key].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	bp := backupPath(path)
	if _, statErr := os.Stat(bp); statErr == nil {
		// Restore backed-up entries
		backupEntries, err := readJSONObject(bp)
		if err != nil {
			return path, 0, fmt.Errorf("read backup: %w", err)
		}
		for name, val := range backupEntries {
			servers[name] = val
		}
		restored = len(backupEntries)
		_ = os.Remove(bp)
	}

	delete(servers, EntryName)
	root[key] = servers
	if err := writeJSONObject(path, root); err != nil {
		return path, restored, err
	}
	return path, restored, nil
}

// DisconnectRestoreOpenCode is the OpenCode-specific variant.
func DisconnectRestoreOpenCode() (path string, restored int, err error) {
	path, key, err := configPath(OpenCode)
	if err != nil {
		return "", 0, err
	}
	if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
		return path, 0, nil
	}

	root, err := readJSONCObject(path)
	if err != nil {
		return path, 0, err
	}
	servers, _ := root[key].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	bp := backupPath(path)
	if _, statErr := os.Stat(bp); statErr == nil {
		backupEntries, err := readJSONObject(bp)
		if err != nil {
			return path, 0, fmt.Errorf("read backup: %w", err)
		}
		for name, val := range backupEntries {
			servers[name] = val
		}
		restored = len(backupEntries)
		_ = os.Remove(bp)
	}

	delete(servers, EntryName)
	root[key] = servers
	if err := writeJSONObject(path, root); err != nil {
		return path, restored, err
	}
	return path, restored, nil
}

// DisconnectRestoreCodex is the Codex-specific (TOML) variant.
func DisconnectRestoreCodex() (path string, restored int, err error) {
	path, _, err = configPath(Codex)
	if err != nil {
		return "", 0, err
	}
	if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
		return path, 0, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return path, 0, err
	}
	var doc map[string]any
	if err := toml.Unmarshal(b, &doc); err != nil {
		return path, 0, fmt.Errorf("parse %s: %w", path, err)
	}

	servers, _ := doc["mcp_servers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	bp := backupPath(path)
	if _, statErr := os.Stat(bp); statErr == nil {
		backupEntries, err := readTOMLBackup(bp)
		if err != nil {
			return path, 0, fmt.Errorf("read backup: %w", err)
		}
		for name, val := range backupEntries {
			servers[name] = val
		}
		restored = len(backupEntries)
		_ = os.Remove(bp)
	}

	delete(servers, EntryName)
	doc["mcp_servers"] = servers

	out, err := toml.Marshal(doc)
	if err != nil {
		return path, restored, err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return path, restored, err
	}
	return path, restored, nil
}

// ---------------------------------------------------------------------------
// Backup helpers
// ---------------------------------------------------------------------------

func backupPath(configPath string) string {
	return configPath + ".mach1-backup"
}

func writeTOMLBackup(path string, entries map[string]any) error {
	b, err := toml.Marshal(entries)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func readTOMLBackup(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := toml.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Legacy additive helpers (kept for compatibility, but NOT used by connect)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Combined config + rules injection
// ---------------------------------------------------------------------------

// ConnectTakeoverWithRules performs ConnectTakeover and then injects the 1MCP
// system directive into the client's rule file (if supported).
// projectDir is optional - if empty, it attempts to find the project root.
func ConnectTakeoverWithRules(kind Kind, entry ServerEntry, projectDir string) (*TakeoverWithRulesResult, error) {
	result := &TakeoverWithRulesResult{}

	// Step 1: Config takeover
	var cfgPath string
	var backup BackupResult
	var err error

	switch kind {
	case OpenCode:
		cfgPath, backup, err = ConnectTakeoverOpenCode(entry)
	case Codex:
		cfgPath, backup, err = ConnectTakeoverCodex(entry)
	default:
		cfgPath, backup, err = ConnectTakeover(kind, entry)
	}

	if err != nil {
		return result, fmt.Errorf("config takeover: %w", err)
	}

	result.ConfigPath = cfgPath
	result.Backup = backup

	// Step 2: Find project directory if not provided
	if projectDir == "" {
		projectDir, err = FindProjectRoot("")
		if err != nil {
			// Non-fatal: rules injection is optional
			return result, nil
		}
	}
	result.ProjectDir = projectDir

	// Step 3: Inject rules
	rulesResult, err := InjectRules(kind, projectDir)
	if err != nil {
		// Non-fatal: rules injection is best-effort
		result.RulesError = err
		return result, nil
	}

	result.RulesPath = rulesResult.Path
	result.RulesInjected = rulesResult.Injected
	result.RulesAlreadyHad = rulesResult.AlreadyHad
	result.RulesCreated = rulesResult.Created

	return result, nil
}

// TakeoverWithRulesResult describes the outcome of ConnectTakeoverWithRules.
type TakeoverWithRulesResult struct {
	ConfigPath      string
	Backup          BackupResult
	ProjectDir      string
	RulesPath       string
	RulesInjected   bool
	RulesAlreadyHad bool
	RulesCreated    bool
	RulesError      error
}

// ConnectOpenCode writes the 1mcp.in entry in OpenCode's config format.
// OpenCode uses a "mcp" key with a different entry shape.
// When entry.Type is "http" or "remote", we use remote format with URL;
// otherwise we use local format with command array.
func ConnectOpenCode(entry ServerEntry) (path string, err error) {
	path, key, err := configPath(OpenCode)
	if err != nil {
		return "", err
	}
	root, err := readJSONCObject(path)
	if err != nil {
		return "", err
	}
	servers, _ := root[key].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	if entry.Type == "http" || entry.Type == "remote" {
		servers[EntryName] = map[string]any{
			"type":    "remote",
			"url":     entry.Command,
			"enabled": true,
		}
	} else {
		cmd := append([]string{entry.Command}, entry.Args...)
		servers[EntryName] = map[string]any{
			"type":    "local",
			"command": cmd,
			"enabled": true,
		}
	}
	root[key] = servers
	if err := writeJSONObject(path, root); err != nil {
		return "", err
	}
	return path, nil
}

// ConnectHTTP writes (or replaces) the 1mcp.in HTTP entry in client kind's
// config file. It uses the URL-based format suitable for HTTP/streamable transport.
func ConnectHTTP(kind Kind, entry HTTPEntry) (path string, err error) {
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
	obj := map[string]any{
		"type": "http",
		"url":  entry.URL,
	}
	if len(entry.Headers) > 0 {
		obj["headers"] = entry.Headers
	}
	servers[EntryName] = obj
	root[key] = servers
	if err := writeJSONObject(path, root); err != nil {
		return "", err
	}
	return path, nil
}

// ConnectCodex writes the 1mcp.in entry in Codex's TOML config format.
// Codex uses ~/.codex/config.toml with [mcp_servers.<name>] tables.
func ConnectCodex(entry ServerEntry) (path string, err error) {
	path, _, err = configPath(Codex)
	if err != nil {
		return "", err
	}

	var doc map[string]any
	if b, err := os.ReadFile(path); err == nil {
		if err := toml.Unmarshal(b, &doc); err != nil {
			return "", fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	if doc == nil {
		doc = map[string]any{}
	}

	servers, _ := doc["mcp_servers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers[EntryName] = map[string]any{
		"command": entry.Command,
		"args":    entry.Args,
	}
	doc["mcp_servers"] = servers

	b, err := toml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("serialize %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return "", err
	}
	return path, os.Rename(tmp, path)
}

// ConnectHTTPCodex writes the 1mcp.in HTTP entry in Codex's TOML config format.
func ConnectHTTPCodex(url string) (path string, err error) {
	path, _, err = configPath(Codex)
	if err != nil {
		return "", err
	}

	var doc map[string]any
	if b, err := os.ReadFile(path); err == nil {
		if err := toml.Unmarshal(b, &doc); err != nil {
			return "", fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	if doc == nil {
		doc = map[string]any{}
	}

	servers, _ := doc["mcp_servers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers[EntryName] = map[string]any{
		"url":  url,
		"type": "http",
	}
	doc["mcp_servers"] = servers

	b, err := toml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("serialize %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return "", err
	}
	return path, os.Rename(tmp, path)
}

// DisconnectCodex removes the 1mcp.in entry from Codex's TOML config.
func DisconnectCodex() (path string, removed bool, err error) {
	path, _, err = configPath(Codex)
	if err != nil {
		return "", false, err
	}
	if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
		return path, false, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return path, false, err
	}
	var doc map[string]any
	if err := toml.Unmarshal(b, &doc); err != nil {
		return path, false, fmt.Errorf("parse %s: %w", path, err)
	}
	servers, _ := doc["mcp_servers"].(map[string]any)
	if servers == nil {
		return path, false, nil
	}
	if _, ok := servers[EntryName]; !ok {
		return path, false, nil
	}
	delete(servers, EntryName)
	doc["mcp_servers"] = servers

	out, err := toml.Marshal(doc)
	if err != nil {
		return path, false, err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return path, false, err
	}
	return path, true, nil
}

// configPath returns the absolute path and the top-level key (e.g. "servers"
// for VS Code, "mcpServers" for Cursor/Claude) for the given client.
func configPath(kind Kind) (string, string, error) {
	switch kind {
	case VSCode:
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "Code", "User", "mcp.json"), "servers", nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "Code", "User", "mcp.json"), "servers", nil
		}
		if runtime.GOOS == "linux" {
			return filepath.Join(home, ".config", "Code", "User", "mcp.json"), "servers", nil
		}
		return filepath.Join(home, "Library", "Application Support", "Code", "User", "mcp.json"), "servers", nil
	case Cursor:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		return filepath.Join(home, ".cursor", "mcp.json"), "mcpServers", nil
	case Claude:
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "Claude", "claude_desktop_config.json"), "mcpServers", nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "Claude", "claude_desktop_config.json"), "mcpServers", nil
		}
		if runtime.GOOS == "linux" {
			return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"), "mcpServers", nil
		}
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), "mcpServers", nil
	case ClaudeCode:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		return filepath.Join(home, ".claude.json"), "mcpServers", nil
	case Windsurf:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		return filepath.Join(home, ".codeium", "windsurf", "mcp_config.json"), "mcpServers", nil
	case Codex:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		return filepath.Join(home, ".codex", "config.toml"), "", nil
	case OpenCode:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		return filepath.Join(home, ".config", "opencode", "opencode.json"), "mcp", nil
	case Antigravity:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		return filepath.Join(home, ".antigravity", "mcp.json"), "mcpServers", nil
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

// readJSONCObject reads a JSONC file (JSON with Comments) and returns the parsed object.
// It strips comments before parsing. Used for OpenCode's opencode.jsonc.
func readJSONCObject(path string) (map[string]any, error) {
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

	// Strip JSONC comments
	cleanJSON := StripJSONC(string(b))
	if len(cleanJSON) == 0 {
		return map[string]any{}, nil
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(cleanJSON), &m); err != nil {
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
