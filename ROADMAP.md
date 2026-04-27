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

- [x] Go module `services/central-mcp` with `centralmcpd` binary.
- [x] Implement MCP server (stdio) speaking JSON-RPC 2.0 per MCP spec.
- [x] In-memory registry loaded from SQLite (`installed_mcps` table).
- [x] Static router: forward `tools/call` to a hard-coded child by name.
- [x] Structured logging (zerolog) + pprof endpoint behind flag.

## Phase 2 — Cloud Infrastructure & Hub UI (current)

- [x] Deploy Cloud API (`mcpapiserver`) to Railway.
- [x] Provision Managed PostgreSQL for user accounts and marketplace metadata.
- [x] Implement Auth bridge (Bcrypt + pgx) in Cloud API.
- [x] Wire Svelte UI (`services/web-ui`) to live Cloud API.
- [ ] Implement Tauri v2 bridge for local FS/Registry operations.
- [ ] Marketplace catalog sync (Cloud → Local SQLite).

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

---

## Enterprise-Grade Improvements (Pre-Launch Audit)

> Notes from architecture review — 2026-04-27. These must be addressed before any public or enterprise launch.

### Security
- [ ] **CORS lockdown**: Replace `Access-Control-Allow-Origin: *` with an allowlist of known origins (Railway domain, `tauri://localhost`, `http://localhost:*`). Current wildcard allows any site to call the API.
- [ ] **Rate limiting**: Add per-IP rate limits on `/api/auth/login` and `/api/auth/register` to prevent brute-force attacks. Use a token bucket or sliding window (e.g. 10 req/min for login).
- [ ] **CSRF protection**: Add CSRF tokens for state-changing requests or switch to `SameSite=Strict` cookies.
- [ ] **Token rotation**: Implement refresh tokens. Current 30-day bearer tokens are long-lived and non-revocable after issuance.
- [ ] **Input sanitization**: Validate email format server-side (regex or net/mail parse). Currently accepts any string as email.
- [ ] **Password policy**: Enforce complexity beyond 8-char minimum (at least 1 uppercase, 1 number, or use zxcvbn scoring).
- [ ] **Audit logging**: Log all auth events (login, register, failed login) to a separate audit table with IP + user-agent.

### Scalability & Reliability
- [ ] **Connection pooling**: pgxpool is configured with defaults. Set explicit `MaxConns`, `MinConns`, `MaxConnLifetime`, and `HealthCheckPeriod` for production loads.
- [ ] **Graceful shutdown**: `mcpapiserver` has no signal handler — `SIGTERM` kills in-flight requests. Add `context.WithCancel` + `server.Shutdown(ctx)`.
- [ ] **Health check depth**: `/health` returns static OK. Add DB ping and upstream dependency checks for real liveness/readiness probes.
- [ ] **Horizontal scaling**: `centralmcpd` is single-process stdio. For multi-user cloud scenarios, add an SSE/WebSocket transport adapter.
- [ ] **Session cleanup**: Expired sessions accumulate forever. Add a periodic job or DB trigger to purge `user_sessions WHERE expires_at < now()`.

### Observability
- [ ] **Structured logging**: `mcpapiserver` uses `slog` but has no request-level middleware. Add HTTP middleware that logs method, path, status, latency, and request ID.
- [ ] **Metrics**: Export Prometheus metrics (request count, latency histogram, active connections, DB pool stats).
- [ ] **Distributed tracing**: Add OpenTelemetry spans for API → DB calls. Essential for debugging in production.
- [ ] **Error reporting**: Integrate Sentry or equivalent for unhandled panics and error aggregation.

### Frontend / UX
- [ ] **Real console integration**: Console currently only shows logs when Tauri events are available. Add a WebSocket or SSE endpoint in `mcpapiserver` that streams router logs to the web UI.
- [ ] **Offline mode**: The web UI silently fails when the API is unreachable. Show a clear offline banner and use Tauri local SQLite as fallback for all views.
- [ ] **Loading states**: Dashboard fetches user count but shows 0 during load. Add skeleton loaders for all async data.
- [ ] **Error boundaries**: No global error boundary in SvelteKit. Add `+error.svelte` pages and an `handleError` hook.
- [ ] **Accessibility**: Add ARIA labels, keyboard navigation for modals, and focus trapping. Currently no a11y testing.

### Data Integrity
- [ ] **DB migrations**: Schema is applied via `CREATE IF NOT EXISTS` at startup. This won't handle column additions or type changes. Adopt a migration tool (golang-migrate, atlas, or goose).
- [ ] **Backup strategy**: No automated backups configured for Railway Postgres. Enable point-in-time recovery.
- [ ] **Data validation**: Marketplace items have no validation on publish. Add schema validation and content moderation.

### CI/CD & Testing
- [ ] **Integration tests**: No test coverage for `mcpapiserver` HTTP handlers or `clouddb` methods. Add table-driven Go tests with testcontainers/pgx mock.
- [ ] **Frontend tests**: Zero test files in `services/web-ui`. Add Playwright E2E tests for login flow, dashboard, and client setup.
- [ ] **Build pipeline**: No CI config. Add GitHub Actions for: lint, test, build binaries, build Tauri app, run E2E.
- [ ] **Release automation**: No versioning strategy. Adopt semver, changelog generation, and signed releases.

### MCP Router Improvements
- [ ] **Concurrent warmup timeout**: All MCPs warm up in parallel with a shared 15s timeout. If one MCP hangs, it doesn't block others (good), but there's no per-MCP circuit breaker to prevent retrying a permanently broken MCP.
- [ ] **Tool deduplication**: If two MCPs expose the same tool name, namespacing handles it (`github__search`, `memory__search`), but there's no conflict detection or user notification.
- [ ] **Resource limits**: No memory/CPU limits on child processes in the process driver. A runaway MCP can consume all system resources.
- [ ] **Streaming support**: The router buffers entire responses. For large tool outputs, add streaming pass-through.
