# @mach1/mcp-manifest

Source-of-truth schema for an installable MCP in 1mcp.in.

- `manifest.schema.json` — JSON Schema (draft 2020-12). Edit this file and only this file.
- Go types: generated into `services/mach1/internal/manifest` (see `services/mach1/Makefile`).
- TS types: generated into `services/web-ui/src/lib/manifest.d.ts`.

## Validation rules worth knowing

- `id` is also the tool-name prefix (`<id>__<toolName>`) used by the router to avoid collisions across child MCPs.
- `entrypoint` is a discriminated union: `{command,args}` for `node`/`python`/`binary`, `{image,args,mounts}` for `docker`.
- `envSchema[].secret=true` values are stored in the OS keychain, never in SQLite.
- `embeddingText` is optional in the catalog; the hub fills it at install time from `name + description + capabilities + tool descriptions`.
