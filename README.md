# 1mcp.in

One local router that makes every approved MCP server available inside VS Code, Cursor, Claude Desktop, Claude Code, and other AI clients.

![1mcp Hub UI demo](https://1mcp.in/assets/hub-demo.gif)

[![CI](https://github.com/SaiAvinashPatoju/1mcp.in/actions/workflows/ci.yml/badge.svg)](https://github.com/SaiAvinashPatoju/1mcp.in/actions/workflows/ci.yml)
![VS Code](https://img.shields.io/badge/VS%20Code-supported-007ACC)
![Cursor](https://img.shields.io/badge/Cursor-supported-111111)
![Claude Desktop](https://img.shields.io/badge/Claude%20Desktop-supported-6B46C1)
![Claude Code](https://img.shields.io/badge/Claude%20Code-supported-6B46C1)
![License](https://img.shields.io/badge/license-MIT-green)

## Install

macOS and Linux:

```bash
curl -fsSL https://install.1mcp.in | sh
```

Windows PowerShell:

```powershell
irm https://install.1mcp.in/windows | iex
```

Then launch the router:

```bash
onemcpctl start
```

## Why 1mcp

MCP is powerful, but managing servers one by one is tedious: every AI client needs its own config, every MCP needs its own credentials, and every teammate repeats the same setup. 1mcp replaces that with one local router process and one Hub UI.

- Install MCPs once, use them from every supported AI client.
- Keep credentials local for the OSS version.
- Route calls through a single low-latency Go process.
- Start MCP servers lazily and shut them down when idle.
- Detect tool definition changes before a changed tool can run.
- Scrub sensitive values before logs or UI output.
- Pin marketplace catalog entries with maintainer-reviewed SHA256 digests.

## Supported Clients

| Client | Status | Setup |
|---|---|---|
| VS Code / GitHub Copilot | Supported | `onemcpctl connect vscode` |
| Cursor | Supported | `onemcpctl connect cursor` |
| Claude Desktop | Supported | `onemcpctl connect claude` |
| Claude Code | Supported | Manual MCP config today |
| OpenCode / Codex / Windsurf | In progress | Manual MCP config today |

## Quick Start From Source

```bash
git clone https://github.com/SaiAvinashPatoju/1mcp.in.git
cd 1mcp.in

# Windows
pwsh -ExecutionPolicy Bypass -File scripts/build.ps1

# macOS/Linux
bash scripts/build.sh
```

List marketplace MCPs:

```bash
bin/onemcpctl catalog list
```

Install one:

```bash
bin/onemcpctl install github
bin/onemcpctl env set github GITHUB_PERSONAL_ACCESS_TOKEN=...
```

Connect a client:

```bash
bin/onemcpctl connect vscode
```

Run the router over stdio for AI clients:

```bash
bin/centralmcpd --db "$HOME/.onemcp/registry.db"
```

Run Streamable HTTP locally:

```bash
bin/centralmcpd --transport http --listen 127.0.0.1:3000
```

Metrics are exposed on `127.0.0.1:3031/metrics` in stdio mode, and on `/metrics` beside the HTTP transport in HTTP mode.

## Hub UI

```bash
cd services/web-ui
npm install
npm run build
npm run tauri build
```

The Hub UI is SvelteKit + Tauri. It manages local installs, credentials, client setup, and marketplace trust labels.

## Marketplace Trust

The registry lives in [packages/registry-index/index.json](packages/registry-index/index.json). Each entry carries a trust label and SHA256 digest:

- `anthropic-official`: official MCPs from the Anthropic / Model Context Protocol catalog.
- `onemcp-verified`: reviewed and tested by 1mcp maintainers.
- `community`: submitted by the community and signed only after maintainer review.

Verify the catalog:

```bash
cd services/central-mcp
go run ./cmd/onemcpsignregistry --check --catalog ../../packages/registry-index/index.json
```

Submit a new MCP through the PR flow in [CONTRIBUTING_MCP.md](CONTRIBUTING_MCP.md). Community entries are never auto-approved.

## Security

- Tool definition hash registry: a changed `description` or `inputSchema` moves the tool to `PENDING_REVIEW` until approved.
- Supply-chain digest verification: installs fail if the marketplace SHA256 no longer matches the canonical manifest.
- PII scrubbing: emails, phone numbers, credit card patterns, GitHub tokens, AWS keys, and JWT-like strings are redacted before logs/UI output.
- Sandbox controls: Docker MCPs run with CPU, memory, PID, read-only filesystem, and network limits unless the manifest grants access.
- OAuth 2.1 surface: `mcpapiserver` includes PKCE, dynamic client registration, resource indicators, and token introspection endpoints for remote MCP readiness.

## Validation

Local release checks used by maintainers:

```bash
cd services/central-mcp
go vet ./...
go run ./cmd/onemcpsignregistry --check --catalog ../../packages/registry-index/index.json
go test ./...

cd ../..
pwsh -ExecutionPolicy Bypass -File scripts/build.ps1
pwsh -ExecutionPolicy Bypass -File scripts/e2e-stub.ps1

cd services/web-ui
npm run check
npm run build

cd src-tauri
cargo check
```

The CI matrix runs Go vet, signed-catalog verification, race-enabled tests, and Tauri compile checks on Ubuntu, macOS, and Windows.

## Team Pro

Team Pro adds shared MCP configuration, credential vaulting, admin approval gates, activity logs, dashboards, and pre-built agents for engineering, sales, support, operations, and marketing teams.

Join the waitlist: [1mcp.in/team](https://1mcp.in/team).

## Repository Layout

```text
services/central-mcp     Go router, API server, CLI, sandbox, supervisor
services/web-ui          SvelteKit + Tauri desktop Hub
packages/mcp-manifest    Manifest JSON Schema
packages/registry-index  Signed marketplace catalog
scripts                  Build, install, and E2E helpers
```

## Star The Project

If 1mcp saves you from one more copy-pasted MCP config file, star the repository. It helps the marketplace grow and tells us which integrations to verify next.

## License

MIT. See [LICENSE](LICENSE).