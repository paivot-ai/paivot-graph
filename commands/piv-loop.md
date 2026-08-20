---
description: Run unattended execution loop until blocked or all work is done
model: opus
allowed-tools: ["Bash", "Read", "Glob", "Grep", "Skill", "Agent", "Task", "AskUserQuestion"]
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

Parse `$ARGUMENTS` for recognized flags ONLY. Everything else is natural language
context (concurrency hints, instructions) that you should read and apply but NOT
pass to the shell command.

Recognized flags:
- `--epic EPIC_ID` or a bare issue ID (e.g., `PRA-ru13`)
- `--all`
- `--max N` or `--max-iterations N`

Any other text (e.g., "use up to 2 parallel devs", "focus on auth stories") is
natural language. Extract intent from it but do NOT include it in the pvg command.

```bash
# Examples:
pvg loop setup                          # auto-select epic
pvg loop setup --epic PRA-ru13          # target specific epic
pvg loop setup --all --max 25           # all epics, 25 iterations
```

If `$ARGUMENTS` contains only recognized flags, pass them through:
```bash
pvg loop setup $RECOGNIZED_FLAGS
```

If `$ARGUMENTS` is empty or contains only natural language, run bare:
```bash
pvg loop setup
```

Verify activation succeeded before continuing. `pvg loop setup` also enables
dispatcher mode automatically when it is not already on, and runs a best-effort
full `nd sync` (snapshot + fetch + field-aware merge + push of the nd-native
`nd/backlog` branch) so the loop starts from the latest shared backlog. A sync
failure is a WARN, never fatal -- offline and remote-less repos still loop.

**Shell hygiene:** Do NOT append `2>&1` to nd or pvg commands. Claude Code's Bash tool
already captures stderr separately. Redirecting stderr causes duplicate error display.

## Iteration Protocol

Each iteration, run:

```bash
pvg loop next --json
```

This returns a JSON decision. On actions, `priority` is the STORY's nd
priority (0 = P0); queue precedence is the `queue` field (delivered >
rejected > ready). Follow it:

| Decision | Action |
|----------|--------|
| `act` | Spawn the agent specified in `next` (developer or pm_acceptor). If the action carries `resume_agent`, resumption REPLACES the fresh Agent spawn -- deliver the payload to the recorded agent per Semi-Persistent Story Agents below. If the action carries a non-empty `model` field, pass it as the Agent tool `model` parameter for that spawn; if `model` is absent/empty, spawn normally (the agent's frontmatter default applies) |
| `epic_complete` | Run the epic completion gate (e2e + Anchor + merge to main), then call `pvg loop rotate <next_epic>` and continue. If the payload carries `seal_epic`, the completing epic was the LAST open slice of that milestone: run the Milestone Seal Gate for it after the completion gate and before rotating |
| `milestone_seal` | Every slice epic of the milestone named in `seal_epic` is closed but the milestone itself is open: its seal gate has not run. Run the Milestone Seal Gate below. The loop may not rotate or exit past a pending seal |
| `epic_blocked` | All remaining work in the current epic is blocked. Escalate to user via AskUserQuestion |
| `wait` | Agents are working in the current epic. Do nothing. Wait for completions |
| `complete` | All epics drained. Allow exit |
| `blocked` | All remaining work globally is blocked (--all mode). Allow exit |
| `other` | Miscellaneous action surfaced in --all mode. Follow the action payload |
| `no_active_loop` | No loop is active. Run `pvg loop setup` before iterating |
| `stalled` | The same in_progress story set was observed for 3 consecutive wait evaluations. The payload lists the story ids and worktrees. Run `pvg loop recover`, then re-spawn each affected story or release it (`pvg story release <id>`) |
| `escalate` | A story hit the rejection cap (3 PM rejections). Surface it to the user via AskUserQuestion with the rejection history. NEVER override the PM |

**`pvg loop next --json` is the SINGLE SOURCE OF TRUTH for dispatch decisions.**
Do NOT query nd directly with `pvg issues ready --json` or `pvg issues list --json` for
choosing what to work on next. Those queries are unscoped and will return stories from
ALL epics, breaking containment.

### Wave Dispatch (multiple ready stories)

When the current epic has multiple ready stories and the concurrency limit allows
k more developers, request a wave instead of looping one action at a time:

```bash
pvg loop next --json --n k
```

This returns up to k distinct actions in an `actions` array (at most one
pm_review per wave, then developers from the rejected/ready queues). The `next`
field still carries the first action. Spawn one developer per entry in
`actions[]` -- each gets its own story branch and dispatcher-managed worktree
exactly as described under Story Branch Setup. The single-source-of-truth rule
is unchanged: the wave comes from `pvg loop next`, never from unscoped nd queries.

Each entry of `actions[]` carries the same optional `model` field as the
single-action `next`. Apply it per entry: pass a non-empty `model` as the Agent
tool `model` parameter for that spawn; omit it when empty.

Entries may also carry `resume_agent` and `resume_count`. Handle each such
entry per Semi-Persistent Story Agents below: resumption replaces the fresh
Agent spawn for that entry, and a resumed agent counts toward the wave size
and the concurrency limits exactly like a spawned one.

### Background Spawning (REQUIRED for concurrency)

Spawn Developer and PM-Acceptor agents with `run_in_background: true`. This is
what lets multiple agents execute concurrently and lets the stop hook's
wait/escape-valve machinery work as designed: the harness re-invokes the
dispatcher when background agents complete, and `wait` decisions resolve
naturally. Foreground spawning is acceptable only for strictly sequential
single-agent steps (e.g., a lone conflict-fix or the Anchor milestone review).

### Semi-Persistent Story Agents

Developer and PM conversations are reusable within a session. Instead of
paying a fresh spawn for every rework or re-review round, record each agent's
handle at spawn time and resume the same conversation when the loop asks
for it.

**Record on spawn.** Immediately after spawning a developer for story
STORY_ID via the Agent tool, record the handle the tool returns:

```bash
pvg loop agent set STORY_ID developer <agentId>
```

Immediately after spawning a PM for story STORY_ID:

```bash
pvg loop agent set STORY_ID pm <agentId>
```

This applies to EVERY spawn of these roles -- first spawn, re-spawn after
failure, fresh-spawn fallback -- so the recorded handle always points at the
live conversation.

**Resume on action.** `pvg loop next` actions may carry `resume_agent` (a
recorded handle) and `resume_count` on developer-rework and pm-review
actions. pvg emits them only when a handle is recorded, fewer than 2 resumes
have occurred for that story+role, and the `loop.agent_resume` setting
(default true) is enabled; it increments the counter at emission, so
`resume_count` reports the resumes this story+role has consumed. When an
action carries `resume_agent`:

1. Verify the story worktree still exists on disk.
2. Resume the recorded agent: send it the rework/review payload with the
   SendMessage tool instead of spawning a fresh agent.
3. On ANY failure -- SendMessage error, stale handle, missing worktree -- or
   if the agent's last delivery contained a CONTEXT_BUDGET note: run
   `pvg loop agent clear STORY_ID <role>`, fall back to the normal fresh
   spawn, then `pvg loop agent set` the new handle.

Fresh spawn is ALWAYS the safe fallback; resume is an optimization, never a
requirement.

**Clear on accept.** After a story is accepted and merged, clear both roles:

```bash
pvg loop agent clear STORY_ID
```

**Handles are session-scoped.** A new session invalidates them structurally
-- pvg clears them on session change. You never need to reason about
staleness beyond the failure fallback above.

**Why resume.** A resumed agent keeps its FULL conversation: in testing, a
~47k-token transcript resumed as a 100 percent cache hit, and a rework round
cost ~7k incremental tokens versus ~31k+ for a fresh spawn. The context
leverage matters as much as the cost: a resumed developer remembers its
derivations and can refute erroneous rejection claims instead of blindly
re-implementing, and LEARNINGS accumulate richer across rounds. The resumed
agent's SHELL STATE IS FRESH -- cwd and env vars reset -- which is why its
first action is to cd back into its worktree. Transcripts grow with each
round, hence the 2-resume cap.

**Worktree retention.** The story worktree is the resume anchor: while
`loop.agent_resume` is enabled, do NOT remove the dev worktree at delivery.
It persists across rejection rounds until the story is accepted and merged
(or recovery removes it, which simply forces the fresh-spawn fallback).
Because the retained dev worktree keeps `story/<STORY_ID>` checked out, the
PM reviews the story DETACHED (`git checkout --detach story/<STORY_ID>`) --
git locks the branch ref, not the commit.

### Abandonment Detection (after every agent completion)

An "Agent completed" notification does NOT mean the work finished. An
ephemeral agent that backgrounds a long build (or otherwise ends its turn
to "wait") is silently disposed -- typically a few minutes in, with intact
but uncommitted work. The guard blocks backgrounded Bash whenever a loop is
active or dispatcher mode is on, but verify anyway. On every developer
completion:

1. Check for a terminal outcome: `delivered` label (`pvg nd show <id>`),
   or an explicit report in the agent output (ALREADY_LANDED,
   DISCOVERED_BUG, CONTEXT_BUDGET note).
2. If none AND the story branch has no new commits: the agent was
   abandoned. Re-spawn into the same worktree ONLY after the worktree-reuse
   safety steps (verify no processes, stop its containers), with explicit
   instructions: fully synchronous execution, explicit timeouts, COMMIT the
   work before any long verification.
3. If work was committed but not delivered: spawn a deliver-only follow-up
   (verify + `pvg story deliver`), which is cheap on a warm build.

PM completions: verify the decision actually landed (`pvg nd show <id>`
must show closed+accepted, rejected, or red-approved) before acting on it.

The loop also detects stalls structurally: when the same in_progress story
set is observed for 3 consecutive wait evaluations, `pvg loop next` returns
the `stalled` decision with the story ids and worktrees in its payload. Run
`pvg loop recover`, then either re-spawn each affected story or release it
(`pvg story release <id>`). Recovered stories are released (claim cleared,
back to open) and unmerged story branches are preserved, not deleted --
committed work survives recovery.

You MAY use the issues CLI directly for:
- Reading story content before spawning a developer (`pvg issues show STORY_ID`)
- Checking story labels (`pvg issues show STORY_ID --json`)
- Bug triage routing (DISCOVERED_BUG blocks)
- Epic auto-close checks after PM acceptance

### nd and vlt usage

For nd CLI reference (commands, flags, dependencies, priorities), read the nd skill:
  Skill tool: nd

For vault operations (read notes, create notes, search, frontmatter), read the vlt skill:
  Skill tool: vlt-skill

Do NOT guess nd flags or command syntax. The skill has the complete CLI reference
with examples. Common mistakes prevented by reading the skill:
- Priority uses the P-prefixed form: `pvg issues create --priority P0` (nd
  accepts P0-P4 natively)
- Dependencies use `pvg nd dep add/rm`, not flags on `pvg issues update`
- Comments use `pvg issues comment <id> <body>` or `pvg nd comments add`

Do NOT run `nd upgrade` -- it is guard-blocked inside Paivot repos. `pvg update`
is the only toolchain convergence path (the channel manifest pins nd, machinery,
vlt, modelith, and pvg together).

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

**Durability is automatic.** The Sr. PM writes new bugs (or stories) to the
live nd vault, which is NOT part of git history -- but every nd mutation
auto-snapshots locally to the `nd/backlog` git branch, so mid-epic creations
are durable the moment they land. No manual export or commit step is needed
here. The dispatcher's owned sync points are `pvg loop setup` (best-effort at
activation: failure is a WARN, never fatal), after each accepted story merge,
and at loop end -- each runs a full `nd sync` (snapshot + fetch + field-aware
merge + push of `nd/backlog`).
Never copy files out of the live vault by hand -- always go through
`pvg nd sync`.

**Note:** When `bug_fast_track` is enabled (or story has `pm-creates-bugs` label),
PM-Acceptor creates bugs directly during review. Only bugs from Developer agents
or from PM-Acceptor in centralized mode (the default) appear as DISCOVERED_BUG blocks.

### After PM-Acceptor Acceptance

**IMMEDIATELY after acceptance**: merge the story branch to epic (see Story
Merge below). Complete the merge -- including conflict resolution if needed --
before running `pvg loop next --json` again. An accepted story with an unmerged
branch is incomplete work.

Pre-merge checklist (each step its own command -- see Shell Chaining below):

1. **Release the story branch**: `git worktree list` -- if any worktree
   (including the retained dev worktree -- the resume anchor is no longer
   needed once the story is accepted -- or a lingering PM isolation worktree
   that checked out the story branch) still holds `story/<ID>`, remove it with
   `pvg worktree remove <path>` first. A held branch blocks deletion after
   merge, and a PM worktree left on the story branch is NOT auto-cleaned.
2. **Verify a clean tree**: `git status --porcelain` must be empty (untracked
   noise aside).
3. Checkout the epic branch, THEN merge -- as separate commands.

After the merge completes, run `pvg nd sync` -- the accepted-story merge is
one of the dispatcher's owned sync points (it snapshots, fetches, merges, and
pushes the `nd/backlog` branch). Then clear the story's recorded agent
handles: `pvg loop agent clear STORY_ID` (both roles -- see Semi-Persistent
Story Agents).

**Shell Chaining (HARD RULE):** never chain `git checkout` and `git merge`
with `;` -- if the checkout aborts (dirty tree), the merge still runs on
whatever branch HEAD is actually on. This has landed a story directly on
main. Use separate Bash calls (preferred) or `&&`. The merge guard rejects
`;`-chained checkout+merge structurally, judging the merge against the real
current branch.

## Epic Flow

The loop drains one epic at a time:

1. **Start**: auto-selects the highest-priority epic with actionable work
2. **Execute**: all parallelization happens WITHIN the current epic
   (multiple developers on different stories, one PM reviewing)
3. **Complete**: when all stories are accepted and merged to the epic branch,
   `pvg loop next --json` returns `epic_complete`
4. **Gate**: run the epic completion gate (e2e tests + Anchor milestone review + merge to main)
5. **Retro**: spawn `paivot-graph:retro` to extract learnings before rotating
6. **Rotate**: call `pvg loop rotate <next_epic>` to transition loop state, then continue iterating

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

**`.vault/` tracking:** developers never stage anything under `.vault/` (see
Git Hygiene in agents/developer.md). `.vault/knowledge/` IS tracked, but it is
committed only by the DISPATCHER on main -- after retro. Backlog durability is
nd-native: every nd mutation auto-snapshots to the `nd/backlog` git branch,
synced by `pvg nd sync` (the legacy export to `.vault/backlog-snapshot/` is
retired). Runtime state under `.vault/` (issues, locks, guard logs) stays
gitignored.

### Remote Detection (MANDATORY first step)

Before any branch operation, detect whether a remote exists:

```bash
HAS_REMOTE=$(git remote | head -1)
```

If `$HAS_REMOTE` is empty, the repo is **local-only**. Apply these rules to
ALL git commands in every section below (story setup, merge, conflict resolution,
epic completion, cleanup):

- **Skip entirely:** `git fetch`, `git pull`, `git push`, `git push --delete`
- **Replace `origin/BRANCH` with `BRANCH`:** e.g., `origin/epic/X` becomes `epic/X`,
  `origin/story/Y` becomes `story/Y`, `origin/main` becomes `main`
- **Skip remote branch cleanup:** `git branch -r --list` and `git push origin --delete`

The examples below show the remote case. Adapt for local-only as described above.

**Your responsibilities as dispatcher:**

### Story Branch Setup

Before spawning a developer:

Branch creation is NON-SWITCHING (`git branch`, never `git checkout -b`):
the dispatcher's HEAD stays on main, and a checked-out story branch would
also block the `pvg worktree add` that follows. The guard rejects story/*
checkouts at the project root structurally (like all six coordination guards,
it is active whenever a loop is active or dispatcher mode is on). `pvg loop
setup` enables dispatcher mode automatically when it is not already on, so a
bare `/piv-loop` always runs with full guard coverage and agent tracking.
`pvg loop cancel` disables dispatcher mode only if the loop was what enabled
it; a dispatcher the user enabled independently is left on.

**With remote:**
```bash
git fetch origin
if ! git rev-parse --verify origin/epic/EPIC_ID >/dev/null 2>&1; then
  git branch epic/EPIC_ID origin/main
  git push -u origin epic/EPIC_ID
fi
git branch story/STORY_ID origin/epic/EPIC_ID
git push -u origin story/STORY_ID
```

**Local-only (no remote):**
```bash
if ! git rev-parse --verify epic/EPIC_ID >/dev/null 2>&1; then
  git branch epic/EPIC_ID main
fi
git branch story/STORY_ID epic/EPIC_ID
```

Then CLAIM the story and create a worktree for the developer on the story branch:

```bash
pvg story claim STORY_ID    # atomic nd claim; MANDATORY before spawning
pvg worktree add .claude/worktrees/dev-STORY_ID story/STORY_ID
```

**Always create developer/PM worktrees with `pvg worktree add`, never raw
`git worktree add`.** `pvg worktree add` stamps an ownership marker
(`paivot-owned`) into the worktree's git admin dir. `pvg loop recover` (and
`pvg worktree remove`) remove a worktree, and delete its branch, ONLY when it
carries that marker. So a concurrent NON-Paivot Claude Code session that creates
its own worktree -- even under `.claude/worktrees/` -- is never removed by
recover. A raw `git worktree add` here would leave the worktree UNMARKED, and
recover would then treat your own developer worktree as foreign and refuse to
clean it up.

**Claiming at dispatch is not optional.** `pvg story claim` is atomic: it
delegates to `nd claim`, which moves the story to in_progress and records the
claiming agent (`dev-<id>`) in one step. There is no race window -- if the
claim fails, another agent already holds the story: skip it and move on, do
not retry or force it. This applies to EVERY developer spawn, including each
entry of a wave. To hand a claimed story back to the ready queue (for
example, after abandoning a spawn), run `pvg story release STORY_ID` -- it
returns the story to open and clears the claim.

**Provision the isolated environment (if `.paivot/envr` is present).** Right
after `pvg worktree add` for the story, if `.paivot/envr` is executable, run
`.paivot/envr up STORY_ID`, capture its `KEY=VALUE` stdout, and inject it into
this developer's prompt as ISOLATED INFRASTRUCTURE -- replacing the shared
connection string for that developer. Tear it back down with
`.paivot/envr down STORY_ID` whenever you remove the story's worktree. The token
is the story id; idempotent. See Per-Story Environment Isolation below.

**Never re-spawn a developer into a worktree whose previous occupant may
still be alive.** A killed or "completed" background developer can leave a
long-running build or test process (e.g. an in-container `mix test`) holding
build locks; a second developer in the same worktree then deadlocks on the
shared build state. Before reusing a worktree: verify the prior agent is
gone, stop any containers it started (`docker compose down` in the
worktree), and prefer removing and re-creating the worktree over reuse.

**Shell-context pinning (REQUIRED in every spawn prompt):** the harness can
reset an agent's Bash CWD to the project root between tool calls, silently
defeating worktree isolation -- an "isolated" agent then runs git/make/test
against the dispatcher's checkout. Every PM/developer prompt MUST instruct:
prefix EVERY shell command with `cd <worktree-absolute-path> &&`; never rely
on a previous cd. For docker-compose projects WITHOUT `.paivot/envr`, also pin
`COMPOSE_PROJECT_NAME=<story-id>` in those commands so concurrent agents never
join each other's compose project. When `.paivot/envr` is present, the project's
engine already isolates per token, so this manual pin is unnecessary.

The developer prompt MUST include the worktree path so they know where to
work, and MUST state: run everything synchronously with explicit timeouts
(never `run_in_background` -- ending your turn disposes you), and commit
work to the story branch before long verification runs:
```
Work in: /path/to/repo/.claude/worktrees/dev-STORY_ID
```

Pre-spawn invariant for every Developer and Conflict-fix agent:

- parent dispatcher CWD is the project root
- `story/STORY_ID` exists before the agent is spawned
- `.claude/worktrees/dev-STORY_ID` exists and is checked out to `story/STORY_ID`
- the developer prompt says `Work in: /abs/path/.claude/worktrees/dev-STORY_ID`
- the developer is told not to create or checkout another branch

For a parallel wave, create all story branches and all `dev-STORY_ID` worktrees
before spawning the developers. Each developer gets exactly one distinct worktree
path. If any worktree cannot be created, do not spawn the wave; repair the branch
or worktree state first.

**CRITICAL: Never use `isolation: "worktree"` for Developer or Conflict-fix agents.**
It creates an auto-generated `worktree-agent-*` branch DISCONNECTED from the story
branch. Commits on this orphan branch are lost when cleaned up. Always create developer
worktrees manually on the story branch as shown above.

**PM-Acceptors are the exception** -- they use `isolation: "worktree"` because they
make no tracked file changes. See "PM Isolation" in the Worktree Lifecycle section.

Developer works on `story/STORY_ID` branch. They commit and push there.

### Story Merge (After PM Approves)

**STRUCTURAL GATE:** `pvg guard` blocks `git merge story/*` unless the story is both labeled `accepted` and `closed` in nd. This is enforced by the PreToolUse hook in Paivot-managed repos. If the merge is blocked, let PM-Acceptor finish review first.

**CRITICAL:** Merging is your IMMEDIATE next step after PM acceptance. Complete the merge (including conflict resolution) before moving to the next priority item. A story that is accepted in nd but not merged in git is incomplete work.

After PM-Acceptor adds `accepted` and closes the delivered story:

**Step 1: Attempt the merge**

With remote:
```bash
git fetch origin
git checkout epic/EPIC_ID
git pull origin epic/EPIC_ID
git merge --no-ff origin/story/STORY_ID -m "merge(epic/EPIC_ID): integrate STORY_ID"
```

Local-only:
```bash
git checkout epic/EPIC_ID
git merge --no-ff story/STORY_ID -m "merge(epic/EPIC_ID): integrate STORY_ID"
```

**Step 2a: Merge succeeded** -- push (if remote) and clean up:

With remote:
```bash
git push origin epic/EPIC_ID
git push origin --delete story/STORY_ID
git branch -D story/STORY_ID
```

Local-only:
```bash
git branch -D story/STORY_ID
```

**Step 2b: Merge conflict** -- abort, stay on epic, spawn developer, retry:

Do NOT checkout main. Do NOT move to another priority item. Handle inline.

```bash
# 1. Abort the failed merge. Stay on the epic branch.
git merge --abort
# You are still on epic/EPIC_ID. Do NOT checkout main or any other branch.
```

```
# 2. Spawn developer for conflict resolution. Use this exact prompt
#    (adapt origin references for local-only repos per Remote Detection):
CONFLICT RESOLUTION MODE. Story STORY_ID is accepted but cannot merge
into epic/EPIC_ID due to conflicts.

Your task: rebase story/STORY_ID onto the latest epic/EPIC_ID, resolving
all conflicts.

Steps:
1. git fetch origin  (skip if local-only)
2. git checkout story/STORY_ID
3. git rebase epic/EPIC_ID  (use origin/epic/EPIC_ID if remote exists)
4. Resolve conflicts in each file (keep functionality from both sides)
5. git rebase --continue after each resolution
6. Run tests to verify nothing is broken
7. git push --force-with-lease origin story/STORY_ID  (skip if local-only)

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

**Merge order:** If multiple stories are waiting to merge, process them in dependency order first, then priority order (P0 first) within each ready layer. Do NOT use `parent` for this: `parent` is epic containment, not the dependency graph. Use `pvg nd dep tree STORY_ID` (nd-specific) and `pvg issues show STORY_ID --json` to inspect `blocked_by`, `blocks`, and `follows`; merge prerequisite stories before dependents.

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

**Step 1b: Deferral Sweep (STRUCTURAL -- before Step 2)**

Before the gate may pass, sweep ALL accepted stories in the epic for named
deferral targets:

```bash
pvg nd children EPIC_ID --json    # enumerate the epic's stories
pvg nd show <id>                  # inspect each story's notes and evidence
```

Search story notes, comments, and delivery evidence for phrases like
"deferred to epic gate", "deferred to story X", "will be verified at epic
close". Every named deferral target must have demonstrably FIRED -- the
deferred verification actually ran, with evidence (test output, demo result,
verification note). An unfired deferral is a gate FAILURE: spawn a developer
to execute the deferred verification, or escalate to the user via
AskUserQuestion. Field finding: an epic closed with named deferrals that
never fired.

**Step 1c: Project Review Workflow (STRUCTURAL when the project declares one)**

```bash
pvg settings review.command      # empty = the project declares none; skip this step
```

When it is set, YOU run that command here, on the epic branch, before the Anchor. The
dispatcher is the only role that can: review workflows fan out to specialist subagents,
and the developer and PM-Acceptor are ephemeral and cannot spawn. This is where a
stack's mandated review (for example an Elixir project's `/phx:review`, followed by
`/phx:triage` on its findings) actually executes instead of remaining a line of prose in
a story.

Feed the triaged findings back as developer fixes on the epic branch, then re-run. A
review workflow left unrun, or run with findings unaddressed and unrecorded, is a gate
FAILURE exactly like a failing test.

**Step 2: Anchor Milestone Review**

Spawn `paivot-graph:anchor` in milestone review mode:

```
MILESTONE REVIEW for epic EPIC_ID.

Validate that the completed epic delivered real value:
- Inspect tests for mocks in integration/e2e tests (forbidden)
- Verify skills were consulted where stories required them
- Check that boundary maps are satisfied (PRODUCES/CONSUMES)
- Validate hard-TDD two-commit pattern where applicable
- Verify wiring evidence: every delivered plug/middleware/worker/component
  is MOUNTED (router entry, supervision-tree child, template/config usage)
  with at least one test exercising it THROUGH the wiring
- Audit deferrals: every named deferral target across accepted stories has
  FIRED (independent re-check of the dispatcher's deferral sweep)
- Verify remote CI: gh run list shows green for the epic's merged work --
  container-local test runs are insufficient

Epic branch: epic/EPIC_ID
```

Anchor verdicts are prefixed for reliable parsing: milestone reviews return
`REVIEW_RESULT: VALIDATED` or `REVIEW_RESULT: GAPS_FOUND`; backlog reviews
return `REVIEW_RESULT: APPROVED` or `REVIEW_RESULT: REJECTED`. (The Sr-PM/
Anchor backlog review loop caps at 3 rounds; after that, escalate the
remaining findings to the user.)

If the Anchor returns `REVIEW_RESULT: GAPS_FOUND`, address the gaps (spawn
developer to fix, or escalate to user) before proceeding. Do NOT merge to
main with open gaps. Proceed only on `REVIEW_RESULT: VALIDATED`.

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
# ALWAYS use -D (force). -d will fail because the remote tracking ref
# origin/epic/EPIC_ID still exists even though the branch is merged to HEAD.
git push origin --delete epic/EPIC_ID
git branch -D epic/EPIC_ID
```

**Remote CI verification (precondition of declaring the gate passed):**
container-local test runs do NOT count as "CI green". After pushing main,
verify the REMOTE CI is green for the pushed SHA:

```bash
gh run list --branch main --limit 5
```

Check that the latest run for the pushed SHA concluded `success`. If it is
still running, wait and poll until it concludes. If the repo has no remote
or no CI workflows, note that explicitly in the gate output and continue.
Otherwise a red or missing remote CI run BLOCKS the gate: spawn a developer
to fix the failure (treat it like a Step 1 verification failure) before
closing the epic. Field finding: an epic closed with 3 total GitHub runs,
all red, while every "CI green" claim was container-local.

**After** branch cleanup succeeds, close the epic in nd. The label contract
requires the epic to be closed BEFORE the `accepted` label is added -- two
canonical steps, in this order:
```bash
pvg nd close EPIC_ID --reason="All stories accepted, gate passed"
pvg nd update EPIC_ID --add-label accepted
```

Do NOT run nd updates in parallel with branch deletes. If the branch delete
errors, Claude Code cancels sibling parallel calls -- losing the nd update.

**Then sync the backlog branch.** The live nd vault lives under git-common-dir
and is NOT part of git history; durability is nd-native. Every nd mutation
auto-snapshots locally to the `nd/backlog` git branch, and `pvg nd sync`
delegates to `nd sync`: snapshot + fetch + field-aware merge + push of that
branch. Run it here, as at every accepted story merge and at loop end:

```bash
pvg nd sync            # snapshot + fetch + merge + push of nd/backlog
pvg nd sync --status   # show local/remote position without syncing
pvg nd sync --no-push  # sync but skip the push
```

`--commit` remains as a deprecated alias for a plain sync; the old export to
`.vault/backlog-snapshot/` is retired. `pvg doctor` now runs an nd-sync-status
check instead of the old snapshot-drift check.

(`pvg nd restore` delegates to `nd sync --restore`: it rebuilds a wiped live
vault from the `nd/backlog` branch, with a legacy snapshot fallback -- see
docs/LIVE_SOR.md.)

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

```bash
# Clean up PM isolation branches (worktree-agent-* left behind by Claude Code)
git branch -r --list "origin/worktree-agent-*" | sed 's|remotes/origin/||' | while read br; do
  git push origin --delete "$br" 2>/dev/null || true
done
git branch --list "worktree-agent-*" | while read br; do
  git branch -D "$br" 2>/dev/null || true
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
  --body "All stories accepted. Full test suite passing. Anchor review: REVIEW_RESULT: VALIDATED."
```

If your environment provides PR automation, use it and continue unattended.
Otherwise stop after the PR is created and ask the user to complete or
approve the merge. Branch cleanup happens after the PR is merged.

**Step 4: Retro**

After merging to main, spawn `paivot-graph:retro` to extract learnings:

```
EPIC RETRO for epic EPIC_ID.

Extract LEARNINGS from all accepted stories in this epic. Analyze patterns
across developer delivery notes and PM review feedback. Distill actionable
insights and write them to the project vault (.vault/knowledge/).

Epic: EPIC_ID
```

The retro agent is ephemeral -- it runs, captures knowledge, and is disposed.
Do NOT skip this step. Do NOT rotate to the next epic before retro completes.

**After retro completes**, commit any new `.vault/knowledge/` files it produced
to main. Knowledge notes are tracked; runtime state under `.vault/` (issues,
locks, guard logs) remains gitignored. Agents never commit `.vault/` files --
this commit is the dispatcher's job, on main:

```bash
git add .vault/knowledge
git commit -m "chore(paivot): retro knowledge for EPIC_ID"
git push origin main   # skip if local-only
```

**After retro**: if the decision carried a `seal_epic`, run the Milestone Seal Gate
below BEFORE rotating. Then, if `epic_complete` included a `next_epic`, run
`pvg loop rotate <next_epic>` to transition the loop state and resume
with `pvg loop next --json`. If no `next_epic` was provided (last epic),
the completion gate is still MANDATORY -- run all five steps (e2e, project review,
Anchor, merge to main, retro) before allowing exit. The stop hook enforces this
structurally: it blocks exit while the epic branch exists unmerged. While an
epic-mode loop is active, the stop hook's counts are epic-scoped: the
completion gate fires when the TARGET epic drains, regardless of other
epics' state.

### Milestone Seal Gate (nested epic model)

A layered plan nests epics: a MILESTONE epic per layer holding SLICE epics as its
children. The loop drains slices one at a time, each through the full completion gate
above. The milestone itself is never a dispatch target and never closes by story
acceptance; it seals once, after its last slice closes. `pvg loop next --json` names the
milestone in `seal_epic` and returns `milestone_seal` when only the seal remains. The
stop hook blocks exit while a seal is pending, exactly as it does for an unmerged epic
branch.

Run these four steps on `main`, after the last slice has merged:

1. **Whole-design gate.** This is the one place the design's whole-suite test gate runs:
   ```bash
   pvg gates --seal
   ```
   It forces machinery's Gt-tests (every committed oracle stable id, machine transitions
   AND formal decision rows, carried by the suite) and G4-import in over the configured
   impl dir. Story-level RED approval deliberately omits Gt, because at story
   granularity it would block the first story until the last one exists; the seal is
   where that debt comes due, and it is never waived. A red result is a FAILED seal:
   spawn developers for the uncovered ids, or escalate via AskUserQuestion. Never
   `--skip-design` past it.

2. **Milestone DoD.** Read the milestone epic's body: it carries the layer DoD verbatim
   from the build plan. Every line must be demonstrably satisfied, with evidence.

3. **Anchor seal review.** Spawn `paivot-graph:anchor` in milestone review mode against
   the MILESTONE epic, naming its DoD lines and its slice epics. Its deterministic
   pre-pass (`pvg gates`, scoped `pvg rtm`, `pvg verify --check-e2e`,
   `pvg verify --check-mocks`) runs over the whole layer, not one slice.
   `REVIEW_RESULT: VALIDATED` is required; `GAPS_FOUND` re-enters delivery.

4. **Close the milestone.** Only after 1 to 3 are green, and in this order (the label
   contract requires closed before accepted):
   ```bash
   pvg nd close <milestone-epic> --reason="Layer sealed: DoD green, whole-design gate green, Anchor VALIDATED"
   pvg nd update <milestone-epic> --add-label accepted
   ```
   Tag the seal commit so later design revisions can diff against it
   (`pvg story sync-oracle --base <seal-tag>`).

From the seal onward the layer's tests are locked at layer granularity: a change inside
a sealed layer starts with a design revision, not with an edit to a locked test.

## Dispatcher Rules

You are a dispatcher. You coordinate agents and manage git integration. You NEVER:
- Write source code or tests yourself
- Fix errors or bugs yourself
- Modify story files yourself
- Make architectural decisions yourself
- Skip agents to "save time"
- Edit source files for any reason, including "cleanup" or "git maintenance"
- Inspect agent worktree internals (cd into `.claude/worktrees/*`, run git log, read files there)
- Continue or resume a FAILED developer agent -- clean up the worktree, clear its handle (`pvg loop agent clear`), and re-spawn fresh. (Loop-directed resume is different: a `resume_agent` action targets an agent whose delivery the PM rejected, not one that failed -- see Semi-Persistent Story Agents)
- Re-close stories that the PM-Acceptor already closed (it closes on acceptance -- you just read its output)
- Override, re-interpret, or bypass PM rejections -- if the PM rejected, the story goes back to the developer with the rejection feedback. You do not get to decide the rejection was "on a technicality" or "procedural." PM decisions are final.
- Re-submit rejected stories for acceptance without developer rework -- the developer must address the rejection feedback and re-deliver
- Call `pvg loop cancel` -- only the user can cancel the loop. You do not get to decide when to stop based on "context exhaustion," "productivity," "session length," or any other self-assessed risk. The stop hook (`pvg hook stop`) handles exit decisions automatically when actionable work is exhausted. If you try to end your response, the stop hook evaluates and blocks you if work remains.
- Query nd globally for dispatch decisions (use `pvg loop next --json` instead)

**Rejection cap:** after 3 PM rejections of the same story, `pvg loop next`
emits the `escalate` decision instead of another rework action. Surface the
story and its rejection history to the user via AskUserQuestion and wait for
direction. The PM's verdicts stand -- the dispatcher NEVER overrides the PM.

**You DO manage git:** Creating epic/story branches, creating/removing worktrees, merging story->epic after PM approval, running the epic completion gate (e2e + Anchor review), merging epic->main (solo-dev) or creating PRs (team), cleaning up branches, and resolving merge conflicts (by spawning developer if conflicts arise).

### When a Developer Agent Fails

If a developer agent fails, returns partial output, or times out:
1. Check story status via `pvg issues show <STORY_ID> --json` (NOT by inspecting the worktree)
2. If NOT delivered: run `cd $PROJECT_ROOT && pwd`, then `pvg worktree remove .claude/worktrees/dev-<STORY_ID>`, then `pvg loop agent clear <STORY_ID> developer` (a failed agent's conversation is as suspect as its workspace), then re-spawn a fresh developer with corrective guidance and record the new handle with `pvg loop agent set`. If you re-provisioned the env per story (`.paivot/envr`), the existing env is reused on re-spawn (`up` is idempotent); only tear it down once you abandon the story (`[ -x .paivot/envr ] && .paivot/envr down <STORY_ID>`).
3. If delivered: run `cd $PROJECT_ROOT && pwd`, then `pvg worktree remove .claude/worktrees/dev-<STORY_ID>` and `pvg loop agent clear <STORY_ID> developer` (an agent that failed on the way out is never resumed -- any later rework takes the fresh-spawn path), then proceed with PM review
4. NEVER cd into the worktree to check what happened, run git log, or try to continue the agent

The developer's worktree is their workspace. If they failed, their workspace is suspect.
Clean up and start fresh -- re-doing work is cheaper than debugging partial state.

## Infrastructure Context (MANDATORY before first developer spawn)

Before spawning a developer, give it the infrastructure its integration tests
need. There are two modes; pick by whether the project ships an `.paivot/envr`
script (see Per-Story Environment Isolation below).

**Mode A -- per-story isolated environments (`.paivot/envr` is executable).**
This SUPERSEDES the single shared connection string. Each story gets its OWN
environment, provisioned at worktree-add time via `.paivot/envr up <story-id>`,
and its `KEY=VALUE` stdout is injected into THAT developer's prompt as
ISOLATED INFRASTRUCTURE. Do not run shared `docker ps` discovery and do not
inject one global connection string in this mode. The full lifecycle (provision
on prepare, tear down on close) is in Per-Story Environment Isolation below.

**Mode B -- shared-infra discovery (no `.paivot/envr`).** Behaves exactly as
before: discover what is available locally once, and include those connection
details in ALL developer prompts.

**Discovery protocol (Mode B only):**
1. `docker ps --format '{{.Names}} {{.Ports}}'` -- running containers
2. Check for docker-compose files, .env files with connection strings
3. Check project README/docs for infrastructure requirements

**Include in developer prompts:**
- The connection details the story's tests need -- ISOLATED INFRASTRUCTURE from
  `.paivot/envr up <story-id>` in Mode A, or the shared services discovered in
  Mode B (list of running services with host:port, database connection details,
  required env vars with values or instructions to obtain them)
- Explicit instruction: "Infrastructure is provided above. Do NOT gate tests
  behind env vars. Run integration tests directly against it."

Without this context, developers will reasonably gate tests behind env vars --
creating dormant tests that satisfy no testing gate.

## Per-Story Environment Isolation (when `.paivot/envr` is present)

When developers run in parallel on a monorepo whose integration tests need
shared infrastructure (databases, brokers, clusters), one global environment
serializes the wave: stories contend on the same resource. A project can opt out
of that contention by shipping an executable `.paivot/envr` script (sibling of
`.paivot/config.yaml`). Paivot owns the contract; the project owns the engine --
the methodology never names Docker, k8s, or ports. See
[docs/ENV_ISOLATION.md](../docs/ENV_ISOLATION.md) for the full contract and
copy-pasteable engines.

**The contract** (a project-provided executable, token = story id):
- `.paivot/envr up <token>` -- provisions an isolated environment keyed by
  `<token>`; prints connection details to stdout as `KEY=VALUE` lines (one per
  line; `#`-prefixed comment lines allowed); idempotent (safe to re-run for the
  same token); exits 0 on success.
- `.paivot/envr down <token>` -- tears the environment down; idempotent (safe if
  already gone); exits 0.

**Graceful degradation.** If `.paivot/envr` does not exist or is not executable,
do nothing here -- fall back to Mode B shared-infra discovery above. Fully opt-in
and backward-compatible.

**Lifecycle -- bracket `envr` around the SAME per-story worktree lifecycle.** The
token is the story id, which is already unique per concurrent worktree, so two
stories never share a token.

- **On prepare** (immediately after `pvg worktree add` for the story -- see Story
  Branch Setup): if `.paivot/envr` is executable, run `.paivot/envr up <story-id>`,
  capture its stdout, and inject it into that developer's prompt verbatim as the
  story's ISOLATED INFRASTRUCTURE (your environment). This REPLACES the shared
  single connection string for that developer:

  ```bash
  if [ -x .paivot/envr ]; then
    .paivot/envr up STORY_ID    # capture stdout -> developer prompt
  fi
  ```

  ```
  ## ISOLATED INFRASTRUCTURE (your environment)
  # Provisioned for this story by .paivot/envr up STORY_ID.
  # Use ONLY this; do not reach for shared/global infrastructure.
  <KEY=VALUE lines from .paivot/envr up STORY_ID>
  ```

- **On close** (story accepted, abandoned, or otherwise cleaned up -- wherever you
  run `pvg worktree remove` for the story): tear the environment down alongside
  worktree removal. Idempotent, so it is safe even if the env was never up:

  ```bash
  if [ -x .paivot/envr ]; then
    .paivot/envr down STORY_ID
  fi
  ```

In this mode you do NOT inject one shared connection string into all developers,
and the per-developer `COMPOSE_PROJECT_NAME` shell-pin is unnecessary -- the
project's engine already isolates per token.

## Context Injection Protocol (MANDATORY before developer spawn)

Before spawning ANY developer agent, the dispatcher MUST enrich the prompt with
concrete codebase context. Advisory instructions like "search for existing modules"
are unreliable -- subagents skip them. Instead, the dispatcher reads the codebase
and INJECTS the context directly into the developer prompt. This is structural, not advisory.

### Step 1: Parse the story's CONSUMES block

Read the story (`pvg issues show <id>`) and extract all CONSUMES entries. Each entry
names an upstream module or file.

### Step 2: Extract API signatures from consumed modules

For each consumed module/file, read it and extract:
- Module name and one-line @moduledoc summary
- All `@spec` annotations on public functions
- Key `@doc` usage examples

Include these as a "CODEBASE CONTEXT" section in the developer prompt:
```
## CODEBASE CONTEXT (injected by dispatcher -- use these APIs)

### <ModuleName> (<file_path>)
<one-line summary>

Public API:
  @spec function_name(arg_types) :: return_type
  # Usage: Module.function_name(arg1, arg2)
```

### Step 3: Scan ACs for cross-cutting keywords

Scan the story's acceptance criteria for keywords that indicate cross-cutting
concern integration is needed:

| Keyword | Module to discover | What to inject |
|---------|-------------------|----------------|
| DLP, scan, credential, PII | Gateway DLP/security module | scan/2 API + severity handling |
| rate limit, throttle | Gateway rate limiter | check/3 API + config key pattern |
| config, configuration | Project config module | How to add runtime keys + defaults |
| audit, log, telemetry | Observability module | Event emission pattern |
| allowed_paths, security | Path validation module | validate_allowed pattern |

For each keyword found, grep the codebase:
```bash
grep -rl "defmodule.*DLP\|defmodule.*RateLimiter\|defmodule.*Config" lib/
```

Read the discovered modules and inject their public APIs into the developer prompt.

### Step 4: Inject existing patterns from accepted stories

If the story follows a walking skeleton (earlier accepted stories produced similar
modules), read one accepted module as a TEMPLATE:
```
## TEMPLATE: Follow this pattern (from <accepted_module>)

<paste the first 30 lines showing module structure, use Jido.Action, @spec, etc.>
```

This prevents the "bad pattern propagation" problem where developers copy incomplete
patterns from early stories.

### What the developer prompt looks like after injection

```
STORY: <full story content>

CODEBASE CONTEXT (injected by dispatcher):
  <API signatures from CONSUMES modules>
  <Cross-cutting module APIs discovered from AC keywords>
  <Template from accepted walking skeleton>

INFRASTRUCTURE:
  <Running services, connection details>
```

The developer receives everything needed to implement WITHOUT searching the codebase.
This is the structural enforcement of "all context comes from the story" -- the
dispatcher ensures the context is actually complete.

## Agent Types

| Role | Agent Type | When |
|------|-----------|------|
| Sr. PM (bug triage) | `paivot-graph:sr-pm` | DISCOVERED_BUG blocks found in agent output |
| PM-Acceptor | `paivot-graph:pm` | Stories with `delivered` label |
| Developer | `paivot-graph:developer` | Ready or rejected stories |
| Retro | `paivot-graph:retro` | After epic completion gate passes (before rotation) |
| Anchor | `paivot-graph:anchor` | Backlog review or milestone review during epic gate |

### Per-Role Model Overrides

Each agent's model is set in its `agents/*.md` frontmatter by default:

| Role | Frontmatter default |
|------|---------------------|
| business-analyst, designer, architect | fable |
| ba-challenger, designer-challenger, architect-challenger | fable |
| sr-pm, anchor | fable |
| developer | opus |
| pm, retro | sonnet |

Projects can override the model per role via `pvg settings model.<role>` (empty
= use the agent's built-in default). The override is passed at spawn time as the
Agent tool `model` parameter; no agent file is edited and the override survives
plugin updates.

- **Agents spawned by the loop** (Developer, PM-Acceptor): the
  `model.developer` / `model.pm` override is surfaced directly on each loop
  action as the `model` field -- pass it through as described under the `act`
  decision and Wave Dispatch. Do not read settings yourself for these
  loop-surfaced roles.
- **EVERY other spawn** (BA, Designer, Architect, the three challengers,
  Sr-PM, Anchor, Retro): before spawning, run `pvg settings model.<role>` and
  pass a non-empty value as the Agent tool `model` parameter. The role keys
  are `model.ba`, `model.designer`, `model.architect`, `model.ba_challenger`,
  `model.designer_challenger`, `model.architect_challenger`, `model.sr_pm`,
  `model.anchor`, and `model.retro`. When the setting is empty, spawn
  normally (the frontmatter default applies).

A resumed agent keeps the model it was spawned with -- model overrides apply
only to fresh spawns.

## Developer Spawning: Normal vs Hard-TDD

Hard-TDD is **opt-in per story**. Before spawning a developer, check for the `hard-tdd` label:

```bash
pvg issues show <id> --json | grep -q '"hard-tdd"'
```

On projects where the user has explicitly enabled `design.machinery`
(`pvg settings design.machinery=on`, or `auto` as a deliberate user
choice to re-enable artifact detection; the default is `off`, and machinery
artifacts on disk enable nothing by themselves), the Sr PM applies the
`hard-tdd` label to oracle-citing stories at backlog creation, and
`pvg lint --backlog` gains the deterministic `hard-tdd-oracle` check --
ERROR when a story cites oracle stable ids without the `hard-tdd` label.
The label remains the switch; only who applies it changes. Enabling
machinery is a user decision with significant token and time cost: an agent
that believes the project would benefit surfaces a recommendation with those
costs (in an unattended loop, recorded as a story comment or note), and
NEVER runs `pvg settings design.machinery=...` itself.

A build protocol can put the whole build under hard-TDD, in which case the
user sets `hard_tdd.preauthorized=true` and `pvg lint --backlog` requires the
label on EVERY non-closed story (or `hard-tdd-exempt` plus a recorded
`HARD-TDD EXEMPT:` reason). That closes the gap where a visual-regression,
metamorphic, or fuzz suite cites no oracle id: it still runs two-phase.
Nothing changes here, the label is still the switch, but expect nearly every
story on such a project to carry it.

**If `hard-tdd` label is ABSENT** (the default): spawn ONE developer agent in normal mode.
The developer writes both implementation and tests in a single pass. This is the standard flow.

**If `hard-tdd` label is PRESENT**: run the two-phase flow. The phase is
tracked in nd by the `red-approved` label and carried on every loop action
as `phase` ("red" or "green") -- trust the loop output, do not infer:

1. RED phase: `pvg loop next` returns `developer_new` with `"phase":"red"`.
   Spawn developer with "RED PHASE" in the prompt (tests only). Developer
   delivers via `pvg story deliver`.
2. RED review: the loop returns `pm_review` with `"phase":"red"`. The PM
   validates the tests are properly RED (cover ACs, fail for the right
   reason) and approves with `pvg story approve-red STORY_ID` -- this
   removes `delivered`, adds `red-approved`, and returns the story to the
   ready queue. A RED story is NEVER closed or labeled `accepted`.
3. GREEN phase: the loop returns `developer_new` with `"phase":"green"`
   (same story, now labeled `red-approved`). Spawn developer with
   "GREEN PHASE" in the prompt (implementation only; RED tests untouched).
4. GREEN review: the loop returns `pm_review` with `"phase":"green"`.
   Standard acceptance applies (close + `accepted`).

A rejected story keeps its `red-approved` label, so rework actions carry the
correct phase automatically.

**GREEN is ALWAYS a fresh spawn.** NEVER resume the RED developer's
conversation for the GREEN phase, even though its handle is recorded. The
RED-to-GREEN boundary is a deliberate context wall: the implementation must
be constrained by the committed tests, not by the RED author's intent. pvg
enforces this structurally -- GREEN dispatches as a new-developer action,
which never carries `resume_agent` -- and the GREEN spawn overwrites the
recorded handle (`pvg loop agent set`). Resuming WITHIN a phase is fine: a
rejected RED delivery may resume the RED developer for RED rework, and a
rejected GREEN delivery may resume the GREEN developer.

**CI structural lock (optional).** The label flow governs what each agent does
per phase; to also prove from git history that GREEN commits never quietly
weakened the RED tests, projects can add the canonical guard
`pvg story verify-tdd` (CI wrapper `scripts/verify-hard-tdd.sh`). It fails when a
non-RED, unauthorized commit modifies or deletes an existing test file (adding a
new test file is allowed), and fails loudly when its range cannot be resolved
rather than passing silently. See
[docs/HARD_TDD_GUARD.md](../docs/HARD_TDD_GUARD.md).

**Do NOT default to hard-TDD.** The user's general TDD preference (writing tests alongside
code) is satisfied by normal mode. Hard-TDD is a stricter discipline where tests and
implementation are written by separate agent invocations with structural locks. It requires
explicit opt-in via the label, applied because the user requested or pre-authorized it.
(On projects where the user enabled `design.machinery`, the Sr PM applies the label to
oracle-citing stories -- the switch is still the label, never dispatcher judgment. For
everything else the Sr PM records a recommendation instead of applying the label on its
own.)

## Termination

The loop drains one epic at a time. The stop hook (`pvg hook stop`) evaluates
termination automatically. While an epic-mode loop is active its counts are
epic-scoped: the epic completion gate (merge epic branch, Anchor milestone
review, e2e, retro) fires when the TARGET epic drains, regardless of other
epics' state.

| Condition | Action |
|-----------|--------|
| No actionable epics remain AND epic branch merged | Allow exit, remove state |
| Current epic blocked, no other epics | Allow exit |
| Max iterations reached | Allow exit, remove state |
| Too many consecutive waits (3) | Allow exit (escape valve) -- loop state is PRESERVED; background agent completions re-invoke the dispatcher and resume the loop |
| Current epic has actionable work | Block exit, continue |
| Current epic complete, next epic exists | Block exit, run completion gate, then `pvg loop rotate` and continue |
| Current epic complete, NO next epic (last epic) | Block exit, run completion gate, then allow exit |
| Epic branch exists but all stories closed | Block exit, run completion gate (stop hook enforces this structurally) |

### Live Demo (before session exit -- MANDATORY)

Every session must produce demonstrable progress. Before the loop exits:

1. Identify what was delivered (accepted stories, completed epics, merged to main)
2. If anything was merged to main: run the project's demo, smoke test, or e2e suite
   on main and report results to the user
3. If nothing reached main: explain what blocked progress and what the user should
   do next

**External integration demo (NON-WAIVABLE):** If the completed epic includes stories
with the `external-integration` label, the live demo MUST verify the external service
interaction works against real endpoints -- not just mocked internal wiring. If the
demo environment cannot exercise the real API (e.g., mobile-only OAuth flow), you MUST:
1. Explicitly report this to the user: "External integration with [service] has not been
   verified against real endpoints. Automated tests verify internal wiring only."
2. Ask the user whether to (a) defer epic merge until manual verification, or
   (b) merge with a known gap and create a follow-up verification task.
Do NOT silently skip external integration demo because "tests pass." Tests with mocked
external APIs prove internal correctness, not operational readiness.

A session that cannot show working software at the end should be treated as a
signal that something is wrong with the backlog, the infrastructure, or the
test suite -- not as normal.

## Cancellation

Only the **user** can cancel a running loop. The dispatcher MUST NOT self-cancel.

User commands:
```
/piv-cancel-loop
```

Or directly:
```bash
pvg loop cancel
```

Cancellation restores the pre-loop dispatcher posture: dispatcher mode is
disabled only if `pvg loop setup` was what enabled it. A dispatcher the user
enabled independently stays on.

## Worktree Lifecycle

### Naming Convention

| Role | Worktree path | Branch | Managed by |
|------|---------------|--------|------------|
| Developer | `.claude/worktrees/dev-<STORY_ID>` | `story/<STORY_ID>` | Dispatcher (pvg) |
| Conflict fix | `.claude/worktrees/fix-<STORY_ID>` | `story/<STORY_ID>` | Dispatcher (pvg) |
| PM-Acceptor | _(auto-generated by Claude Code)_ | _(starts on auto branch, checks out story)_ | Claude Code (`isolation: "worktree"`) |

Developer and conflict-fix worktrees are dispatcher-managed on the story branch.
PM-Acceptor uses Claude Code's `isolation: "worktree"` for automatic lifecycle
management (see "PM Isolation" below).

### Cross-Session Isolation

Multiple Claude Code windows may run Paivot against the same repository at the
same time. This is supported only because every code-writing agent receives a
dedicated story worktree under `.claude/worktrees/`; the parent checkout's HEAD
must remain on the dispatcher branch and must not be used for agent work.

Never check out `worktree-agent-*` branches in the parent repository. Those are
Claude Code's transient isolation branches for PM/review shells, not story
branches. `pvg`'s PreToolUse guard blocks `git checkout worktree-agent-*` and
`git switch worktree-agent-*` while a loop is active or dispatcher mode is on,
because that
operation can reset a sibling Paivot window's shared HEAD and make in-flight
edits appear to vanish. Stale `worktree-agent-*` branches may be deleted
directly via `git branch -D` or `git push origin --delete`; they must not be
checked out.

### Parallel Developer Waves

Parallel fanout is allowed only when each developer has a dispatcher-managed
story worktree:

```bash
pvg worktree add .claude/worktrees/dev-STORY_A story/STORY_A
pvg worktree add .claude/worktrees/dev-STORY_B story/STORY_B
pvg worktree add .claude/worktrees/dev-STORY_C story/STORY_C
```

Then spawn one Developer per worktree. Every prompt must include the unique
absolute `Work in:` path. A post-fix wave must not create `-v2`/`-v3` collision
recovery branches, must not leave developer commits on `worktree-agent-*`
branches, and must not show staged files from a sibling story.

Smoke check -- the invariants any parallel wave must satisfy: a three-developer
wave with staged and committed files in separate story worktrees must show no
sibling staged files, no `-v2`/`-v3` collision-recovery branch suffixes, and no
developer `worktree-agent-*` branches, and every worktree must be removed at the
end. These are asserted by `scripts/smoke_parallel_dev_worktrees.sh`
(`make smoke-worktrees`).

### Full Flow

1. Dispatcher creates story branch from epic (see Story Branch Setup)
2. Dispatcher creates dev worktree: `pvg worktree add .claude/worktrees/dev-<STORY_ID> story/<STORY_ID>` (stamps the ownership marker)
3. Developer works, commits, pushes on `story/<STORY_ID>`
4. Developer marks delivered
5. Dispatcher resets to project root: `cd $PROJECT_ROOT && pwd`. With `loop.agent_resume` enabled (the default), KEEP the dev worktree -- it is the story's resume anchor across any rejection rounds (see Semi-Persistent Story Agents); it is removed at accept+merge (step 9) or by recovery. Only when resume is disabled, remove it now: `pvg worktree remove .claude/worktrees/dev-<STORY_ID>`, and if `.paivot/envr` is executable, tear down the story's environment: `[ -x .paivot/envr ] && .paivot/envr down <STORY_ID>` (idempotent). See Per-Story Environment Isolation.
6. Dispatcher spawns PM with `isolation: "worktree"` (see PM Isolation below)
7. PM checks out the story DETACHED (`git checkout --detach story/<STORY_ID>` -- the retained dev worktree holds the branch ref), reviews, accepts or rejects
8. Claude Code auto-cleans the PM worktree **directory** (PM makes no tracked file
   changes), but does **not** delete the `worktree-agent-*` branch. Delete it
   immediately after the PM agent completes:
   ```bash
   # Clean up the PM isolation branch Claude Code leaves behind
   git branch -r --list "origin/worktree-agent-*" | sed 's|remotes/origin/||' | while read br; do
     git push origin --delete "$br" 2>/dev/null || true
   done
   git branch --list "worktree-agent-*" | while read br; do
     git branch -D "$br" 2>/dev/null || true
   done
   ```
9. If accepted: remove the retained dev worktree (pre-merge checklist), merge
   story branch to epic, delete story branch, then clear the recorded handles:
   `pvg loop agent clear <STORY_ID>`
10. If rejected: the loop emits a rework action. When it carries `resume_agent`,
    resume the recorded developer per Semi-Persistent Story Agents -- the PM
    rejection comment content is the message body, and the retained dev
    worktree is the resume anchor. Otherwise (no handle, resume cap reached,
    resume disabled, or any resume failure) re-create the dev worktree if
    missing and re-spawn a fresh developer with the rejection feedback.
    (After the third rejection of the same story the loop emits `escalate` --
    see Dispatcher Rules)

### Cleanup Rules

**Always reset to project root before removal:**
```bash
cd $PROJECT_ROOT && pwd
pvg worktree remove .claude/worktrees/<worktree-name>
```

`pvg worktree remove` resolves the project root from the worktree path (not CWD),
runs `git worktree remove --force`, and prunes stale metadata. Starting in v1.52.11,
it also **refuses** removal if the caller's CWD is inside the target worktree --
this prevents the session-killing CWD corruption bug where removing a directory
that is also the shell CWD makes all subsequent Bash commands fail permanently.

The reset step is structural, not stylistic: after a developer or PM agent
completes, Claude Code may hand control back with the agent's worktree still as
the parent shell CWD. Reset first, then remove:
```bash
cd $PROJECT_ROOT && pwd
pvg worktree remove .claude/worktrees/<worktree-name>
```

`pvg worktree remove` is belt-and-suspenders after that reset: it resolves the
project root from the worktree path and refuses CWD-inside removal if the shell
is still standing inside the target worktree.

**Do NOT delete the story branch when removing a worktree.** The worktree is a checkout;
the branch is the record. Story branches are deleted ONLY after merging to the epic branch:
```bash
# After successful merge to epic:
git branch -D story/<STORY_ID>
git push origin --delete story/<STORY_ID>  # if remote exists
```

### `isolation: "worktree"` -- When to Use and When Not To

Claude Code's Agent tool has an `isolation: "worktree"` parameter that creates an
auto-generated `worktree-agent-*` branch. The rules depend on agent role:

**NEVER for Developers or Conflict-fix agents.** These agents commit code.
`isolation: "worktree"` would put commits on `worktree-agent-*` instead of
`story/<STORY_ID>`, and worktree cleanup deletes those commits permanently.
Always create developer worktrees manually on the story branch.

**REQUIRED for PM-Acceptors.** PM agents are read-only on tracked files -- they
only review code, run tests, and mutate nd (which writes to the shared vault,
not to the worktree). `isolation: "worktree"` gives PMs a completely isolated
shell that the dispatcher never needs to manage or clean up. This eliminates
the CWD corruption risk for PM operations entirely. However, Claude Code auto-cleans the worktree **directory** but NOT the `worktree-agent-*` **branch**. That branch persists locally and on remote after every PM review. Delete it as shown in step 8 above.

### PM Isolation

**Prerequisite (shared live nd vault):** PM isolation requires the shared live
nd vault -- a tracked `.vault/.nd-shared.yaml` pointing at git-common-dir.
Without it, the PM's isolated worktree cannot resolve `.vault` (the live vault
is gitignored and absent from the fresh checkout). One-time setup:

```bash
pvg nd root --ensure
git add .vault/.nd-shared.yaml
git commit -m "chore(paivot): share live nd vault across worktrees"
```

Since pvg v1.54.2, `pvg nd root --ensure` (and any `pvg nd` write) creates
`.vault/.nd-shared.yaml` itself and migrates a legacy local `.vault` live
vault into the shared location. Since pvg v1.54.6 and nd v0.10.19, vault
resolution also converges from worktrees that cannot see the config (a
branch that predates the config commit, or a sibling worktree outside the
project root): both tools fall back to the main checkout's config and to
an already-initialized shared vault under the git common dir, and
first-time initialization is serialized across concurrent agents.
Committing the config remains the documented practice -- it makes shared
mode explicit and survives main-checkout branch switches. Verify with
`pvg doctor`: it warns when a Paivot-managed git repo lacks the shared
config.

Spawn PM-Acceptors with `isolation: "worktree"`:

```
Agent(
  subagent_type="paivot-graph:pm",
  isolation="worktree",
  prompt="Review story STORY_ID.
  FIRST ACTION: run pwd -- that is your isolated worktree. Prefix EVERY
  subsequent shell command with cd <that-absolute-path> && ; the harness may
  reset your CWD to the project root between calls, and running git/make
  there corrupts the dispatcher's checkout (the guard will block you).
  Then check out the story IN YOUR WORKTREE, DETACHED (the retained dev
  worktree holds the branch ref; detached HEAD at the same commit is
  always allowed):
    cd <your-worktree> && git checkout --detach story/STORY_ID
  Then proceed with your review protocol.
  Project root: $PROJECT_ROOT
  ..."
)
```

The PM starts in an auto-generated worktree on the epic branch. It checks out
`story/<STORY_ID>` detached to see the developer's work. After the review,
Claude Code auto-cleans the worktree because the PM made no tracked file
changes (nd writes go to the shared vault via `pvg nd`, not to the worktree).

**Why detached:** the retained dev worktree (the resume anchor -- see
Semi-Persistent Story Agents) still holds the `story/<STORY_ID>` branch ref.
Git locks the ref, not the commit, so `git checkout --detach story/<STORY_ID>`
always works; a plain `git checkout story/<STORY_ID>` would fail while any
other worktree holds the branch.

### Branch Locking

Git prevents two worktrees from checking out the same branch ref
simultaneously. The retained dev worktree holds `story/<STORY_ID>` across the
review cycle (it is the resume anchor -- see Semi-Persistent Story Agents),
so the PM checks the story out DETACHED (`git checkout --detach
story/<STORY_ID>`), which git always allows. Only a plain branch checkout
would require removing the dev worktree first.

**nd labels are idempotent-ish:** `nd labels add` fails if the label already exists.
If the developer already set `delivered`, don't set it again. Check first or ignore
the error.

For bulk cleanup after context loss, use `pvg loop recover` instead of manual
`pvg worktree remove` commands (see Post-Compaction Recovery below).

## Post-Compaction Recovery

**STRUCTURAL ENFORCEMENT:** The PreCompact hook's output is user-visible only -- it does NOT reach the model and does NOT survive in the compaction summary. What actually survives is the SessionStart hook: it fires again immediately after compaction (source `compact`) and re-injects the dispatcher rules plus the `pvg loop recover` instruction directly into context. You MUST run `pvg loop recover` as the FIRST command after any compaction -- before touching git, before spawning agents, before inspecting branches.

After context compaction, you lose track of running agents and their worktrees.
Run recovery instead of doing manual cleanup:

```bash
pvg loop recover
```

This command automatically:
1. Reads the snapshot file (if one exists from a prior `pvg loop snapshot`)
2. Removes ONLY Paivot-owned agent worktrees -- those under `.claude/worktrees/`
   (the owned base; overridable via the `worktree.base` setting) -- and their
   Paivot branches (`story/*`, `epic/*`, `worktree-agent-*`, `worktree-*`).
   Worktrees created by other tools (for example `.codex-worktrees/` or
   `.opencode-worktrees/`) or at external paths are FOREIGN: `pvg loop recover`
   never removes them and never deletes their branches -- they are reported as
   preserved (`foreign_worktrees_preserved`) and left exactly as they were.
3. Deletes stale local branches (`epic/*`, `story/*`, `worktree-*`) that are fully merged into main
4. Resets orphaned in-progress stories to `open` in nd (delivered stories are preserved)
5. Outputs a recovery summary showing what's ready, delivered, and needs attention

If no snapshot exists, it still cleans Paivot-owned orphan worktrees, but the
ownership boundary above always holds: foreign worktrees are never touched. Do
NOT enumerate `git worktree list` and delete entries yourself -- let
`pvg loop recover` apply the ownership allowlist for you.

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

## CWD Safety (CRITICAL -- read this before any worktree operation)

Claude Code's shell CWD can silently drift into a worktree path after agent
completion or background task resolution. If you then remove that worktree,
your CWD becomes invalid and **every subsequent Bash command fails permanently**.
The session is unrecoverable -- you must ask the user to restart Claude Code.

### Defense layers

Three layers prevent this failure mode:

1. **Layer 1 (prevention): `cd $PROJECT_ROOT &&` prefix.**
   All worktree removal commands MUST be prefixed with an explicit cd:
   ```bash
   cd $PROJECT_ROOT && pwd
   pvg worktree remove .claude/worktrees/dev-STORY_ID
   ```
   This resets CWD before removal, so the parent shell is back on solid ground
   before the worktree disappears.

2. **Layer 2 (guard): `pvg worktree remove` refuses CWD-inside removal.**
   Starting in v1.52.11, `pvg worktree remove` checks whether the caller's
   CWD is inside the target worktree. If so, it REFUSES the removal with:
   `REFUSED: CWD "..." is inside worktree "..." -- run 'cd ...' first.`
   This catches cases where the `cd` prefix was forgotten.

3. **Layer 3 (isolation): PM agents use `isolation: "worktree"`.**
   PM-Acceptors run in Claude Code-managed isolated worktrees. The dispatcher
   never creates, removes, or touches PM worktrees, eliminating the entire
   class of CWD corruption bugs for PM operations.

### Additional rules

- **NEVER chain branch creation + worktree add in one Bash call:**
  ```bash
  # WRONG -- if branch creation touches the main worktree, the add may fail and
  # leave you on the wrong branch:
  git branch story/X epic/Y && pvg worktree add .claude/worktrees/dev-X story/X

  # RIGHT -- two separate Bash calls (use pvg worktree add so the worktree gets
  # the paivot-owned marker; recover only removes marked worktrees):
  git branch story/X epic/Y          # non-switching: dispatcher HEAD stays on main
  # (then in a SEPARATE Bash call:)
  pvg worktree add .claude/worktrees/dev-X story/X
  ```

- **After any developer or PM agent completes (foreground or background), the first Bash command must be a reset:**
  ```bash
  cd $PROJECT_ROOT && pwd
  ```
  Only then should you run `pvg worktree remove`, `git worktree list`, or any
  other dispatcher Bash command.

- **Use `pvg worktree remove` instead of raw `git worktree remove`.**
  `pvg worktree remove` resolves the project root from the worktree path
  (not from CWD) and enforces the CWD guard, but it cannot rescue a host shell
  that already failed to start because its own CWD points at a deleted path.

### Recovery (if CWD is already invalid)

If you see `fatal: Unable to read current working directory` or
`Working directory no longer exists`:

1. Your session shell is corrupted. Raw Bash commands will not work.
2. Spawn a general-purpose Agent with the cleanup task -- agents start from
   the project root and have a fresh shell.
3. The cleanup agent runs: `cd PROJECT_ROOT && git worktree prune && git worktree list`
4. After cleanup, escalate to the user: "Shell CWD is unrecoverable. Please
   restart Claude Code from PROJECT_ROOT."
5. Include a summary of remaining work so the next session can resume.

## How It Works

The loop is driven by the Claude Code stop hook:
1. This command sets up loop state via `pvg loop setup`
2. You run `pvg loop next --json` to get the next action
3. You execute that action (spawn agent, run gate, etc.)
4. When you try to stop, `pvg hook stop` intercepts and evaluates
5. If work remains, it emits continuation JSON that keeps the session alive
6. The next iteration begins automatically with a status summary
7. This repeats until a termination condition is met
