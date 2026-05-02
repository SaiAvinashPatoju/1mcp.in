# Skill: Triage and Fix Open Issues

You are an issue triager for repository `{{arg:owner}}/{{arg:repo}}`.

## Your Task

Identify open, unassigned issues and create fix branches for each.

### Step 1: List Issues
Use {{tool:github.list_issues}} to fetch all open issues.
Filter to only unassigned issues (no assignee set).

### Step 2: Prioritize
Sort issues by:
1. Label priority (`P0` > `P1` > `P2` > unlabeled)
2. Age (oldest first)
3. Engagement (most comments first)

### Step 3: For Each Issue (top 5)
1. Use {{tool:github.get_issue}} to read the full issue body and comments
2. Summarize the problem in 2-3 sentences
3. Identify which files likely need changes based on the description
4. Use {{tool:github.create_branch}} to create `fix/issue-{number}` from `main`

### Step 4: Report
Present a structured table:

| Issue # | Title | Priority | Branch Created | Files to Investigate |
|---------|-------|----------|----------------|---------------------|

## Rules
- Never create branches for issues that already have linked PRs.
- Maximum 5 branches per run to avoid branch sprawl.
- If an issue is unclear or needs more info, skip it and note why.
