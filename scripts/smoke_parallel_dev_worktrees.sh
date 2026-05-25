#!/bin/sh
set -eu

root="$(mktemp -d)"
cleanup() {
  if [ -d "$root/repo" ]; then
    git -C "$root/repo" worktree remove --force "$root/repo/.claude/worktrees/dev-HXT-a1" >/dev/null 2>&1 || true
    git -C "$root/repo" worktree remove --force "$root/repo/.claude/worktrees/dev-HXT-b2" >/dev/null 2>&1 || true
    git -C "$root/repo" worktree remove --force "$root/repo/.claude/worktrees/dev-HXT-c3" >/dev/null 2>&1 || true
    git -C "$root/repo" worktree prune >/dev/null 2>&1 || true
  fi
  rm -rf "$root"
}
trap cleanup EXIT

repo="$root/repo"
mkdir -p "$repo"
git -C "$repo" init -b main >/dev/null
git -C "$repo" config user.email "paivot-smoke@example.invalid"
git -C "$repo" config user.name "Paivot Smoke"

printf ".claude/worktrees/\n" > "$repo/.gitignore"
printf "root\n" > "$repo/README.md"
git -C "$repo" add .gitignore README.md
git -C "$repo" commit -m "init" >/dev/null
git -C "$repo" branch epic/HXT-smoke main

for id in HXT-a1 HXT-b2 HXT-c3; do
  git -C "$repo" branch "story/$id" epic/HXT-smoke
  git -C "$repo" worktree add "$repo/.claude/worktrees/dev-$id" "story/$id" >/dev/null
done

for id in HXT-a1 HXT-b2 HXT-c3; do
  wt="$repo/.claude/worktrees/dev-$id"
  printf "%s\n" "$id" > "$wt/$id.txt"
  git -C "$wt" add "$id.txt"
done

for id in HXT-a1 HXT-b2 HXT-c3; do
  wt="$repo/.claude/worktrees/dev-$id"
  status="$(git -C "$wt" status --short)"
  expected="A  $id.txt"
  if [ "$status" != "$expected" ]; then
    printf "unexpected staged state for %s:\n%s\n" "$id" "$status" >&2
    exit 1
  fi
done

for id in HXT-a1 HXT-b2 HXT-c3; do
  wt="$repo/.claude/worktrees/dev-$id"
  git -C "$wt" commit -m "feat($id): smoke isolated worktree" >/dev/null
done

for id in HXT-a1 HXT-b2 HXT-c3; do
  git -C "$repo" worktree remove "$repo/.claude/worktrees/dev-$id" >/dev/null
done
git -C "$repo" worktree prune >/dev/null

git -C "$repo" checkout epic/HXT-smoke >/dev/null
for id in HXT-a1 HXT-b2 HXT-c3; do
  git -C "$repo" merge --no-ff "story/$id" -m "merge(epic/HXT-smoke): integrate $id" >/dev/null
done

if git -C "$repo" branch --list | grep -Eq '(^|[[:space:]])(worktree-agent-|.*-v[0-9]+$)'; then
  git -C "$repo" branch --list >&2
  exit 1
fi

if [ -n "$(git -C "$repo" worktree list --porcelain | awk '/^worktree / { count++ } END { if (count > 1) print count }')" ]; then
  git -C "$repo" worktree list >&2
  exit 1
fi

for id in HXT-a1 HXT-b2 HXT-c3; do
  test -f "$repo/$id.txt"
done

printf "PASS: three dispatcher-managed developer worktrees stayed isolated\n"
