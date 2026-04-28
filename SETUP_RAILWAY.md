# Railway Deployment Setup

This guide walks through setting up Railway for automated cloud deployment via GitHub Actions.

## Prerequisites
- GitHub repository access (Settings â†’ Secrets)
- Railway CLI installed: `npm install -g @railway/cli`
- Railway account linked locally: `railway login`

## Step 1: Link Your Railway Project

```bash
cd 1mcp.in
railway link
# Follow prompts to select your project (or create new)
# Output: .railway/ directory with project ID
```

Confirm:
```bash
railway status
# Output: Project: 1mcp.in, Environment: production, Service: mcpapiserver
```

## Step 2: Create a Railway API Token

Generate a **personal API token** for GitHub Actions:

```bash
# Option A: Via Railway CLI
railway token

# Option B: Via Railway Dashboard
# 1. Go to https://railway.app/account/tokens
# 2. Click "Create Token"
# 3. Copy the token (you'll only see it once!)
```

The token looks like: `rw_xxx_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

## Step 3: Add Token to GitHub Secrets

1. Go to your GitHub repo: `github.com/SaiAvinashPatoju/1mcp.in`
2. Settings â†’ Secrets and Variables â†’ Actions
3. Click "New repository secret"
4. **Name:** `RAILWAY_TOKEN`
5. **Value:** Paste the token from Step 2
6. Click "Add secret"

## Step 4: (Optional) Database Configuration

If your PostgreSQL isn't yet provisioned:

```bash
railway add
# Select PostgreSQL
# Follow prompts to add a new service

# Confirm variables are set:
railway env
```

The environment variables (`DATABASE_URL`, `DATABASE_PUBLIC_URL`) are auto-set by Railway.

## Step 5: Verify CI/CD

Push a commit or tag:
```bash
git push origin main
# or
git tag v0.2.2
git push origin main --tags
```

Then check:
1. **CI workflow**: https://github.com/SaiAvinashPatoju/1mcp.in/actions â†’ Workflows â†’ CI
2. **Release workflow** (on tag): â†’ Release
3. **Deploy logs**: Railway Dashboard â†’ mcpapiserver â†’ Deployments

Expected flow:
- `ci.yml` runs: Go tests, version check, Tauri builds
- `release.yml` runs (on tag): Go test â†’ Deploy to Railway â†’ Publish GitHub Release

## Troubleshooting

### `Invalid RAILWAY_TOKEN`
- Token expired or doesn't have permissions
- **Fix**: Regenerate token via `railway token` and update GitHub secret

### Deploy fails: `Database connection refused`
- DATABASE_PUBLIC_URL env var not set in Railway environment
- **Fix**: `railway env set DATABASE_PUBLIC_URL $(...)`
  Or ensure PostgreSQL service is running: `railway status`

### CI passes but release fails
- Check `.github/workflows/release.yml` for correct service name
- Verify Railway CLI version: `railway --version` (need v4.0+)
- Check logs: Railway Dashboard â†’ Service â†’ Deployments

### "Service not found" in GitHub Actions
- `.railway/config.json` might not be committed or is stale
- **Fix**: Recommit: `git add .railway/config.json && git commit -m "chore: railway config"`

---

## Local Testing (Before Pushing)

Test the deploy locally without GitHub:

```bash
cd services/mach1
railway run bash -c 'go run ./cmd/mcpapiserver'
# or
railway up --service mcpapiserver --detach
```

Check status:
```bash
railway status
railway logs mcpapiserver
```

---

## Notes

- **Token Security**: Never commit `.railway/` or tokens to git. GitHub secrets are encrypted.
- **Production Env**: Railway separates environments; you can have staging/prod branches deploy differently.
- **Rollback**: Old releases stay on GitHub; use `railway env` to pick a prior version.

For more: https://docs.railway.app/reference/cli
