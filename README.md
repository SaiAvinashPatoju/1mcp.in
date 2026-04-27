# 1mcp.in

The single entry point for all your Model Context Protocol (MCP) servers.

1mcp provides a lightweight router that multiplexes multiple MCP servers into a single interface. It handles the lifecycle, sandboxing, and discovery of MCPs so your AI agents only need to talk to one endpoint.

## Architecture

- **`centralmcpd`**: A high-performance Go router that sits between your agent and your MCPs. Handles tool namespacing, semantic ranking, lazy-start, and idle shutdown.
- **Hub UI**: A SvelteKit + Tauri desktop application for managing, configuring, and discovering MCPs.
- **Cloud API**: Syncs your settings and provides a curated marketplace backed by PostgreSQL.

## Quick Start

### 1. Build the Router
```bash
cd services/central-mcp
go build -o ../../bin/centralmcpd ./cmd/centralmcpd
```

Or build everything:
```powershell
powershell -ExecutionPolicy Bypass -File scripts/build.ps1
```

### 2. Run the UI
```bash
cd services/web-ui
npm install && npm run dev
```

### 3. Connect to Your AI Client

**Automatic (via Hub UI):** Open the **Clients** tab and click **Setup 1mcp** on your preferred client.

**Manual configuration for any MCP client:**
```json
{
  "mcpServers": {
    "1mcp": {
      "command": "centralmcpd",
      "args": ["--db", "<path-to-registry.db>"]
    }
  }
}
```

Supported clients: VS Code, Cursor, Claude Desktop, Claude Code, Codex, Windsurf, and more.

### 4. Install MCP Servers

Use the Hub UI marketplace or the CLI:
```bash
onemcpctl install github
onemcpctl install memory
onemcpctl env set github GITHUB_TOKEN <your-token>
```

## Performance

| Metric | Value |
|---|---|
| Cold initialize (npx MCPs) | ~13s |
| Warm tools/call | ~2ms |
| E2E stub round-trip | <1s |
| Idle RAM (router) | <30MB |

## Project Structure

- `services/central-mcp` — Core Go router, supervisor, and CLI tools
- `services/web-ui` — SvelteKit + Tauri hub UI
- `packages/mcp-manifest` — Shared schema (JSON Schema)
- `packages/registry-index` — Static catalog for MVP marketplace

## License
MIT

