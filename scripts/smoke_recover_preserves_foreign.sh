#!/usr/bin/env bash
#
# smoke_recover_preserves_foreign.sh -- regression smoke for the worktree
# ownership invariant: `pvg loop recover` removes ONLY worktrees that carry
# Paivot's ownership marker (created via `pvg worktree add`) and NEVER removes or
# touches a worktree without the marker -- regardless of its path. This proves
# ownership is the MARKER, not the path.
#
# It builds a throwaway git repo with:
#   - an OWNED worktree    .claude/worktrees/dev-X        (created via `pvg worktree add`
#                                                          -> carries the paivot-owned marker)
#   - a FOREIGN worktree   .codex-worktrees/foreign       (raw `git worktree add`, NO marker,
#                                                          OUTSIDE Paivot's base)
#   - a FOREIGN worktree   .claude/worktrees/foreign-session (raw `git worktree add`, NO marker,
#                                                          INSIDE Paivot's base -- a concurrent
#                                                          non-Paivot session would create this)
# runs `pvg loop recover`, and asserts:
#   1. the codex foreign worktree DIRECTORY still exists           (never removed)
#   2. the codex foreign BRANCH feature/foreign still exists        (never deleted)
#   3. the INSIDE-base foreign worktree DIRECTORY still exists      (marker, not path, decides)
#   4. the INSIDE-base foreign BRANCH feature/intruder still exists (never deleted)
#   5. the owned worktree .claude/worktrees/dev-X is gone           (legitimate cleanup)
#
# Requires the `pvg` binary on PATH (the ownership marker logic lives in pvg).
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

printf 'Recover marker-ownership preservation smoke (%s)\n' "$ROOT"

# --- Seed a repo on main with one commit ---
g init -q -b main
g config commit.gpgsign false
g commit -q --allow-empty -m base

# --- Owned worktree: .claude/worktrees/dev-X on story/X, created via pvg so it
#     carries the paivot-owned ownership marker. ---
g branch story/X main
mkdir -p "$ROOT/.claude/worktrees"
( cd "$ROOT" && pvg worktree add .claude/worktrees/dev-X story/X ) \
  || fail "pvg worktree add failed for the owned worktree"
# The marker must exist in the worktree's git admin dir.
adm="$(git -C "$ROOT/.claude/worktrees/dev-X" rev-parse --git-dir)"
case "$adm" in
  /*) : ;;                                  # absolute -- use as-is
  *)  adm="$ROOT/.claude/worktrees/dev-X/$adm" ;;
esac
[ -f "$adm/paivot-owned" ] || fail "owned worktree is missing the paivot-owned marker after pvg worktree add"
ok "owned worktree created and marked via pvg worktree add"

# --- Foreign worktree OUTSIDE the base: .codex-worktrees/foreign (raw git, no marker) ---
g branch feature/foreign main
mkdir -p "$ROOT/.codex-worktrees"
g worktree add -q "$ROOT/.codex-worktrees/foreign" feature/foreign

# --- Foreign worktree INSIDE the base: .claude/worktrees/foreign-session (raw git, no marker).
#     This is the case the marker closes: a concurrent NON-Paivot session creating a worktree
#     under Paivot's own directory. Path is under the base, but there is NO marker. ---
g branch feature/intruder main
g worktree add -q "$ROOT/.claude/worktrees/foreign-session" feature/intruder

# Sanity: all three exist before recovery.
[ -d "$ROOT/.claude/worktrees/dev-X" ] || fail "owned worktree missing before recover"
[ -d "$ROOT/.codex-worktrees/foreign" ] || fail "codex foreign worktree missing before recover"
[ -d "$ROOT/.claude/worktrees/foreign-session" ] || fail "inside-base foreign worktree missing before recover"

# --- Run recovery from the project root (no snapshot exists) ---
( cd "$ROOT" && pvg loop recover ) || true   # nd-vault warnings -> nonzero is OK

# Invariant 1: codex foreign worktree directory survives
[ -d "$ROOT/.codex-worktrees/foreign" ] \
  || fail "foreign worktree .codex-worktrees/foreign was REMOVED (data loss!)"
ok "codex foreign worktree directory preserved"

# Invariant 2: codex foreign branch survives
g rev-parse --verify feature/foreign >/dev/null 2>&1 \
  || fail "foreign branch feature/foreign was DELETED (data loss!)"
ok "codex foreign branch preserved"

# Invariant 3: INSIDE-base foreign worktree directory survives -- the decisive case
[ -d "$ROOT/.claude/worktrees/foreign-session" ] \
  || fail "UNMARKED worktree INSIDE .claude/worktrees/ was REMOVED (marker check failed -- data loss!)"
ok "inside-base foreign worktree directory preserved (marker, not path, decides ownership)"

# Invariant 4: INSIDE-base foreign branch survives
g rev-parse --verify feature/intruder >/dev/null 2>&1 \
  || fail "foreign branch feature/intruder was DELETED (data loss!)"
ok "inside-base foreign branch preserved"

# Invariant 5: owned (marked) worktree was cleaned up
if [ -d "$ROOT/.claude/worktrees/dev-X" ]; then
  fail "owned worktree .claude/worktrees/dev-X was NOT removed (recover did not clean up the marked worktree)"
fi
ok "owned (marked) worktree removed (legitimate cleanup)"

printf 'SMOKE PASS: pvg loop recover removes only marked worktrees; both foreign worktrees (outside AND inside .claude/worktrees/) preserved\n'
