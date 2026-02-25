#!/usr/bin/env bash
# vault-scope-guard.sh -- Block direct writes to system-scoped vault notes.
#
# PreToolUse hook for Edit and Write tools. Reads the tool input JSON from stdin,
# checks if the target file is inside the vault's methodology/ or conventions/
# directories, and blocks the operation if so.
#
# This makes knowledge governance structural, not advisory.
# System notes must be changed via /vault-triage proposals, not direct edits.
#
# Exit codes:
#   0 = allow the tool call
#   2 = block the tool call (with reason on stdout)

set -euo pipefail

VAULT_DIR="$HOME/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude"

# Read hook input
hook_input="$(cat)"

# Extract the file path from the tool input (Edit uses file_path, Write uses file_path)
file_path="$(printf '%s' "$hook_input" | python3 -c "
import sys, json
data = json.load(sys.stdin)
# Tool input is nested under 'tool_input'
tool_input = data.get('tool_input', {})
print(tool_input.get('file_path', ''))
" 2>/dev/null || echo "")"

# If we couldn't extract a path, allow (don't block on parse failures)
if [ -z "$file_path" ]; then
    exit 0
fi

# Check if the path targets a protected vault directory
protected_dirs=(
    "$VAULT_DIR/methodology/"
    "$VAULT_DIR/conventions/"
)

for protected in "${protected_dirs[@]}"; do
    if [[ "$file_path" == "$protected"* ]]; then
        note_name="$(basename "$file_path")"
        cat <<BLOCKED
BLOCKED: Direct modification of system-scoped vault note "$note_name".

System notes (methodology/, conventions/) are protected by knowledge governance.
To change this note:
  1. Run /vault-evolve to create a proposal
  2. Run /vault-triage to review and apply it

This ensures all projects using this note are aware of the change.
BLOCKED
        exit 2
    fi
done

# Not a protected path -- allow
exit 0
