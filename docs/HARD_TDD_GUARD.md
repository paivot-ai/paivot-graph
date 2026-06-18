# Hard-TDD Guard

Hard-TDD is opt-in per story via the `hard-tdd` label. The runtime flow (RED
tests authored first, PM approves with `pvg story approve-red`, then GREEN
implementation) is enforced by the loop and the PM/Anchor agents. This guard is
the **CI-time structural lock** that complements it: it proves, from git
history, that test files were not quietly edited to make a failing
implementation pass.

## The rule it enforces

Within a commit range, an existing test file may only be **modified or deleted**
in:

- a **RED commit** -- its message contains the `tdd-red` marker (this is where
  failing tests are authored), or
- a commit explicitly authorized to touch tests during GREEN -- its message
  contains the `[test-edit-authorized]` marker (e.g. repairing a genuinely
  flaky test).

Any other commit that modifies or deletes a test file is a violation.

**Adding a brand-new test file is always allowed**, in any commit, with no
marker. A pure addition cannot weaken the frozen RED tests -- they still run and
must pass -- so GREEN is free to add coverage or CI tests in new files. Only
edits and deletions touch the RED set. Renames are detected via `--no-renames`,
so a renamed RED test surfaces as a delete of its old path and is still caught.

Merge commits are skipped: a merge carries no per-commit marker of its own, and
its non-merge constituents (which do) are checked directly.

## Running it

The engine is a `pvg` subcommand, so every Paivot-managed project shares one
implementation:

```bash
pvg story verify-tdd --range <base>..HEAD      # explicit range
pvg story verify-tdd --base origin/main        # merge-base(origin/main, HEAD)..HEAD
pvg story verify-tdd --base "$EPIC_BRANCH" --json
```

Flags:

| Flag | Meaning |
|------|---------|
| `--range A..B` | Explicit git range. Takes precedence over `--base`. |
| `--base REF` | Compute `merge-base(REF, HEAD)..HEAD`. `$TDD_BASE` is the env fallback. |
| `--test-glob SUBSTR` | Add a test-path substring (repeatable). Defaults cover Go, Elixir, Python, JS/TS, Ruby, Java, C#. |
| `--red-marker TOKEN` | Override the RED marker (default `tdd-red`). |
| `--authz-marker TOKEN` | Override the authorized-edit marker (default `[test-edit-authorized]`). |
| `--json` | Machine-readable output. |

It exits non-zero on a violation, and also **fails loudly** (non-zero, with
guidance) when the range cannot be resolved -- it never inspects nothing and
reports success. That silent-pass mode is exactly what this guard exists to
prevent.

## CI wiring

`scripts/verify-hard-tdd.sh` is a thin wrapper that resolves the range from the
CI environment (PR base branch, push `before` SHA, or `$TDD_BASE`/`origin/main`)
and calls `pvg story verify-tdd`. Drop it into a workflow step:

```yaml
- name: Hard-TDD guard
  run: scripts/verify-hard-tdd.sh
```

Why a binary subcommand plus a thin wrapper, rather than a per-repo script: range
resolution against worktree checkouts and merge-to-main pushes is the fragile
part, and solving it once in pvg (with tests) keeps every project from
re-discovering the worktree silent-pass and merge-commit false-fail.
