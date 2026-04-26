# OneMCP — Execution Roadmap

> Stack contract (non-negotiable): Tauri v2 + SvelteKit + Tailwind/shadcn (hub UI) · Go (central router) · Docker (sandbox, MVP) · LanceDB + ONNX MiniLM (semantic routing) · TS thin VS Code shim.
> Performance budget: cold route < 50ms · idle RAM (router) < 30MB · hub binary < 15MB.

---

## Repo Layout (target)

```
OneMcp/
├── apps/
│   ├── hub/                  # Tauri v2 + SvelteKit (marketplace + admin)
│   │   ├── src/              # Svelte UI
│   │   └── src-tauri/        # Rust: install/download/fs ops
│   └── vscode-shim/          # TS extension: registers central MCP as single entry
├── services/
│   └── central-mcp/          # Go: router, registry, sandbox supervisor, RAG
│       ├── cmd/centralmcpd/
│       └── internal/
│           ├── registry/     # installed MCP manifest store (SQLite)
│           ├── router/       # MCP protocol multiplexer (stdio/SSE)
│           ├── sandbox/      # docker driver (later: wasm/deno)
│           ├── semantic/     # lancedb + onnx embedder
│           └── secrets/      # OS keychain bridge
├── packages/
│   ├── mcp-manifest/         # shared schema (JSON Schema + Go/TS types)
│   └── registry-index/       # static catalog JSON for MVP marketplace
└── ROADMAP.md
```

---

## Phase 0 — Foundations (plan)

- [ ] Lock `mcp-manifest` schema: `id, name, version, transport(stdio|sse), entrypoint, runtime(node|python|docker), capabilities[], envSchema[], permissions{net,fs}, embeddingText`.
- [ ] Define wire contract: VS Code shim ⇄ central-mcp (stdio MCP) ⇄ child MCP (stdio in sandbox).
- [ ] Decision log: Docker for MVP sandbox; WASM/Deno deferred to Phase 2.
- [ ] Bench harness skeleton (route latency, mem RSS, cold-start).

**Exit:** schema frozen, ADR written, bench script runs against stub.

## Phase 1 — Central MCP skeleton (build core)

- [ ] Go module `services/central-mcp` with `centralmcpd` binary.
- [ ] Implement MCP server (stdio) speaking JSON-RPC 2.0 per MCP spec.
- [ ] In-memory registry loaded from SQLite (`installed_mcps` table).
- [ ] Static router: forward `tools/call` to a hard-coded child by name.
- [ ] Structured logging (zerolog) + pprof endpoint behind flag.

**Exit:** `centralmcpd` proxies one real upstream MCP (e.g. `@modelcontextprotocol/server-filesystem`) end-to-end via stdio.

## Phase 2 — MCP download & install (test mcp download)

- [ ] Static `registry-index` JSON (10 curated MCPs) served from Tauri bundle + GitHub raw fallback.
- [ ] Tauri Rust commands: `install_mcp(id)`, `uninstall_mcp(id)`, `list_installed()`.
  - Pulls manifest, materializes config under `%APPDATA%/OneMcp/mcps/<id>/`.
  - For Docker MCPs: `docker pull` via shellout; for node/python: pin version, no global install.
- [ ] SvelteKit marketplace page: grid + detail drawer + install progress (Tauri events).
- [ ] Persist install record to central-mcp SQLite (shared DB path).

**Exit:** click install in UI → file present on disk → row in registry → `centralmcpd` lists it.

## Phase 3 — MCP management (api key / delete / toggle)

- [ ] Manifest `envSchema` drives a dynamic settings form in Svelte.
- [ ] Secrets stored via OS keychain (Tauri `keyring` crate; central-mcp reads via IPC at launch only).
- [ ] Per-MCP toggle: `enabled`, `autostart`, `permissions` overrides.
- [ ] Delete flow: stop sandbox → remove files → purge secrets → drop registry row.

**Exit:** user can paste a GitHub PAT, enable the MCP, restart router, and the child receives the env.

## Phase 4 — Connect central MCP to client

- [ ] `apps/vscode-shim`: minimal TS extension that registers `centralmcpd` (absolute path) as an MCP server in the user's `mcp.json`.
- [ ] Tauri "Connect to client" wizard: detects VS Code / Cursor / Claude Desktop config files; writes a single entry pointing to `centralmcpd`.
- [ ] Health check command in UI that pings the running router.

**Exit:** fresh machine → install hub → click Connect → VS Code agent sees ONE server (`onemcp`) exposing all installed tools.

## Phase 5 — Sandbox supervisor + on-demand activation

- [ ] `sandbox/docker.go`: lazy-start child MCP on first `tools/call`, idle-shutdown after N seconds.
- [ ] Resource caps: `--memory`, `--cpus`, `--network=none` unless manifest grants net.
- [ ] Stream multiplexer: one stdio pair per active child, fan-in/out by request id.
- [ ] Crash supervisor with exponential backoff + circuit breaker.

**Exit:** 20 installed MCPs, only the invoked one is running; idle RAM stays flat.

## Phase 6 — Semantic routing (RAG brain)

- [ ] Embed `manifest.embeddingText + tool descriptions` at install time → LanceDB table `tool_vectors`.
- [ ] ONNX Runtime Go bindings + `all-MiniLM-L6-v2` (quantized int8) bundled.
- [ ] Hybrid scorer: BM25 (sqlite FTS5) + cosine, weighted; top-k tool surfacing.
- [ ] Optional `tools/list` filter mode: only expose top-N relevant tools per agent turn (reduces context bloat).

**Exit:** p50 routing decision < 50ms cold, < 5ms warm; measured by bench harness.

## Phase 7 — Test all MCPs working (E2E)

- [ ] E2E harness: spawn `centralmcpd`, drive it as an MCP client, iterate registry, call one canonical tool per MCP, assert non-error response.
- [ ] Latency report artifact (markdown table) committed per release.
- [ ] Smoke test in CI on Win + macOS runners.

**Exit:** green matrix across curated registry; regression gate before release.

---

## Phase 8+ — Next upgrades (post-MVP)

- [ ] **Public MCP Registry service** (separate repo): signed manifests, versioning, search API; hub falls back to it beyond the static index.
- [ ] **OneMCP SDK** (`@onemcp/sdk` for TS, `onemcp-go` for Go): lets any client (VS Code, Cursor, custom agents) discover + list OneMCP-managed tools without rewriting MCP plumbing.
- [ ] **WASM sandbox driver** behind same `sandbox.Driver` interface — drop-in once components-model MCPs exist.
- [ ] **Deno isolate driver** for JS/TS MCPs (sub-100ms cold start, permission flags from manifest).
- [ ] Telemetry-free local analytics dashboard (per-tool latency, error rate).
- [ ] Signed-manifest verification + capability prompts (least-privilege UX).

---

## Risk register

| Risk | Mitigation |
|---|---|
| Docker not installed on user machine | Detect at install; gate Docker-only MCPs; offer node/python path first. |
| MCP spec churn | Pin to dated spec rev; isolate protocol layer behind `internal/router/proto`. |
| ONNX binary size bloats hub | Ship embedder inside `centralmcpd` (Go), not Tauri; lazy-load model. |
| stdio multiplexing deadlocks | Per-child writer goroutine + bounded channel; fuzz test. |
