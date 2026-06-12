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

# Solo developer workflow: merge epics directly to main, no PRs
# When true (default), epic branches merge to main and are deleted after completion.
# When false, epic completion creates a PR for team review before merging.
# Options: true, false
workflow.solo_dev: true

# Bug creation model: PM-Acceptor fast-track vs centralized Sr PM
# When false (default), only Sr PM creates bugs (ensures consistency and completeness)
# When true, PM-Acceptor can create bugs directly during story review (faster, less overhead)
# Options: true, false
bug_fast_track: false

# Workflow FSM -- structural enforcement of nd status transitions
# This is an nd-native status FSM, separate from Paivot's label-based
# delivered/accepted/rejected contract.
# When enabled, pvg guard blocks nd commands that skip nd status steps.
# The Paivot contract still uses labels:
#   delivered = nd status in_progress + delivered label
#   accepted  = nd status closed + accepted label
#   rejected  = nd status open + rejected label
workflow.fsm: false
workflow.sequence: open,in_progress,closed
workflow.exit_rules: blocked:open,in_progress;deferred:open,in_progress
workflow.custom_statuses:

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
# When true (default), the loop survives session boundaries -- background agent
# completions resume it where it left off
# When false, loop state is cleared when session exits, even if work remains
# Options: true, false
loop.persist_across_sessions: true

# Extra quality-gate patterns (pipe-separated) that the walking-skeleton check
# of `pvg lint --backlog` requires in every skeleton's AC, on top of its
# generic defaults. Populate from the project hard rules extracted in the
# Sr PM's Phase 1 ingestion.
# Example: "no.skip.if.missing|no mocks? in integration|always TDD"
lint.quality_gates:

# Force the paths-exist lint check on (brownfield mode), regardless of the
# >50-commits heuristic. The check verifies every path referenced in a story
# body exists on disk or in a PRODUCES block.
# Options: true, false
lint.brownfield: false

# Whether the session-start hook prints a one-line nudge when the installed
# pvg version is behind the distribution channel (channel/stable.json).
# Updates are never applied automatically; the nudge suggests `pvg update`.
# Options: true, false
update.nudge: true

# Per-role model overrides for Paivot agents. Each agent's model is set in its
# agents/*.md frontmatter by default; these settings override it at spawn time
# WITHOUT editing any agent file (the override survives plugin updates).
# Empty (default) = no override, the agent's built-in model wins.
# Allowed values: opus, sonnet, haiku, fable, inherit, or a full claude-* model id.
model.developer:
model.pm:
model.sr_pm:
model.anchor:
model.retro:
model.ba:
model.designer:
model.architect:
model.ba_challenger:
model.designer_challenger:
model.architect_challenger:

# Metric quality gates on delivered code (pvg gates). These compute code
# metrics by shelling out to real analyzers, compare them to thresholds, and
# return PASS/FAIL. A BLOCK finding fails the gate (exit 1); WARN findings are
# reported but do not fail. When an analyzer tool is ABSENT, the gate is
# SKIPPED and noted -- never a silent pass.
#
# Mode keys take: off | warn | block.
#   complexity  -- cyclomatic complexity per function (lizard, then gocyclo/radon)
#   duplication -- copy-paste duplication (jscpd)
#   file_loc    -- non-blank lines per file (built-in, no external tool)
gates.complexity: block
gates.complexity.warn_cc: 15
gates.complexity.block_cc: 30
gates.duplication: block
gates.duplication.max_pct: 10
gates.duplication.min_lines: 50
gates.file_loc: warn
gates.file_loc.max: 400
# Comma-separated globs/path-substrings dropped before any metric runs.
gates.exclude: vendor/,node_modules/,*.generated.*,*.pb.go,migrations/,*.lock,*.min.*,dist/,build/
```

## Step 2: Present Current Configuration

Show the user the current state:

```
## Vault Settings (<project>)

| Setting                  | Value     | Description                                      |
|--------------------------|-----------|--------------------------------------------------|
| project_vault_git        | ask       | Git tracking for .vault/knowledge/ notes only    |
| default_scope            | system    | Default scope when ambiguous                     |
| proposal_expiry_days     | 30        | Days before proposals are flagged stale          |
| session_start_max_notes  | 10        | Max notes summarized per subfolder at start      |
| auto_init_project_vault  | ask       | Create .vault/knowledge/ on first capture        |
| auto_capture             | true      | Automatically capture knowledge notes during work |
| staleness_days           | 30        | Days before vault notes are considered stale     |
| stack_detection          | false     | Detect and output project tech stack at start    |
| bug_fast_track           | false     | PM-Acceptor can create bugs directly during review |
| workflow.solo_dev        | true      | Merge epics to main directly, no PRs, clean branches |
| workflow.fsm             | false     | Structural enforcement of nd status transitions  |
| workflow.sequence        | open,...  | Ordered nd status pipeline (forward=+1, backward=any) |
| workflow.exit_rules      | ...       | Escape rules for blocked/deferred statuses        |
| workflow.custom_statuses | ...       | Extra nd statuses, if your project explicitly uses them |
| dnf.specialist_review    | false     | Adversarial challengers review each D&F document |
| dnf.max_iterations       | 3         | Max challenger review loops before user escalation |
| architecture.c4          | false     | C4 model + Architecture Contract alongside ARCHITECTURE.md |
| loop.persist_across_sessions | true  | Loop survives session boundaries; background completions resume it |
| lint.quality_gates       | (empty)   | Pipe-separated extra patterns the walking-skeleton lint check requires |
| lint.brownfield          | false     | Force the paths-exist lint check on (brownfield mode) |
| update.nudge             | true      | Session-start nudge when pvg is behind the channel |
| model.<role>             | (empty)   | Per-role agent model override; empty = agent's built-in default |
| gates.complexity         | block     | Cyclomatic-complexity gate mode (off/warn/block)  |
| gates.complexity.warn_cc | 15        | CCN at/above which a complexity WARN fires        |
| gates.complexity.block_cc| 30        | CCN at/above which a complexity BLOCK fires (block mode) |
| gates.duplication        | block     | Copy-paste duplication gate mode (off/warn/block) |
| gates.duplication.max_pct| 10        | Total duplication % at/above which a finding fires |
| gates.duplication.min_lines | 50     | A single clone of >= this many lines fires a finding |
| gates.file_loc           | warn      | File-size gate mode (off/warn/block)              |
| gates.file_loc.max       | 400       | Non-blank lines per file at/above which a finding fires |
| gates.exclude            | vendor/,...| Comma-separated globs/path-substrings dropped before metrics run |

Settings file: .vault/knowledge/.settings.yaml
```

## Step 3: Ask What to Change

If the user provided arguments (e.g., `/vault-settings project_vault_git=tracked`), apply them directly.

`project_vault_git` affects `.vault/knowledge/` only. It does not determine the
live nd backlog location for execution.

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

**If `workflow.solo_dev` was changed:**
- `true` (enable -- default):
  1. Report: "Solo-dev workflow enabled. Epics merge directly to main after completion gate (e2e + Anchor review). No PRs."
  2. Report: "Branch cleanup: epic and story branches deleted after merge to main."
- `false` (disable):
  1. Report: "Team workflow enabled. Epic completion creates a PR for review before merging to main."
  2. Report: "Branch cleanup happens after the PR is merged."

**If `workflow.fsm` was changed:**
- `true` (enable):
  1. `pvg settings workflow.fsm=true` (pvg auto-syncs nd)
  2. Verify nd is initialized: `pvg nd stats` (nd-specific)
  3. Report: "FSM enabled. pvg guard will enforce nd status transitions: <sequence>."
  4. Report: "Paivot contract labels remain unchanged: delivered stays on in_progress, accepted on closed, rejected on open."
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
- `true` (enable -- default):
  1. Report: "Loop state persistence enabled. The loop survives session boundaries; background agent completions resume it from where it left off."
- `false` (disable):
  1. Report: "Loop state persistence disabled. The execution loop will clear its state on session exit, even if work remains."
  2. No side effects -- takes effect on next loop stop.

**If `lint.quality_gates` was changed:**
- Report: "Walking-skeleton lint check will additionally require these patterns in every skeleton's AC: <patterns>."
- No side effects -- `pvg lint --backlog` reads this at runtime.

**If `lint.brownfield` was changed:**
- `true`: Report: "Brownfield mode forced on. The paths-exist lint check will run regardless of commit count."
- `false`: Report: "Brownfield mode not forced. The paths-exist lint check falls back to the >50-commits heuristic."

**If a `model.<role>` key was changed:**
- Sets the model used when that role's agent is spawned, overriding the agent's
  `agents/*.md` frontmatter without editing any file (the override survives
  plugin updates).
- Allowed values: `opus`, `sonnet`, `haiku`, `fable`, `inherit`, or a full
  `claude-*` model id. Empty clears the override (the agent's built-in default
  wins). Invalid values (e.g. a typo like `sonet`) are rejected by `pvg settings`.
- Roles: `developer`, `pm`, `sr_pm`, `anchor`, `retro`, `ba`, `designer`,
  `architect`, `ba_challenger`, `designer_challenger`, `architect_challenger`.
- For Developer and PM-Acceptor, the loop surfaces the override on each
  `pvg loop next` action as a `model` field; the dispatcher passes it as the
  Agent tool `model` parameter. For agents spawned outside the loop, the
  dispatcher reads `pvg settings model.<role>` and passes it at spawn time.
- Example: `pvg settings model.developer=sonnet`

**If a `gates.*` key was changed:**
- These configure `pvg gates` -- deterministic metric quality gates on
  delivered code (cyclomatic complexity, copy-paste duplication, file size).
  A `[BLOCK]` finding fails the gate (exit 1); `[WARN]` findings are reported
  but do not fail. When an analyzer tool is ABSENT, the gate is SKIPPED and
  noted in the report (`[SKIP] complexity: lizard not found`) -- never a silent
  pass and never a failure.
- Mode keys (`gates.complexity`, `gates.duplication`, `gates.file_loc`) take
  `off`, `warn`, or `block`. `pvg settings` rejects any other value (e.g. a
  typo like `blok`). The threshold keys take integers.
  - `gates.complexity` (default `block`): per-function cyclomatic complexity via
    `lizard` (multi-language), falling back to `gocyclo` (.go) and `radon`
    (.py). `gates.complexity.warn_cc` (default 15) and `gates.complexity.block_cc`
    (default 30) set the WARN/BLOCK bands. In `warn` mode no BLOCK finding is
    ever produced, even above `block_cc`.
  - `gates.duplication` (default `block`): copy-paste detection via `jscpd`. A
    finding fires when total duplication `>= gates.duplication.max_pct`
    (default 10) OR any single clone has `>= gates.duplication.min_lines`
    (default 50) duplicated lines.
  - `gates.file_loc` (default `warn`): built-in non-blank line count per file;
    a finding fires when a file's LOC `>= gates.file_loc.max` (default 400). No
    external tool, so this gate is never skipped.
  - `gates.exclude` (default
    `vendor/,node_modules/,*.generated.*,*.pb.go,migrations/,*.lock,*.min.*,dist/,build/`):
    comma-separated globs/path-substrings; any in-scope file matching is dropped
    before any metric runs. Directory substrings (`vendor/`) and basename globs
    (`*.pb.go`) are both supported.
- No side effects -- `pvg gates` reads these at runtime.
- Example: `pvg settings gates.duplication=warn`

**Installing the analyzers (the complexity and duplication gates need them):**

| Tool    | Purpose                          | Languages                              | Install                                                  | In Ubuntu apt? |
|---------|----------------------------------|----------------------------------------|----------------------------------------------------------|----------------|
| lizard  | cyclomatic complexity            | multi (C/C++, Java, JS/TS, Python, Go, Swift, ...) | `pip install lizard`                          | NO             |
| jscpd   | duplication / copy-paste         | multi                                  | `npm install -g jscpd`                                   | NO             |
| gocyclo | complexity (Go fallback)         | Go only                                | `go install github.com/fzipp/gocyclo/cmd/gocyclo@latest` | NO             |
| radon   | complexity (Python fallback)     | Python only                            | `pip install radon`                                      | YES (`apt install python3-radon`) |

apt alone is not enough: only `radon` ships in the Ubuntu repos. The two
recommended, multi-language tools come from pip (`lizard`) and npm (`jscpd`),
which are present on most dev machines. Installing just `lizard` + `jscpd`
lights up the full gate on virtually any stack; gocyclo/radon are niche
single-language fallbacks you only need if you skip lizard. Run `pvg doctor` to
see which analyzers are present, and `pvg setup` nudges you to install missing
ones.

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
