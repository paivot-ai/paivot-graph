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
fresh clone does not contain it. Durability is nd-native:

- Every nd mutation auto-snapshots the vault locally to the `nd/backlog` git
  branch -- an auto-snapshot journal. At most the single most recent command
  can be lost.
- `pvg nd sync` delegates to `nd sync`: snapshot + fetch + field-aware merge +
  push of the `nd/backlog` branch. `--status` shows the local branch's
  position relative to the remote; `--no-push` skips the push.
- `pvg nd restore` delegates to `nd sync --restore`: it rebuilds a wiped live
  vault from the `nd/backlog` branch, falling back to a legacy
  `.vault/backlog-snapshot/` export if the branch is absent.

The branch is a snapshot journal, never the live queue. Agents keep reading
and writing through `pvg nd` against the shared live vault; the branch exists
so the backlog survives clone boundaries and machine loss.

### The branch is a snapshot, not the source of truth

Reconcile "files vs backlog" lints against the LIVE vault via
`pvg issues list --json` (or `pvg nd` directly), NEVER against the
`nd/backlog` branch contents. A lint that reconciles against the branch will
report phantom drift for anything newer than the last snapshot.

The dispatcher's owned sync points are `pvg loop setup` (pulls remote backlog
state), after each accepted story merge, and at loop end. `pvg doctor` runs an
nd-sync-status check -- it warns when the local `nd/backlog` branch is behind
or unpushed, with remedy `pvg nd sync`. (This replaces the old
`snapshot-drift` check; the export-to-`.vault/backlog-snapshot/` model is
retired, and `pvg nd sync --commit` remains only as a deprecated alias for a
plain sync.)

Manually copying files out of the live vault (under `git-common-dir/...`) into
`.vault/` is NOT a supported mechanism -- a hand copy will diverge from what
`pvg nd restore` expects. Always use `pvg nd sync`.

## Dependency Link Lifecycle

A dependency edge is not deleted when it is satisfied -- it is archived. When a
blocking issue closes, nd moves the edge from `blocked_by` to `was_blocked_by`
(and mirrors the resulting execution order in `follows`). A satisfied edge is
still an edge of the planned DAG.

`pvg issues show --json` and `pvg issues list --json` therefore expose:

- `was_blocked_by` -- the archived (already-satisfied) blockers.
- `all_blocked_by` -- the deduplicated, sorted lifetime union of `blocked_by` +
  `was_blocked_by`.

Lints and gates that consume pvg JSON to reconcile a DAG across an epic's
lifetime MUST read `all_blocked_by`, not `blocked_by` alone -- otherwise they
lose edges as the epic completes and blockers close. pvg's own backlog lints
already union these internally, so this guidance is for DOWNSTREAM consumers.

(Quirk: the legacy edge fields serialize in CamelCase -- `BlockedBy`, `Blocks`
-- while the two lifecycle fields are snake_case.)

## Separation

- `.vault/knowledge/` is project knowledge and has its own git policy
- live nd state is execution state and should be shared across worktrees
- runtime lock files and guard logs must never be tracked
