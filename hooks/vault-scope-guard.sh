#!/usr/bin/env bash
# vault-scope-guard.sh -- Block direct writes to system-scoped vault notes.
#
# PreToolUse hook for Edit, Write, and Bash tools. Reads the tool input JSON
# from stdin, checks if the operation targets a protected vault directory,
# and blocks it if so.
#
# For Edit/Write: checks the file_path parameter.
# For Bash: checks if the command contains a redirect (>, >>, tee) targeting
# the vault path. This is a heuristic -- it catches common patterns but a
# determined actor could still bypass it. The goal is to catch accidental
# writes, not to be an unbreakable sandbox.
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

# Determine which tool is being called and extract the relevant content
tool_and_path="$(printf '%s' "$hook_input" | python3 -c "
import sys, json
data = json.load(sys.stdin)
tool_name = data.get('tool_name', '')
tool_input = data.get('tool_input', {})
if tool_name in ('Edit', 'Write'):
    print('file|' + tool_input.get('file_path', ''))
elif tool_name == 'Bash':
    print('bash|' + tool_input.get('command', ''))
else:
    print('unknown|')
" 2>/dev/null || echo "unknown|")"

tool_type="${tool_and_path%%|*}"
tool_content="${tool_and_path#*|}"

# If we couldn't parse, allow (don't block on parse failures)
if [ -z "$tool_content" ] || [ "$tool_type" = "unknown" ]; then
    exit 0
fi

# Protected vault directories (everything except _inbox/ and _templates/)
# _inbox/ must remain writable for proposals and new captures
protected_dirs=(
    "$VAULT_DIR/methodology/"
    "$VAULT_DIR/conventions/"
    "$VAULT_DIR/decisions/"
    "$VAULT_DIR/patterns/"
    "$VAULT_DIR/debug/"
    "$VAULT_DIR/concepts/"
    "$VAULT_DIR/projects/"
    "$VAULT_DIR/people/"
)

block_with_message() {
    local target="$1"
    local folder
    folder="$(echo "$target" | sed "s|$VAULT_DIR/||" | cut -d'/' -f1)"
    cat <<BLOCKED
BLOCKED: Direct modification of system-scoped vault content in ${folder}/.

System vault directories are protected by knowledge governance.
To change system notes:
  1. Run /vault-evolve to create a proposal
  2. Run /vault-triage to review and apply it

Only _inbox/ is writable directly (for proposals and new captures).
BLOCKED
    exit 2
}

if [ "$tool_type" = "file" ]; then
    # Edit or Write -- check file_path against protected dirs
    for protected in "${protected_dirs[@]}"; do
        if [[ "$tool_content" == "$protected"* ]]; then
            block_with_message "$tool_content"
        fi
    done
elif [ "$tool_type" = "bash" ]; then
    # Bash -- check if command contains a redirect to a protected vault path
    # Catch patterns: > "vault/path", >> "vault/path", tee "vault/path",
    # cat > vault/path, echo > vault/path, vlt ... (vlt is allowed -- it's
    # the intended mechanism for proposals)
    #
    # We specifically look for shell redirects and file-writing commands
    # targeting vault protected dirs. vlt commands are allowed through.
    if echo "$tool_content" | grep -q "^vlt " || echo "$tool_content" | grep -q "^vlt "; then
        # vlt commands are the intended mechanism -- allow
        exit 0
    fi
    for protected in "${protected_dirs[@]}"; do
        # Check for redirects: >, >>, or tee targeting protected dirs
        # Use a broad pattern match -- the vault path contains spaces so
        # it will typically be quoted in commands
        # shellcheck disable=SC2016
        vault_escaped="$(printf '%s' "$protected" | sed 's/[.[\*^$()+?{|]/\\&/g')"
        if echo "$tool_content" | grep -qE "(>|>>|tee\s).*$vault_escaped"; then
            block_with_message "$protected"
        fi
        # Also check for cp/mv targeting the vault
        if echo "$tool_content" | grep -qE "(cp|mv)\s.*$vault_escaped"; then
            block_with_message "$protected"
        fi
    done
fi

# Not a protected operation -- allow
exit 0
