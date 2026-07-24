#!/usr/bin/env bash
# run-evals.sh: experimental agent-prompt eval harness.
#
# Usage: scripts/run-evals.sh [scenario-name]
#
# For each scenario under evals/ (or the one named in $1):
#   1. Create a scratch workspace under evals/.results/<scenario>/workspace
#      and copy the scenario's inputs/ into it.
#   2. Run the agent under test headlessly: claude -p with a prompt composed
#      of the agent .md file, a separator, and task.md, executed from inside
#      the scratch workspace. Stdout is captured to transcript.md.
#   3. Grade: a second claude -p call judges the workspace against the
#      weighted checklist in criteria.json and emits JSON to grade.json.
#
# WARNING: every run consumes real Claude tokens (one agent run plus one
# grading run per scenario). This harness is experimental: not wired into
# CI, and no release gate depends on it.
#
# EVAL_TIMEOUT (seconds, default 900) bounds each claude invocation. macOS
# has no timeout binary by default, so the bound is implemented with a
# background watcher.

set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EVALS_DIR="$ROOT/evals"
RESULTS_DIR="$EVALS_DIR/.results"
EVAL_TIMEOUT="${EVAL_TIMEOUT:-900}"

# Guard rail: this harness must never hard-fail a machine without the CLI.
if ! command -v claude >/dev/null 2>&1; then
  echo "SKIP: claude CLI not found on PATH; the eval harness needs Claude Code."
  echo "SKIP: nothing was run. Install claude and re-run to execute evals."
  exit 0
fi

# Portable timeout: run "$@" with a watcher that kills it after EVAL_TIMEOUT
# seconds (macOS ships no timeout binary; prefer it when present).
run_with_timeout() {
  if command -v timeout >/dev/null 2>&1; then
    timeout "$EVAL_TIMEOUT" "$@"
    return $?
  fi
  "$@" &
  local cmd_pid=$!
  (
    sleep "$EVAL_TIMEOUT"
    kill -TERM "$cmd_pid" 2>/dev/null
  ) &
  local watcher_pid=$!
  local status=0
  wait "$cmd_pid" || status=$?
  kill "$watcher_pid" 2>/dev/null
  wait "$watcher_pid" 2>/dev/null
  return "$status"
}

# Extract a top-level string value from a small JSON file (python3 when
# available, sed as a last resort).
json_get() {
  local file="$1" key="$2"
  if command -v python3 >/dev/null 2>&1; then
    python3 -c "import json,sys; print(json.load(open(sys.argv[1]))[sys.argv[2]])" "$file" "$key"
  else
    sed -n "s/.*\"$key\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p" "$file" | head -n 1
  fi
}

run_scenario() {
  local sdir="$1"
  local name
  name="$(basename "$sdir")"
  local out="$RESULTS_DIR/$name"
  local workspace="$out/workspace"

  echo "=== Scenario: $name ==="

  local agent
  agent="$(json_get "$sdir/scenario.json" agent)"
  if [ -z "$agent" ] || [ ! -f "$ROOT/agents/$agent" ]; then
    echo "FAIL: $name: agent '$agent' not found under agents/"
    return 1
  fi

  rm -rf "$out"
  mkdir -p "$workspace"
  if [ -d "$sdir/inputs" ]; then
    cp -R "$sdir/inputs/." "$workspace/"
  fi

  # Compose the prompt: agent playbook, separator, task.
  local prompt
  prompt="$(cat "$ROOT/agents/$agent"; printf '\n\n===== EVAL TASK (from the orchestrator) =====\n\n'; cat "$sdir/task.md")"

  echo "--- Running agent $agent (timeout ${EVAL_TIMEOUT}s, consumes real tokens)"
  (
    cd "$workspace" && run_with_timeout claude -p "$prompt" --permission-mode acceptEdits
  ) > "$out/transcript.md" 2> "$out/stderr.log"
  local agent_status=$?
  if [ "$agent_status" -ne 0 ]; then
    echo "WARN: agent run exited $agent_status (timeout or error); grading anyway. See $out/stderr.log"
  fi

  # Build the grading prompt: rubric + task + workspace listing + contents.
  local gp="$out/grading-prompt.md"
  {
    echo "You are grading the output of an AI agent against a weighted checklist rubric."
    echo "Score each criterion from 0 to its max_score, with a one-line evidence citation."
    echo "Respond with ONLY a JSON object, no prose and no code fences, shaped as:"
    echo '{"criteria": [{"name": "...", "score": N, "max_score": N, "evidence": "..."}], "total": N, "max_total": N}'
    echo
    echo "## Rubric (criteria.json)"
    echo
    cat "$sdir/criteria.json"
    echo
    echo "## Task the agent was given"
    echo
    cat "$sdir/task.md"
    echo
    echo "## Workspace file listing"
    echo
    (cd "$workspace" && find . -type f | sort | head -n 200)
    echo
    echo "## Workspace file contents (each file capped at 200 lines)"
    while IFS= read -r f; do
      echo
      echo "----- FILE: ${f#"$workspace"/} -----"
      head -n 200 "$f"
    done < <(find "$workspace" -type f | sort | head -n 200)
    echo
    echo "## Agent transcript (stdout, capped at 200 lines)"
    echo
    head -n 200 "$out/transcript.md"
  } > "$gp"

  echo "--- Grading (timeout ${EVAL_TIMEOUT}s, consumes real tokens)"
  run_with_timeout claude -p "$(cat "$gp")" > "$out/grade.raw" 2>> "$out/stderr.log"
  # Strip stray code fences if the judge added them anyway.
  sed '/^[[:space:]]*```/d' "$out/grade.raw" > "$out/grade.json"

  echo "--- Results: $out"
  if command -v python3 >/dev/null 2>&1; then
    python3 - "$out/grade.json" <<'PYEOF' || echo "WARN: grade.json did not parse; inspect it manually"
import json, sys
with open(sys.argv[1]) as fh:
    g = json.load(fh)
rows = g.get("criteria", [])
width = max([len(r.get("name", "")) for r in rows] + [9])
print(f"  {'criterion'.ljust(width)}  score")
for r in rows:
    print(f"  {r.get('name','?').ljust(width)}  {r.get('score','?')}/{r.get('max_score','?')}")
print(f"  {'TOTAL'.ljust(width)}  {g.get('total','?')}/{g.get('max_total','?')}")
PYEOF
  else
    echo "  (python3 not found; raw grade follows)"
    cat "$out/grade.json"
  fi
  echo
}

# Collect scenarios: $1 names one, otherwise every directory under evals/
# that contains a scenario.json (evals/.results is skipped by construction).
failures=0
if [ "$#" -ge 1 ] && [ -n "$1" ]; then
  if [ ! -f "$EVALS_DIR/$1/scenario.json" ]; then
    echo "ERROR: no scenario '$1' under evals/ (expected evals/$1/scenario.json)"
    exit 1
  fi
  run_scenario "$EVALS_DIR/$1" || failures=$((failures + 1))
else
  found=0
  for sdir in "$EVALS_DIR"/*/; do
    [ -f "$sdir/scenario.json" ] || continue
    found=1
    run_scenario "${sdir%/}" || failures=$((failures + 1))
  done
  if [ "$found" -eq 0 ]; then
    echo "No scenarios found under evals/."
  fi
fi

if [ "$failures" -gt 0 ]; then
  echo "Done with $failures scenario failure(s)."
  exit 1
fi
echo "Done. Results under evals/.results/."
