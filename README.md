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
| 2 | Tauri hub (download/install) | ⏳ next |
| 3 | MCP management (keys, delete) | ⏳ |
| 4 | Client connect (VS Code shim + wizard) | ⏳ |
| 5 | Sandbox + lazy start | ⏳ |
| 6 | Semantic routing (LanceDB + ONNX) | ⏳ |
| 7 | E2E test matrix | ⏳ |

## Quick start (Phase 1)

Requires Go 1.22+ and Node.js (only for the example child MCPs via `npx`).

```powershell
cd services\central-mcp
go build -o ..\..\bin\centralmcpd.exe .\cmd\centralmcpd
..\..\bin\centralmcpd.exe --config .\config.example.json --log debug
```

Then connect any MCP client (e.g. `npx @modelcontextprotocol/inspector`) to
the binary. You should see tools namespaced as `fs__*` and `memory__*`.
