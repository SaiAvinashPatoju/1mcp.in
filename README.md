# 1mcp.in

One local router that makes every approved MCP server available inside VS Code, Cursor, Claude Desktop, Claude Code, Windsurf, and other AI clients.

![1mcp Hub UI demo](https://1mcp.in/assets/hub-demo.gif)

[![CI](https://github.com/SaiAvinashPatoju/1mcp.in/actions/workflows/ci.yml/badge.svg)](https://github.com/SaiAvinashPatoju/1mcp.in/actions/workflows/ci.yml)
[![Playwright](https://img.shields.io/badge/tests-131%20passing-brightgreen)](https://github.com/SaiAvinashPatoju/1mcp.in/actions/workflows/ci.yml)
![VS Code](https://img.shields.io/badge/VS%20Code-supported-007ACC)
![Cursor](https://img.shields.io/badge/Cursor-supported-111111)
![Claude Desktop](https://img.shields.io/badge/Claude%20Desktop-supported-6B46C1)
![Claude Code](https://img.shields.io/badge/Claude%20Code-supported-6B46C1)
![Windsurf](https://img.shields.io/badge/Windsurf-supported-0078D4)
![License](https://img.shields.io/badge/license-Apache%202.0-blue)

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
mach1ctl start
```

Open the Hub UI at `http://localhost:5173` to browse the marketplace, manage servers, and connect clients.

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
| VS Code / GitHub Copilot | Supported | `mach1ctl connect vscode` |
| Cursor | Supported | `mach1ctl connect cursor` |
| Claude Desktop | Supported | `mach1ctl connect claude` |
| Claude Code | Supported | `mach1ctl connect claude-code` |
| Windsurf | Supported | Manual MCP config |
| OpenCode / Codex | Supported | Manual MCP config |

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
bin/mach1ctl catalog list
```

Install one:

```bash
bin/mach1ctl install github
bin/mach1ctl env set github GITHUB_PERSONAL_ACCESS_TOKEN=...
```

Connect a client:

```bash
bin/mach1ctl connect vscode
```

Run the router over stdio for AI clients:

```bash
bin/mach1 --db "$HOME/.mach1/registry.db"
```

Run Streamable HTTP locally:

```bash
bin/mach1 --transport http --listen 127.0.0.1:3000
```

Metrics are exposed on `127.0.0.1:3031/metrics` in stdio mode, and on `/metrics` beside the HTTP transport in HTTP mode.

## Hub UI

```bash
cd services/web-ui
npm install
npm run build        # production build → build/
npm run tauri build  # desktop app bundle
```

The Hub UI is SvelteKit + Tauri. It manages local installs, credentials, client setup, and marketplace trust labels.

**Test commands:**

```bash
npm run test                  # unit tests (Vitest)
npm run test:e2e:smoke        # smoke tests (11 tests)
npm run test:e2e:quality      # quality tests (108 tests)
npm run test:e2e:stress       # stress tests
npm run check                 # Svelte type-check
```

## Marketplace Trust

The registry lives in [packages/registry-index/index.json](packages/registry-index/index.json). Each entry carries a trust label and SHA256 digest:

- `anthropic-official`: official MCPs from the Anthropic / Model Context Protocol catalog.
- `1mcp.in-verified`: reviewed and tested by 1mcp maintainers.
- `community`: submitted by the community and signed only after maintainer review.

Verify the catalog:

```bash
cd services/mach1
go run ./cmd/mach1signregistry --check --catalog ../../packages/registry-index/index.json
```

Submit a new MCP through the PR flow in [CONTRIBUTING_MCP.md](CONTRIBUTING_MCP.md). Community entries are never auto-approved.

## Security

- **Tool definition hash registry**: a changed `description` or `inputSchema` moves the tool to `PENDING_REVIEW` until approved.
- **Supply-chain digest verification**: installs fail if the marketplace SHA256 no longer matches the canonical manifest.
- **PII scrubbing**: emails, phone numbers, credit card patterns, GitHub tokens, AWS keys, and JWT-like strings are redacted before logs/UI output.
- **Sandbox controls**: Docker MCPs run with CPU, memory, PID, read-only filesystem, and network limits unless the manifest grants access.
- **OAuth 2.1 surface**: `mcpapiserver` includes PKCE, dynamic client registration, resource indicators, and token introspection endpoints for remote MCP readiness.

## Validation

Local release checks used by maintainers:

```bash
cd services/mach1
go vet ./...
go run ./cmd/mach1signregistry --check --catalog ../../packages/registry-index/index.json
go test ./...

cd ../..
pwsh -ExecutionPolicy Bypass -File scripts/build.ps1
pwsh -ExecutionPolicy Bypass -File scripts/e2e-stub.ps1

cd services/web-ui
npm run check
npm run test
npm run test:e2e:smoke
npm run test:e2e:quality

cd src-tauri
cargo check
```

The CI matrix runs Go vet, signed-catalog verification, race-enabled tests, Playwright smoke/quality/stress suites, and Tauri compile checks on Ubuntu, macOS, and Windows.

## Team Pro

Team Pro adds shared MCP configuration, credential vaulting, admin approval gates, activity logs, dashboards, and pre-built agents for engineering, sales, support, operations, and marketing teams.

Join the waitlist: [1mcp.in/team](https://1mcp.in/team).

## Repository Layout

```text
services/
  mach1/                   Go router, API server, CLI, sandbox, supervisor
  web-ui/                  SvelteKit + Tauri desktop Hub
    src/routes/            App pages (dashboard, servers, discover, clients, settings)
    e2e/                   E2E test suites (smoke, quality, stress, integration)
packages/
  mcp-manifest/            Manifest JSON Schema
  registry-index/          Signed marketplace catalog
scripts/                   Build, install, and E2E helpers
.github/workflows/         CI + release pipelines
```

## Star The Project

If 1mcp saves you from one more copy-pasted MCP config file, star the repository. It helps the marketplace grow and tells us which integrations to verify next.

## License

Apache 2.0. See [LICENSE](LICENSE).
