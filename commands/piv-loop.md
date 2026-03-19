---
description: Run unattended execution loop until blocked or all work is done
allowed-tools: ["Bash", "Read", "Glob", "Grep", "Skill", "Task", "AskUserQuestion"]
args: "[EPIC_ID] [--all] [--max-iterations|--max N]"
---

# piv-loop -- Unattended Execution Loop

Run the backlog forward one epic at a time without manual intervention. The loop
drains each epic fully (all stories accepted, merged, e2e verified) before rotating
to the next. Parallelization happens WITHIN the current epic, not across epics.

## Defaults and Settings

| Setting | Default | Override |
|---------|---------|----------|
| Epic selection | Auto (highest-priority with actionable work) | `--epic EPIC_ID` |
| Scope | Single epic at a time | `--all` (legacy, no containment) |
| Auto-rotate | On (rotate to next epic after completion gate) | Inherent to epic mode |
| Max iterations | 50 | `--max N` (0 = unlimited) |
| Concurrency | Within current epic only | Stack-dependent limits |

The dispatcher NEVER picks stories from outside the current epic. `pvg loop next --json`
enforces this structurally -- it only returns stories scoped to the active epic.

## Setup

If `$ARGUMENTS` is non-empty, run:
```bash
pvg loop setup $ARGUMENTS
```

If `$ARGUMENTS` is empty, run the bare command (auto-selects the highest-priority epic):
```bash
pvg loop setup
```

To target a specific epic: `pvg loop setup --epic EPIC_ID`
To run across all epics without containment (not recommended): `pvg loop setup --all`

Verify activation succeeded before continuing.

**Shell hygiene:** Do NOT append `2>&1` to nd or pvg commands. Claude Code's Bash tool
already captures stderr separately. Redirecting stderr causes duplicate error display.

## Iteration Protocol

Each iteration, run:

```bash
pvg loop next --json
```

This returns a JSON decision. Follow it:

| Decision | Action |
|----------|--------|
| `act` | Spawn the agent specified in `next` (developer or pm_acceptor) |
| `epic_complete` | Run the epic completion gate (e2e + Anchor + merge to main), then rotate |
| `epic_blocked` | All remaining work in the current epic is blocked. Escalate to user via AskUserQuestion |
| `wait` | Agents are working in the current epic. Do nothing. Wait for completions |
| `rotate` | Epic is done and gate passed. Update loop state to the new epic in `next_epic` |
| `complete` | All epics drained. Allow exit |
| `blocked` | All remaining work globally is blocked (--all mode). Allow exit |

**`pvg loop next --json` is the SINGLE SOURCE OF TRUTH for dispatch decisions.**
Do NOT query nd directly with `nd ready --json` or `nd list --json` for choosing what
to work on next. Those queries are unscoped and will return stories from ALL epics,
breaking containment.

You MAY use nd directly for:
- Reading story content before spawning a developer (`nd show STORY_ID`)
- Checking story labels (`nd show STORY_ID --json`)
- Bug triage routing (DISCOVERED_BUG blocks)
- Epic auto-close checks after PM acceptance

### Bug Triage (Overrides Iteration Protocol)

After any Developer or PM-Acceptor agent completes, scan its output for
`DISCOVERED_BUG:` blocks BEFORE running `pvg loop next --json`. If found,
collect ALL bug reports and spawn `paivot-graph:sr-pm` with:

```
BUG TRIAGE MODE. Create properly structured bugs for these discovered issues:
<paste all DISCOVERED_BUG blocks>
```

Wait for Sr. PM to finish before continuing. Bugs need epic placement and
dependency chains before other work can be prioritized correctly.

**Note:** When `bug_fast_track` is enabled (or story has `pm-creates-bugs` label),
PM-Acceptor creates bugs directly during review. Only bugs from Developer agents
or from PM-Acceptor in centralized mode (the default) appear as DISCOVERED_BUG blocks.

### After PM-Acceptor Acceptance

**IMMEDIATELY after acceptance**: merge the story branch to epic (see Story
Merge below). Complete the merge -- including conflict resolution if needed --
before running `pvg loop next --json` again. An accepted story with an unmerged
branch is incomplete work.

## Epic Flow

The loop drains one epic at a time:

1. **Start**: auto-selects the highest-priority epic with actionable work
2. **Execute**: all parallelization happens WITHIN the current epic
   (multiple developers on different stories, one PM reviewing)
3. **Complete**: when all stories are accepted and merged to the epic branch,
   `pvg loop next --json` returns `epic_complete`
4. **Gate**: run the epic completion gate (e2e tests + Anchor milestone review + merge to main)
5. **Rotate**: `pvg loop next --json` returns `rotate` with `next_epic` -- update state and continue

Epic completion is a GATE, not a passthrough. The full gate (e2e, Anchor, merge to main)
MUST finish before rotation. There is no cherry-picking across epics.

## Concurrency Limits (HARD RULE)

All concurrency is WITHIN the current epic.

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

Paivot uses a two-level branching strategy: `main -> epic -> story`. See [[Two-Level Branch Model]] for complete details.

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

When `pvg loop next --json` returns `epic_complete`, the epic enters a three-step
completion gate before merging to main. All three steps are structural -- no step
may be skipped.

**Step 1: Epic Verification Gate (STRUCTURAL -- always on)**

Run the FULL test suite on the merged epic branch. This catches integration
failures that passed in isolation on individual story branches but break when
combined. **No epic is done without passing e2e tests. Period.**

```bash
git fetch origin
git checkout epic/EPIC_ID
git pull origin epic/EPIC_ID

# Run the project's full test suite (unit + integration + e2e)
# Use the project's standard test command (make test, pytest, go test ./..., etc.)
```

**After running the test suite, verify e2e tests exist and ran:**

```bash
pvg verify --check-e2e
```

If `pvg verify --check-e2e` reports zero e2e test files, the gate FAILS --
even if all other tests passed. "0 e2e failures" with 0 e2e tests is not
passing, it is missing. Spawn a developer to write the e2e tests before
proceeding.

Every test must pass -- unit, integration, AND e2e. If any test fails:

1. Spawn `paivot-graph:developer` with:
   ```
   EPIC VERIFICATION FIX. Tests fail on the merged epic/EPIC_ID branch after
   all stories were integrated. Your task: fix the failing tests on the epic
   branch directly. This is NOT a story -- do not create nd issues. Run the
   full test suite after fixing and report results.

   Failing tests: <paste test output>
   Infrastructure: <paste connection details>
   ```
2. After the developer fix, re-run the full test suite.
3. If tests still fail after 2 developer attempts, escalate to user via AskUserQuestion.

Do NOT skip this gate. Do NOT proceed to Step 2 with failing tests.

**Step 2: Anchor Milestone Review**

Spawn `paivot-graph:anchor` in milestone review mode:

```
MILESTONE REVIEW for epic EPIC_ID.

Validate that the completed epic delivered real value:
- Inspect tests for mocks in integration/e2e tests (forbidden)
- Verify skills were consulted where stories required them
- Check that boundary maps are satisfied (PRODUCES/CONSUMES)
- Validate hard-TDD two-commit pattern where applicable

Epic branch: epic/EPIC_ID
```

If the Anchor returns GAPS_FOUND, address the gaps (spawn developer to fix,
or escalate to user) before proceeding. Do NOT merge to main with open gaps.

**Step 3: Merge to Main**

Check the project workflow setting:

```bash
pvg settings workflow.solo_dev
```

**If `workflow.solo_dev=true`** (default -- solo developer, no PRs):

```bash
# Safety: ensure we have the latest main
git checkout main
git pull origin main

# Merge with --no-ff to preserve epic history
git merge --no-ff epic/EPIC_ID -m "merge(main): complete EPIC_ID"
git push origin main

# Clean up epic branch (local + remote)
git branch -D epic/EPIC_ID
git push origin --delete epic/EPIC_ID
```

Then clean up all story branches for this epic:

```bash
# Delete remote story branches
for branch in $(git branch -r --list "origin/story/*" | sed 's|origin/||'); do
  git push origin --delete "$branch" 2>/dev/null || true
done

# Delete local story branches
for branch in $(git branch --list "story/*"); do
  git branch -D "$branch" 2>/dev/null || true
done
```

**If `workflow.solo_dev=false`** (team workflow, PRs required):

```bash
git fetch origin
git checkout epic/EPIC_ID
git pull origin epic/EPIC_ID

# Create PR for epic -> main (requires gh CLI)
gh pr create --base main --head "epic/EPIC_ID" \
  --title "merge(main): complete EPIC_ID" \
  --body "All stories accepted. Full test suite passing. Anchor review: VALIDATED."
```

If your environment provides PR automation, use it and continue unattended.
Otherwise stop after the PR is created and ask the user to complete or
approve the merge. Branch cleanup happens after the PR is merged.

**After merge to main**: run `pvg loop next --json` again. It will return
either `rotate` (with the next epic) or `complete` (all done).

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
- Query nd globally for dispatch decisions (use `pvg loop next --json` instead)

**You DO manage git:** Creating epic/story branches, merging story->epic after PM approval, running the epic completion gate (e2e + Anchor review), merging epic->main (solo-dev) or creating PRs (team), cleaning up branches, and resolving merge conflicts (by spawning developer if conflicts arise).

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

## Termination

The loop drains one epic at a time. The stop hook (`pvg hook stop`) evaluates
termination automatically:

| Condition | Action |
|-----------|--------|
| No actionable epics remain | Allow exit, remove state |
| Current epic blocked, no other epics | Allow exit |
| Max iterations reached | Allow exit, remove state |
| Too many consecutive waits (3) | Allow exit |
| Current epic has actionable work | Block exit, continue |
| Current epic complete, next epic exists | Block exit, rotate |

### Live Demo (before session exit)

Every session must produce demonstrable progress. Before the loop exits:

1. Identify what was delivered (accepted stories, completed epics, merged to main)
2. If anything was merged to main: run the project's demo, smoke test, or e2e suite
   on main and report results to the user
3. If nothing reached main: explain what blocked progress and what the user should
   do next

A session that cannot show working software at the end should be treated as a
signal that something is wrong with the backlog, the infrastructure, or the
test suite -- not as normal.

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
2. You run `pvg loop next --json` to get the next action
3. You execute that action (spawn agent, run gate, etc.)
4. When you try to stop, `pvg hook stop` intercepts and evaluates
5. If work remains, it emits continuation JSON that keeps the session alive
6. The next iteration begins automatically with a status summary
7. This repeats until a termination condition is met
