#!/usr/bin/env bash
#
# smoke_recover_preserves_foreign.sh -- regression smoke for the worktree
# ownership invariant: `pvg loop recover` removes ONLY Paivot-owned worktrees
# (under .claude/worktrees/) and NEVER removes or touches worktrees created by
# other tools (e.g. .codex-worktrees/) or at external paths.
#
# It builds a throwaway git repo with:
#   - an OWNED worktree   .claude/worktrees/dev-X  on branch story/X
#   - a FOREIGN worktree  .codex-worktrees/foreign on branch feature/foreign
# runs `pvg loop recover`, and asserts:
#   1. the foreign worktree DIRECTORY still exists       (never removed)
#   2. the foreign BRANCH feature/foreign still exists    (never deleted)
#   3. the owned worktree .claude/worktrees/dev-X is gone (legitimate cleanup)
#
# Requires the `pvg` binary on PATH (the ownership allowlist lives in pvg).
# Pure git + pvg -- no nd vault is configured, so recover prints nd warnings
# (expected and harmless); the worktree-safety assertions are what matter.
# Exits non-zero with a clear message on the first violated invariant.

set -euo pipefail

fail() { printf 'SMOKE FAIL: %s\n' "$*" >&2; exit 1; }
ok()   { printf '  ok: %s\n' "$*"; }

command -v pvg >/dev/null 2>&1 || fail "pvg not on PATH (build it: cd ../pvg && make build, then add to PATH)"

ROOT="$(mktemp -d "${TMPDIR:-/tmp}/pvg-recover-smoke-XXXXXX")"
trap 'rm -rf "$ROOT"' EXIT

export GIT_AUTHOR_NAME=smoke GIT_AUTHOR_EMAIL=smoke@test.local
export GIT_COMMITTER_NAME=smoke GIT_COMMITTER_EMAIL=smoke@test.local

g() { git -C "$ROOT" "$@"; }

printf 'Recover foreign-worktree preservation smoke (%s)\n' "$ROOT"

# --- Seed a repo on main with one commit ---
g init -q -b main
g config commit.gpgsign false
g commit -q --allow-empty -m base

# --- Owned worktree: .claude/worktrees/dev-X on story/X ---
g branch story/X main
mkdir -p "$ROOT/.claude/worktrees"
g worktree add -q "$ROOT/.claude/worktrees/dev-X" story/X

# --- Foreign worktree: .codex-worktrees/foreign on feature/foreign ---
g branch feature/foreign main
mkdir -p "$ROOT/.codex-worktrees"
g worktree add -q "$ROOT/.codex-worktrees/foreign" feature/foreign

# Sanity: both exist before recovery.
[ -d "$ROOT/.claude/worktrees/dev-X" ] || fail "owned worktree missing before recover"
[ -d "$ROOT/.codex-worktrees/foreign" ] || fail "foreign worktree missing before recover"

# --- Run recovery from the project root (no snapshot exists) ---
( cd "$ROOT" && pvg loop recover ) || true   # nd-vault warnings -> nonzero is OK

# Invariant 1: foreign worktree directory survives
[ -d "$ROOT/.codex-worktrees/foreign" ] \
  || fail "foreign worktree .codex-worktrees/foreign was REMOVED (data loss!)"
ok "foreign worktree directory preserved"

# Invariant 2: foreign branch survives
g rev-parse --verify feature/foreign >/dev/null 2>&1 \
  || fail "foreign branch feature/foreign was DELETED (data loss!)"
ok "foreign branch preserved"

# Invariant 3: owned worktree was cleaned up
if [ -d "$ROOT/.claude/worktrees/dev-X" ]; then
  fail "owned worktree .claude/worktrees/dev-X was NOT removed (recover did not clean up)"
fi
ok "owned worktree removed (legitimate cleanup)"

printf 'SMOKE PASS: pvg loop recover preserves foreign worktrees and branches\n'
