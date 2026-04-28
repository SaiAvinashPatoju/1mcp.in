# Contributing MCPs to the 1mcp Marketplace

The marketplace grows through pull requests. Community submissions are never auto-approved or auto-verified: a maintainer reviews the MCP, tests it, signs the catalog digest, and merges it.

## Submission Flow

1. Fork the repository.
2. Add one JSON object to `packages/registry-index/index.json`.
3. Set `verification` to `community`.
4. Leave `sha256` and `signature` empty. Maintainers fill these after review.
5. Open a pull request with a short test note and links to the MCP source/package.

CI validates manifest structure, duplicate IDs, and maintainer-pinned SHA256 values. A maintainer will run the MCP locally, inspect install behavior, check for suspicious code paths, then sign the entry with:

```bash
cd services/mach1
go run ./cmd/mach1signregistry --catalog ../../packages/registry-index/index.json
```

## Manifest Template

```json
{
  "id": "notion-enhanced",
  "name": "Notion Enhanced",
  "version": "1.0.0",
  "description": "Full Notion API coverage including databases, pages, and blocks.",
  "homepage": "https://github.com/contributor/notion-enhanced-mcp",
  "license": "MIT",
  "tags": ["notion", "docs", "community"],
  "transport": "stdio",
  "runtime": "node",
  "entrypoint": {
    "command": "npx",
    "args": ["-y", "notion-enhanced-mcp"]
  },
  "envSchema": [
    {
      "name": "NOTION_API_KEY",
      "label": "Notion API Key",
      "secret": true,
      "required": true
    }
  ],
  "permissions": { "network": true },
  "verification": "community",
  "sha256": "",
  "signature": ""
}
```

## Trust Labels

- `anthropic-official`: official MCPs from the Model Context Protocol / Anthropic catalog.
- `1mcp.in-verified`: reviewed and tested by 1mcp maintainers.
- `community`: submitted by the community and signed only after maintainer review.

## Security Rules

- Do not submit obfuscated install commands.
- Do not request broad filesystem access unless the MCP genuinely needs it.
- Declare secrets in `envSchema` with `"secret": true`.
- Add `toolAnnotations` when known so admins can identify read-only, destructive, and idempotent tools.
- Maintainers, not contributors, set verification upgrades and SHA256 values.