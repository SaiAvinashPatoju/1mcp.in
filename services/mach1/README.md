# mach1

The 1mcp.in central router. Speaks MCP over stdio or Streamable HTTP to one client (e.g. VS Code, Cursor, Claude Desktop), aggregates tools from N installed child MCPs, and routes `tools/call` to the right child by namespaced tool name.

## Build

```bash
cd services/mach1

# All binaries
go build -trimpath -ldflags "-s -w" -o ../../bin/mach1 ./cmd/mach1
go build -trimpath -ldflags "-s -w" -o ../../bin/mach1ctl ./cmd/mach1ctl
go build -trimpath -ldflags "-s -w" -o ../../bin/mcpapiserver ./cmd/mcpapiserver

# Or use the repo build script
../../scripts/build.sh        # macOS/Linux
../../scripts/build.ps1       # Windows
```

The `modernc.org/sqlite` driver is pure Go, so no CGO toolchain is required.

## Run

### stdio mode (default, for AI clients)

```bash
../../bin/mach1 --db "$HOME/.mach1/registry.db" --log debug
```

`stdout` is the MCP wire — do not interleave anything else there. All logs go to `stderr`.

### Streamable HTTP mode

```bash
../../bin/mach1 --transport http --listen 127.0.0.1:3000
```

### Dev mode with file config

```bash
../../bin/mach1 --config config.example.json --log debug
```

## CLI Usage

```bash
../../bin/mach1ctl catalog list          # List marketplace MCPs
../../bin/mach1ctl install <id>          # Install an MCP
../../bin/mach1ctl uninstall <id>        # Uninstall an MCP
../../bin/mach1ctl list                  # Show installed MCPs
../../bin/mach1ctl env set <id> <key>=<val>  # Set env vars
../../bin/mach1ctl connect <client>      # Auto-configure an AI client
../../bin/mach1ctl doctor                # Verify installation
../../bin/mach1ctl start                 # Launch router + Hub UI
```

## Wiring into VS Code

Add to your user `mcp.json`:

```jsonc
{
  "servers": {
    "mach1": {
      "command": "/path/to/bin/mach1",
      "args": ["--db", "/path/to/registry.db"]
    }
  }
}
```

Or use the auto-connect command:

```bash
mach1ctl connect vscode
```

## Smoke Test

1. Build the binary.
2. Run with `config.example.json` (requires Node.js for `npx`).
3. From a second terminal, drive it with any MCP client:

```bash
npx @modelcontextprotocol/inspector ./bin/mach1 --config services/mach1/config.example.json
```

4. In the inspector you should see tools named `fs__read_file`, `memory__create_entities`, etc. Call one and confirm the response round-trips.

## Layout

```
cmd/
  mach1/           Router entrypoint + flags
  mach1ctl/        CLI (install, list, env, connect, start)
  mcpapiserver/    Cloud API server
  mach1e2e/        E2E test harness
  stubmcp/         Stub MCP for testing
internal/
  router/          MCP stdio server, tool aggregation, tools/call routing
  registry/        SQLite-backed list of installed MCPs
  sandbox/         Docker and process sandbox drivers
  supervisor/      Process lifecycle, semantic ranking
  transport/       stdio and Streamable HTTP transport
  clouddb/         PostgreSQL schema (marketplace, auth)
  proto/           JSON-RPC 2.0 + MCP message types
  framing/         Newline-delimited JSON reader/writer
  upstream/        Child MCP process client (spawn, JSON-RPC, handshake)
  secrets/         OS keychain bridge
  clients/         AI client config auto-detection
```

## Design Invariants

- `stdout` is sacred: only MCP frames. Logs always go to `stderr`.
- Tool names exposed to the client are `<upstreamId>__<toolName>`. The router splits on the first `__` and forwards the suffix unchanged.
- Upstream `tools/call` results are forwarded as raw JSON — we never re-encode them, so any extension fields the upstream returns are preserved.
- One slow upstream cannot block another: each client request runs on its own goroutine, and each upstream has its own pending-request map.
