#!/usr/bin/env bash
# vault-session-start.sh -- Consult the Obsidian vault for project context on session start.
#
# Reads the SessionStart hook JSON from stdin, extracts cwd, detects the project name,
# searches the vault, and outputs relevant context to stdout (injected into Claude's awareness).
#
# Always exits 0 -- never blocks session start.

set -euo pipefail

VAULT="Claude"

# ---------------------------------------------------------------------------
# 1. Read hook input and extract cwd
# ---------------------------------------------------------------------------
hook_input="$(cat)"
cwd="$(printf '%s' "$hook_input" | python3 -c "import sys,json; print(json.load(sys.stdin).get('cwd',''))" 2>/dev/null || echo "")"

if [ -z "$cwd" ]; then
    cwd="$(pwd)"
fi

# ---------------------------------------------------------------------------
# 2. Detect project name (git remote basename > directory name)
# ---------------------------------------------------------------------------
project=""
if [ -d "$cwd/.git" ] || git -C "$cwd" rev-parse --git-dir >/dev/null 2>&1; then
    remote_url="$(git -C "$cwd" remote get-url origin 2>/dev/null || echo "")"
    if [ -n "$remote_url" ]; then
        project="$(basename "$remote_url" .git)"
    fi
fi

if [ -z "$project" ]; then
    project="$(basename "$cwd")"
fi

# ---------------------------------------------------------------------------
# 3. Check if obsidian CLI is available
# ---------------------------------------------------------------------------
if ! command -v obsidian >/dev/null 2>&1; then
    echo "[VAULT] Obsidian CLI not available -- vault consultation skipped."
    echo "Install: https://github.com/Acylation/obsidian-cli"
    exit 0
fi

# ---------------------------------------------------------------------------
# 4. Search vault for project context
# ---------------------------------------------------------------------------
# Filter obsidian CLI noise (app loading messages, timestamps)
search_results="$(obsidian vault="$VAULT" search query="$project" 2>/dev/null \
    | grep -v "^[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}" \
    | grep -v "^Loading " \
    | grep -v "^No matches found" \
    || echo "")"

# Trim whitespace
search_results="$(printf '%s' "$search_results" | sed '/^$/d')"

if [ -z "$search_results" ]; then
    echo "[VAULT] Project: $project"
    echo "No existing vault knowledge found for this project."
    echo "Consider creating a project note: obsidian vault=\"$VAULT\" create name=\"$project\" path=\"projects/$project.md\" content=\"---"
    echo "type: project"
    echo "project: $project"
    echo "status: active"
    echo "created: $(date +%Y-%m-%d)"
    echo "---"
    echo ""
    echo "# $project\" silent"
    exit 0
fi

# ---------------------------------------------------------------------------
# 5. Output structured context
# ---------------------------------------------------------------------------
echo "[VAULT] Project: $project"
echo "Relevant vault knowledge found:"
echo ""
echo "$search_results"
echo ""
echo "Consult these notes before starting work:"
echo "  obsidian vault=\"$VAULT\" read file=\"$project\""
echo ""
echo "Remember: Update the vault with decisions, patterns, and debug insights as you work."

exit 0
