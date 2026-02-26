---
description: View and configure paivot-graph settings for the current project -- project vault behavior, default scope, proposal expiry, gitignore preferences
allowed-tools: ["Bash", "Read", "Grep", "Glob"]
---

# Vault Settings

Manage paivot-graph configuration for the current project. Settings are stored in `.vault/knowledge/.settings.yaml` and affect how knowledge governance behaves in this project.

## Step 1: Load Current Settings

Check if settings file exists and read it:
```bash
bin/pvg settings
```

If the pvg binary is not available, read the file directly:
```bash
cat .vault/knowledge/.settings.yaml 2>/dev/null || echo "not found"
```

If not found, use these defaults:

```yaml
# paivot-graph project settings
# Managed by /vault-settings

# Whether project vault notes should be tracked in git
# Options: tracked, ignored, ask (prompt on first capture)
project_vault_git: ask

# Default scope for new notes when ambiguous
# Options: system, project
default_scope: system

# Days before a proposal is flagged as stale in /vault-triage
proposal_expiry_days: 30

# Maximum notes to summarize at session start (per subfolder)
# Higher values provide more context but consume more of the context window
session_start_max_notes: 10

# Whether to create .vault/knowledge/ on first /vault-capture
# Options: auto, ask, never
auto_init_project_vault: ask

# Whether session-start detects and outputs the project's tech stack
# Options: true, false
stack_detection: false
```

## Step 2: Present Current Configuration

Show the user the current state:

```
## Vault Settings (<project>)

| Setting                  | Value     | Description                                      |
|--------------------------|-----------|--------------------------------------------------|
| project_vault_git        | ask       | Git tracking for .vault/knowledge/ notes         |
| default_scope            | system    | Default scope when ambiguous                     |
| proposal_expiry_days     | 30        | Days before proposals are flagged stale          |
| session_start_max_notes  | 10        | Max notes summarized per subfolder at start      |
| auto_init_project_vault  | ask       | Create .vault/knowledge/ on first capture        |
| stack_detection          | false     | Detect and output project tech stack at start    |

Settings file: .vault/knowledge/.settings.yaml
```

## Step 3: Ask What to Change

If the user provided arguments (e.g., `/vault-settings project_vault_git=tracked`), apply them directly.

Otherwise, ask the user which setting they want to change and what value to set.

## Step 4: Apply Changes

Use the pvg binary to apply settings changes:
```bash
bin/pvg settings <key>=<value>
```

For example:
```bash
bin/pvg settings project_vault_git=tracked
bin/pvg settings proposal_expiry_days=14
```

**If `project_vault_git` was changed:**
- `tracked`: ensure `.vault/knowledge/` is NOT in `.gitignore`
- `ignored`: add `.vault/knowledge/*.md` to `.gitignore` (keep .settings.yaml tracked)
- `ask`: no gitignore changes (will prompt on first capture)

**If `proposal_expiry_days` was changed:**
- No side effects -- `/vault-triage` reads this at runtime

## Step 5: Report

```
## Vault Settings Updated

Changed:
- <setting>: <old value> -> <new value>

Settings file: .vault/knowledge/.settings.yaml
```

## Reading Settings from Other Commands

Other commands (`/vault-capture`, `/vault-triage`, session-start hook) should check for `.vault/knowledge/.settings.yaml` and use its values. If the file doesn't exist, use defaults.

To read a setting programmatically:
```bash
bin/pvg settings <key>
```
