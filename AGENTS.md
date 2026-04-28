# AGENTS.md — 1mcp.in

> This file is the single source of truth for any AI agent or developer working on this codebase.
> Read this entirely before writing any code, making any architecture decision, or suggesting any change.

---

Application Name : 1mcp.in
centralMcp Name : mach1

## What This Product Is

**1mcp.in** is a local-first MCP (Model Context Protocol) hub and router for teams and individuals.

It solves two distinct problems:

**For individuals (OSS free tier):**
Managing multiple MCP servers is painful — separate installs, separate credentials, separate configs per AI client. 1mcp replaces all of that with a single router process (`mach1`) and a desktop Hub UI. One install. Every AI client (Codex, Cursor, VS Code, Codex Desktop) connects to one endpoint.

**For teams (Team Pro):**
Every developer on a team configures MCPs individually — 10 developers means 10 copies of the same setup, 10x credential sprawl, zero visibility into what AI is doing. 1mcp Team Pro gives an admin one control plane: install once, credential once, every member gets it automatically. Add agents and shared workflows on top. The team's AI clients become a unified, governed agentic layer.

---

## What This Product Is NOT

- Not a replacement for Codex, Cursor, or any AI client. It is infrastructure beneath them.
- Not an LLM. It does not generate responses.
- Not a workflow builder (like n8n or Zapier). Workflows are composed from agents, not visual flowcharts.
- Not enterprise-first. The GTM target is SMB teams of 5–50 people.

---

## North Star

> Every member of a team should be able to use every AI tool the team has approved — with zero individual setup, full admin visibility, and pre-built agents for their job function — from inside the AI client they already use.

---

## Repository Structure

```
1mcp.in/                          ← PUBLIC (open source, MIT)
├── services/
│   ├── mach1/              ← Go router, supervisor, sandbox, CLI
│   │   ├── cmd/
│   │   │   ├── mach1/      ← Main router binary
│   │   │   ├── mcpapiserver/     ← Local API (auth bridge, marketplace)
│   │   │   └── mach1ctl/        ← CLI (install, list, env)
│   │   └── internal/
│   │       ├── router/           ← Request routing, tool namespacing
│   │       ├── supervisor/       ← Process lifecycle, semantic ranking
│   │       ├── sandbox/          ← Process isolation, stdio tunneling
│   │       ├── transport/        ← stdio (current), HTTP streamable (add)
│   │       └── clouddb/          ← PostgreSQL schema (marketplace, auth)
│   └── web-ui/                   ← SvelteKit + Tauri desktop app
│       ├── src/                  ← Svelte pages, components, stores
│       ├── src-tauri/            ← Rust backend (auto-update, sqlite)
│       └── vite.config.ts
├── packages/
│   ├── mcp-manifest/             ← JSON Schema for MCP config
│   └── registry-index/           ← Curated MCP catalog (18 servers)
├── scripts/
│   ├── build.ps1                 ← Windows build (existing)
│   └── build.sh                  ← macOS/Linux build (ADD THIS)
└── .github/workflows/
    ├── ci.yml
    └── release.yml

1mcp-cloud/                       ← PRIVATE (closed source, Team Pro backend)
├── services/
│   ├── team-api/                 ← Go — workspace, members, vault, agents
│   │   ├── cmd/teamd/
│   │   └── internal/
│   │       ├── workspace/        ← Org management
│   │       ├── members/          ← Invites, roles, permissions
│   │       ├── vault/            ← Encrypted credential store
│   │       ├── agents/           ← Agent library, custom builder
│   │       ├── workflows/        ← Workflow templates
│   │       ├── activitylog/      ← Immutable audit trail
│   │       └── sync/             ← Config push to local routers
│   └── team-ui/                  ← SvelteKit web-only (no Tauri)
│       └── src/
│           ├── workspace/        ← Admin dashboard
│           ├── members/          ← Seat management
│           ├── agents/           ← Agent library UI
│           ├── vault/            ← Credential management
│           └── activity/         ← Feed + analytics
```

Railway project: 1mcp
├── Service: mcpapiserver    ← OSS cloud API (marketplace, auth)
├── Service: teamd           ← Team Pro API (workspace, vault, agents)  [ADD]
├── Service: PostgreSQL      ← Shared DB (two schemas: public + team)
└── Service: Redis           ← Job queue + config pub/sub              [ADD]
---

## Tech Stack

| Layer | Tech | Notes |
|---|---|---|
| Router | Go 1.22+ | `mach1` binary |
| CLI | Go | `mach1ctl` binary |
| Desktop UI | SvelteKit + Tauri (Rust) | Local Hub UI |
| Team API | Go | `teamd` binary, private repo |
| Team UI | SvelteKit (web) | No Tauri, browser only |
| Local DB | SQLite | Router registry, session state |
| Cloud DB | PostgreSQL (Railway) | Marketplace, auth, team data |
| Credential Vault | AES-256-GCM | Keys stored encrypted server-side |
| Transport (current) | stdio | MCP 2024-11-05 spec |
| Transport (add) | Streamable HTTP | MCP 2025-03-26 spec |
| Auth | JWT (current) → OAuth 2.1 (add) | Per MCP 2025-06-18 spec |
| Telemetry | OpenTelemetry (add) | Spans per tool call |
| CI/CD | GitHub Actions | Build + release on vX.Y.Z tag |
| Hosting | Railway | mcpapiserver + teamd |

---

## Current Performance Baselines (do not regress)

| Metric | Value |
|---|---|
| Cold initialize (npx MCPs) | ~741ms |
| Warm tool call | ~1–2ms |
| Router RAM footprint | <30MB |
| Supported AI clients | 7+ |
| Marketplace MCPs | 18 |

---

## Protocol Compliance Targets

The codebase must track the MCP specification. Current gaps and required upgrades:

### Streamable HTTP Transport (MCP 2025-03-26)
**Status:** Not implemented. stdio only.
**Why it matters:** stdio requires the MCP server to run on the same machine as the client. No remote deployment. No shared server across multiple clients. Kills team use case.
**What to build:**
- Add `--transport http` flag to `mach1`
- Implement single-endpoint Streamable HTTP (replaces dual-channel HTTP+SSE)
- Support version negotiation via request headers (legacy clients fall back to stdio mode)
- Target file: `services/mach1/internal/transport/`

### OAuth 2.1 (MCP 2025-06-18)
**Status:** Not implemented. Custom JWT session auth only.
**Why it matters:** Required for any remote MCP deployment. Without it, remote tool calls are unauthenticated.
**What to build:**
- Implement OAuth 2.1 with PKCE + HTTPS on `mcpapiserver`
- Add `mcp:read` and `mcp:write` scopes
- Implement Resource Indicators (RFC 8707) — prevents token leakage to rogue servers
- Add Step-Up Authorization: server returns 403 + required scopes if token insufficient
- Support dynamic client registration
- Target file: `services/mach1/cmd/mcpapiserver/`

### Tool Annotations (MCP 2025-03-26)
**Status:** Not implemented.
**Why it matters:** Lets the router enforce permissions intelligently. A `read-only` tool cannot be blocked by admin; a `destructive` tool requires approval gate.
**What to build:**
- Add `readOnly`, `destructive`, `idempotent` flags to tool manifest schema
- Router reads annotations and applies policy from admin config
- Target file: `packages/mcp-manifest/`

---

## Security Requirements

These are not optional. Ship before any public OSS release.

### Tool Definition Hash Registry
**Problem:** MCP tools can change their descriptions after install (rug pull attack). If a tool's `description` or `inputSchema` changes post-install, the router must detect it, surface an alert in Hub UI, and require admin re-approval before the tool is usable again.
**Implementation:**
- On install, SHA256 hash each tool's manifest and store in local SQLite
- On every `tools/list` response from an MCP server, recompute hash and compare
- If mismatch: mark tool as `PENDING_REVIEW`, surface alert in Hub UI, block tool calls until re-approved

### Supply Chain Verification
**Problem:** `mach1ctl install github` runs arbitrary npx packages. No integrity check.
**Implementation:**
- Add `sha256` and `signature` fields to every entry in `packages/registry-index/index.json`
- Before executing any install, verify hash matches
- Log verification status in install output

### PII Scrubbing
**Problem:** Live console in Hub UI streams raw stdio. Tool responses can contain API keys, passwords, personal data.
**Implementation:**
- Add a scrubbing middleware in the router between MCP response and log/UI output
- Redact patterns: email, phone, credit card, AWS key format, GitHub token format, JWT pattern
- Never log raw tool call results to persistent storage (only scrubbed versions)

### Process Isolation
**Problem:** MCP sub-processes run with the same filesystem access as the router. A malicious MCP can read `~/.ssh/`.
**Implementation (short-term):** Apply Go `syscall` restrictions to sub-processes — restrict filesystem paths accessible to each MCP process.
**Implementation (medium-term):** Wrap each MCP in a Docker container via Tauri backend. CPU cap: 1 core. Memory cap: 512MB. No host filesystem mount by default.

---

## OSS GTM — What Needs to Change Before Public Launch

The current codebase is not ready for OSS distribution. These are blockers:

### 1. macOS Build (P0 BLOCKER)
`build.ps1` is PowerShell. The majority of developers are on macOS.

**Create `scripts/build.sh`:**
```bash
#!/usr/bin/env bash
set -euo pipefail

echo "Building mach1..."
cd services/mach1
go build -o ../../bin/mach1 ./cmd/mach1
go build -o ../../bin/mcpapiserver ./cmd/mcpapiserver
go build -o ../../bin/mach1ctl ./cmd/mach1ctl
cd ../..

echo "Building Tauri app..."
cd services/web-ui
npm install
npm run tauri build
cd ../..

echo "Done. Binaries in ./bin/"
```

**Add to GitHub Actions CI matrix:**
```yaml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]
```

**Add Homebrew tap:**
```ruby
# Formula/1mcp.rb
class Onemcp < Formula
  desc "Local-first MCP hub and router"
  homepage "https://1mcp.in"
  url "https://github.com/SaiAvinashPatoju/1mcp.in/releases/download/vX.Y.Z/mach1-darwin-amd64.tar.gz"
  ...
end
```

### 2. One-Command Install
Nobody reads multi-terminal setup instructions. Ship:

```bash
# macOS/Linux
curl -fsSL https://install.1mcp.in | sh

# Windows
irm https://install.1mcp.in/windows | iex
```

The install script must:
- Detect OS and arch
- Download the correct binary
- Add `mach1ctl` to PATH
- Print: "Run `mach1ctl start` to launch 1mcp"

### 3. README Rewrite for OSS Distribution
The current README is a developer reference. The OSS README needs:
- One-line value prop (first line, no jargon)
- One GIF showing Hub UI with MCPs running
- One-command install block
- Supported clients list with badges
- Link to Team Pro waitlist
- Star request

### 4. Expand Marketplace from 18 → 50+ MCPs
**Community submission flow:**
- Create `CONTRIBUTING_MCP.md` with manifest template
- Any contributor opens a PR adding one JSON entry to `packages/registry-index/index.json`
- CI validates schema + required fields
- Maintainer reviews, signs (adds SHA256), merges
- Auto-syncs on next release build

### 5. Cross-Platform CI Release Pipeline
```yaml
# .github/workflows/release.yml additions
- name: Build macOS arm64
  runs-on: macos-latest
  steps:
    - run: bash scripts/build.sh
    - uses: actions/upload-artifact@v3

- name: Build Linux amd64
  runs-on: ubuntu-latest
  ...
```

---

## Team Pro — Feature Specification

### Workspace
- One workspace per organization
- Admin creates workspace via team-ui (web)
- Workspace has: `name`, `slug`, `plan`, `seat_count`, `created_at`
- Members join via invite link (email-gated or open link with expiry)

### Member Roles
| Role | Can do |
|---|---|
| Admin | Everything |
| Member | Use approved MCPs + agents. Cannot install new MCPs or create agents. |
| Viewer | Read-only dashboard access. Cannot run tools. |

### Shared MCP Config Sync
- Admin installs MCPs in team workspace via team-ui
- `teamd` maintains a workspace MCP manifest
- Each member's local `mach1` polls team-api every 5 minutes for config changes (or on router start)
- On change: router hot-reloads MCP config without restart
- Member never touches `mach1ctl install` or `mach1ctl env set`

### Credential Vault
- Admin sets secrets in team-ui: `GITHUB_TOKEN`, `SLACK_BOT_TOKEN`, etc.
- Stored encrypted (AES-256-GCM) in `teamd` database
- Member's router fetches secrets at runtime via authenticated API call
- Secrets are injected as env vars into MCP sub-processes
- Members cannot read raw secret values — only use them via MCP tool calls
- Secret rotation: admin updates value, all members get it on next sync

### Agent Library

#### Pre-built Agents (ship at launch)

**Engineering agents:**

`standup-agent`
- Trigger: manual or scheduled (e.g., 9:00 AM daily)
- Tools used: `github__list_commits`, `linear__list_issues`, `slack__post_message`
- Behavior: pulls commits from last 24hrs for calling user, pulls their assigned open tickets, formats standup, optionally posts to Slack channel
- Output: markdown standup report

`pr-review-agent`
- Trigger: on PR open (via webhook) or manual with PR URL
- Tools used: `github__get_pull_request`, `github__list_files`, `github__create_review`
- Behavior: reads diff, checks against team coding standards (from workspace config), posts structured code review comment
- Output: GitHub PR review comment

`deploy-notifier-agent`
- Trigger: on merge to main (via webhook)
- Tools used: `github__compare_commits`, `linear__get_issue`, `slack__post_message`
- Behavior: reads commit list, maps commit messages to Linear ticket IDs, formats release notes, posts to Slack
- Output: Slack message with release summary

**Sales agents:**

`lead-research-agent`
- Trigger: manual with company name input
- Tools used: `brave_search`, `fetch`, `memory`
- Behavior: searches company website, LinkedIn, recent news; extracts: company size, funding, tech stack, key contacts, recent announcements; stores in memory for follow-up
- Output: structured one-page company brief (markdown)

`outreach-draft-agent`
- Trigger: manual, takes lead brief as input
- Tools used: `memory`, `filesystem` (reads past outreach templates)
- Behavior: reads stored lead brief, reads team outreach templates, generates personalized first-touch email
- Output: email draft

`crm-update-agent`
- Trigger: manual with meeting transcript or notes as input
- Tools used: `notion` or relevant CRM MCP
- Behavior: extracts action items, next steps, contact info from notes; logs structured entry to CRM/Notion
- Output: CRM record created/updated

**Operations/Support agents:**

`ticket-triage-agent`
- Trigger: scheduled (every 30 min) or webhook on new email
- Tools used: `fetch` (email/support inbox), `notion` or `linear__create_issue`
- Behavior: reads new support messages, classifies by urgency + product area, drafts response, creates ticket if new issue, escalates if critical
- Output: drafted responses + created tickets

`invoice-extract-agent`
- Trigger: manual with PDF input or folder watch
- Tools used: `filesystem`, `sequential_thinking`
- Behavior: reads invoice PDF, extracts vendor, amount, date, line items, payment terms, appends to tracking spreadsheet
- Output: structured row in Google Sheets / CSV

**Marketing agents:**

`content-brief-agent`
- Trigger: manual with topic input
- Tools used: `brave_search`, `fetch`, `sequential_thinking`
- Behavior: searches top-ranking content for topic, identifies content gaps, pulls search trend data, generates structured content brief
- Output: content brief doc (markdown)

#### Custom Agent Builder
Admin can create custom agents in team-ui:
- Select MCPs from workspace library as available tools
- Write system prompt (with variable slots: `{{user_name}}`, `{{date}}`, `{{input}}`)
- Set trigger: manual / cron schedule / webhook URL
- Set output: return to caller / post to Slack / write to Notion / create GitHub issue
- Save as named agent — immediately available to all workspace members

#### Agent Execution
- Agents run via `teamd` (server-side orchestration)
- Each agent call creates an `agent_run` record: `agent_id`, `member_id`, `trigger`, `status`, `started_at`, `completed_at`, `output_preview`
- Output is returned to the member's AI client session that invoked it (via MCP tool result)
- Failed runs are retried once automatically; second failure creates an alert in activity feed

### Activity Feed
Every tool call and agent run is logged:
```
timestamp | member | tool_or_agent | mcp_server | status | duration_ms
```
- Immutable (insert-only table)
- Searchable by member, MCP, agent, date range, status
- Exportable as CSV

### Usage Dashboard
- Per-member: tool call count, agent runs, most-used MCPs, weekly trend
- Per-agent: total runs, success rate, avg duration
- Per-MCP: call volume, error rate, most-called tools
- Workspace-wide: total tool calls this month vs last month, active members, top agents

### Approval Gates
Admin can mark specific tools as requiring human approval:
- Configured per tool name (e.g., `filesystem__write_file`, `github__merge_pull_request`)
- When member triggers a tool call that hits a gated tool:
  - Tool call is paused (router holds the request)
  - Admin sees pending approval in team-ui with: member, tool, arguments, timestamp
  - Admin approves → router resumes call → result returned to member
  - Admin denies → router returns error to member with denial reason
  - Auto-deny after 30 minutes if no admin action

### Budget Controls
- Admin sets monthly token spend limit per workspace or per member
- `teamd` tracks LLM API costs via usage metadata (if model returns token counts)
- Alert at 80% of limit (Slack + email)
- Hard stop at 100% — router rejects new agent runs, returns budget error to member
- Admin can adjust limit or reset mid-month

---

## User Flows

These flows describe what real humans do with this product. Every feature decision must map back to one of these flows.

---

### Flow 1: Individual Developer — First Install

**Who:** Solo developer using Codex or Cursor. Currently has GitHub MCP installed manually.

**Goal:** Replace manual MCP management with 1mcp.

```
1. Developer visits 1mcp.in
2. Runs: curl -fsSL https://install.1mcp.in | sh
3. Runs: mach1ctl start
   → mach1 starts, Hub UI opens at localhost:5173
4. Opens Hub UI → Marketplace tab
5. Clicks "Install" on GitHub MCP
6. Hub UI prompts: "Enter GITHUB_TOKEN"
7. Developer pastes token → clicks Save
8. Hub UI shows GitHub MCP: Running ✓
9. Developer opens their AI client config (Codex Desktop / Cursor)
10. Hub UI → Clients tab → clicks "Setup 1mcp" on their client
    → auto-patches client config file
11. Developer restarts AI client
12. In their AI session, they can now call GitHub tools
    (e.g., "list my open PRs" → Codex uses github__list_pull_requests)
13. Developer installs 3 more MCPs from marketplace — same flow
14. All 4 MCPs now available in every AI client via single router
```

**What the agent needs to understand:** The user's goal is zero-friction MCP access in their existing AI client. The Hub UI and router are invisible infrastructure. Success = user forgets 1mcp exists and just uses their AI client normally.

---

### Flow 2: Team Admin — Onboarding a Team

**Who:** Engineering lead at a 12-person startup. Team uses Codex. Tired of every developer asking "what's the GitHub token?"

**Goal:** Set up shared MCPs for the whole team.

```
1. Admin visits 1mcp.in → clicks "Start Team Pro"
2. Signs up → creates workspace: "Acme Engineering"
3. Connects payment (₹1,999/seat/month × 12 seats)
4. Admin opens Workspace Settings → MCPs tab
5. Admin installs: GitHub, Linear, Slack, Notion
6. For each MCP, admin enters team credentials (stored in vault)
7. Admin opens Members tab → clicks "Invite Members"
8. Pastes 11 email addresses → sends invites
9. Each member receives email → clicks link → downloads 1mcp
10. Member runs: mach1ctl join acme-engineering --token <invite_token>
    → local router authenticates with teamd
    → fetches workspace MCP config
    → fetches credentials from vault
    → starts all 4 MCPs automatically
11. Member opens Codex → GitHub/Linear/Slack/Notion tools are available
    (member configured nothing themselves)
12. Admin adds Postgres MCP next week
    → within 5 minutes, all 12 members automatically have it
```

**What the agent needs to understand:** Admin's job is configuration + governance. Member's job is to do their actual work. The sync mechanism is the core value — admin sets it once, everyone gets it.

---

### Flow 3: Team Member — Daily Use (Engineering)

**Who:** Backend developer on the Acme team. Has 1mcp installed and joined workspace.

**Goal:** Use AI to do real work faster.

```
Morning:
1. Developer opens Codex
2. Types: "What did I work on yesterday and what should I focus on today?"
   → Codex uses github__list_commits (filtered to user, last 24hrs)
   → Codex uses linear__list_issues (assigned to user, in progress)
   → Codex returns: standup summary + prioritized today's tasks

During work:
3. Developer is reviewing a PR
4. Types: "Review this PR for code quality and security issues: <PR URL>"
   → Codex uses github__get_pull_request
   → Codex uses github__list_files
   → Codex returns: structured review with specific line comments
   → Developer asks: "Post this review as a GitHub comment"
   → Codex uses github__create_review (this tool has approval gate set by admin)
   → Tool call pauses → admin sees request in team dashboard
   → Admin approves
   → Comment posted to GitHub

End of day:
5. Developer merges a PR
   → Deploy Notifier Agent triggers automatically (webhook)
   → github__compare_commits pulls changed files
   → linear__get_issue maps commit messages to ticket IDs
   → slack__post_message posts release notes to #deploys channel
   → Developer did not initiate this — it happened automatically
```

**What the agent needs to understand:** Members don't interact with 1mcp directly. They interact with their AI client (Codex, Cursor). 1mcp is the invisible layer that makes tools available and routes calls. The AI client is the interface. 1mcp is the infrastructure.

---

### Flow 4: Team Member — Daily Use (Sales)

**Who:** SDR at a B2B SaaS company. Team has 1mcp with Brave Search, Fetch, HubSpot, Memory MCPs installed.

**Goal:** Research leads faster, write better outreach, update CRM without switching tabs.

```
9:00 AM — Lead Research:
1. SDR gets list of 5 companies to research
2. Opens Codex or Codex Desktop
3. Types: "Research Acme Corp for an outreach call — I need their size, 
   recent news, tech stack, and likely pain points"
4. Lead Research Agent runs:
   → brave_search: "Acme Corp funding news 2025"
   → fetch: acmecorp.com/about, LinkedIn company page
   → sequential_thinking: synthesizes data
   → memory__store: saves brief under "acme-corp"
5. Returns: structured one-page brief in 15 seconds
   (previously: 25 minutes of manual research)

10:00 AM — Outreach:
6. Types: "Draft a first email to the CTO of Acme Corp based on my research"
   → memory__get: fetches acme-corp brief
   → Returns: personalized cold email draft referencing their recent Series B,
     tech stack (Node.js/AWS), and expansion into APAC
7. SDR edits lightly → sends

3:00 PM — Post-call CRM Update:
8. SDR pastes call notes into Codex
9. Types: "Log this call to HubSpot and create follow-up tasks"
   → crm-update-agent extracts: contact info, pain points, next steps, 
     decision timeline, budget signals
   → hubspot__create_note, hubspot__create_task run
   → CRM updated
   (previously: 15 minutes of manual CRM entry after every call)
```

**What the agent needs to understand:** The SDR's entire workflow — research, outreach, CRM — happens inside one AI chat session. They never open a browser to research, never copy-paste between tools, never manually update CRM. The agent orchestrates tool calls that the SDR doesn't see. The SDR just talks to their AI client.

---

### Flow 5: Admin — Monitoring and Control

**Who:** Engineering lead who set up the team workspace. Wants to know what AI is doing.

**Goal:** Understand team AI usage. Prevent misuse. Control costs.

```
Weekly review:
1. Admin opens team-ui dashboard
2. Activity Feed shows: 847 tool calls this week across 12 members
3. Top tools: github__list_pull_requests (312), linear__list_issues (198), 
   slack__post_message (89)
4. Admin notices: one member made 45 calls to filesystem__write_file
5. Admin clicks member name → sees full activity log for that member
6. Determines: member is using an agent that writes test files — expected, fine

Approval gate review:
7. Admin sees 3 pending approvals: github__merge_pull_request
8. Reviews each: sees who requested it, which PR, what the diff summary is
9. Approves 2, denies 1 (wrong branch)

Budget review:
10. Workspace at 73% of monthly token budget
11. Admin adjusts: raises limit by 20% (team is productive, worth it)

New MCP request:
12. Developer Slack message: "Can we get the Postgres MCP?"
13. Admin opens team-ui → Marketplace → installs Postgres MCP
14. Vault: adds DATABASE_URL for dev/staging environments
15. Sets approval gate on: postgres__execute_query (safety — require review for writes)
16. All 12 members have Postgres MCP available within 5 minutes
```

**What the agent needs to understand:** The admin's job is governance, not development. The team-ui is their control plane. They should never need to SSH into servers or edit config files. Every action — install MCP, add credential, invite member, set approval gate — must be a UI click.

---

### Flow 6: Developer Contributing an MCP to the Marketplace

**Who:** Open source contributor. Built an MCP server for Notion (better than existing one). Wants to add it.

**Goal:** Get their MCP listed in the 1mcp marketplace.

```
1. Contributor forks 1mcp.in on GitHub
2. Reads CONTRIBUTING_MCP.md
3. Creates: packages/registry-index/entries/notion-enhanced.json
   {
     "id": "notion-enhanced",
     "name": "Notion Enhanced",
     "description": "Full Notion API coverage including databases, pages, blocks",
     "runtime": "node",
     "install": "npx notion-enhanced-mcp",
     "env": ["NOTION_API_KEY"],
     "verification": "community",
     "homepage": "https://github.com/contributor/notion-enhanced-mcp",
     "license": "MIT"
   }
4. Opens PR with the JSON file
5. CI validates: schema check, required fields, no duplicate ID
6. Maintainer reviews: tests the MCP manually, checks for malicious patterns
7. Maintainer approves → adds SHA256 hash to entry → merges
8. Next release build auto-syncs to marketplace
9. All 1mcp users see "Notion Enhanced" in marketplace on next app update
```

**What the agent needs to understand:** The marketplace grows through community PRs. Never auto-approve marketplace entries. Always require human maintainer review and signature before an MCP is listed. The SHA256 hash is computed by the maintainer, not the contributor.

---

## Build Priority Order

Build in this exact sequence. Do not skip ahead.

### Phase 1 — OSS Launch Readiness (build first, everything else depends on this)
- [ ] `scripts/build.sh` — macOS/Linux build script
- [ ] GitHub Actions CI matrix (ubuntu, macos, windows)
- [ ] One-command curl installer (`install.1mcp.in`)
- [ ] Homebrew tap
- [ ] Tool definition hash registry (security — ship before OSS launch)
- [ ] Supply chain SHA256 verification in marketplace
- [ ] PII scrubbing middleware in router
- [ ] README rewrite for OSS distribution

### Phase 2 — Team Pro Foundation
- [ ] `1mcp-cloud` private repo setup
- [ ] Workspace + member invite + role model (`team-api`)
- [ ] Config sync endpoint (workspace MCP manifest → local router)
- [ ] Credential vault (AES-256-GCM)
- [ ] Team UI: workspace dashboard, member management, MCP management, vault UI

### Phase 3 — Agent Library
- [ ] Agent execution engine in `teamd`
- [ ] `standup-agent` (engineering, high daily use)
- [ ] `lead-research-agent` (sales, high daily use)
- [ ] `ticket-triage-agent` (operations)
- [ ] Activity feed (agent runs + tool calls)

### Phase 4 — Governance + Productivity Layer
- [ ] Approval gates (router-side hold + teamd approval flow)
- [ ] Budget controls + alerts
- [ ] Usage dashboard (per-member, per-agent, per-MCP)
- [ ] Custom agent builder (admin UI)
- [ ] Webhook triggers for agents

### Phase 5 — Protocol Compliance (required before enterprise tier)
- [ ] Streamable HTTP transport in `mach1`
- [ ] OAuth 2.1 + PKCE on `mcpapiserver`
- [ ] Tool annotations (readOnly, destructive, idempotent)
- [ ] Process isolation (Docker or seccomp)
- [ ] OpenTelemetry spans + `/metrics` Prometheus endpoint

---

## Pricing

| Plan | Seats | MCPs | Agents | Vault | Dashboard | Approval Gates | Price |
|---|---|---|---|---|---|---|---|
| Free (OSS) | 1 | Unlimited | None | None | None | None | ₹0 |
| Team Pro | 5–50 | Unlimited | Full library + custom | ✅ | ✅ | ✅ | ₹1,999/seat/month |
| Enterprise | 50+ | Unlimited | All + dedicated | ✅ SOC2 | ✅ Advanced | ✅ + audit log | Custom |

Team Pro billing: seat-based. Every member who joins the workspace consumes one seat.
Admin does not consume a seat (admin role is always free).

---

## What NOT to Build (Explicit Exclusions)

- Do not build a visual workflow builder (drag-and-drop). That is n8n's problem.
- Do not build an LLM. Route to existing ones.
- Do not build a Chrome extension. Focus on desktop.
- Do not build enterprise SSO (Okta, Entra) until Team Pro is profitable.
- Do not build mobile apps.
- Do not open-source the team-api or team-ui. Ever.
- Do not auto-approve marketplace MCP submissions. Always require human review.
- Do not store raw secret values in logs, UI, or tool call results (only redacted).

---

## Competitive Positioning

| Competitor | What they do | Why they don't win our users |
|---|---|---|
| n8n | Visual workflow builder | Requires setup per workflow. Not in the user's AI client. |
| Zapier Agents | 8000+ app automation | Browser-based. Not in Codex or Cursor. Expensive at volume. |
| Composio | MCP gateway, large catalog | Cloud-only. No local router. No desktop-first story. |
| MintMCP | Enterprise MCP gateway | US-priced, enterprise-focused, no team agent library. |
| Microsoft Copilot | M365 AI assistant | Locked to Microsoft stack. $30/user/month. Not for dev tools. |
| CrewAI AMP | Multi-agent framework | Developer framework, not a product. Requires Python expertise. |

**1mcp wins because:** it lives inside the tools developers already use (Codex, Cursor), runs locally (low latency, offline-capable), and bridges MCP management + team agents in one product — at a price SMB teams in India can actually pay.

---

## Agent Working Instructions

If you are an AI agent working on this codebase:

1. **Never break the warm tool call latency.** 1–2ms is the baseline. Measure before and after any router change.
2. **Never store raw credentials in logs, console output, or UI.** Run the PII scrubber on all tool call results before they leave the router.
3. **Never auto-approve marketplace entries.** The `verified` flag in registry-index must only be set by a maintainer, never by automated tooling.
4. **Always check tool definition hashes.** If a tool's manifest changes post-install, block it and surface an alert. Do not silently continue.
5. **Closed source stays closed.** Never commit team-api or team-ui code to the public 1mcp.in repository.
6. **Build order matters.** Do not start Phase 2 until Phase 1 is complete and the OSS version is published. Traction precedes revenue features.
7. **Members configure nothing.** Every feature targeting team members must work with zero config on their side. If a member has to run a command or edit a file, the feature is not done.
8. **Admin clicks, not commands.** Every admin action must be achievable through team-ui. No SSH, no config file edits, no CLI commands for admins.
9. **macOS first.** When in doubt about platform support, macOS is the primary target. Windows is secondary. Linux is tertiary.
10. **The AI client is the interface.** 1mcp is infrastructure. Any feature that requires the user to open 1mcp Hub UI to do their actual work is a design failure.

### OpenCode Behavioral Rules

11. **Subagent delegation for Go work.** When asked to add a new backend feature, ALWAYS delegate to the `@go-architect` subagent defined in `opencode.json`. It has direct access to Go tooling, Postgres MCP, and GitHub MCP.
12. **Subagent delegation for UI work.** When asked to update the Local Hub UI or Team UI, ALWAYS delegate to the `@ui-builder` subagent. It handles SvelteKit, TypeScript, and Tauri (Rust) tasks.
13. **Plan before Build.** Before executing any bash command to change architecture, map out the change to ensure it doesn't violate process isolation (Docker/seccomp) or OpenTelemetry requirements.
14. **Subagent scope is read-only.** When invoking `@go-architect` or `@ui-builder`, their file writes are limited to their domain. Cross-boundary changes (e.g., a UI subagent modifying Go router code) must be reviewed before write.
