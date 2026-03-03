---
description: Run unattended execution loop until complete or blocked
allowed-tools: ["Bash", "Read", "Glob", "Grep", "Skill", "Task", "AskUserQuestion"]
args: "[EPIC_ID] [--all] [--max-iterations|--max N]"
---

# piv-loop -- Unattended Execution Loop

Run the backlog to completion without manual intervention. Spawns developer and PM agents
in priority order until all work is done, blocked, or max iterations reached.

## Setup

Activate the loop via pvg:

```bash
pvg loop setup $ARGUMENTS
```

If `$ARGUMENTS` is empty, ask the user:
- "Run all ready work (`--all`) or target a specific epic (provide the EPIC_ID)?"
- "Max iterations? (default: 50, 0 for unlimited)"

Verify activation succeeded before continuing.

## Priority Order

Each iteration, pick work in this order:

0. **Sr. PM for bug triage** (highest priority -- discovered bugs need structure)
   After any Developer or PM-Acceptor agent completes, scan its output for
   `DISCOVERED_BUG:` blocks. If found, collect ALL bug reports from that agent
   and spawn `paivot-graph:sr-pm` with:
   ```
   BUG TRIAGE MODE. Create properly structured bugs for these discovered issues:
   <paste all DISCOVERED_BUG blocks>
   ```
   Wait for Sr. PM to finish before continuing. Bugs need epic placement and
   dependency chains before other work can be prioritized correctly.

1. **PM-Acceptor for delivered stories** (unblock the pipeline)
   ```bash
   nd list --status in_progress --label delivered --json
   ```
   For each: spawn `paivot-graph:pm` agent to review and accept/reject.
   **After each acceptance**: the PM-Acceptor runs epic auto-close (see pm.md).

2. **Developer for rejected stories** (fix before starting new work)
   ```bash
   nd list --status in_progress --label rejected --json
   ```
   For each: spawn `paivot-graph:developer` agent to address rejection notes.

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

## Dispatcher Rules

You are a dispatcher. You coordinate agents. You NEVER:
- Write source code or tests yourself
- Fix errors or bugs yourself
- Modify story files yourself
- Make architectural decisions yourself
- Skip agents to "save time"
- Resolve merge conflicts yourself (spawn a developer -- conflict resolution requires code judgment)
- Edit source files for any reason, including "cleanup" or "git maintenance"

If an agent fails, re-spawn it with corrective guidance. Do not do its work.

## Agent Types

| Role | Agent Type | When |
|------|-----------|------|
| Sr. PM (bug triage) | `paivot-graph:sr-pm` | DISCOVERED_BUG blocks found in agent output |
| PM-Acceptor | `paivot-graph:pm` | Stories with `delivered` label |
| Developer | `paivot-graph:developer` | Ready or rejected stories |

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

## Post-Compaction Recovery

After context compaction, you lose track of running agents and their worktrees.
Do NOT investigate old worktrees or try to continue partial work. Instead:

1. Check `nd list --status in_progress --json` for stories that were being worked on
2. Discard any stale worktrees: `git worktree list` then `git worktree remove --force <path>` for agent worktrees
3. Re-spawn fresh developer agents for in-progress stories -- they start clean from main
4. Never spawn an agent whose cwd is inside another agent's worktree (causes nesting)

The agents' worktree isolation means partial uncommitted work is lost on compaction.
This is acceptable -- re-doing work is cheaper than untangling nested worktrees.

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
