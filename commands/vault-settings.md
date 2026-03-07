---
description: View and configure paivot-graph settings for the current project -- project vault behavior, default scope, proposal expiry, gitignore preferences
allowed-tools: ["Bash", "Read", "Grep", "Glob"]
---

# Vault Settings

Manage paivot-graph configuration for the current project. Settings are stored in `.vault/knowledge/.settings.yaml` and affect how knowledge governance behaves in this project.

## Step 1: Load Current Settings

Check if settings file exists and read it:
```bash
pvg settings
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

# Whether to automatically capture knowledge notes during work
# Options: true, false
auto_capture: true

# Days before vault notes are considered stale (used by maintenance)
staleness_days: 30

# Whether session-start detects and outputs the project's tech stack
# Options: true, false
stack_detection: false

# Bug creation model: PM-Acceptor fast-track vs centralized Sr PM
# When false (default), only Sr PM creates bugs (ensures consistency and completeness)
# When true, PM-Acceptor can create bugs directly during story review (faster, less overhead)
# Options: true, false
bug_fast_track: false

# Workflow FSM -- structural enforcement of nd status transitions
# When enabled, pvg guard blocks nd commands that skip workflow steps
workflow.fsm: false
workflow.sequence: open,in_progress,delivered,review,closed
workflow.exit_rules: blocked:open,in_progress;rejected:in_progress
workflow.custom_statuses: delivered,review,rejected

# D&F specialist review: adversarial challengers review each BLT document
# When false (default), only Anchor reviews the final backlog (cost-optimized)
# When true, specialist challengers review BUSINESS.md, DESIGN.md, ARCHITECTURE.md
# individually before proceeding to the next BLT step (up to 3 iterations each)
# Options: true, false
dnf.specialist_review: false

# Maximum iterations for each specialist challenger review loop
# If a challenger rejects after this many iterations, escalate to user
# Range: 1-5 (default: 3)
dnf.max_iterations: 3

# C4 architecture model alongside ARCHITECTURE.md
# When enabled, Architect maintains workspace.dsl and Architecture Contract
# Options: true, false
architecture.c4: false

# Whether to persist execution loop state across sessions
# When false (default), loop state is cleared when session exits, even if work remains
# When true, loop can resume from where it left off in the next session
# Options: true, false
loop.persist_across_sessions: false
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
| auto_capture             | true      | Automatically capture knowledge notes during work |
| staleness_days           | 30        | Days before vault notes are considered stale     |
| stack_detection          | false     | Detect and output project tech stack at start    |
| bug_fast_track           | false     | PM-Acceptor can create bugs directly during review |
| workflow.fsm             | false     | Structural enforcement of nd status transitions  |
| workflow.sequence        | open,...  | Ordered status pipeline (forward=+1, backward=any) |
| workflow.exit_rules      | ...       | Escape rules for blocked/rejected statuses        |
| workflow.custom_statuses | ...       | Custom statuses registered with nd for display    |
| dnf.specialist_review    | false     | Adversarial challengers review each D&F document |
| dnf.max_iterations       | 3         | Max challenger review loops before user escalation |
| architecture.c4          | false     | C4 model + Architecture Contract alongside ARCHITECTURE.md |
| loop.persist_across_sessions | false | Whether execution loop state persists across sessions |

Settings file: .vault/knowledge/.settings.yaml
```

## Step 3: Ask What to Change

If the user provided arguments (e.g., `/vault-settings project_vault_git=tracked`), apply them directly.

Otherwise, ask the user which setting they want to change and what value to set.

## Step 4: Apply Changes

Use the pvg binary to apply settings changes:
```bash
pvg settings <key>=<value>
```

For example:
```bash
pvg settings project_vault_git=tracked
pvg settings proposal_expiry_days=14
```

**If `project_vault_git` was changed:**
- `tracked`: ensure `.vault/knowledge/` is NOT in `.gitignore`
- `ignored`: add `.vault/knowledge/*.md` to `.gitignore` (keep .settings.yaml tracked)
- `ask`: no gitignore changes (will prompt on first capture)

**If `proposal_expiry_days` was changed:**
- No side effects -- `/vault-triage` reads this at runtime

**If `bug_fast_track` was changed:**
- `true` (enable):
  1. Report: "Bug fast-track enabled. PM-Acceptor can now create bugs directly during story review."
  2. Report: "Guardrails enforced: P0 hardcoded, parent epic auto-detected, discovered-by-pm label."
- `false` (disable):
  1. Report: "Bug fast-track disabled. All bugs route through Sr PM (centralized model)."
  2. No side effects -- PM-Acceptor reverts to DISCOVERED_BUG blocks.

**If `dnf.specialist_review` was changed:**
- `true` (enable):
  1. Report: "D&F specialist review enabled. Each BLT document will be adversarially reviewed before proceeding."
  2. Report: "Challengers: BA Challenger (BUSINESS.md), Designer Challenger (DESIGN.md), Architect Challenger (ARCHITECTURE.md)."
  3. Report: "Max iterations per document: <dnf.max_iterations> (default 3). After exhaustion, escalates to user."
- `false` (disable):
  1. Report: "D&F specialist review disabled. Only Anchor reviews the final backlog (cost-optimized model)."
  2. No side effects -- challengers simply won't be spawned.

**If `dnf.max_iterations` was changed:**
- Validate value is between 1 and 5. If out of range, reject and report valid range.
- Report: "Max challenger iterations set to <value>. Each D&F document may be reviewed up to <value> times before user escalation."

**If `workflow.fsm` was changed:**
- `true` (enable):
  1. `pvg settings workflow.fsm=true` (pvg auto-syncs nd)
  2. Verify nd is initialized: `nd stats 2>/dev/null || echo "warning: nd not initialized"`
  3. Report: "FSM enabled. pvg guard will enforce status transitions: <sequence>"
- `false` (disable):
  1. `pvg settings workflow.fsm=false` (pvg auto-syncs nd)
  2. Report: "FSM disabled. Status transitions are no longer enforced."

**If `architecture.c4` was changed:**
- `true` (enable):
  1. Report: "C4 architecture model enabled. Architect will maintain workspace.dsl and Architecture Contract."
  2. If `workspace.dsl` does not exist, note: "Architect will create workspace.dsl on next D&F or architecture update."
  3. The `c4` skill will be discovered by agents via normal skill discovery.
- `false` (disable):
  1. Report: "C4 architecture model disabled. Existing workspace.dsl is preserved but not maintained."
  2. No files are deleted.

**If `loop.persist_across_sessions` was changed:**
- `true` (enable):
  1. Report: "Loop state persistence enabled. The execution loop will resume from where it left off in the next session."
- `false` (disable):
  1. Report: "Loop state persistence disabled. The execution loop will clear its state on session exit."
  2. No side effects -- takes effect on next loop stop.

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
pvg settings <key>
```
