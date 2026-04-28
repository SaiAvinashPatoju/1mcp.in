// mcpapiserver is the 1mcp.in cloud HTTP API.  It bridges the Railway Postgres
// database (shared user accounts, marketplace metadata) with the Svelte web-UI
// and Tauri desktop app.
//
// Configuration (all via environment variables):
//
//	DATABASE_PUBLIC_URL  Ã¢â‚¬â€œ Railway Postgres public proxy URL (required)
//	PORT                 Ã¢â‚¬â€œ listen port (default: 8080)
//
// Endpoints:
//
//	GET  /health                  liveness probe
//	GET  /api/stats               { total_users: N }
//	GET  /api/marketplace         list all marketplace MCPs
//	POST /api/auth/register       { name, email, password } Ã¢â€ â€™ { token, user }
//	POST /api/auth/login          { email, password }       Ã¢â€ â€™ { token, user }
//	GET  /api/auth/me             Authorization: Bearer <token> Ã¢â€ â€™ { user }
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/clouddb"
)

//go:embed data/registry-index.json
var registryIndexJSON []byte

func main() {
	dsn := os.Getenv("DATABASE_PUBLIC_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		slog.Error("DATABASE_PUBLIC_URL env var is required")
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	issuer := os.Getenv("OAUTH_ISSUER")
	if issuer == "" {
		issuer = "http://localhost:" + port
	}

	ctx := context.Background()
	db, err := clouddb.Open(ctx, dsn)
	if err != nil {
		slog.Error("open clouddb", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/stats", handleStats(db))
	mux.HandleFunc("GET /api/marketplace", handleMarketplace(db))
	mux.HandleFunc("POST /api/auth/register", handleRegister(db))
	mux.HandleFunc("POST /api/auth/login", handleLogin(db))
	mux.HandleFunc("GET /api/auth/me", handleMe(db))
	registerOAuthHandlers(mux, newOAuthStore(), strings.TrimRight(issuer, "/"))

	// Seed marketplace from embedded registry-index on startup
	if err := seedMarketplace(ctx, db); err != nil {
		slog.Warn("marketplace seed failed (non-fatal)", "err", err)
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      cors(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	slog.Info("mcpapiserver ready", "port", port)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

// cors adds permissive CORS headers so the Svelte SPA can reach the API from
// any origin during development and from the Railway deployment domain in
// production.
func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// Ã¢â€â‚¬Ã¢â€â‚¬ /api/stats Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬

func handleStats(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n, err := db.GetUserCount(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"total_users": n})
	}
}

// Ã¢â€â‚¬Ã¢â€â‚¬ /api/auth/register Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬

func handleRegister(db *clouddb.DB) http.HandlerFunc {
	type request struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		if body.Name == "" || body.Email == "" || body.Password == "" {
			writeJSON(w, http.StatusBadRequest, errBody("name, email and password are required"))
			return
		}
		if len(body.Password) < 8 {
			writeJSON(w, http.StatusBadRequest, errBody("password must be at least 8 characters"))
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		id, err := db.RegisterUser(r.Context(), body.Email, body.Name, string(hash))
		if err != nil {
			if strings.Contains(err.Error(), "already registered") {
				writeJSON(w, http.StatusConflict, errBody("email already registered"))
				return
			}
			slog.Error("register user", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		token, err := db.CreateSession(r.Context(), id)
		if err != nil {
			slog.Error("create session", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"token": token,
			"user":  map[string]string{"id": id, "name": body.Name, "email": body.Email},
		})
	}
}

// Ã¢â€â‚¬Ã¢â€â‚¬ /api/auth/login Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬

func handleLogin(db *clouddb.DB) http.HandlerFunc {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		if body.Email == "" || body.Password == "" {
			writeJSON(w, http.StatusBadRequest, errBody("email and password are required"))
			return
		}

		u, err := db.FindUserByEmail(r.Context(), body.Email)
		if err != nil {
			// Don't leak whether the email exists.
			writeJSON(w, http.StatusUnauthorized, errBody("invalid credentials"))
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(body.Password)); err != nil {
			writeJSON(w, http.StatusUnauthorized, errBody("invalid credentials"))
			return
		}

		token, err := db.CreateSession(r.Context(), u.ID)
		if err != nil {
			slog.Error("create session", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"token": token,
			"user":  map[string]string{"id": u.ID, "name": u.Name, "email": u.Email},
		})
	}
}

// Ã¢â€â‚¬Ã¢â€â‚¬ /api/auth/me Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬

func handleMe(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, errBody("missing authorization token"))
			return
		}
		u, err := db.ValidateSession(r.Context(), token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, errBody("invalid or expired session"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"user": map[string]string{"id": u.ID, "name": u.Name, "email": u.Email},
		})
	}
}

func errBody(msg string) map[string]string { return map[string]string{"error": msg} }

// Ã¢â€â‚¬Ã¢â€â‚¬ /api/marketplace Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬Ã¢â€â‚¬

func handleMarketplace(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := db.GetMarketplaceItems(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

// seedMarketplace upserts the embedded registry-index into the DB so the
// cloud marketplace is always in sync with the static catalog.
func seedMarketplace(ctx context.Context, db *clouddb.DB) error {
	var items []clouddb.MarketplaceItem
	if err := json.Unmarshal(registryIndexJSON, &items); err != nil {
		return err
	}
	return db.UpsertMarketplaceItems(ctx, items)
}
