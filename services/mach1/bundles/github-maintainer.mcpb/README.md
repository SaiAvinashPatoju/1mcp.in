# github-maintainer.mcpb

> Autonomous Repo Maintainer — end-to-end repository lifecycle management.

## What This Bundle Does

This bundle packages GitHub tools with procedural knowledge (skills) and automated
workflows (macros) for repository maintenance:

- **Triage open issues** and create fix branches automatically
- **Review PRs** for security issues (hardcoded secrets, CI status) and merge if clean
- **Audit repositories** for common security vulnerabilities

## Skills (LLM-Guided — Architecture 1)

Skills load as MCP prompts. When triggered, they inject procedural knowledge into
your AI client's context so it follows a tested, step-by-step workflow.

| Skill | Prompt Name | Description |
|-------|-------------|-------------|
| Review & Merge | `skill_review_and_merge` | Full PR review: diff analysis → CI check → secret scan → approve/reject → merge |
| Triage & Fix | `skill_triage_and_fix` | Issue prioritization → detail extraction → branch creation |
| Security Audit | `skill_security_audit` | Enumerate attack surface → scan for vulns → generate threat report |

**Usage:** In your AI client, call `list_prompts` then `get_prompt("github-maintainer/skill_review_and_merge", { "pr_number": "42", "owner": "acme", "repo": "api" })`.

## Macros (Go-Automated — Architecture 2)

Macros execute entirely server-side. One tool call triggers a multi-step Go workflow
with retry logic, timeouts, and a full audit trail.

| Macro | Description |
|-------|-------------|
| `1mcp_triage_and_fix` | Lists open unassigned issues, fetches details, creates `fix/issue-{id}` branches |
| `1mcp_review_and_merge` | Fetches PR diff → checks CI → scans for secrets → approves → squash merges → deletes branch |

**Usage:**
```json
{
  "tool": "1mcp_execute",
  "arguments": {
    "macro": "1mcp_review_and_merge",
    "params": {
      "owner": "acme",
      "repo": "api",
      "pr_number": 42
    }
  }
}
```

## Required MCP Servers

This bundle requires the `github` MCP server to be installed and running in your
mach1 router. Install it via:

```bash
mach1ctl install github
mach1ctl env set github GITHUB_TOKEN=ghp_...
```

## Bundle Structure

```
github-maintainer.mcpb/
├── manifest.json
├── skills/
│   ├── review_and_merge.md
│   ├── triage_and_fix.md
│   └── security_audit.md
└── README.md
```
