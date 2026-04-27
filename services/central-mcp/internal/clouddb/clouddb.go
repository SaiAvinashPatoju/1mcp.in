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
	"encoding/hex"
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
    author_email TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

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

// CreateSession generates a random opaque token, stores it, and returns it.
func (d *DB) CreateSession(ctx context.Context, userID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("clouddb: create session token: %w", err)
	}
	token := hex.EncodeToString(b)
	_, err := d.pool.Exec(ctx,
		`INSERT INTO user_sessions (token, user_id) VALUES ($1, $2)`,
		token, userID,
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
		token,
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
