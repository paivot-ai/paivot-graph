#!/usr/bin/env bash
# vault-stop.sh -- Soft capture reminder when Claude tries to stop.
#
# Reads the Stop Capture Checklist from the vault (or uses static fallback).
# Outputs a reminder to stdout. Exits 0 (soft reminder, not hard block).
#
# To make this a hard block, change exit 0 to exit 2 and update the vault
# note to say "DO NOT STOP UNTIL CAPTURE IS CONFIRMED".

set -euo pipefail

VAULT_DIR="$HOME/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude"

# ---------------------------------------------------------------------------
# Try to read the checklist from the vault (prefer vlt, fallback to cat)
# ---------------------------------------------------------------------------
checklist=""
if command -v vlt >/dev/null 2>&1; then
    checklist="$(vlt vault="Claude" read file="Stop Capture Checklist" 2>/dev/null || echo "")"
fi
if [ -z "$checklist" ]; then
    checklist_file="$VAULT_DIR/conventions/Stop Capture Checklist.md"
    if [ -f "$checklist_file" ]; then
        checklist="$(cat "$checklist_file")"
    fi
fi

# ---------------------------------------------------------------------------
# Output checklist (vault or fallback)
# ---------------------------------------------------------------------------
if [ -n "$checklist" ]; then
    echo "[VAULT] Stop capture check (from vault):"
    echo ""
    echo "$checklist"
else
    cat <<'FALLBACK'
[VAULT] Stop capture check:

Before ending this session, confirm you have considered each of these:

- [ ] Did you capture any DECISIONS made this session?
- [ ] Did you capture any PATTERNS discovered?
- [ ] Did you capture any DEBUG INSIGHTS?
- [ ] Did you update the PROJECT INDEX NOTE?
- [ ] Did you capture project-specific knowledge to .vault/knowledge/?

If none apply (trivial session), that is fine -- but confirm it was considered.

Use: vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="..." silent
FALLBACK
fi

# Two-tier reminder (only if project vault exists)
if [ -d ".vault/knowledge" ]; then
    cat <<'TIER'

[VAULT] Remember: save to the right tier.
  - Universal insights -> global vault (_inbox/)
  - Project-specific insights -> .vault/knowledge/ (local)
TIER
fi

exit 0
