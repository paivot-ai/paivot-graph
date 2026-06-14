#!/usr/bin/env bash
#
# Canonical Paivot hard-TDD CI guard.
#
# Thin wrapper around `pvg story verify-tdd` that resolves a robust commit range
# from the CI environment and FAILS LOUDLY when the range cannot be determined
# (never warn-and-pass). The marker-checking and merge-commit handling live in
# pvg, so every Paivot-managed project shares one implementation instead of
# re-solving range resolution in a per-repo shell script.
#
# Range resolution precedence:
#   1. $TDD_RANGE                      explicit "<base>..<tip>"
#   2. $GITHUB_BASE_REF (PR build)     merge-base(origin/<base>, HEAD)..HEAD
#   3. $GITHUB_EVENT_BEFORE (push)     <before>..HEAD  (merge commits are
#                                      skipped inside pvg, so a merge-to-main
#                                      push is handled cleanly)
#   4. $TDD_BASE or origin/main        merge-base fallback
#
# Any extra arguments are passed through to `pvg story verify-tdd`
# (e.g. --json, --test-glob, --red-marker, --authz-marker).
set -euo pipefail

if ! command -v pvg >/dev/null 2>&1; then
  echo "[hard-tdd] pvg not found on PATH -- install it (see docs/CONTAINER_TOOLCHAIN.md)" >&2
  exit 2
fi

args=()
if [[ -n "${TDD_RANGE:-}" ]]; then
  args+=(--range "$TDD_RANGE")
elif [[ -n "${GITHUB_BASE_REF:-}" ]]; then
  git fetch --quiet origin "$GITHUB_BASE_REF" 2>/dev/null || true
  args+=(--base "origin/${GITHUB_BASE_REF}")
elif [[ -n "${GITHUB_EVENT_BEFORE:-}" && "${GITHUB_EVENT_BEFORE}" != "0000000000000000000000000000000000000000" ]]; then
  args+=(--range "${GITHUB_EVENT_BEFORE}..HEAD")
else
  args+=(--base "${TDD_BASE:-origin/main}")
fi

exec pvg story verify-tdd "${args[@]}" "$@"
