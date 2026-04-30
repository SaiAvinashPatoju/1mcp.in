# 1mcp.in — Execution Roadmap

> Performance budget: cold route < 50ms — idle RAM (router) < 30MB — hub binary < 15MB.

---

## Repo Layout

```
services/
  mach1/                    Go router, registry, sandbox supervisor, API server
    cmd/
      mach1/                Router entrypoint + flags
      mach1ctl/             CLI (install, list, env, connect)
      mcpapiserver/         Cloud API server
      mach1e2e/             E2E test harness
      stubmcp/              Stub MCP for testing
    internal/
      router/               MCP protocol multiplexer (stdio/SSE)
      registry/             SQLite-backed install manifest store
      sandbox/              Docker sandbox driver
      secrets/              OS keychain bridge
      clouddb/              PostgreSQL auth + marketplace
      transport/            stdio + Streamable HTTP
      supervisor/           Process lifecycle, semantic ranking
  web-ui/                   SvelteKit + Tauri desktop Hub
    src/routes/             Svelte pages: dashboard, servers, discover, clients, settings
    e2e/                    E2E suites: smoke (11), quality (108), stress, integration
packages/
  mcp-manifest/             Shared schema (JSON Schema)
  registry-index/           Signed marketplace catalog (18 MCPs, 3 trust tiers)
scripts/                    Build, install, and E2E helpers
.github/workflows/          CI + release pipelines
```

---

## Phase 0 — Foundations

- [x] Lock `mcp-manifest` schema: `id, name, version, transport(stdio|sse), entrypoint, runtime(node|python|docker), capabilities[], envSchema[], permissions{net,fs}, embeddingText`.
- [x] Define wire contract: AI client → mach1 (stdio MCP) → child MCP (stdio in sandbox).
- [x] Decision log: Docker for MVP sandbox; WASM/Deno deferred.
- [x] Bench harness skeleton (route latency, mem RSS, cold-start).

## Phase 1 — Central MCP skeleton

- [x] Go module `services/mach1` with `mach1` binary.
- [x] Implement MCP server (stdio) speaking JSON-RPC 2.0 per MCP spec.
- [x] In-memory registry loaded from SQLite (`installed_mcps` table).
- [x] Static router: forward `tools/call` to child MCP by namespaced name.
- [x] Structured logging (zerolog) + pprof endpoint behind flag.

## Phase 2 — Cloud Infrastructure & Hub UI

- [x] Deploy Cloud API (`mcpapiserver`) to Railway.
- [x] Provision Managed PostgreSQL for user accounts and marketplace metadata.
- [x] Implement Auth bridge (Bcrypt + pgx) in Cloud API.
- [x] Wire Svelte UI (`services/web-ui`) to live Cloud API.
- [x] Implement Tauri v2 bridge for local FS/Registry operations.
- [ ] Marketplace catalog sync (Cloud → Local SQLite).

## Phase 2b — MCP download & install

- [x] Static `registry-index` JSON (18 curated MCPs, 3 trust tiers) served from Tauri bundle + GitHub raw fallback.
- [x] Tauri Rust commands: `install_mcp`, `uninstall_mcp`, `list_installed`.
- [x] SvelteKit marketplace page with grid + detail drawer + install progress.
- [x] Persist install record to mach1 SQLite (shared DB path).

## Phase 3 — MCP management (env / toggle / delete)

- [x] Manifest `envSchema` drives dynamic settings form in UI.
- [x] Secrets stored via OS keychain (Tauri `keyring` crate; mach1 reads via IPC at launch only).
- [x] Per-MCP toggle: `enabled`, `autostart`, `permissions` overrides.
- [x] Delete flow: uninstall → remove files → purge secrets → drop registry row.

## Phase 4 — Connect central MCP to client

- [ ] `apps/vscode-shim`: TS extension that registers mach1 as a single MCP entry.
- [x] In-app "Connect to client" wizard: detects VS Code / Cursor / Claude Desktop config files; writes a single entry pointing to mach1.
- [x] Health check command in UI that pings the running router.

## Phase 5 — Sandbox supervisor + on-demand activation

- [x] Docker sandbox driver: lazy-start child MCP on first `tools/call`, idle-shutdown after N seconds.
- [ ] Resource caps: `--memory`, `--cpus`, `--network=none` unless manifest grants net.
- [x] Stream multiplexer: one stdio pair per active child, fan-in/out by request id.
- [ ] Crash supervisor with exponential backoff + circuit breaker.

## Phase 6 — Semantic routing (RAG brain)

- [ ] Embed `manifest.embeddingText + tool descriptions` at install time.
- [ ] ONNX Runtime Go bindings + `all-MiniLM-L6-v2` (quantized int8) bundled.
- [ ] Hybrid scorer: BM25 (sqlite FTS5) + cosine, weighted; top-k tool surfacing.
- [ ] Optional `tools/list` filter mode: only expose top-N relevant tools per agent turn.

**Exit:** p50 routing decision < 50ms cold, < 5ms warm.

## Phase 7 — E2E testing

- [x] Playwright E2E harness: smoke (11 tests), quality (108 tests), stress (30 tests).
- [ ] Go E2E harness: spawn mach1, drive as MCP client, iterate registry, call canonical tools.
- [ ] Latency report artifact per release.
- [x] Smoke test in CI on Win + macOS runners.

---

## Phase 8+ — Next upgrades (post-MVP)

- [ ] **Public MCP Registry service** (separate repo): signed manifests, versioning, search API.
- [ ] **WASM sandbox driver** behind same `sandbox.Driver` interface.
- [ ] **Deno isolate driver** for JS/TS MCPs (sub-100ms cold start, permission flags from manifest).
- [ ] Telemetry-free local analytics dashboard (per-tool latency, error rate).
- [ ] Signed-manifest verification + capability prompts (least-privilege UX).

---

## Risk Register

| Risk | Mitigation |
|---|---|
| Docker not installed on user machine | Detect at install; gate Docker-only MCPs; offer node/python path first. |
| MCP spec churn | Pin to dated spec rev; isolate protocol layer behind `internal/router/proto`. |
| ONNX binary size bloats hub | Ship embedder inside `mach1` (Go), not Tauri; lazy-load model. |
| stdio multiplexing deadlocks | Per-child writer goroutine + bounded channel; fuzz test. |

---

## Enterprise-Grade Improvements (Pre-Launch Audit)

> Notes from architecture review — 2026-04-27. Must be addressed before any public or enterprise launch.

### Security
- [ ] **CORS lockdown**: Replace `Access-Control-Allow-Origin: *` with an allowlist of known origins (Railway domain, `tauri://localhost`, `http://localhost:*`).
- [ ] **Rate limiting**: Add per-IP rate limits on `/api/auth/login` and `/api/auth/register` to prevent brute-force attacks.
- [ ] **CSRF protection**: Add CSRF tokens for state-changing requests or switch to `SameSite=Strict` cookies.
- [ ] **Token rotation**: Implement refresh tokens. Current 30-day bearer tokens are long-lived and non-revocable.
- [ ] **Input sanitization**: Validate email format server-side (regex or net/mail parse).
- [ ] **Password policy**: Enforce complexity beyond 8-char minimum.
- [ ] **Audit logging**: Log all auth events (login, register, failed login) to a separate audit table with IP + user-agent.

### Scalability & Reliability
- [ ] **Connection pooling**: pgxpool is configured with defaults. Set explicit `MaxConns`, `MinConns`, `MaxConnLifetime`, and `HealthCheckPeriod`.
- [ ] **Graceful shutdown**: `mcpapiserver` has no signal handler — `SIGTERM` kills in-flight requests.
- [ ] **Health check depth**: `/health` returns static OK. Add DB ping and upstream dependency checks.
- [ ] **Horizontal scaling**: mach1 is single-process stdio. For multi-user cloud scenarios, add an SSE/WebSocket transport adapter.
- [ ] **Session cleanup**: Expired sessions accumulate forever. Add a periodic job or DB trigger to purge `user_sessions WHERE expires_at < now()`.

### Observability
- [ ] **Structured logging**: `mcpapiserver` uses `slog` but has no request-level middleware.
- [ ] **Metrics**: Export Prometheus metrics (request count, latency histogram, active connections, DB pool stats).
- [ ] **Distributed tracing**: Add OpenTelemetry spans for API → DB calls.
- [ ] **Error reporting**: Integrate Sentry or equivalent for unhandled panics.

### Frontend / UX
- [ ] **Real console integration**: Console currently only shows logs when Tauri events are available. Add a WebSocket or SSE endpoint in `mcpapiserver` that streams router logs to the web UI.
- [ ] **Offline mode**: The web UI silently fails when the API is unreachable. Show a clear offline banner and use Tauri local SQLite as fallback.
- [ ] **Error boundaries**: Add `+error.svelte` pages and a `handleError` hook.
- [ ] **Accessibility**: Add ARIA labels, keyboard navigation for modals, and focus trapping.

### Data Integrity
- [ ] **DB migrations**: Schema is applied via `CREATE IF NOT EXISTS` at startup. Adopt a migration tool (golang-migrate, atlas, or goose).
- [ ] **Backup strategy**: No automated backups configured for Railway Postgres.
- [ ] **Data validation**: Marketplace items have no validation on publish.

### CI/CD & Testing
- [ ] **Integration tests**: No test coverage for `mcpapiserver` HTTP handlers or `clouddb` methods.
- [ ] **Go E2E tests**: Zero Go-level E2E tests. Only Playwright browser tests exist.
- [ ] **Release automation**: No versioning strategy. Adopt semver, changelog generation, and signed releases.

### MCP Router Improvements
- [ ] **Per-MCP circuit breaker**: Prevent retrying a permanently broken MCP.
- [ ] **Tool conflict detection**: If two MCPs expose the same tool name, notify the user.
- [ ] **Resource limits**: No memory/CPU limits on child processes in the process driver.
- [ ] **Streaming support**: The router buffers entire responses. Add streaming pass-through for large tool outputs.
