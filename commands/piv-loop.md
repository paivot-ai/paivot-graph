---
description: Run unattended execution loop until blocked or protected-main handoff
allowed-tools: ["Bash", "Read", "Glob", "Grep", "Skill", "Task", "AskUserQuestion"]
args: "[EPIC_ID] [--all] [--max-iterations|--max N]"
---

# piv-loop -- Unattended Execution Loop

Run the backlog forward without manual intervention until all dispatchable work is
done, the loop is blocked, a protected-main handoff is required, or max
iterations are reached. Spawns developer and PM agents in priority order.

## Setup

**IMPORTANT:** `pvg loop setup` REQUIRES either `--all` or `--epic EPIC_ID`. Running it
without these flags will fail. Do NOT attempt the bare command.

If `$ARGUMENTS` is non-empty, run:
```bash
pvg loop setup $ARGUMENTS
```

If `$ARGUMENTS` is empty, ask the user FIRST (via AskUserQuestion):
- "Run all ready work (`--all`) or target a specific epic (provide the EPIC_ID)?"
- "Max iterations? (default: 50, 0 for unlimited)"

Then run `pvg loop setup` with the user's chosen flags. Verify activation succeeded
before continuing.

**Shell hygiene:** Do NOT append `2>&1` to nd or pvg commands. Claude Code's Bash tool
already captures stderr separately. Redirecting stderr causes duplicate error display.

## Priority Order

Each iteration, pick work in this order:

0. **Bug triage** (highest priority -- discovered bugs need structure)
   After any Developer or PM-Acceptor agent completes, scan its output for
   `DISCOVERED_BUG:` blocks. If found, collect ALL bug reports from that agent
   and spawn `paivot-graph:sr-pm` with:
   ```
   BUG TRIAGE MODE. Create properly structured bugs for these discovered issues:
   <paste all DISCOVERED_BUG blocks>
   ```
   Wait for Sr. PM to finish before continuing. Bugs need epic placement and
   dependency chains before other work can be prioritized correctly.

   **Note:** When `bug_fast_track` is enabled (or story has `pm-creates-bugs` label),
   PM-Acceptor creates bugs directly during review -- there will be no DISCOVERED_BUG
   blocks to route to Sr PM for those stories. Only bugs from Developer agents or from
   PM-Acceptor in centralized mode (the default) appear as DISCOVERED_BUG blocks.

1. **PM-Acceptor for delivered stories** (unblock the pipeline)
   ```bash
   nd list --status in_progress --label delivered --json
   ```
   For each: spawn `paivot-graph:pm` agent to review and accept/reject.
   **The PM-Acceptor closes the story itself** (`nd close --reason`). Do NOT
   re-close stories after the PM-Acceptor finishes -- they are already closed.
   **After each acceptance**: the PM-Acceptor runs epic auto-close (see pm.md).
   **IMMEDIATELY after acceptance**: merge the story branch to epic (see Story
   Merge below). Complete the merge -- including conflict resolution if needed --
   before moving to the next priority item. An accepted story with an unmerged
   branch is incomplete work.

2. **Developer for rejected stories** (fix before starting new work)
   ```bash
   nd list --status open --label rejected --json
   ```
   For each: spawn `paivot-graph:developer` agent to address rejection notes, claim the story, and clear the `rejected` label before re-delivery.

3. **Developer for ready stories** (new work)
   ```bash
   nd ready --sort priority --json
   ```
   Pick the highest-priority item from results (P0 first, then P1, etc.).
   For each: spawn `paivot-graph:developer` agent to implement.
   **An empty result from this query is the ONLY signal that work is done.**
   If it returns items at ANY priority, keep working.

**nd filter cheat sheet** (prevents wasted queries with wrong flags):
- Priority: `--priority 0` (not `--label P0` -- priority is not a label)
- Labels: `--label delivered`, `--label rejected`, `--label hard-tdd`
- Type: `--type bug`, `--type task`, `--type epic`
- Parent: `--parent <epic-id>`

**Epic-scoped queries**: When targeting a specific epic, scope queries with `--parent`.
But remember: the loop runs across the ENTIRE backlog, not just one epic (see Termination).

```bash
nd ready --parent <epic-id> --json                        # Ready work in the epic
nd children <epic-id> --json                              # All stories in the epic
nd list --parent <epic-id> --status in_progress --json    # Filtered within epic
```

As of nd v0.7.0, `nd ready` supports the same filter flags as `nd list`:
`--parent`, `--status`, `--label`, `--type`, `--assignee`, `--priority`,
`--no-parent`, `--sort`, `--reverse`, `--limit`, date range filters, `--json`.
Run `nd <command> --help` if unsure about available flags.

## Concurrency Limits (HARD RULE)

Limits are stack-dependent. Detect from project files (Cargo.toml, *.xcodeproj,
*.csproj, wrangler.toml/wrangler.jsonc, pyproject.toml, package.json, etc.).

Heavy stacks (Rust, iOS/Swift, C#, CloudFlare Workers):
- Maximum 2 developer agents simultaneously
- Maximum 1 PM-Acceptor agent simultaneously
- Total active subagents (all types) must not exceed 3

Light stacks (Python, non-CF TypeScript/JavaScript):
- Maximum 4 developer agents simultaneously
- Maximum 2 PM-Acceptor agents simultaneously
- Total active subagents (all types) must not exceed 6

When a project mixes stacks, use the most restrictive limit.
- Wait for an agent to finish before spawning another if at the limit

These limits prevent context and machine resource exhaustion.

## Branch Management (Two-Level Model)

Paivot uses a two-level branching strategy: `main → epic → story`. See [[Two-Level Branch Model]] for complete details.

The branch model does not change the live source of record requirement: when nd
backs execution, the mutable backlog must live in a branch-independent vault
shared across worktrees, not in branch-local `.vault/issues/` copies.

**Your responsibilities as dispatcher:**

### Story Branch Setup

Before spawning a developer:

```bash
# Ensure epic branch exists (create if needed)
git fetch origin
if ! git rev-parse --verify origin/epic/EPIC_ID >/dev/null 2>&1; then
  git checkout -b epic/EPIC_ID origin/main
  git push -u origin epic/EPIC_ID
fi

# Create story branch from epic
git checkout -b story/STORY_ID origin/epic/EPIC_ID
git push -u origin story/STORY_ID
```

Developer receives worktree rooted at `story/STORY_ID`. They work in isolation, cannot accidentally push to epic or main.

### Story Merge (After PM Approves)

**STRUCTURAL GATE:** `pvg guard` blocks `git merge story/*` unless the story is both labeled `accepted` and `closed` in nd. This is enforced by the PreToolUse hook in Paivot-managed repos. If the merge is blocked, let PM-Acceptor finish review first.

**CRITICAL:** Merging is your IMMEDIATE next step after PM acceptance. Complete the merge (including conflict resolution) before moving to the next priority item. A story that is accepted in nd but not merged in git is incomplete work.

After PM-Acceptor adds `accepted` and closes the delivered story:

**Step 1: Attempt the merge**

```bash
git fetch origin
git checkout epic/EPIC_ID
git pull origin epic/EPIC_ID
git merge --no-ff origin/story/STORY_ID -m "merge(epic/EPIC_ID): integrate STORY_ID"
```

**Step 2a: Merge succeeded** -- push and clean up:

```bash
git push origin epic/EPIC_ID
git branch -D story/STORY_ID
git push origin --delete story/STORY_ID
```

**Step 2b: Merge conflict** -- abort, stay on epic, spawn developer, retry:

Do NOT checkout main. Do NOT move to another priority item. Handle inline.

```bash
# 1. Abort the failed merge. Stay on the epic branch.
git merge --abort
# You are still on epic/EPIC_ID. Do NOT checkout main or any other branch.
```

```
# 2. Spawn developer for conflict resolution. Use this exact prompt:
CONFLICT RESOLUTION MODE. Story STORY_ID is accepted but cannot merge
into epic/EPIC_ID due to conflicts.

Your task: rebase story/STORY_ID onto the latest epic/EPIC_ID, resolving
all conflicts.

Steps:
1. git fetch origin
2. git checkout story/STORY_ID
3. git rebase origin/epic/EPIC_ID
4. Resolve conflicts in each file (keep functionality from both sides)
5. git rebase --continue after each resolution
6. Run tests to verify nothing is broken
7. git push --force-with-lease origin story/STORY_ID

Do NOT update nd -- the story is already accepted and closed.
Report: list of conflicting files, resolution decisions, test results.
```

```bash
# 3. After developer completes, retry the merge from the epic branch:
git fetch origin
git checkout epic/EPIC_ID
git pull origin epic/EPIC_ID
git merge --no-ff origin/story/STORY_ID -m "merge(epic/EPIC_ID): integrate STORY_ID"
```

```bash
# 4. If retry succeeds: push and clean up (same as Step 2a).
# 5. If retry STILL fails: escalate to user via AskUserQuestion:
#    "Merge conflict persists for STORY_ID into epic/EPIC_ID after developer
#     rebase. Please resolve manually or provide guidance."
```

**Canonical branch names:** use `epic/<EPIC_ID>` and `story/<STORY_ID>` exactly. Do not append descriptive suffixes. The dispatcher, merge gate, and recovery flow all assume IDs are the full branch key.

**Merge order:** If multiple stories are waiting to merge, process them in dependency order first, then priority order (P0 first) within each ready layer. Do NOT use `parent` for this: `parent` is epic containment, not the dependency graph. Use `nd dep tree STORY_ID` and `nd show STORY_ID --json` to inspect `blocked_by`, `blocks`, and `follows`; merge prerequisite stories before dependents.

### Epic Completion (All Stories Merged)

When all stories in the epic have been approved and merged to the epic branch,
prepare the epic for merge back to `main` through the repository's protected-main
flow. Do NOT merge `epic/*` directly into `main` from the dispatcher.

```bash
git fetch origin
git checkout epic/EPIC_ID
git pull origin epic/EPIC_ID

# Then hand off to the repo's PR/integration process for epic -> main.
# Main is protected; do not push a direct merge from the dispatcher.
```

If your environment provides PR automation, use it and continue unattended.
Otherwise stop after the epic branch is ready and ask the user to complete or
approve the protected-main merge.

## Dispatcher Rules

You are a dispatcher. You coordinate agents and manage git integration. You NEVER:
- Write source code or tests yourself
- Fix errors or bugs yourself
- Modify story files yourself
- Make architectural decisions yourself
- Skip agents to "save time"
- Edit source files for any reason, including "cleanup" or "git maintenance"
- Inspect agent worktree internals (cd into `.claude/worktrees/agent-*`, run git log, read files there)
- Re-close stories that the PM-Acceptor already closed (it closes on acceptance -- you just read its output)

**You DO manage git:** Creating epic/story branches, merging story→epic after PM approval, preparing epic branches for protected-main integration, and resolving merge conflicts (by spawning developer if conflicts arise).

If an agent fails, re-spawn it with corrective guidance. Do not do its work.

## Infrastructure Context (MANDATORY before first developer spawn)

Before spawning the first developer agent in a session, discover what infrastructure
is available locally and include connection details in ALL developer agent prompts.

**Discovery protocol:**
1. `docker ps --format '{{.Names}} {{.Ports}}'` -- running containers
2. Check for docker-compose files, .env files with connection strings
3. Check project README/docs for infrastructure requirements

**Include in developer prompts:**
- List of running services with host:port
- Database connection details
- Required env vars with values (or instructions to obtain them)
- Explicit instruction: "Infrastructure is running. Do NOT gate tests behind env
  vars. Run integration tests directly against these services."

Without this context, developers will reasonably gate tests behind env vars --
creating dormant tests that satisfy no testing gate.

## Agent Types

| Role | Agent Type | When |
|------|-----------|------|
| Sr. PM (bug triage) | `paivot-graph:sr-pm` | DISCOVERED_BUG blocks found in agent output |
| PM-Acceptor | `paivot-graph:pm` | Stories with `delivered` label |
| Developer | `paivot-graph:developer` | Ready or rejected stories |

## Developer Spawning: Normal vs Hard-TDD

Hard-TDD is **opt-in per story**. Before spawning a developer, check for the `hard-tdd` label:

```bash
nd show <id> --json | grep -q '"hard-tdd"'
```

**If `hard-tdd` label is ABSENT** (the default): spawn ONE developer agent in normal mode.
The developer writes both implementation and tests in a single pass. This is the standard flow.

**If `hard-tdd` label is PRESENT**: run the two-phase flow:
1. RED phase: spawn developer with "RED PHASE" in the prompt (tests only)
2. PM-Acceptor reviews tests
3. GREEN phase: spawn developer with "GREEN PHASE" in the prompt (implementation only)
4. PM-Acceptor reviews implementation

**Do NOT default to hard-TDD.** The user's general TDD preference (writing tests alongside
code) is satisfied by normal mode. Hard-TDD is a stricter discipline where tests and
implementation are written by separate agent invocations with structural locks. It requires
explicit opt-in via the label.

## Termination Conditions

**The loop is permanent.** It runs across the ENTIRE backlog, not a single epic.
When an epic completes, the loop moves to the next epic with ready work.
The loop only stops when the backlog is empty or fully blocked.

The stop hook (`pvg hook stop`) evaluates these automatically:

| Condition | Action |
|-----------|--------|
| Entire backlog complete (nothing open anywhere) | Allow exit, remove state |
| All remaining work blocked (no actionable items) | Allow exit, remove state |
| Max iterations reached (if set) | Allow exit, remove state |
| Too many consecutive waits (3) | Allow exit, remove state |
| Actionable work exists anywhere in backlog | Block exit, continue loop |
| Only in-progress work (waiting) | Block exit, increment wait counter |

**Epic completion is NOT a termination event.** When an epic's last story is
accepted, the PM-Acceptor closes the epic (auto-close), and the loop moves on
to the next piece of ready work in the backlog. The loop keeps running.

## Cancellation

To cancel a running loop:
```
/piv-cancel-loop
```

Or directly:
```bash
pvg loop cancel
```

## Worktree Cleanup (after developer agent completes)

After merging a developer's worktree branch, clean up in ONE command:

```bash
git worktree remove --force .claude/worktrees/<agent-id> && git branch -D worktree-<agent-id>
```

**Always use `--force` and `-D`:**
- `--force` because worktrees always have build artifacts (.pyc, __pycache__, .pytest_cache)
- `-D` (not `-d`) because the branch is merged to HEAD but not to origin/main

Do NOT use `git worktree remove` without `--force` or `git branch -d` without `-D`.
These will always fail and waste tool calls.

**nd labels are idempotent-ish:** `nd labels add` fails if the label already exists.
If the developer already set `delivered`, don't set it again. Check first or ignore
the error.

For bulk cleanup after context loss, use `pvg loop recover` instead of manual
`git worktree remove` commands (see Post-Compaction Recovery below).

## Post-Compaction Recovery

**STRUCTURAL ENFORCEMENT:** The `pvg` pre-compact hook now emits a mandatory `pvg loop recover` reminder before every compaction. This reminder survives in the compaction summary. You MUST run `pvg loop recover` as the FIRST command after any compaction -- before touching git, before spawning agents, before inspecting branches.

After context compaction, you lose track of running agents and their worktrees.
Run recovery instead of doing manual cleanup:

```bash
pvg loop recover
```

This command automatically:
1. Reads the snapshot file (if one exists from a prior `pvg loop snapshot`)
2. Removes all agent worktrees and their branches
3. Deletes stale local branches (`epic/*`, `story/*`, `worktree-*`) that are fully merged into main
4. Resets orphaned in-progress stories to `open` in nd (delivered stories are preserved)
5. Outputs a recovery summary showing what's ready, delivered, and needs attention

If no snapshot exists, it still cleans orphan worktrees from `git worktree list`.

**Before compaction (optional but recommended):** take a snapshot to preserve agent state:
```bash
pvg loop snapshot --agent STORY-a1b=developer --agent STORY-c3d=pm-acceptor
```

Re-doing work is always cheaper than untangling nested worktrees.
Never spawn an agent whose cwd is inside another agent's worktree.

## Shell Usage

Do NOT redirect stderr on nd or pvg commands:
- No `2>&1` -- causes duplicate error display in Claude Code
- No `2>/dev/null` -- hides errors you need to see

Claude Code's Bash tool already captures stderr separately. Run commands bare.

## How It Works

The loop is driven by the Claude Code stop hook:
1. This command sets up loop state via `pvg loop setup`
2. You execute one iteration of work (spawn agents per priority order)
3. When you try to stop, `pvg hook stop` intercepts and evaluates
4. If work remains, it emits continuation JSON that keeps the session alive
5. The next iteration begins automatically with a status summary
6. This repeats until a termination condition is met
