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

# Workflow FSM -- structural enforcement of nd status transitions
# When enabled, pvg guard blocks nd commands that skip workflow steps
workflow.fsm: false
workflow.sequence: open,in_progress,delivered,review,closed
workflow.exit_rules: blocked:open,in_progress;rejected:in_progress
workflow.custom_statuses: delivered,review,rejected

# C4 architecture model alongside ARCHITECTURE.md
# When enabled, Architect maintains workspace.dsl and Architecture Contract
# Options: true, false
architecture.c4: false
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
| workflow.fsm             | false     | Structural enforcement of nd status transitions  |
| workflow.sequence        | open,...  | Ordered status pipeline (forward=+1, backward=any) |
| workflow.exit_rules      | ...       | Escape rules for blocked/rejected statuses        |
| workflow.custom_statuses | ...       | Custom statuses registered with nd for display    |
| architecture.c4          | false     | C4 model + Architecture Contract alongside ARCHITECTURE.md |

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

**If `workflow.fsm` was changed:**
- `true` (enable):
  1. `bin/pvg settings workflow.fsm=true` (pvg auto-syncs nd)
  2. Verify nd is initialized: `nd stats 2>/dev/null || echo "warning: nd not initialized"`
  3. Report: "FSM enabled. pvg guard will enforce status transitions: <sequence>"
- `false` (disable):
  1. `bin/pvg settings workflow.fsm=false` (pvg auto-syncs nd)
  2. Report: "FSM disabled. Status transitions are no longer enforced."

**If `architecture.c4` was changed:**
- `true` (enable):
  1. Report: "C4 architecture model enabled. Architect will maintain workspace.dsl and Architecture Contract."
  2. If `workspace.dsl` does not exist, note: "Architect will create workspace.dsl on next D&F or architecture update."
  3. The `c4` skill will be discovered by agents via normal skill discovery.
- `false` (disable):
  1. Report: "C4 architecture model disabled. Existing workspace.dsl is preserved but not maintained."
  2. No files are deleted.

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
