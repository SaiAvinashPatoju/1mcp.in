# 1mcp — One router for all your AI tools

Install MCP servers once, connect every AI client. No duplicated configs, no scattered credentials.

![1mcp Hub UI demo](https://1mcp.in/assets/hub-demo.gif)

[![CI](https://github.com/SaiAvinashPatoju/1mcp.in/actions/workflows/ci.yml/badge.svg)](https://github.com/SaiAvinashPatoju/1mcp.in/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)
[![Release](https://img.shields.io/github/v/release/SaiAvinashPatoju/1mcp.in)](https://github.com/SaiAvinashPatoju/1mcp.in/releases)

## Install

```bash
curl -fsSL https://install.1mcp.in | sh   # macOS/Linux
irm https://install.1mcp.in/windows | iex  # Windows
```

```bash
mach1ctl start                 # launch router + Hub UI
mach1ctl install github        # install GitHub MCP
mach1ctl connect cursor        # connect Cursor (replaces direct MCPs)
mach1ctl doctor                # verify everything works
```

## Why this exists

Every AI client (Cursor, VS Code, Claude Desktop, Claude Code, Windsurf, Codex) needs its own MCP config file. If you use GitHub, Slack, Linear, Notion, and Postgres MCPs with 4 clients, you configure 20 entries and paste credentials 20 times.

1mcp replaces all of them with **one router process** and **one Hub UI**. Install each MCP once, connect every client with one command. When mach1 connects to a client, it **takes over** the MCP config — removing all direct entries so the AI client has no choice but to route every tool call through mach1. Your original config is backed up and restored when you disconnect.

## Quick start

```bash
mach1ctl install github
mach1ctl env set github GITHUB_TOKEN=ghp_...
mach1ctl connect cursor        # replaces all cursor MCPs with mach1
mach1ctl connect claude        # replaces all claude MCPs with mach1
```

One install. One credential entry. Every client uses `github__list_pull_requests`, `github__create_issue`, etc.

To undo:
```bash
mach1ctl disconnect cursor     # restores your original MCP config
```

## Supported clients

| Client | Connect | Status |
|---|---|---|
| VS Code / GitHub Copilot | `mach1ctl connect vscode` | ✅ Supported |
| Cursor | `mach1ctl connect cursor` | ✅ Supported |
| Claude Desktop | `mach1ctl connect claude` | ✅ Supported |
| Claude Code | `mach1ctl connect claudecode` | ✅ Supported |
| Windsurf | `mach1ctl connect windsurf` | ✅ Supported |
| Codex | `mach1ctl connect codex` | ✅ Supported |
| OpenCode | `mach1ctl connect opencode` | ✅ Supported |

## Security model

| Layer | What 1mcp does |
|---|---|
| Tool integrity | SHA256 of every tool manifest is pinned on install. Changes require re-approval. |
| Supply chain | Every marketplace entry has a maintainer-signed SHA256 digest. Install fails on mismatch. |
| PII scrubbing | Emails, phones, credit cards, API keys, tokens redacted before logs/UI output. |
| Process sandbox | Docker MCPs capped at 1 CPU, 512MB RAM, read-only fs, no network unless granted. |
| Credential storage | Secrets stored locally in OSS version, never logged. Team Pro uses AES-256-GCM vault. |
| Catalog signing | Community entries reviewed and signed by maintainers. No auto-approval. |

## Marketplace

The catalog at `packages/registry-index/index.json` has 50+ MCPs with trust labels:

- **anthropic-official** — from the Anthropic MCP catalog
- **1mcp.in-verified** — reviewed and tested by maintainers
- **community** — submitted via PR, signed only after human review

To add an MCP to the marketplace, see [CONTRIBUTING_MCP.md](CONTRIBUTING_MCP.md).

## Team Pro

Shared MCP configuration across your team, credential vaulting, admin approval gates, activity logs, custom agents, and usage dashboards. 5-50 seats at ₹1,999/seat/month.

[1mcp.in/team](https://1mcp.in/team) — waitlist open.

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
| One-command connect | ✅ | ❌ | ❌ |

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

---

If 1mcp saves you time, [star the repo](https://github.com/SaiAvinashPatoju/1mcp.in) ⭐
