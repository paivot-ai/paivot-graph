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
- "Run all ready work (`--all`) or target a specific epic (`--epic EPIC_ID`)?"
- "Max iterations? (default: 50, 0 for unlimited)"

Verify activation succeeded before continuing.

## Priority Order

Each iteration, pick work in this order:

1. **PM-Acceptor for delivered stories** (highest priority -- unblock the pipeline)
   ```bash
   nd list --status in_progress --label delivered --json
   ```
   For each: spawn `paivot-graph:pm` agent to review and accept/reject.

2. **Developer for rejected stories** (fix before starting new work)
   ```bash
   nd list --status in_progress --label rejected --json
   ```
   For each: spawn `paivot-graph:developer` agent to address rejection notes.

3. **Developer for ready stories** (new work)
   ```bash
   nd ready --json
   ```
   For each: spawn `paivot-graph:developer` agent to implement.

**Epic-scoped queries**: When targeting a specific epic, scope all queries to that epic:
```bash
nd children <epic-id> --json            # All stories in the epic
nd list --parent <epic-id> --status in_progress --json  # Filter within epic
```
IMPORTANT: nd does NOT have an `--epic` flag. Use `--parent` or `nd children` instead.

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

If an agent fails, re-spawn it with corrective guidance. Do not do its work.

## Agent Types

| Role | Agent Type |
|------|-----------|
| Developer | `paivot-graph:developer` |
| PM-Acceptor | `paivot-graph:pm` |

## Termination Conditions

The stop hook (`pvg hook stop`) evaluates these automatically:

| Condition | Action |
|-----------|--------|
| All work complete (nothing in any state) | Allow exit, remove state |
| All remaining work blocked | Allow exit, remove state |
| Max iterations reached | Allow exit, remove state |
| Too many consecutive waits (3) | Allow exit, remove state |
| Actionable work exists (ready or delivered) | Block exit, continue loop |
| Only in-progress work (waiting) | Block exit, increment wait counter |

## Cancellation

To cancel a running loop:
```
/piv-cancel-loop
```

Or directly:
```bash
pvg loop cancel
```

## Post-Compaction Recovery

After context compaction, you lose track of running agents and their worktrees.
Do NOT investigate old worktrees or try to continue partial work. Instead:

1. Check `nd list --status in_progress --json` for stories that were being worked on
2. Discard any stale worktrees: `git worktree list` then `git worktree remove <path>` for agent worktrees
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
