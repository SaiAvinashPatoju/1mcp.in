# 1mcp.in Getting Started Guide

1mcp.in is a lightweight, high-performance router that lets you manage multiple Model Context Protocol (MCP) servers behind a single entry point.

## 1. Install

### One-command install (recommended)

```bash
# macOS / Linux
curl -fsSL https://install.1mcp.in | sh

# Windows PowerShell
irm https://install.1mcp.in/windows | iex
```

### Build from source

```bash
git clone https://github.com/SaiAvinashPatoju/1mcp.in.git
cd 1mcp.in

# Windows
pwsh -ExecutionPolicy Bypass -File scripts/build.ps1

# macOS / Linux
bash scripts/build.sh

# Add bin/ to your PATH for convenience
$env:Path += ";$pwd\bin"           # PowerShell
export PATH="$PWD/bin:$PATH"       # bash/zsh
```

Verify the installation:

```bash
mach1ctl doctor
```

## 2. Start the Router

```bash
mach1ctl start
```

This launches the `mach1` router and the Hub UI at `http://localhost:5173`.

## 3. Browse & Install MCPs from the Marketplace

Open the Hub UI (`http://localhost:5173`) and go to the **Discover** tab. Browse 18+ curated MCPs and click **Install** on any server.

Or use the CLI:

```bash
# List available MCPs
mach1ctl catalog list

# Install one (e.g., Knowledge Graph Memory)
mach1ctl install memory

# Configure required environment variables
mach1ctl env set filesystem MACH1_FS_ROOT="C:\my-allowed-folder"
```

## 4. Connect to Your AI Clients

1mcp.in auto-configures your AI tools to use the central router.

```bash
# VS Code (GitHub Copilot / Continue / Roo Code)
mach1ctl connect vscode

# Cursor
mach1ctl connect cursor

# Claude Desktop
mach1ctl connect claude

# Claude Code
mach1ctl connect claude-code
```

After connecting, restart your AI client. It will see a single MCP server (`mach1`) with all your installed MCP tools namespaced (e.g., `github__list_pull_requests`, `memory__create_entities`).

## 5. Manage Installed Servers

```bash
# List active router configuration
mach1ctl list

# Enable / Disable / Uninstall
mach1ctl enable filesystem
mach1ctl disable filesystem
mach1ctl uninstall filesystem
```

All of these operations are also available from the Hub UI (Servers / Discover tabs).

## 6. Run the Router Manually (Advanced)

```bash
# stdio mode (default)
mach1 --db "$HOME/.mach1/registry.db" --log debug

# Streamable HTTP mode
mach1 --transport http --listen 127.0.0.1:3000
```

Metrics are available at `127.0.0.1:3031/metrics` in stdio mode.

## 7. Run Tests

```bash
cd services/web-ui

# Unit tests
npm run test

# E2E smoke tests
npm run test:e2e:smoke

# Full quality suite (108 tests)
npm run test:e2e:quality

# Go tests
cd services/mach1
go test ./...
```
