# Contributing an MCP to the 1mcp.in Marketplace

Thank you for contributing to the 1mcp.in MCP registry! Every new MCP makes the platform more useful for every user.

## Prerequisites

- Your MCP server must be published and installable via `npx`, `uvx`, `pipx`, or a binary download
- You must be the maintainer or have the maintainer's approval to submit
- Your MCP must not contain malware, credential stealers, or any form of supply-chain attack

## How to submit

### 1. Fork the repository

```bash
gh repo fork SaiAvinashPatoju/1mcp.in --clone
```

### 2. Create a manifest entry

Add one JSON object to `packages/registry-index/index.json`. Insert it in alphabetical order by `id`.

### 3. Manifest schema

Every entry must conform to the JSON Schema at `packages/mcp-manifest/manifest.schema.json` and include these fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | string | yes | Stable unique ID (`kebab-case`, lowercase). Used as tool name prefix in clients. |
| `name` | string | yes | Human-readable name (max 80 chars). |
| `version` | string | yes | Semver string (`X.Y.Z`). |
| `description` | string | no | Short description of what this MCP does. |
| `homepage` | string (uri) | no | Project homepage or repository. |
| `license` | string | no | SPDX license identifier (e.g. MIT, Apache-2.0). |
| `tags` | string[] | no | Up to 32 tags for filtering. |
| `transport` | string | yes | One of: `stdio`, `sse`, `http`. |
| `runtime` | string | yes | One of: `node`, `python`, `docker`, `binary`. |
| `entrypoint` | object | yes | How to launch this MCP (see below). |
| `envSchema` | array | no | Environment variables the user must configure. |
| `permissions` | object | no | Declared permissions (network, filesystem). |
| `verification` | string | no | Set to `"community"` for your submission. |

### 4. Entrypoint formats

**Node (npx):**
```json
"entrypoint": {
  "command": "npx",
  "args": ["-y", "@your-scope/mcp-server"]
}
```

**Python (uvx):**
```json
"entrypoint": {
  "command": "uvx",
  "args": ["your-mcp-server"]
}
```

**Docker:**
```json
"entrypoint": {
  "image": "docker.io/your-image:mcp-latest",
  "args": []
}
```

**Binary:**
```json
"entrypoint": {
  "command": "your-mcp-binary",
  "args": ["--flag", "value"]
}
```

### 5. Environment variables (envSchema)

List every environment variable your MCP requires:

```json
"envSchema": [
  {
    "name": "MY_API_KEY",
    "label": "My API Key",
    "description": "Get one at https://example.com/settings/tokens",
    "secret": true,
    "required": true
  },
  {
    "name": "MY_REGION",
    "label": "Region",
    "default": "us-east-1"
  }
]
```

- `secret: true` values are encrypted in the vault and never logged
- The user is prompted for `required: true` env vars during installation

### 6. Example entry

```json
{
  "id": "my-service",
  "name": "My Service",
  "version": "1.0.0",
  "description": "Do something useful via the My Service API.",
  "homepage": "https://github.com/you/my-service-mcp",
  "license": "MIT",
  "tags": ["my-service", "utility"],
  "transport": "stdio",
  "runtime": "node",
  "entrypoint": {
    "command": "npx",
    "args": ["-y", "@your-scope/mcp-server"]
  },
  "envSchema": [
    {
      "name": "MY_SERVICE_API_KEY",
      "label": "My Service API Key",
      "secret": true,
      "required": true
    }
  ],
  "permissions": {
    "network": true
  },
  "verification": "community",
  "sha256": "0000000000000000000000000000000000000000000000000000000000000000"
}
```

### 7. Open a Pull Request

Open a PR with your single-entry addition. The CI will:

1. ✅ Validate the JSON Schema
2. ✅ Check required fields are present
3. ✅ Verify all IDs are unique and lowercase kebab-case
4. ⏳ Flag the entry as `community` — requires maintainer review

### 8. Maintainer review

A maintainer will:
1. Download and test your MCP manually
2. Scan for malicious patterns (network calls to unknown hosts, filesystem exfiltration, crypto miners)
3. Compute the SHA256 hash of your canonical manifest
4. Replace the placeholder hash with the computed hash
5. Optionally sign the entry
6. Approve and merge

Once merged, your MCP will appear in the 1mcp.in marketplace on the next release build.

## Do NOT

- Submit entries with placeholder SHA256 hashes that contain malicious code
- Submit entries that require elevated OS permissions (sudo, root access)
- Submit entries that exfiltrate user data
- Submit multiple entries in a single PR (one PR = one MCP)

## Questions?

Open a discussion at https://github.com/SaiAvinashPatoju/1mcp.in/discussions or join our community chat.
