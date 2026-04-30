// Package clouddb manages the Railway-hosted Postgres connection used for
// shared state: user accounts, sessions, and marketplace metadata.
//
// Local MCP configuration (installed plugins, tokens, offline caching) is
// intentionally kept in SQLite on the device — this package only touches the
// cloud tier.
package clouddb

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// schema is applied once at startup (CREATE IF NOT EXISTS is idempotent).
const schema = `
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE NOT NULL,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_sessions (
    token      TEXT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '30 days'
);

CREATE TABLE IF NOT EXISTS marketplace_items (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    version      TEXT NOT NULL,
    runtime      TEXT NOT NULL,
    transport    TEXT NOT NULL DEFAULT 'stdio',
    author_email TEXT NOT NULL DEFAULT '',
    tags         TEXT NOT NULL DEFAULT '[]',
    homepage     TEXT NOT NULL DEFAULT '',
    license      TEXT NOT NULL DEFAULT '',
	verification TEXT NOT NULL DEFAULT 'community',
	sha256       TEXT NOT NULL DEFAULT '',
	signature    TEXT NOT NULL DEFAULT '',
    entrypoint_command TEXT NOT NULL DEFAULT '',
    entrypoint_args    TEXT NOT NULL DEFAULT '[]',
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS skills (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    icon         TEXT NOT NULL DEFAULT '',
    mcp_ids      TEXT NOT NULL DEFAULT '[]',
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

const marketplaceTrustMigration = `
ALTER TABLE marketplace_items ADD COLUMN IF NOT EXISTS author_email TEXT NOT NULL DEFAULT '';
ALTER TABLE marketplace_items ADD COLUMN IF NOT EXISTS tags TEXT NOT NULL DEFAULT '[]';
ALTER TABLE marketplace_items ADD COLUMN IF NOT EXISTS homepage TEXT NOT NULL DEFAULT '';
ALTER TABLE marketplace_items ADD COLUMN IF NOT EXISTS license TEXT NOT NULL DEFAULT '';
ALTER TABLE marketplace_items ADD COLUMN IF NOT EXISTS verification TEXT NOT NULL DEFAULT 'community';
ALTER TABLE marketplace_items ADD COLUMN IF NOT EXISTS sha256 TEXT NOT NULL DEFAULT '';
ALTER TABLE marketplace_items ADD COLUMN IF NOT EXISTS signature TEXT NOT NULL DEFAULT '';
ALTER TABLE marketplace_items ADD COLUMN IF NOT EXISTS transport TEXT NOT NULL DEFAULT 'stdio';
ALTER TABLE marketplace_items ADD COLUMN IF NOT EXISTS entrypoint_command TEXT NOT NULL DEFAULT '';
ALTER TABLE marketplace_items ADD COLUMN IF NOT EXISTS entrypoint_args TEXT NOT NULL DEFAULT '[]';
`

// MarketplaceItem is a row in marketplace_items. Tags is a JSON array string.
type MarketplaceItem struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Version      string     `json:"version"`
	Runtime      string     `json:"runtime"`
	Transport    string     `json:"transport,omitempty"`
	AuthorEmail  string     `json:"author_email,omitempty"`
	Tags         []string   `json:"tags"`
	Homepage     string     `json:"homepage,omitempty"`
	License      string     `json:"license,omitempty"`
	Verification string     `json:"verification,omitempty"`
	SHA256       string     `json:"sha256,omitempty"`
	Signature    string     `json:"signature,omitempty"`
	Entrypoint   Entrypoint `json:"entrypoint,omitempty"`
}

type Entrypoint struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Cwd     string   `json:"cwd,omitempty"`
}

type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	MCPIDs      []string `json:"mcp_ids"`
}

// User is a registered account row.
type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// DB wraps a pgxpool with typed helpers. Open it once at startup and Close it
// on shutdown. All methods accept a context so callers control timeouts.
type DB struct {
	pool *pgxpool.Pool
}

// Open connects to the Postgres DSN, pings, and runs schema migrations.
// DSN should be the DATABASE_PUBLIC_URL from Railway (external proxy address).
func Open(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("clouddb: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("clouddb: ping: %w", err)
	}
	db := &DB{pool: pool}
	if err := db.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return db, nil
}

// Close releases all pooled connections.
func (d *DB) Close() { d.pool.Close() }

func (d *DB) migrate(ctx context.Context) error {
	if _, err := d.pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("clouddb: migrate: %w", err)
	}
	if _, err := d.pool.Exec(ctx, marketplaceTrustMigration); err != nil {
		return fmt.Errorf("clouddb: trust migrate: %w", err)
	}
	return nil
}

// RegisterUser inserts a new user and returns the generated UUID.
// Returns an error containing "already registered" if the email is taken.
func (d *DB) RegisterUser(ctx context.Context, email, name, passwordHash string) (string, error) {
	var id string
	err := d.pool.QueryRow(ctx,
		`INSERT INTO users (email, name, password_hash)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (email) DO NOTHING
		 RETURNING id`,
		email, name, passwordHash,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("clouddb: register user: %w", err)
	}
	if id == "" {
		return "", fmt.Errorf("clouddb: email already registered")
	}
	return id, nil
}

// FindUserByEmail looks up a user by email. Returns an error wrapping
// pgx.ErrNoRows if not found.
func (d *DB) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := d.pool.QueryRow(ctx,
		`SELECT id, name, email, password_hash, created_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("clouddb: find user: %w", err)
	}
	return u, nil
}

// CreateSession generates a random opaque token, stores only its SHA-256 hash,
// and returns the raw token once to the caller.
func (d *DB) CreateSession(ctx context.Context, userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("clouddb: create session token: %w", err)
	}
	token := hex.EncodeToString(b)
	_, err := d.pool.Exec(ctx,
		`INSERT INTO user_sessions (token, user_id) VALUES ($1, $2)`,
		sessionTokenHash(token), userID,
	)
	if err != nil {
		return "", fmt.Errorf("clouddb: create session: %w", err)
	}
	return token, nil
}

// ValidateSession returns the user associated with token if the session is
// still valid (not expired). Returns an error if invalid or expired.
func (d *DB) ValidateSession(ctx context.Context, token string) (*User, error) {
	u := &User{}
	err := d.pool.QueryRow(ctx,
		`SELECT u.id, u.name, u.email, u.password_hash, u.created_at
		 FROM user_sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token = $1 AND s.expires_at > now()`,
		sessionTokenHash(token),
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("clouddb: validate session: %w", err)
	}
	return u, nil
}

// GetUserCount returns the total number of registered users.
func (d *DB) GetUserCount(ctx context.Context) (int64, error) {
	var n int64
	if err := d.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return 0, fmt.Errorf("clouddb: user count: %w", err)
	}
	return n, nil
}

// UpsertMarketplaceItems inserts or updates a batch of marketplace items.
func (d *DB) UpsertMarketplaceItems(ctx context.Context, items []MarketplaceItem) error {
	for _, item := range items {
		tagsJSON, _ := json.Marshal(item.Tags)
		entrypointArgsJSON, _ := json.Marshal(item.Entrypoint.Args)
		_, err := d.pool.Exec(ctx, `
			INSERT INTO marketplace_items (id, name, description, version, runtime, transport, author_email, tags, homepage, license, verification, sha256, signature, entrypoint_command, entrypoint_args, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, now())
			ON CONFLICT (id) DO UPDATE SET
				name         = EXCLUDED.name,
				description  = EXCLUDED.description,
				version      = EXCLUDED.version,
				runtime      = EXCLUDED.runtime,
				transport    = EXCLUDED.transport,
				tags         = EXCLUDED.tags,
				homepage     = EXCLUDED.homepage,
				license      = EXCLUDED.license,
				verification = EXCLUDED.verification,
				sha256       = EXCLUDED.sha256,
				signature    = EXCLUDED.signature,
				entrypoint_command = EXCLUDED.entrypoint_command,
				entrypoint_args = EXCLUDED.entrypoint_args,
				updated_at   = now()`,
			item.ID, item.Name, item.Description, item.Version, item.Runtime, item.Transport,
			item.AuthorEmail, string(tagsJSON), item.Homepage, item.License, item.Verification, item.SHA256, item.Signature,
			item.Entrypoint.Command, string(entrypointArgsJSON),
		)
		if err != nil {
			return fmt.Errorf("clouddb: upsert marketplace item %s: %w", item.ID, err)
		}
	}
	return nil
}

func sessionTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GetMarketplaceItems returns all marketplace items ordered by name.
func (d *DB) GetMarketplaceItems(ctx context.Context) ([]MarketplaceItem, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, name, description, version, runtime, transport, author_email, tags, homepage, license, verification, sha256, signature, entrypoint_command, entrypoint_args
		FROM marketplace_items
		ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("clouddb: list marketplace: %w", err)
	}
	defer rows.Close()

	var out []MarketplaceItem
	for rows.Next() {
		var item MarketplaceItem
		var tagsJSON string
		var entrypointArgsJSON string
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Version,
			&item.Runtime, &item.Transport, &item.AuthorEmail, &tagsJSON, &item.Homepage, &item.License, &item.Verification, &item.SHA256, &item.Signature, &item.Entrypoint.Command, &entrypointArgsJSON); err != nil {
			return nil, fmt.Errorf("clouddb: scan marketplace item: %w", err)
		}
		_ = json.Unmarshal([]byte(tagsJSON), &item.Tags)
		_ = json.Unmarshal([]byte(entrypointArgsJSON), &item.Entrypoint.Args)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (d *DB) UpsertSkills(ctx context.Context, skills []Skill) error {
	for _, skill := range skills {
		mcpIDsJSON, _ := json.Marshal(skill.MCPIDs)
		_, err := d.pool.Exec(ctx, `
			INSERT INTO skills (id, name, description, icon, mcp_ids, updated_at)
			VALUES ($1, $2, $3, $4, $5, now())
			ON CONFLICT (id) DO UPDATE SET
				name        = EXCLUDED.name,
				description = EXCLUDED.description,
				icon        = EXCLUDED.icon,
				mcp_ids     = EXCLUDED.mcp_ids,
				updated_at  = now()`,
			skill.ID, skill.Name, skill.Description, skill.Icon, string(mcpIDsJSON),
		)
		if err != nil {
			return fmt.Errorf("clouddb: upsert skill %s: %w", skill.ID, err)
		}
	}
	return nil
}

func (d *DB) GetSkills(ctx context.Context) ([]Skill, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, name, description, icon, mcp_ids
		FROM skills
		ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("clouddb: list skills: %w", err)
	}
	defer rows.Close()

	var out []Skill
	for rows.Next() {
		var skill Skill
		var mcpIDsJSON string
		if err := rows.Scan(&skill.ID, &skill.Name, &skill.Description, &skill.Icon, &mcpIDsJSON); err != nil {
			return nil, fmt.Errorf("clouddb: scan skill: %w", err)
		}
		_ = json.Unmarshal([]byte(mcpIDsJSON), &skill.MCPIDs)
		out = append(out, skill)
	}
	return out, rows.Err()
}

func (d *DB) UpdateUserProfile(ctx context.Context, userID, name, email string) (*User, error) {
	u := &User{}
	err := d.pool.QueryRow(ctx,
		`UPDATE users
		 SET name = $2, email = $3
		 WHERE id = $1
		 RETURNING id, name, email, password_hash, created_at`,
		userID, name, email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("clouddb: update user profile: %w", err)
	}
	return u, nil
}

func (d *DB) UpdateUserPasswordHash(ctx context.Context, userID, passwordHash string) error {
	if _, err := d.pool.Exec(ctx,
		`UPDATE users
		 SET password_hash = $2
		 WHERE id = $1`,
		userID, passwordHash,
	); err != nil {
		return fmt.Errorf("clouddb: update user password: %w", err)
	}
	return nil
}
