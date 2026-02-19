#!/usr/bin/env bash
# vault-session-end.sh -- Append session log entry to the project index note.
#
# Fires on session end. Updates the project index note with a session log entry.
# Fire-and-forget: always exits 0, silently skips if vault is unavailable.

set -euo pipefail

VAULT_DIR="$HOME/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude"
TODAY="$(date +%Y-%m-%d)"

# ---------------------------------------------------------------------------
# 1. Read hook input and extract cwd
# ---------------------------------------------------------------------------
hook_input="$(cat)"
cwd="$(printf '%s' "$hook_input" | python3 -c "import sys,json; print(json.load(sys.stdin).get('cwd',''))" 2>/dev/null || echo "")"

if [ -z "$cwd" ]; then
    cwd="$(pwd)"
fi

# ---------------------------------------------------------------------------
# 2. Detect project name
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
# 3. Append session log entry (prefer vlt, fallback to direct file ops)
# ---------------------------------------------------------------------------
session_entry="$(printf '\n\n## Session log (%s)\n- Session ended normally\n' "$TODAY")"

if command -v vlt >/dev/null 2>&1; then
    vlt vault="Claude" append file="$project" content="$session_entry" 2>/dev/null || true
    exit 0
fi

if [ ! -d "$VAULT_DIR" ]; then
    exit 0
fi

# Fallback: direct file ops
project_note=""
if [ -f "$VAULT_DIR/projects/${project}.md" ]; then
    project_note="$VAULT_DIR/projects/${project}.md"
elif [ -f "$VAULT_DIR/${project}.md" ]; then
    project_note="$VAULT_DIR/${project}.md"
fi

if [ -n "$project_note" ]; then
    printf '%s' "$session_entry" >> "$project_note"
fi

exit 0
