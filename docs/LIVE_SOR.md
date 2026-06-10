# Live Source Of Record

Paivot's live execution queue must stay canonical even when multiple agents are
working in different branches or worktrees for the same repository.

## Rule

For nd-backed Paivot variants, the mutable backlog lives in a branch-independent
nd vault resolved from the repository's git common dir.

Example location:

```bash
$(git rev-parse --git-common-dir)/paivot/nd-vault
```

The worktree advertises that shared vault through:

```yaml
# .vault/.nd-shared.yaml
mode: git_common_dir
path: paivot/nd-vault
```

That vault is the live source of record for:

- issue status
- labels such as `delivered`, `accepted`, `rejected`
- append-only notes and proof
- dependency edges and epic relationships

## Why

Tracked issue files inside each story branch cannot remain canonical once two
branches mutate them independently. One of the branches will eventually carry a
divergent tracker history, and merge order will determine which state survives.

Branch-local mutable state is therefore the wrong place for the live backlog
when Paivot is running multi-agent execution.

## What Git Still Does

Git remains useful for:

- code branches and merges
- explicit backlog snapshots (`nd archive`)
- exported audit artifacts

But those snapshots are exports of the live queue, not the live queue itself.

## Durability

The live vault lives under git-common-dir and is not part of git history -- a
fresh clone does not contain it. Durability comes from snapshots:

- `pvg nd sync` exports the live vault into a tracked snapshot at
  `.vault/backlog-snapshot/`. The dispatcher runs it at each epic completion
  gate and commits the snapshot on main.
- `pvg nd restore` re-imports the snapshot into an empty live vault after a
  fresh clone.

The snapshot is an export, never the live queue. Agents keep reading and
writing through `pvg nd` against the shared live vault; the snapshot exists
only so the backlog survives clone boundaries and machine loss.

## Separation

- `.vault/knowledge/` is project knowledge and has its own git policy
- live nd state is execution state and should be shared across worktrees
- runtime lock files and guard logs must never be tracked
