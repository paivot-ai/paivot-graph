#!/usr/bin/env bash
#
# smoke_parallel_dev_worktrees.sh -- regression smoke for the parallel-developer
# wave invariants documented in docs/PARALLEL_DEV_WORKTREES.md.
#
# It builds a throwaway repo, simulates a three-developer wave using
# dispatcher-managed story worktrees (manual `git worktree add` on story
# branches -- never Claude Code's isolation: "worktree"), and asserts the
# isolation guarantees the methodology depends on:
#
#   1. the dispatcher (parent) checkout stays on its branch, never a story branch
#   2. each developer worktree is checked out to its own story branch
#   3. each worktree's index stages ONLY its own file (no sibling leak)
#   4. each story commit contains ONLY its own file
#   5. all three stories merge cleanly back into the epic branch
#   6. no worktree-agent-* developer branch exists (manual story worktrees only)
#   7. no -vN collision-recovery branch suffix exists
#   8. every dev worktree is removed cleanly at the end
#
# Pure git -- no pvg/nd/network dependency, so it runs in CI and standalone.
# Exits non-zero with a clear message on the first violated invariant.

set -euo pipefail

readonly STORIES=(STORY-a1b STORY-c3d STORY-e5f)
readonly EPIC="epic/EPIC-001"

fail() { printf 'SMOKE FAIL: %s\n' "$*" >&2; exit 1; }
ok()   { printf '  ok: %s\n' "$*"; }

ROOT="$(mktemp -d "${TMPDIR:-/tmp}/pvg-wt-smoke-XXXXXX")"
trap 'rm -rf "$ROOT"' EXIT

# Deterministic identity; never sign commits (CI has no key).
export GIT_AUTHOR_NAME=smoke GIT_AUTHOR_EMAIL=smoke@test.local
export GIT_COMMITTER_NAME=smoke GIT_COMMITTER_EMAIL=smoke@test.local

g() { git -C "$ROOT" "$@"; }
wt() { printf '%s/.claude/worktrees/dev-%s' "$ROOT" "$1"; }

printf 'Parallel-dev worktree smoke (%s)\n' "$ROOT"

# --- Seed a repo with an epic branched off main (non-switching, like dispatch) ---
g init -q -b main
g config commit.gpgsign false
g commit -q --allow-empty -m "base"
g branch "$EPIC" main

# --- Dispatcher: create every story branch + dev worktree BEFORE any spawn ---
for s in "${STORIES[@]}"; do
  g branch "story/$s" "$EPIC"
  g worktree add -q "$(wt "$s")" "story/$s"
done

# Invariant 1: parent checkout stays on main, not dragged onto a story branch
head="$(g rev-parse --abbrev-ref HEAD)"
[ "$head" = "main" ] || fail "dispatcher checkout is on '$head', expected 'main'"
ok "dispatcher checkout stayed on main"

# Invariant 2: each worktree is checked out to its own story branch
for s in "${STORIES[@]}"; do
  wh="$(git -C "$(wt "$s")" rev-parse --abbrev-ref HEAD)"
  [ "$wh" = "story/$s" ] || fail "worktree dev-$s is on '$wh', expected 'story/$s'"
done
ok "each worktree checked out its own story branch"

# --- Each developer stages one unique file in its own worktree ---
for s in "${STORIES[@]}"; do
  printf '%s payload\n' "$s" > "$(wt "$s")/$s.txt"
  git -C "$(wt "$s")" add "$s.txt"
done

# Invariant 3: each worktree's index holds ONLY its own file
for s in "${STORIES[@]}"; do
  staged="$(git -C "$(wt "$s")" diff --cached --name-only)"
  [ "$staged" = "$s.txt" ] || fail "dev-$s staged [$staged], expected [$s.txt] (sibling leak)"
done
ok "each worktree staged only its own file"

# --- Each developer commits on its own story branch ---
for s in "${STORIES[@]}"; do
  git -C "$(wt "$s")" commit -q -m "feat($s): add $s.txt"
done

# Invariant 4: each story tree contains ONLY its own file
for s in "${STORIES[@]}"; do
  n="$(g ls-tree -r --name-only "story/$s" | grep -c '\.txt$' || true)"
  [ "$n" -eq 1 ] || fail "story/$s tree has $n .txt files, expected 1 (sibling content leaked)"
done
ok "each story branch committed only its own file"

# --- Dispatcher merges every story back into the epic branch ---
g checkout -q "$EPIC"
for s in "${STORIES[@]}"; do
  g merge --no-ff -q "story/$s" -m "merge(epic): integrate $s" \
    || fail "merge of story/$s into $EPIC conflicted"
done

# Invariant 5: the epic integrates all three stories
n="$(g ls-tree -r --name-only "$EPIC" | grep -c '\.txt$' || true)"
[ "$n" -eq 3 ] || fail "epic has $n .txt files after merging 3 stories, expected 3"
ok "epic integrated all three stories"

# Invariant 6: no Claude-Code isolation branch leaked in
if g for-each-ref --format='%(refname:short)' 'refs/heads/worktree-agent-*' | grep -q .; then
  fail "found a worktree-agent-* branch (developers must use manual story worktrees)"
fi
ok "no worktree-agent-* branches"

# Invariant 7: no -vN collision-recovery branch suffix
if g for-each-ref --format='%(refname:short)' 'refs/heads/story/*' | grep -qE -- '-v[0-9]+$'; then
  fail "found a -vN collision-recovery story branch"
fi
ok "no -vN collision-recovery branches"

# --- Dispatcher removes every dev worktree ---
for s in "${STORIES[@]}"; do
  g worktree remove --force "$(wt "$s")"
done
g worktree prune

# Invariant 8: nothing left behind
left="$(g worktree list | grep -c '/dev-' || true)"
[ "$left" -eq 0 ] || fail "$left dev worktree(s) still registered after removal"
if find "$ROOT/.claude/worktrees" -mindepth 1 2>/dev/null | grep -q .; then
  fail "leftover paths under .claude/worktrees after removal"
fi
ok "all dev worktrees removed cleanly"

printf 'SMOKE PASS: parallel-dev worktree isolation holds (3 developers)\n'
