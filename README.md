# 1mcp — local-first MCP control plane

Install MCP servers once, approve them once, and use them safely across Cursor, Claude Desktop, VS Code, Claude Code, Windsurf, and more.

No duplicated configs. No scattered credentials. No silent tool changes.

![1mcp Hub UI demo](https://1mcp.in/assets/hub-demo.gif)

[![CI](https://github.com/SaiAvinashPatoju/1mcp.in/actions/workflows/ci.yml/badge.svg)](https://github.com/SaiAvinashPatoju/1mcp.in/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

## Install

```bash
curl -fsSL https://install.1mcp.in | sh   # macOS/Linux
irm https://install.1mcp.in/windows | iex  # Windows
```

```bash
mach1ctl start                 # launch router + Hub UI
mach1ctl install github        # install GitHub MCP
mach1ctl connect cursor        # connect Cursor
mach1ctl connect claude        # connect Claude Desktop
mach1ctl doctor                # verify everything works
```

## Why this exists

MCP is becoming the USB-C of AI tools, but local setup is still a mess. Every AI client needs its own config file. Every MCP needs its own credentials pasted N times. Every teammate repeats the same ceremony.

1mcp replaces all of that with a single local router process and one Hub UI. Install once. Approve once. Connect everywhere.

## Killer workflow

**The problem:** You use Cursor, Claude Desktop, VS Code, and Claude Code. You configure GitHub MCP four times, paste your token four times, and update four configs when something breaks.

**With 1mcp:**

```bash
mach1ctl install github
mach1ctl env set github GITHUB_TOKEN=ghp_...
mach1ctl connect all
```

One install. One credential entry. Every client can now call `github__list_pull_requests`, `github__create_issue`, etc.

**When a tool changes:**

Upstream MCP silently renames `create_issue` to `create_issue_and_execute_hook`? 1mcp detects the hash mismatch, marks the tool `PENDING_REVIEW`, and blocks calls until you approve or reject the diff. No surprises.

## Security model

| Layer | What 1mcp does |
|---|---|
| Tool integrity | SHA256 of every tool manifest is pinned on install. Changes require re-approval. |
| Supply chain | Every marketplace entry has a maintainer-signed SHA256 digest. Install fails on mismatch. |
| PII scrubbing | Emails, phones, credit cards, API keys, tokens redacted before logs/UI output. |
| Process sandbox | Docker MCPs capped at 1 CPU, 512MB RAM, read-only fs, no network unless granted. |
| Credential storage | Secrets stored locally in OSS version, never logged. Team Pro uses AES-256-GCM vault. |
| Catalog signing | Community entries reviewed and signed by maintainers. No auto-approval. |

## Supported clients

| Client | Connect |
|---|---|
| VS Code / GitHub Copilot | `mach1ctl connect vscode` |
| Cursor | `mach1ctl connect cursor` |
| Claude Desktop | `mach1ctl connect claude` |
| Claude Code | `mach1ctl connect claude-code` |
| Windsurf | `mach1ctl connect windsurf` |
| OpenCode / Codex | `mach1ctl connect opencode` |

## Marketplace trust

The catalog at `packages/registry-index/index.json` carries trust labels:

- **anthropic-official** — from the Anthropic MCP catalog
- **1mcp.in-verified** — reviewed and tested by maintainers
- **community** — submitted via PR, signed only after human review

18 servers ship now. To add one, see [CONTRIBUTING_MCP.md](CONTRIBUTING_MCP.md).

## Verified packs

```bash
mach1ctl pack install dev    # GitHub, Filesystem, Playwright, Context7
mach1ctl pack install data   # Postgres, SQLite, BigQuery, DuckDB
mach1ctl pack install web    # Fetch, Firecrawl, Browserbase
```

## Comparison

| Capability | 1mcp | Raw client configs | Enterprise gateways |
|---|---|---|---|
| Local-first | ✅ | ✅ | Usually ❌ |
| One config for many clients | ✅ | ❌ | Sometimes |
| Hub UI | ✅ | ❌ | ✅ |
| Tool change review | ✅ | ❌ | Sometimes |
| Signed catalog installs | ✅ | ❌ | Sometimes |
| Local credential storage | ✅ | Manual | Usually centralized |
| Lazy MCP startup | ✅ | ❌ | Depends |
| Simple dev install | ✅ | ❌ | ❌ |

## Team Pro

Shared MCP configuration across your team, credential vaulting, admin approval gates, activity logs, usage dashboards, and pre-built agents for engineering, sales, and operations.

[1mcp.in/team](https://1mcp.in/team) — waitlist open.

## From source

```bash
git clone https://github.com/SaiAvinashPatoju/1mcp.in.git
cd 1mcp.in

# Windows
powershell -ExecutionPolicy Bypass -File scripts\build.ps1

# macOS/Linux
bash scripts/build.sh
```

See [GETTING_STARTED.md](GETTING_STARTED.md) for detailed development setup.

## Repository

```
services/mach1/          Go router, API server, CLI
services/web-ui/         SvelteKit + Tauri desktop Hub
packages/mcp-manifest/   Manifest JSON Schema
packages/registry-index/ Signed marketplace catalog
scripts/                 Build, install, E2E helpers
```

## License

Apache 2.0. See [LICENSE](LICENSE).
