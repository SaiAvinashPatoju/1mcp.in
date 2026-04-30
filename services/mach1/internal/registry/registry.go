// Package registry persists the list of installed MCPs in SQLite (modernc.org
// pure-Go driver, no CGO). The hub writes rows on install/uninstall; the
// router reads them at startup to know which children to launch.
//
// Phase 1: schema is small and stable. Future phases will add columns for
// idle-shutdown timing, sandbox driver, and embedding-vector references.
package registry

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Schema is applied at Open. CREATE IF NOT EXISTS makes it idempotent; bumps
// to schema must go through migrations (see TODO at bottom).
const Schema = `
CREATE TABLE IF NOT EXISTS installed_mcps (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    version       TEXT NOT NULL,
    enabled       INTEGER NOT NULL DEFAULT 1,
    runtime       TEXT NOT NULL,
    command       TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    args_json     TEXT NOT NULL DEFAULT '[]',
    env_json      TEXT NOT NULL DEFAULT '{}',
    cwd           TEXT NOT NULL DEFAULT '',
    manifest_json TEXT NOT NULL,
    installed_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS tool_definition_hashes (
	mcp_id        TEXT NOT NULL,
	tool_name     TEXT NOT NULL,
	approved_hash TEXT NOT NULL,
	current_hash  TEXT NOT NULL,
	status        TEXT NOT NULL DEFAULT 'APPROVED',
	first_seen_at INTEGER NOT NULL,
	updated_at    INTEGER NOT NULL,
	PRIMARY KEY (mcp_id, tool_name)
);
`

// Entry is the persisted record for a single installed MCP.
type Entry struct {
	ID      string
	Name    string
	Version string
	Enabled bool
	Runtime string
	Command string
	Args    []string
	Env     map[string]string
	Cwd     string
}

const (
	ToolStatusApproved      = "APPROVED"
	ToolStatusPendingReview = "PENDING_REVIEW"
)

type ToolDefinition struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

type ToolReview struct {
	MCPID        string
	ToolName     string
	ApprovedHash string
	CurrentHash  string
	Status       string
	UpdatedAt    int64
}

// DB wraps a sql.DB with typed helpers.
type DB struct {
	sql *sql.DB
}

// Open opens (and creates if needed) the registry database at path.
func Open(path string) (*DB, error) {
	sdb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// modernc/sqlite: keep a small pool to avoid SQLITE_BUSY under contention.
	sdb.SetMaxOpenConns(4)
	// WAL mode enables concurrent readers even while a writer is active.
	// The Tauri Hub UI holds a connection to the same DB, so this prevents
	// SQLITE_BUSY when both processes access the file at the same time.
	if _, err := sdb.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;"); err != nil {
		_ = sdb.Close()
		return nil, fmt.Errorf("set WAL: %w", err)
	}
	if _, err := sdb.Exec(Schema); err != nil {
		_ = sdb.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &DB{sql: sdb}, nil
}

func (d *DB) Close() error { return d.sql.Close() }

// ListEnabled returns enabled MCPs sorted by id for stable router behavior.
func (d *DB) ListEnabled(ctx context.Context) ([]Entry, error) {
	rows, err := d.sql.QueryContext(ctx, `
        SELECT id, name, version, enabled, runtime, command, args_json, env_json, cwd
        FROM installed_mcps
        WHERE enabled = 1
        ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Entry
	for rows.Next() {
		var e Entry
		var enabled int
		var argsJSON, envJSON string
		if err := rows.Scan(&e.ID, &e.Name, &e.Version, &enabled, &e.Runtime, &e.Command, &argsJSON, &envJSON, &e.Cwd); err != nil {
			return nil, err
		}
		e.Enabled = enabled == 1
		if err := json.Unmarshal([]byte(argsJSON), &e.Args); err != nil {
			return nil, fmt.Errorf("decode args for %s: %w", e.ID, err)
		}
		if err := json.Unmarshal([]byte(envJSON), &e.Env); err != nil {
			return nil, fmt.Errorf("decode env for %s: %w", e.ID, err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Upsert inserts or replaces an entry. Used by the hub at install time and by
// the dev seeder.
func (d *DB) Upsert(ctx context.Context, e Entry, manifestJSON []byte, installedAtUnix int64) error {
	if e.ID == "" {
		return errors.New("registry: empty id")
	}
	if e.Args == nil {
		e.Args = []string{}
	}
	if e.Env == nil {
		e.Env = map[string]string{}
	}
	args, _ := json.Marshal(e.Args)
	env, _ := json.Marshal(e.Env)
	enabled := 0
	if e.Enabled {
		enabled = 1
	}
	_, err := d.sql.ExecContext(ctx, `
        INSERT INTO installed_mcps
            (id, name, version, enabled, runtime, command, args_json, env_json, cwd, manifest_json, installed_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            name=excluded.name,
            version=excluded.version,
            enabled=excluded.enabled,
            runtime=excluded.runtime,
            command=excluded.command,
            args_json=excluded.args_json,
            env_json=excluded.env_json,
            cwd=excluded.cwd,
            manifest_json=excluded.manifest_json
    `, e.ID, e.Name, e.Version, enabled, e.Runtime, e.Command, string(args), string(env), e.Cwd, string(manifestJSON), installedAtUnix)
	return err
}

// Delete removes an entry by id.
func (d *DB) Delete(ctx context.Context, id string) error {
	_, err := d.sql.ExecContext(ctx, `DELETE FROM installed_mcps WHERE id = ?`, id)
	return err
}

// Get returns a single entry by id (including disabled). Returns sql.ErrNoRows
// if not present.
func (d *DB) Get(ctx context.Context, id string) (*Entry, []byte, error) {
	row := d.sql.QueryRowContext(ctx, `
        SELECT id, name, version, enabled, runtime, command, args_json, env_json, cwd, manifest_json
        FROM installed_mcps WHERE id = ?`, id)
	var e Entry
	var enabled int
	var argsJSON, envJSON, manifestJSON string
	if err := row.Scan(&e.ID, &e.Name, &e.Version, &enabled, &e.Runtime, &e.Command, &argsJSON, &envJSON, &e.Cwd, &manifestJSON); err != nil {
		return nil, nil, err
	}
	e.Enabled = enabled == 1
	if err := json.Unmarshal([]byte(argsJSON), &e.Args); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal([]byte(envJSON), &e.Env); err != nil {
		return nil, nil, err
	}
	return &e, []byte(manifestJSON), nil
}

// ListAll returns every installed entry regardless of enabled state.
func (d *DB) ListAll(ctx context.Context) ([]Entry, error) {
	rows, err := d.sql.QueryContext(ctx, `
        SELECT id, name, version, enabled, runtime, command, args_json, env_json, cwd
        FROM installed_mcps ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		var enabled int
		var argsJSON, envJSON string
		if err := rows.Scan(&e.ID, &e.Name, &e.Version, &enabled, &e.Runtime, &e.Command, &argsJSON, &envJSON, &e.Cwd); err != nil {
			return nil, err
		}
		e.Enabled = enabled == 1
		_ = json.Unmarshal([]byte(argsJSON), &e.Args)
		_ = json.Unmarshal([]byte(envJSON), &e.Env)
		out = append(out, e)
	}
	return out, rows.Err()
}

// SetEnabled flips the enabled flag without touching anything else.
func (d *DB) SetEnabled(ctx context.Context, id string, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	res, err := d.sql.ExecContext(ctx, `UPDATE installed_mcps SET enabled=? WHERE id=?`, v, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("registry: not installed")
	}
	return nil
}

// SetEnv replaces the non-secret env_json blob.
func (d *DB) SetEnv(ctx context.Context, id string, env map[string]string) error {
	if env == nil {
		env = map[string]string{}
	}
	b, _ := json.Marshal(env)
	res, err := d.sql.ExecContext(ctx, `UPDATE installed_mcps SET env_json=? WHERE id=?`, string(b), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("registry: not installed")
	}
	return nil
}

// VerifyToolDefinitions stores first-seen tool definition hashes and marks a
// tool as PENDING_REVIEW when its description or input schema changes later.
func (d *DB) VerifyToolDefinitions(ctx context.Context, mcpID string, defs []ToolDefinition) (map[string]ToolReview, error) {
	now := time.Now().Unix()
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	out := make(map[string]ToolReview, len(defs))
	for _, def := range defs {
		hash, err := HashToolDefinition(def)
		if err != nil {
			return nil, err
		}
		var existing ToolReview
		row := tx.QueryRowContext(ctx, `
			SELECT approved_hash, current_hash, status, updated_at
			FROM tool_definition_hashes
			WHERE mcp_id = ? AND tool_name = ?`, mcpID, def.Name)
		err = row.Scan(&existing.ApprovedHash, &existing.CurrentHash, &existing.Status, &existing.UpdatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO tool_definition_hashes (mcp_id, tool_name, approved_hash, current_hash, status, first_seen_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)`, mcpID, def.Name, hash, hash, ToolStatusApproved, now, now)
			if err != nil {
				return nil, err
			}
			out[def.Name] = ToolReview{MCPID: mcpID, ToolName: def.Name, ApprovedHash: hash, CurrentHash: hash, Status: ToolStatusApproved, UpdatedAt: now}
			continue
		}
		if err != nil {
			return nil, err
		}
		status := ToolStatusApproved
		if existing.ApprovedHash != hash {
			status = ToolStatusPendingReview
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE tool_definition_hashes
			SET current_hash = ?, status = ?, updated_at = ?
			WHERE mcp_id = ? AND tool_name = ?`, hash, status, now, mcpID, def.Name)
		if err != nil {
			return nil, err
		}
		out[def.Name] = ToolReview{MCPID: mcpID, ToolName: def.Name, ApprovedHash: existing.ApprovedHash, CurrentHash: hash, Status: status, UpdatedAt: now}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (d *DB) ListPendingToolReviews(ctx context.Context) ([]ToolReview, error) {
	rows, err := d.sql.QueryContext(ctx, `
		SELECT mcp_id, tool_name, approved_hash, current_hash, status, updated_at
		FROM tool_definition_hashes
		WHERE status = ?
		ORDER BY updated_at DESC`, ToolStatusPendingReview)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ToolReview
	for rows.Next() {
		var r ToolReview
		if err := rows.Scan(&r.MCPID, &r.ToolName, &r.ApprovedHash, &r.CurrentHash, &r.Status, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (d *DB) ApproveToolDefinition(ctx context.Context, mcpID, toolName string) error {
	res, err := d.sql.ExecContext(ctx, `
		UPDATE tool_definition_hashes
		SET approved_hash = current_hash, status = ?, updated_at = ?
		WHERE mcp_id = ? AND tool_name = ?`, ToolStatusApproved, time.Now().Unix(), mcpID, toolName)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("registry: tool definition not found")
	}
	return nil
}

func HashToolDefinition(def ToolDefinition) (string, error) {
	inputSchema := any(map[string]any{})
	if len(def.InputSchema) > 0 {
		if err := json.Unmarshal(def.InputSchema, &inputSchema); err != nil {
			return "", fmt.Errorf("decode input schema for %s: %w", def.Name, err)
		}
	}
	canonical, err := json.Marshal(struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		InputSchema any    `json:"inputSchema"`
	}{Name: def.Name, Description: def.Description, InputSchema: inputSchema})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// TODO(phase 3+): add a `schema_version` table and a migration runner before
// adding columns. Do not silently ALTER TABLE without a version bump.
