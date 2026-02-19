#!/usr/bin/env bash
# vault-pre-compact.sh -- Remind Claude to capture knowledge before context compaction.
#
# Reads the Pre-Compact Checklist from the vault (or uses static fallback).
# This is the last chance to save what was learned before memory is lost.
# Outputs a structured reminder to stdout. Always exits 0.

set -euo pipefail

VAULT_DIR="$HOME/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude"

# ---------------------------------------------------------------------------
# Try to read the checklist from the vault (prefer vlt, fallback to cat)
# ---------------------------------------------------------------------------
checklist=""
if command -v vlt >/dev/null 2>&1; then
    checklist="$(vlt vault="Claude" read file="Pre-Compact Checklist" 2>/dev/null || echo "")"
fi
if [ -z "$checklist" ]; then
    checklist_file="$VAULT_DIR/conventions/Pre-Compact Checklist.md"
    if [ -f "$checklist_file" ]; then
        checklist="$(cat "$checklist_file")"
    fi
fi

# ---------------------------------------------------------------------------
# Output checklist (vault or fallback)
# ---------------------------------------------------------------------------
if [ -n "$checklist" ]; then
    echo "[VAULT] Context compaction imminent -- capture knowledge now."
    echo ""
    echo "$checklist"
else
    # Static fallback
    cat <<'EOF'
[VAULT] Context compaction imminent -- capture knowledge now.

Before this context is compacted, save anything worth remembering:

1. DECISIONS made this session (with rationale and alternatives considered):
   vlt vault="Claude" create name="<Decision Title>" path="_inbox/<Decision Title>.md" content="..." silent

2. PATTERNS discovered (reusable solutions):
   vlt vault="Claude" create name="<Pattern Name>" path="_inbox/<Pattern Name>.md" content="..." silent

3. DEBUG INSIGHTS (problems solved):
   vlt vault="Claude" create name="<Bug Title>" path="_inbox/<Bug Title>.md" content="..." silent

4. PROJECT UPDATES (progress, state changes):
   vlt vault="Claude" append file="<Project>" content="## Session update (<date>)\n- <what was accomplished>"

All notes must have frontmatter: type, project, status, created.

Do this NOW -- after compaction, the details will be lost.
EOF
fi

exit 0
