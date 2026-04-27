# OneMCP

The single entry point for all your Model Context Protocol (MCP) servers.

OneMCP provides a lightweight router that multiplexes multiple MCP servers into a single interface. It handles the lifecycle, sandboxing, and discovery of MCPs so your AI agents only need to talk to one endpoint.

## Architecture

- **`centralmcpd`**: A high-performance Go router that sits between your agent and your MCPs.
- **Hub UI**: A SvelteKit + Tauri desktop application for managing, configuring, and discovering MCPs.
- **Cloud API**: Syncs your settings and provides a curated marketplace.

## Quick Start

### 1. Build the Router
```bash
cd services/central-mcp
go build -o ../../bin/centralmcpd ./cmd/centralmcpd
```

### 2. Run the UI
```bash
cd services/web-ui
npm install && npm run dev
```

### 3. Connect to Agent
Connect your MCP client (VS Code, Claude, etc.) to the `centralmcpd` binary.

## Project Structure

- `services/central-mcp`: The core Go router and supervisor.
- `services/web-ui`: The management dashboard (SvelteKit).
- `services/mcpapiserver`: Cloud backend for auth and marketplace.
- `packages/mcp-manifest`: Shared schema and types.

## License
MIT

