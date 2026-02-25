---
description: View and configure paivot-graph settings for the current project -- project vault behavior, default scope, proposal expiry, gitignore preferences
allowed-tools: ["Bash", "Read", "Write", "Edit", "Glob", "Grep"]
---

# Vault Settings

Manage paivot-graph configuration for the current project. Settings are stored in `.vault/knowledge/.settings.yaml` and affect how knowledge governance behaves in this project.

## Step 1: Load Current Settings

Check if settings file exists:
```bash
test -f .vault/knowledge/.settings.yaml && echo "exists" || echo "not found"
```

If it exists, read it:
```bash
cat .vault/knowledge/.settings.yaml
```

If not, use these defaults:

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

Settings file: .vault/knowledge/.settings.yaml
```

## Step 3: Ask What to Change

If the user provided arguments (e.g., `/vault-settings project_vault_git=tracked`), apply them directly.

Otherwise, ask the user which setting they want to change and what value to set.

## Step 4: Apply Changes

1. Create `.vault/knowledge/` directory if it doesn't exist:
   ```bash
   mkdir -p .vault/knowledge
   ```

2. Write or update `.vault/knowledge/.settings.yaml` with the new values. Preserve comments.

3. **If `project_vault_git` was changed:**
   - `tracked`: ensure `.vault/knowledge/` is NOT in `.gitignore`
   - `ignored`: add `.vault/knowledge/*.md` to `.gitignore` (keep .settings.yaml tracked)
   - `ask`: no gitignore changes (will prompt on first capture)

4. **If `proposal_expiry_days` was changed:**
   - No side effects -- `/vault-triage` reads this at runtime

5. Report what was changed.

## Step 5: Report

```
## Vault Settings Updated

Changed:
- <setting>: <old value> -> <new value>

Settings file: .vault/knowledge/.settings.yaml
```

## Reading Settings from Other Commands

Other commands (`/vault-capture`, `/vault-triage`, session-start hook) should check for `.vault/knowledge/.settings.yaml` and use its values. If the file doesn't exist, use defaults.

To read a setting from a shell hook:
```bash
if [ -f ".vault/knowledge/.settings.yaml" ]; then
    value="$(grep '^proposal_expiry_days:' .vault/knowledge/.settings.yaml | awk '{print $2}')"
fi
value="${value:-30}"  # default
```
