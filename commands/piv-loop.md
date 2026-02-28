---
description: Run unattended execution loop until complete or blocked
allowed-tools: ["Bash", "Read", "Glob", "Grep", "Skill", "Task", "AskUserQuestion"]
args: "[EPIC_ID] [--all] [--max-iterations N]"
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

## Concurrency Limits (HARD RULE)

- Maximum 2 developer agents running simultaneously
- Maximum 1 PM-Acceptor agent running simultaneously
- Total active subagents (all types) must not exceed 3
- Wait for an agent to finish before spawning another if at the limit

These limits prevent context exhaustion. Violating them risks losing the entire session.

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

## How It Works

The loop is driven by the Claude Code stop hook:
1. This command sets up loop state via `pvg loop setup`
2. You execute one iteration of work (spawn agents per priority order)
3. When you try to stop, `pvg hook stop` intercepts and evaluates
4. If work remains, it emits continuation JSON that keeps the session alive
5. The next iteration begins automatically with a status summary
6. This repeats until a termination condition is met
