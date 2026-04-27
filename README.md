# OneMCP

Single MCP entry point for AI agents. Browse a marketplace, install MCPs into
a managed environment, and expose all of them through one lightweight router
process — `centralmcpd` — that the agent talks to over stdio.

See [ROADMAP.md](ROADMAP.md) for the phased plan and [plan.md](plan.md) for
the original product brief.

## Status

| Phase | Component | State |
|---|---|---|
| 0 | Manifest schema | ✅ `packages/mcp-manifest/manifest.schema.json` |
| 1 | Central MCP (Go) | ✅ `services/central-mcp` — stdio router, SQLite registry, multi-upstream proxy |
| 2 | Cloud API (Go/Postgres) | ✅ `mcpapiserver` — Auth, Marketplace stats, Live on Railway |
| 3 | Tauri hub (UI) | ✅ `services/web-ui` — SvelteKit Dashboard, real Cloud Auth integration |
| 4 | MCP management (keys, delete) | ⏳ |
| 5 | Client connect (VS Code shim + wizard) | ⏳ |
| 6 | Sandbox + lazy start | ⏳ |
| 7 | Semantic routing (LanceDB + ONNX) | ⏳ |
| 8 | E2E test matrix | ⏳ |

## Cloud API (Live)

The central marketplace and user session data are handled by the Cloud API:
- **Base URL**: `https://mcpapiserver-production.up.railway.app`
- **Health**: `/health`
- **Stack**: Go 1.25, pgx/v5, PostgreSQL (Railway)

## Quick start (Phase 1 & 2)

Requires Go 1.25+ and Node.js.

### Local Router
```powershell
cd services\central-mcp
go build -o ..\..\bin\centralmcpd.exe .\cmd\centralmcpd
..\..\bin\centralmcpd.exe --config .\config.example.json --log debug
```

### Web UI
```powershell
cd services\web-ui
npm install
npm run dev
```

Then connect any MCP client (e.g. `npx @modelcontextprotocol/inspector`) to
the binary. You should see tools namespaced as `fs__*` and `memory__*`.
