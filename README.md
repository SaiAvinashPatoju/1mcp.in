# 1mcp.in – Enterprise Model Context Protocol Hub

A production-grade Model Context Protocol (MCP) hub and router that centralizes lifecycle management, tool discovery, and routing for AI agents across multiple MCP servers. Built for reliability, performance, and scale.

## Overview

**1mcp** solves the orchestration problem: instead of your AI applications juggling multiple MCP server connections, sandboxing, tool discovery, and routing—you get a single, high-performance entry point that handles all of it.

- **Semantic routing**: Tools are ranked by relevance and prioritized based on the agent's intent
- **Lazy initialization & idle shutdown**: MCP servers start on-demand and shut down after 5 minutes of inactivity
- **Marketplace sync**: Curated catalog of 18+ vetted MCPs with one-click install
- **Remember-me sessions**: Users stay logged in across restarts; offline-capable app
- **Background auto-updates**: Updates checked every 4 hours; seamless restart prompts

### Key Metrics
| Metric | Value |
|--------|-------|
| Cold initialize (npx MCPs) | ~741ms |
| Warm tool call | ~1–2ms |
| Router RAM footprint | <30MB |
| Supported clients | 7+ (VS Code, Cursor, Claude Desktop, etc.) |
| MCP servers in marketplace | 18 |

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│ AI Agent / Client (VS Code, Cursor, Claude Desktop...)  │
└──────────────┬──────────────────────────────────────────┘
               │ MCP (stdio)
               ▼
┌─────────────────────────────────────────────────────────┐
│ centralmcpd – Go Router (semantic + lazy init + sandbox)│
│  ├─ Namespaces tools (tool → mcp-id__tool)              │
│  ├─ Semantic ranking (intent-based tool selection)      │
│  ├─ Process supervision (restart on crash)              │
│  └─ Idle shutdown (5min inactivity)                     │
└──────────────┬──────────────────────────────────────────┘
               │
    ┌──────────┼──────────────┐
    ▼          ▼              ▼
┌────────┐ ┌────────┐  ┌──────────────────┐
│ GitHub │ │ Memory │  │ Filesystem / SQL │
│  MCP   │ │  MCP   │  │      MCPs…       │
└────────┘ └────────┘  └──────────────────┘
    │          │              │
    └──────────┴──────────────┘
              │
    ┌─────────┴─────────────────┐
    ▼                           ▼
┌──────────────────┐     ┌────────────────────┐
│ Hub UI (Tauri +  │     │ Cloud API (Railway)│
│ SvelteKit)       │     │ • Auth             │
│ • Manage MCPs    │     │ • Marketplace sync │
│ • Install/config │     │ • PostgreSQL store │
│ • Live console   │     └────────────────────┘
└──────────────────┘

```

## Quick Start

### Prerequisites
- **Go 1.22+** (for router)
- **Node.js 18+** (for UI)
- **Rust** (for Tauri desktop app) or just use web preview

### 1. Clone & Install Dependencies

```bash
git clone https://github.com/SaiAvinashPatoju/1mcp.in.git
cd 1mcp.in

# Build Go binaries
pwsh -ExecutionPolicy Bypass -File scripts/build.ps1
# Output: bin/centralmcpd.exe, bin/mcpapiserver.exe, bin/onemcpctl.exe

# Install web UI dependencies
cd services/web-ui
npm install
```

### 2. Run Locally (Development)

**Terminal 1 – Router:**
```bash
cd 1mcp.in/services/central-mcp
go run ./cmd/centralmcpd
```

**Terminal 2 – API Server (for auth + marketplace):**
```bash
cd 1mcp.in/services/central-mcp
DATABASE_URL="postgres://user:pass@localhost:5432/1mcp" \
go run ./cmd/mcpapiserver
```

Or connect to Railway PostgreSQL:
```bash
railway run bash -c \
  'go run ./cmd/mcpapiserver'
```

**Terminal 3 – Web UI:**
```bash
cd 1mcp.in/services/web-ui
VITE_API_URL=http://localhost:8080 npm run dev
```

Open **http://localhost:5173** in your browser or Tauri app.

### 3. Connect Your AI Client

#### Automatic (via Hub UI)
1. Open 1mcp Hub → **Clients** tab
2. Click **Setup 1mcp** on your preferred client (VS Code, Cursor, Claude Desktop)
3. 1mcp auto-patches the client config file with the router endpoint

#### Manual Configuration

**VS Code** (`%APPDATA%/Code/User/settings.json`):
```json
{
  "mcp.servers": {
    "1mcp": {
      "command": "path/to/centralmcpd",
      "args": ["--listen", "127.0.0.1:3000"]
    }
  }
}
```

**Cursor** (`~/.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "1mcp": {
      "command": "path/to/centralmcpd"
    }
  }
}
```

**Claude Desktop** (`%APPDATA%/Claude/claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "1mcp": {
      "command": "path/to/centralmcpd"
    }
  }
}
```

See [modelcontextprotocol.io](https://modelcontextprotocol.io) for full client setup docs.

### 4. Install MCPs

Via Hub UI (**Marketplace** tab) or CLI:

```bash
onemcpctl install github
onemcpctl env set github GITHUB_TOKEN sk-...
onemcpctl install filesystem
onemcpctl env set filesystem ROOT_DIR C:/Users/YourName/Documents
```

---

## Configuration

### Environment Variables

**mcpapiserver (Cloud API)**:
```bash
DATABASE_PUBLIC_URL    # PostgreSQL connection string (Railway or local)
PORT                   # HTTP listen port (default: 8080)
```

**centralmcpd (Router)**:
```bash
REGISTRY_DB            # Path to local MCP registry SQLite DB
LOG_LEVEL              # debug, info, warn, error (default: info)
```

**Web UI (SvelteKit)**:
```bash
VITE_API_URL           # mcpapiserver endpoint (default: http://localhost:8080)
```

### Production Deployment

#### Deploy to Railway

1. **Link your repo:**
   ```bash
   railway link
   railway up --service mcpapiserver --detach
   ```

2. **Set PostgreSQL secret:**
   ```bash
   railway env set DATABASE_PUBLIC_URL $RAILWAY_DATABASE_URL
   ```

3. **GitHub Actions auto-deploys on git push to `main` + releases on `vX.Y.Z` tags**
   - Requires `RAILWAY_TOKEN` secret in GitHub (Settings → Secrets)
   - See `.github/workflows/release.yml` for details

#### Self-Hosted (Docker)

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o mcpapiserver ./cmd/mcpapiserver

FROM alpine:latest
COPY --from=build /app/mcpapiserver /usr/local/bin/
EXPOSE 8080
CMD ["mcpapiserver"]
```

Deploy with `docker run -e DATABASE_PUBLIC_URL=... -p 8080:8080 your-image:latest`

---

## Marketplace

18 enterprise-grade MCPs available:

**Data & Databases:**
- PostgreSQL, SQLite, Google Drive, AWS Knowledge Base

**Integration & Communication:**
- GitHub, Slack, Linear, Notion, Jira

**Web & Automation:**
- Fetch, Brave Search, Playwright, Puppeteer

**Utilities:**
- Filesystem, Memory (Knowledge Graph), Git, Sequential Thinking, Time/World Clock

**Plus:** Everything (test/demo server)

Each MCP is configured with:
- Proper runtime (Node.js, Python, binary)
- Required environment secrets
- Verification status (official vs community)
- Installation instructions

---

## Project Structure

```
1mcp.in/
├─ services/
│  ├─ central-mcp/          # Go router, API, CLI, supervisor
│  │  ├─ cmd/
│  │  │  ├─ centralmcpd     # Main router binary
│  │  │  ├─ mcpapiserver    # Cloud API (auth, marketplace, stats)
│  │  │  └─ onemcpctl       # CLI tool (install, list, env)
│  │  └─ internal/
│  │     ├─ router/         # Request routing & tool namespacing
│  │     ├─ supervisor/     # Process lifecycle & semantic ranking
│  │     ├─ sandbox/        # Process isolation & stdio tunneling
│  │     ├─ clouddb/        # PostgreSQL schema & queries
│  │     └─ ...
│  └─ web-ui/               # SvelteKit + Tauri desktop app
│     ├─ src/               # Svelte pages, components, stores
│     ├─ src-tauri/         # Tauri Rust backend (auto-update, sqlite)
│     ├─ build/             # Pre-built static assets (gitignored)
│     └─ vite.config.ts
├─ packages/
│  ├─ mcp-manifest/         # JSON Schema for MCP config
│  └─ registry-index/       # Catalog of vetted MCPs (18 servers)
├─ scripts/
│  └─ build.ps1             # Windows build script (go + cargo)
├─ .github/workflows/
│  ├─ ci.yml                # Lint, test, version-check on PR/push
│  └─ release.yml           # Deploy on vX.Y.Z tag push
└─ README.md (this file)
```

---

## Development

### Build Everything
```powershell
pwsh -ExecutionPolicy Bypass -File scripts/build.ps1
```
Outputs: `bin/centralmcpd.exe`, `bin/mcpapiserver.exe`, `bin/onemcpctl.exe`

### Run Tests
```bash
cd services/central-mcp
go test ./...

cd ../web-ui
npm run test
```

### Linting & Formatting
```bash
cd services/central-mcp
go vet ./...
go fmt ./...

cd ../web-ui
npm run lint
npx prettier --write src/
```

### Update MCP Marketplace
1. Edit `packages/registry-index/index.json` (add/update MCPs)
2. Run `build.ps1` (auto-syncs to `services/central-mcp/cmd/mcpapiserver/data/`)
3. Commit & push; CI auto-tests and release job deploys

---

## Troubleshooting

### "Failed to fetch" on login/signup
- Ensure `mcpapiserver` is running (Terminal 2 above)
- Check `VITE_API_URL=http://localhost:8080` in your `.env.local`
- Check browser console (F12) for actual error message

### Router not found or won't start
- Verify `centralmcpd` binary exists: `ls bin/centralmcpd.exe` or `which centralmcpd`
- Check Go version: `go version` (need 1.22+)
- Check logs for permission errors

### MCP not installing
- Run `onemcpctl list` to see available MCPs
- Check environment variables: `onemcpctl env set github GITHUB_TOKEN <token>`
- Review `services/central-mcp/internal/install/install.go` for install logic

### PostgreSQL connection fails
- Confirm `DATABASE_PUBLIC_URL` is set and reachable
- Test with: `psql $DATABASE_PUBLIC_URL -c "SELECT 1"`
- For Railway: `railway run bash` → test directly in Railway shell

---

## Performance Tuning

**Router memory:** Adjust MCP idle timeout in [supervisor.go](services/central-mcp/internal/supervisor/supervisor.go):
```go
const idleTimeout = 5 * time.Minute  // Tune as needed
```

**Database connection pool:** Edit `clouddb.Open()` in [clouddb.go](services/central-mcp/internal/clouddb/clouddb.go):
```go
config.MaxConns = 25  // Default; increase for high concurrency
```

**Semantic ranking:** Tune threshold and algorithm in [rank.go](services/central-mcp/internal/supervisor/rank.go)

---

## Contributing

1. **Fork & clone** this repo
2. **Create a branch** for your feature (`git checkout -b feat/my-feature`)
3. **Make changes** and test locally
4. **Push & open a PR** with a clear description
5. **CI/CD runs tests automatically**; address any failures

See [ROADMAP.md](ROADMAP.md) for planned improvements and enterprise enhancements.

---

## License

MIT License – See [LICENSE](LICENSE) for details

---

## Support & Community

- 📖 **Docs**: [modelcontextprotocol.io](https://modelcontextprotocol.io)
- 🐛 **Issues**: [GitHub Issues](https://github.com/SaiAvinashPatoju/1mcp.in/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/SaiAvinashPatoju/1mcp.in/discussions)
- 🚀 **Roadmap**: [ROADMAP.md](ROADMAP.md)


