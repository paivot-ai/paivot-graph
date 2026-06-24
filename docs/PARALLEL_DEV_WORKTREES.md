# Parallel Developer Worktrees

Paivot developers that write tracked files must not share the dispatcher main
worktree. The dispatcher creates one branch and one git worktree per story, then
passes the absolute worktree path in the developer prompt.

## Required Developer Flow

For each developer story:

```bash
git branch story/STORY_ID origin/main
pvg worktree add .claude/worktrees/dev-STORY_ID story/STORY_ID
```

Create the worktree with `pvg worktree add`, never raw `git worktree add`. See
[Marker = Ownership](#marker--ownership) below: `pvg worktree add` stamps the
ownership marker that lets `pvg loop recover` clean the worktree up. A raw
`git worktree add` leaves the worktree unmarked, and recover then treats it as
foreign and refuses to remove it.

The developer prompt must include:

```text
Work in: /absolute/path/to/repo/.claude/worktrees/dev-STORY_ID
Do not create or checkout another branch. Commit to story/STORY_ID only.
```

For a parallel wave, prepare all branches and worktrees before spawning any
developer. If one setup step fails, stop the wave and repair branch or worktree
state first.

## Why Native Agent Worktree Isolation Is Not Used For Developers

Claude Code's native `isolation: "worktree"` creates an automatic
`worktree-agent-*` branch. That is useful for PM-Acceptor review because PMs do
not commit tracked files. It is unsafe for Developers and Conflict-fix agents:
code commits would land on an automatic branch instead of `story/STORY_ID`, and
cleanup can remove the only easy reference to those commits.

Use native `isolation: "worktree"` for PM-Acceptor only. Use dispatcher-managed
`pvg worktree add` directories for code-writing agents.

## Marker = Ownership

`pvg loop recover` cleans up Paivot's leftover worktrees after a context loss.
It must never delete a worktree another tool -- or a concurrent NON-Paivot
Claude Code session -- created. Ownership is therefore a **marker**, not a path:

- `pvg worktree add <path> <branch>` creates the worktree via git, then writes a
  `paivot-owned` marker file into the worktree's git admin dir
  (`.git/worktrees/<name>/paivot-owned`). This is the ONLY way a Paivot worktree
  gets marked.
- `pvg loop recover` and `pvg worktree remove` remove a worktree, and delete its
  branch, **only if it carries that marker**. A worktree with no marker is
  treated as foreign and preserved untouched -- **regardless of its path**
  (including one created under `.claude/worktrees/` by a non-Paivot session) and
  regardless of its branch name.
- The marker lives in the git admin dir so git owns its lifecycle:
  `git worktree add` creates the admin dir; `git worktree remove`/`prune` deletes
  it together with the marker. There is nothing extra to clean up.

Consequence: ALWAYS create developer/Conflict-fix worktrees with
`pvg worktree add`. A raw `git worktree add` produces an UNMARKED worktree that
recover will refuse to clean up (it looks foreign), leaving stale worktrees and
branches behind.

## Reproduction Recipe For The Old Bug

This reproduces the shared-worktree failure that HXT-jabf records:

1. Start three developer sessions from the same project root without creating
   `.claude/worktrees/dev-STORY_ID` directories.
2. In session A, run `git checkout -b story/A origin/main`, create `a.txt`, and
   stage it.
3. In session B, run `git checkout -b story/B origin/main`.
4. In session C, run `git checkout -b story/C origin/main`.
5. Return to session A and run `git status --short`.

At least one failure appears: A's staged file is gone or visible to a sibling,
the branch moved under another session, or a developer creates a `-v2`/`-v3`
recovery branch to escape the collision.

## Regression Smoke

A parallel-dev wave is correct only when, after three dispatcher-managed
worktrees each stage and commit one unique file and the stories merge back to
the epic branch, all of these hold:

- each worktree sees only its own staged file before commit
- no `worktree-agent-*` developer branch exists
- no `-v2` or `-v3` collision-recovery branch exists
- all worktrees are removed at the end

These invariants are asserted automatically by
`scripts/smoke_parallel_dev_worktrees.sh` (run `make smoke-worktrees`), which the
`worktree-smoke` CI workflow runs on every push and pull request.
