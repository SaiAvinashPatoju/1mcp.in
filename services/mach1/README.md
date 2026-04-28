# mach1 (`mach1`)

The 1mcp.in central router. Speaks MCP over stdio to one client (e.g. VS Code,
Cursor, Claude Desktop), aggregates tools from N installed child MCPs, and
routes `tools/call` to the right child by namespaced tool name.

> Phase 1 scope only. No sandbox, no semantic routing, no lazy-start yet ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¬ÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡Ãƒâ€šÃ‚Â¬ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â
> see [../../ROADMAP.md](../../ROADMAP.md).

## Build

```powershell
cd services/mach1
go build -o ..\..\bin\mach1.exe .\cmd\mach1
```

The `modernc.org/sqlite` driver is pure Go, so no CGO toolchain is required.

## Run (dev, file config)

```powershell
.\bin\mach1.exe --config services\mach1\config.example.json --log debug
```

`stdout` is the MCP wire ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¬ÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡Ãƒâ€šÃ‚Â¬ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â do not interleave anything else there. All logs go
to `stderr`.

## Run (registry-backed, what the hub will use)

```powershell
.\bin\mach1.exe --db "$env:APPDATA\Mach1\registry.db"
```

If the DB does not exist it is created with an empty schema; `tools/list` will
return an empty array until the hub installs something.

## Wiring into VS Code

Add to your user `mcp.json`:

```jsonc
{
  "servers": {
    "mach1": {
      "command": "C:\\path\\to\\bin\\mach1.exe",
      "args": ["--db", "C:\\Users\\you\\AppData\\Roaming\\Mach1\\registry.db"]
    }
  }
}
```

## Smoke test (Phase 1 exit)

1. Build the binary.
2. Run it with `config.example.json` (requires Node.js for `npx`).
3. From a second terminal, drive it with any MCP client. Quickest is the
   official inspector:

   ```powershell
   npx @modelcontextprotocol/inspector .\bin\mach1.exe --config services\mach1\config.example.json
   ```

4. In the inspector you should see tools named `fs__read_file`,
   `fs__list_directory`, `memory__create_entities`, etc. Call one and confirm
   the response round-trips.

## Layout

```
cmd/mach1/      entrypoint + flags
internal/proto/       JSON-RPC 2.0 + MCP message types
internal/framing/     newline-delimited JSON reader/writer
internal/upstream/    child MCP process client (spawn, JSON-RPC, handshake)
internal/router/      stdio MCP server, tool aggregation, tools/call routing
internal/registry/    SQLite-backed list of installed MCPs
```

## Design invariants

- `stdout` is sacred: only MCP frames. Logs always go to `stderr`.
- Tool names exposed to the client are `<upstreamId>__<toolName>`. The router
  splits on the first `__` and forwards the suffix unchanged.
- Upstream `tools/call` results are forwarded as raw JSON ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¬ÃƒÆ’Ã†â€™Ãƒâ€šÃ‚Â¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡Ãƒâ€šÃ‚Â¬ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â we never re-encode
  them, so any extension fields the upstream returns are preserved.
- One slow upstream cannot block another: each client request runs on its own
  goroutine, and each upstream has its own pending-request map.
