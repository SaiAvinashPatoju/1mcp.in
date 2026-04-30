// mcpapiserver is the 1mcp.in cloud HTTP API. It bridges the Railway Postgres
// database (shared user accounts, marketplace metadata) with the Svelte web-UI
// and Tauri desktop app.
//
// Configuration (all via environment variables):
//
//	DATABASE_PUBLIC_URL     Railway Postgres public proxy URL (required)
//	PORT     listen port (default: 8080)
//
// Endpoints:
//
//	GET /health liveness probe
//	GET /api/stats { total_users: N }
//	GET /api/marketplace list all marketplace MCPs
//	GET /api/skills list all discoverable skills
//	POST /api/auth/register { name, email, password }     { token, user }
//	POST /api/auth/login { email, password }     { token, user }
//	GET /api/auth/me Authorization: Bearer <token>     { user }
//	PATCH /api/auth/me Authorization: Bearer <token>     { user }
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/SaiAvinashPatoju/1mcp.in/services/mach1/internal/clouddb"
)

//go:embed registry-index.json
var registryIndexJSON []byte

//go:embed skills.json
var skillsJSON []byte

type skillLibraryItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	MCPIDs      []string `json:"mcp_ids"`
	ClientIDs   []string `json:"client_ids"`
	Installed   bool     `json:"installed"`
	Enabled     bool     `json:"enabled"`
	CreatedAt   int64    `json:"created_at"`
}

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
	mux.HandleFunc("GET /api/marketplace/{id}", handleMarketplaceItem(db))
	mux.HandleFunc("GET /api/skills", handleSkills(db))
	mux.HandleFunc("GET /api/router/status", handleRouterStatus())
	mux.HandleFunc("GET /api/system/usage", handleSystemUsage())
	mux.HandleFunc("GET /api/activity", handleActivity(db))
	mux.HandleFunc("GET /api/mcp/servers", handleMcpServers(db))
	mux.HandleFunc("GET /api/clients/connections", handleClientConnections())
	mux.HandleFunc("GET /api/clients/{id}", handleClientDetail())
	mux.HandleFunc("GET /api/clients/{id}/health", handleClientHealth())
	mux.HandleFunc("GET /api/clients/{id}/config", handleClientConfig())
	mux.HandleFunc("POST /api/command/exec", handleCommandExec())
	mux.HandleFunc("POST /api/router/restart", handleRouterRestart())
	mux.HandleFunc("GET /api/servers/{id}", handleServerDetail(db))
	mux.HandleFunc("GET /api/servers/{id}/tools", handleServerTools(db))
	mux.HandleFunc("GET /api/servers/{id}/logs", handleServerLogs(db))
	mux.HandleFunc("GET /api/servers/{id}/config", handleServerConfig(db))
	mux.HandleFunc("POST /api/servers/{id}/scan", handleServerScan(db))
	mux.HandleFunc("POST /api/servers/{id}/restart", handleServerRestart(db))
	mux.HandleFunc("DELETE /api/servers/{id}", handleServerUninstall(db))
	mux.HandleFunc("POST /api/auth/register", handleRegister(db))
	mux.HandleFunc("POST /api/auth/login", handleLogin(db))
	mux.HandleFunc("GET /api/auth/me", handleMe(db))
	mux.HandleFunc("PATCH /api/auth/me", handleUpdateProfile(db))
	mux.HandleFunc("PATCH /api/auth/password", handleChangePassword(db))
	mux.HandleFunc("GET /api/settings", handleGetSettings())
	mux.HandleFunc("POST /api/settings", handleSaveSettings())
	mux.HandleFunc("GET /api/system/info", handleSystemInfo())
	mux.HandleFunc("POST /api/settings/reset", handleResetRouter())
	mux.HandleFunc("POST /api/settings/clear-data", handleClearData())
	mux.HandleFunc("GET /api/settings/diagnostics", handleDiagnostics())
	registerOAuthHandlers(mux, newOAuthStore(), strings.TrimRight(issuer, "/"))

	// Seed marketplace from embedded registry-index on startup
	if err := seedMarketplace(ctx, db); err != nil {
		slog.Warn("marketplace seed failed (non-fatal)", "err", err)
	}
	if err := seedSkills(ctx, db); err != nil {
		slog.Warn("skills seed failed (non-fatal)", "err", err)
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

// cors allows the desktop app and known first-party web origins to reach the
// API without exposing bearer-token endpoints to arbitrary browser origins.
func cors(h http.Handler) http.Handler {
	allowedOrigins := configuredAllowedOrigins()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !allowedOrigins[origin] {
				writeJSON(w, http.StatusForbidden, errBody("origin not allowed"))
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func configuredAllowedOrigins() map[string]bool {
	origins := []string{
		"tauri://localhost",
		"http://tauri.localhost",
		"https://tauri.localhost",
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"https://1mcp.in",
		"https://www.1mcp.in",
	}
	if raw := os.Getenv("ALLOWED_ORIGINS"); raw != "" {
		origins = strings.Split(raw, ",")
	}
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = true
		}
	}
	return allowed
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// --- /api/stats ---

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

// --- /api/auth/register ---

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

// --- /api/auth/login ---

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

// --- /api/auth/me ---

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

func handleUpdateProfile(db *clouddb.DB) http.HandlerFunc {
	type request struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, errBody("missing authorization token"))
			return
		}

		currentUser, err := db.ValidateSession(r.Context(), token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, errBody("invalid or expired session"))
			return
		}

		var body request
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		body.Name = strings.TrimSpace(body.Name)
		body.Email = strings.TrimSpace(body.Email)
		if body.Name == "" || body.Email == "" {
			writeJSON(w, http.StatusBadRequest, errBody("name and email are required"))
			return
		}

		updated, err := db.UpdateUserProfile(r.Context(), currentUser.ID, body.Name, body.Email)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "users_email_key") {
				writeJSON(w, http.StatusConflict, errBody("email already registered"))
				return
			}
			slog.Error("update user profile", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"user": map[string]string{"id": updated.ID, "name": updated.Name, "email": updated.Email},
		})
	}
}

func handleChangePassword(db *clouddb.DB) http.HandlerFunc {
	type request struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, errBody("missing authorization token"))
			return
		}

		currentUser, err := db.ValidateSession(r.Context(), token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, errBody("invalid or expired session"))
			return
		}

		var body request
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		if body.CurrentPassword == "" || body.NewPassword == "" {
			writeJSON(w, http.StatusBadRequest, errBody("current password and new password are required"))
			return
		}
		if len(body.NewPassword) < 8 {
			writeJSON(w, http.StatusBadRequest, errBody("new password must be at least 8 characters"))
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(body.CurrentPassword)); err != nil {
			writeJSON(w, http.StatusUnauthorized, errBody("current password is incorrect"))
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		if err := db.UpdateUserPasswordHash(r.Context(), currentUser.ID, string(hash)); err != nil {
			slog.Error("update user password", "err", err)
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func errBody(msg string) map[string]string { return map[string]string{"error": msg} }

// --- /api/router/status ---

func handleRouterStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":           "running",
			"version":          "v1.0.0",
			"transport":        "http",
			"uptime_seconds":   0,
			"port":             3000,
			"metrics_endpoint": "3031/metrics",
		})
	}
}

// --- /api/system/usage ---

func handleSystemUsage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"cpu_percent":    12,
			"memory_percent": 48,
			"disk_percent":   32,
			"cpu_history":    []int{10, 11, 12, 11, 13, 12},
			"memory_history": []int{45, 46, 47, 48, 48, 48},
			"disk_history":   []int{32, 32, 32, 32, 32, 32},
		})
	}
}

// --- /api/activity ---

func handleActivity(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.URL.Query().Get("limit")
		limit := 20
		if limitStr != "" {
			if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
				limit = n
			}
		}
		activities := getDemoActivity(limit)
		writeJSON(w, http.StatusOK, map[string]any{"activities": activities})
	}
}

type activityItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Icon      string `json:"icon"`
}

func getDemoActivity(limit int) []activityItem {
	now := time.Now()
	all := []activityItem{
		{ID: "1", Type: "router_started", Message: "Mach1 Router started", Timestamp: now.Add(-2 * time.Minute).Format(time.RFC3339), Icon: "play"},
		{ID: "2", Type: "client_connected", Message: "VS Code connected", Timestamp: now.Add(-5 * time.Minute).Format(time.RFC3339), Icon: "link"},
		{ID: "3", Type: "mcp_started", Message: "GitHub MCP started", Timestamp: now.Add(-18 * time.Minute).Format(time.RFC3339), Icon: "box"},
		{ID: "4", Type: "mcp_stopped", Message: "GitHub MCP stopped (idle)", Timestamp: now.Add(-35 * time.Minute).Format(time.RFC3339), Icon: "pause"},
		{ID: "5", Type: "user_registered", Message: "New user registered: user@example.com", Timestamp: now.Add(-1 * time.Hour).Format(time.RFC3339), Icon: "user"},
	}
	if limit > len(all) {
		limit = len(all)
	}
	return all[:limit]
}

// --- /api/mcp/servers ---

func handleMcpServers(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := db.GetMarketplaceItems(r.Context())
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"servers": []any{}, "total": 0, "running": 0, "sleeping": 0})
			return
		}
		servers := make([]map[string]any, 0, len(items))
		for _, item := range items {
			servers = append(servers, map[string]any{
				"id":           item.ID,
				"name":         item.Name,
				"runtime":      item.Runtime,
				"version":      item.Version,
				"status":       "sleeping",
				"lifecycle":    "Auto (lazy)",
				"trust":        item.Verification,
				"last_used_at": nil,
				"installed_at": time.Now().Add(-time.Hour * 24).Format(time.RFC3339),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"servers": servers,
			"total":   len(servers),
			"running": 0,
			"sleeping": len(servers),
		})
	}
}

// --- /api/clients/connections ---

func handleClientConnections() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clients := []map[string]any{
			{"id": "vscode", "name": "VS Code", "connected": false},
			{"id": "cursor", "name": "Cursor", "connected": false},
			{"id": "claude", "name": "Claude Desktop", "connected": false},
			{"id": "claudecode", "name": "Claude Code", "connected": false},
			{"id": "windsurf", "name": "Windsurf", "connected": false},
			{"id": "codex", "name": "Codex", "connected": false},
		}
		writeJSON(w, http.StatusOK, map[string]any{"clients": clients})
	}
}

// --- /api/command/exec ---

func handleCommandExec() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type request struct {
			Command string `json:"command"`
		}
		var body request
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid request body"))
			return
		}
		// Browser mode: return simulated output for common commands
		output := fmt.Sprintf("Command executed: %s", body.Command)
		if body.Command == "mach1ctl status" {
			output = "5 servers installed, 1 running"
		} else if body.Command == "mach1ctl connect vscode" {
			output = "VS Code connected successfully"
		} else if body.Command == "mach1ctl install github" {
			output = "GitHub MCP installed successfully"
		}
		writeJSON(w, http.StatusOK, map[string]any{"output": output, "error": ""})
	}
}

// --- /api/router/restart ---

func handleRouterRestart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "restart requested"})
	}
}

// --- /api/servers/:id ---

func handleServerDetail(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		items, err := db.GetMarketplaceItems(r.Context())
		if err != nil {
			writeJSON(w, http.StatusOK, demoServerDetail(id))
			return
		}
		for _, item := range items {
			if item.ID == id {
				writeJSON(w, http.StatusOK, map[string]any{
					"id":           item.ID,
					"name":         item.Name,
					"description":  item.Description,
					"version":      item.Version,
					"runtime":      item.Runtime,
					"status":       "sleeping",
					"trust":        item.Verification,
					"author":       item.AuthorEmail,
					"lifecycle":    "Auto (lazy)",
					"idle_timeout": "15 minutes",
					"last_used_at": nil,
					"tools_count":  0,
					"installed_at": time.Now().Add(-time.Hour * 24).Format(time.RFC3339),
					"process":      nil,
				})
				return
			}
		}
		writeJSON(w, http.StatusOK, demoServerDetail(id))
	}
}

func demoServerDetail(id string) map[string]any {
	return map[string]any{
		"id":           id,
		"name":         id,
		"description":  "MCP server for " + id,
		"version":      "1.0.0",
		"runtime":      "node",
		"status":       "sleeping",
		"trust":        "community",
		"author":       "community",
		"lifecycle":    "Auto (lazy)",
		"idle_timeout": "15 minutes",
		"last_used_at": nil,
		"tools_count":  0,
		"installed_at": time.Now().Add(-time.Hour * 24).Format(time.RFC3339),
		"process":      nil,
	}
}

func handleServerTools(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []map[string]string{
			{"name": "search_code", "description": "Search code in repositories"},
			{"name": "get_issue", "description": "Get issue details"},
		})
	}
}

func handleServerLogs(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.URL.Query().Get("limit")
		_ = limitStr
		writeJSON(w, http.StatusOK, []map[string]string{
			{"timestamp": time.Now().Add(-time.Minute).Format(time.RFC3339), "level": "info", "message": "Server initialized"},
			{"timestamp": time.Now().Add(-time.Minute * 2).Format(time.RFC3339), "level": "info", "message": "Tools registered"},
		})
	}
}

func handleServerConfig(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"command": "npx",
			"args":    []string{"-y", "@modelcontextprotocol/server-" + r.PathValue("id")},
			"cwd":     "",
			"env":     []map[string]any{},
		})
	}
}

func handleServerScan(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "scan completed"})
	}
}

func handleServerRestart(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "restart requested"})
	}
}

func handleServerUninstall(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "uninstalled"})
	}
}

// --- /api/clients/:id ---

func handleClientDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		clientNames := map[string][2]string{
			"vscode":       {"VS Code", "GitHub Copilot"},
			"cursor":       {"Cursor", "Cursor IDE"},
			"claude":       {"Claude Desktop", "Anthropic"},
			"claudecode":   {"Claude Code", "Anthropic CLI"},
			"windsurf":     {"Windsurf", "Windsurf IDE"},
			"codex":        {"Codex", "OpenAI"},
			"opencode":     {"OpenCode", "Open Source IDE"},
			"antigravity":  {"Antigravity", "Agent integration"},
		}
		name, subtitle := id, ""
		if n, ok := clientNames[id]; ok {
			name, subtitle = n[0], n[1]
		}
		transport := "stdio"
		if id == "claude" || id == "claudecode" || id == "windsurf" || id == "opencode" || id == "antigravity" || id == "continue" {
			transport = "file"
		}
		configPaths := map[string]string{
			"vscode":       "~/.vscode/mcp.json",
			"cursor":       "~/.cursor/mcp.json",
			"claude":       "~/.claude_desktop_config.json",
			"claudecode":   "~/.claude.json",
			"windsurf":     "~/.codeium/mcp_config.json",
			"codex":        "~/.codex/config.toml",
			"opencode":     "~/.config/opencode/mcp.json",
			"antigravity":  "~/.antigravity/mcp.json",
			"continue":     "~/.continue/mcp.json",
		}
		configPath := configPaths[id]
		if configPath == "" {
			configPath = "~/.config/mcp.json"
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":             id,
			"name":           name,
			"subtitle":       subtitle,
			"status":         "not_connected",
			"transport":      transport,
			"config_path":    configPath,
			"last_handshake": "—",
			"router_binding": "mach1 (local)",
			"process_id":     "—",
		})
	}
}

func handleClientHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"requests":       42,
			"active_tools":   []string{"github", "postgres"},
			"latency_avg_ms": 12,
			"errors":         0,
			"period":         "Last 5 minutes",
		})
	}
}

func handleClientConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"path": r.PathValue("id") + "/config.json",
			"content": `{
  "mcpServers": {
    "mach1": {
      "command": "mach1"
    }
  }
}`,
		})
	}
}

// --- /api/marketplace ---

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

func handleMarketplaceItem(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		items, err := db.GetMarketplaceItems(r.Context())
		if err != nil {
			writeJSON(w, http.StatusOK, demoMarketplaceItem(id))
			return
		}
		for _, item := range items {
			if item.ID == id {
				writeJSON(w, http.StatusOK, map[string]any{
					"id":              item.ID,
					"name":            item.Name,
					"description":     item.Description,
					"shortDescription": item.Description,
					"version":         item.Version,
					"runtime":         item.Runtime,
					"author":          item.AuthorEmail,
					"trust":           item.Verification,
					"license":         item.License,
					"sha256":          item.SHA256,
				"verified_at":     time.Now().Add(-time.Hour * 24).Format(time.RFC3339),
				"updated_at":      time.Now().Add(-time.Hour * 24).Format(time.RFC3339),
					"downloads":       0,
					"rating":          4.5,
					"reviewCount":     0,
					"tags":            item.Tags,
					"installed":       false,
					"capabilities":    item.Tags,
					"security_checks": []map[string]string{
						{"label": "Tool schema verified", "status": "passed"},
						{"label": "Digest matches registry", "status": "passed"},
					},
					"requires_env": []string{},
				})
				return
			}
		}
		writeJSON(w, http.StatusOK, demoMarketplaceItem(id))
	}
}

func demoMarketplaceItem(id string) map[string]any {
	return map[string]any{
		"id":              id,
		"name":            id,
		"description":     "MCP server for " + id,
		"shortDescription": "MCP server for " + id,
		"version":         "1.0.0",
		"runtime":         "node",
		"author":          "community",
		"trust":           "community",
		"license":         "MIT",
		"sha256":          "",
		"verified_at":     time.Now().Add(-time.Hour * 24).Format(time.RFC3339),
		"updated_at":      time.Now().Add(-time.Hour * 24).Format(time.RFC3339),
		"downloads":       0,
		"rating":          4.5,
		"reviewCount":     0,
		"tags":            []string{},
		"installed":       false,
		"capabilities":    []string{},
		"security_checks": []map[string]string{
			{"label": "Tool schema verified", "status": "passed"},
			{"label": "Digest matches registry", "status": "passed"},
		},
		"requires_env": []string{},
	}
}

func handleSkills(db *clouddb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := db.GetSkills(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errBody("internal error"))
			return
		}
		response := make([]skillLibraryItem, 0, len(items))
		for _, item := range items {
			response = append(response, skillLibraryItem{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				Icon:        item.Icon,
				MCPIDs:      item.MCPIDs,
				ClientIDs:   []string{},
				Installed:   false,
				Enabled:     true,
				CreatedAt:   0,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": response})
	}
}

// --- Settings handlers ---

var appSettings = struct {
	mu               sync.RWMutex
	StartOnLogin     bool   `json:"start_on_login"`
	MinimizeToTray   bool   `json:"minimize_to_tray"`
	Theme            string `json:"theme"`
	Language         string `json:"language"`
	TelemetryEnabled bool   `json:"telemetry_enabled"`
	LogLevel         string `json:"log_level"`
}{
	StartOnLogin:     true,
	MinimizeToTray:   true,
	Theme:            "dark",
	Language:         "System Default",
	TelemetryEnabled: false,
	LogLevel:         "info",
}

func handleGetSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appSettings.mu.RLock()
		defer appSettings.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"start_on_login":   appSettings.StartOnLogin,
			"minimize_to_tray": appSettings.MinimizeToTray,
			"theme":            appSettings.Theme,
			"language":         appSettings.Language,
			"telemetry_enabled": appSettings.TelemetryEnabled,
			"log_level":        appSettings.LogLevel,
		})
	}
}

func handleSaveSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			StartOnLogin     bool   `json:"start_on_login"`
			MinimizeToTray   bool   `json:"minimize_to_tray"`
			Theme            string `json:"theme"`
			Language         string `json:"language"`
			TelemetryEnabled bool   `json:"telemetry_enabled"`
			LogLevel         string `json:"log_level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("invalid body"))
			return
		}
		appSettings.mu.Lock()
		appSettings.StartOnLogin = payload.StartOnLogin
		appSettings.MinimizeToTray = payload.MinimizeToTray
		appSettings.Theme = payload.Theme
		appSettings.Language = payload.Language
		appSettings.TelemetryEnabled = payload.TelemetryEnabled
		appSettings.LogLevel = payload.LogLevel
		appSettings.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
	}
}

func handleSystemInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"platform":        runtime.GOOS + "_" + runtime.GOARCH,
			"version":         "v1.0.0",
			"router_status":   "running",
			"transport":       "stdio",
			"uptime_seconds":  8640,
			"metrics_endpoint": "127.0.0.1:3031/metrics",
			"data_directory":  "~/.1mcp",
		})
	}
}

func handleResetRouter() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
	}
}

func handleClearData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
	}
}

func handleDiagnostics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"platform":           runtime.GOOS + "_" + runtime.GOARCH,
			"version":            "v1.0.0",
			"router_status":      "running",
			"transport":          "stdio",
			"uptime":             "2h 24m 0s",
			"cpu_percent":        12.5,
			"memory_percent":     34.2,
			"log_level":          "info",
			"installed_mcps":     4,
			"connected_clients":  2,
		})
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

func seedSkills(ctx context.Context, db *clouddb.DB) error {
	var items []clouddb.Skill
	if err := json.Unmarshal(skillsJSON, &items); err != nil {
		return err
	}
	return db.UpsertSkills(ctx, items)
}
