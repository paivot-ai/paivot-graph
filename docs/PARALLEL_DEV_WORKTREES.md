# Parallel Developer Worktrees

Paivot developers that write tracked files must not share the dispatcher main
worktree. The dispatcher creates one branch and one git worktree per story, then
passes the absolute worktree path in the developer prompt.

## Required Developer Flow

For each developer story:

```bash
git branch story/STORY_ID origin/main
git worktree add .claude/worktrees/dev-STORY_ID story/STORY_ID
```

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
`git worktree add` directories for code-writing agents.

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

Run:

```bash
scripts/smoke_parallel_dev_worktrees.sh
```

The smoke creates a temporary repo, prepares three story branches and three
dispatcher-managed worktrees, stages and commits one unique file per worktree,
merges the stories back to the epic branch, and verifies:

- each worktree sees only its own staged file before commit
- no `worktree-agent-*` developer branch exists
- no `-v2` or `-v3` collision-recovery branch exists
- all worktrees are removed at the end
