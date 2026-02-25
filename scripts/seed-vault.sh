#!/usr/bin/env bash
# seed-vault.sh -- Write vault notes directly to disk.
#
# Writes files directly to the vault directory on disk. Obsidian picks them up
# automatically via iCloud sync. No obsidian CLI needed.
#
# Usage:
#   seed-vault.sh           Idempotent: creates missing notes, skips existing.
#   seed-vault.sh --force   Overwrites existing notes with latest content.

set -euo pipefail

TODAY="$(date +%Y-%m-%d)"
FORCE=false
if [[ "${1:-}" == "--force" ]]; then
    FORCE=true
fi

# Vault on disk
VAULT_DIR="$HOME/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude"
# Resolve AGENT_SRC dynamically: find the latest paivot-claude cache with agents/
if [ -z "${AGENT_SRC:-}" ]; then
    AGENT_SRC="$(find "$HOME/.claude/plugins/cache/paivot-claude" -maxdepth 3 -type d -name agents 2>/dev/null \
        | sort -V | tail -1 || echo "")"
    if [ -z "$AGENT_SRC" ]; then
        echo "ERROR: Could not find paivot-claude agents directory in plugin cache."
        echo "       Set AGENT_SRC=/path/to/agents manually, or install paivot-claude first."
        exit 1
    fi
fi
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PLUGIN_DIR="$(dirname "$SCRIPT_DIR")"

# Counters use temp files to survive subshells (pipes create subshells)
count_dir="$(mktemp -d)"
trap 'rm -rf "$count_dir"' EXIT
echo 0 > "$count_dir/created"
echo 0 > "$count_dir/updated"
echo 0 > "$count_dir/skipped"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

extract_body() {
    local file="$1"
    awk 'BEGIN{c=0} /^---[[:space:]]*$/{c++; next} c>=2{print}' "$file"
}

inc_created() { echo $(( $(cat "$count_dir/created") + 1 )) > "$count_dir/created"; }
inc_updated() { echo $(( $(cat "$count_dir/updated") + 1 )) > "$count_dir/updated"; }
inc_skipped() { echo $(( $(cat "$count_dir/skipped") + 1 )) > "$count_dir/skipped"; }

write_note() {
    local dest="$1"  # relative path within vault
    local full_path="$VAULT_DIR/$dest"

    if [ -f "$full_path" ]; then
        if [ "$FORCE" = true ]; then
            cat > "$full_path"
            echo "  UPDATED: $dest"
            inc_updated
            return 0
        else
            echo "  SKIP: $dest (already exists)"
            inc_skipped
            return 0
        fi
    fi

    # Ensure parent directory exists
    mkdir -p "$(dirname "$full_path")"

    # Write from stdin
    cat > "$full_path"
    echo "  CREATED: $dest"
    inc_created
}

# ---------------------------------------------------------------------------
# Preflight
# ---------------------------------------------------------------------------

echo "paivot-graph vault seeder"
echo "========================="
if [ "$FORCE" = true ]; then
    echo "Mode: force (overwriting existing notes)"
else
    echo "Mode: safe (skipping existing notes)"
fi
echo ""

if [ ! -d "$VAULT_DIR" ]; then
    echo "ERROR: Vault directory not found at $VAULT_DIR"
    exit 1
fi

if [ ! -d "$AGENT_SRC" ]; then
    echo "ERROR: Agent source not found at $AGENT_SRC"
    exit 1
fi

# ---------------------------------------------------------------------------
# 1. Agent prompts (8 agents)
# ---------------------------------------------------------------------------

echo "Seeding agent prompts..."

seed_agent() {
    local slug="$1"
    local vault_name="$2"
    local src_file="$AGENT_SRC/${slug}.md"

    if [ ! -f "$src_file" ]; then
        echo "  WARN: $src_file not found, skipping $vault_name"
        inc_skipped
        return 0
    fi

    local body
    body="$(extract_body "$src_file")"

    cat <<AGENT_EOF | write_note "methodology/${vault_name}.md"
---
type: methodology
scope: system
project: paivot
stack: [claude-code]
domain: developer-tools
status: active
created: $TODAY
---

$body

## Changelog

- $TODAY: Seeded from paivot-graph plugin (initial version)
AGENT_EOF
}

seed_agent "paivot-sr-pm"             "Sr PM Agent"
seed_agent "paivot-pm"                "PM Acceptor Agent"
seed_agent "paivot-developer"         "Developer Agent"
seed_agent "paivot-architect"         "Architect Agent"
seed_agent "paivot-designer"          "Designer Agent"
seed_agent "paivot-business-analyst"  "Business Analyst Agent"
seed_agent "paivot-anchor"            "Anchor Agent"
seed_agent "paivot-retro"             "Retro Agent"

# ---------------------------------------------------------------------------
# 2. Skill content
# ---------------------------------------------------------------------------

echo ""
echo "Seeding skill content..."

skill_src="$PLUGIN_DIR/skills/vault-knowledge/SKILL.md"
if [ -f "$skill_src" ]; then
    skill_body="$(extract_body "$skill_src")"

    cat <<SKILL_EOF | write_note "conventions/Vault Knowledge Skill.md"
---
type: convention
scope: system
project: paivot-graph
stack: [claude-code, obsidian]
domain: developer-tools
status: active
created: $TODAY
---

$skill_body

## Changelog

- $TODAY: Seeded from paivot-graph plugin (initial version)
SKILL_EOF
else
    echo "  WARN: $skill_src not found"
    inc_skipped
fi

# ---------------------------------------------------------------------------
# 3. Behavioral notes
# ---------------------------------------------------------------------------

echo ""
echo "Seeding behavioral notes..."

cat <<SOM_EOF | write_note "conventions/Session Operating Mode.md"
---
type: convention
scope: system
project: paivot-graph
stack: [claude-code, obsidian]
domain: developer-tools
status: active
created: $TODAY
---

# Session Operating Mode

CONCURRENCY LIMITS (HARD RULE -- unless user explicitly overrides):
  - Maximum 2 developer agents running simultaneously
  - Maximum 1 PM-Acceptor agent running simultaneously
  - Total active subagents (all types) must not exceed 3
  These limits prevent context exhaustion. Violating them risks losing the entire session.

BEFORE STARTING: Read the vault notes listed above. Do not rediscover what is already known.
  vlt vault="Claude" read file="<note>"

WHILE WORKING: Capture knowledge as it emerges -- do not wait for the end.
  - After making a decision (chose X over Y): create a decision note
  - After solving a non-obvious bug: create a debug note
  - After discovering a reusable pattern: create a pattern note
  Use: vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="..." silent

BEFORE ENDING: Update the project index note with what was accomplished.
  vlt vault="Claude" append file="<Project>" content="## Session update (<date>)\n- <what was done>"

This is not optional. Knowledge that is not captured is knowledge that will be rediscovered at cost.

## Related

- [[paivot-graph]] -- Plugin that reads this note at session start
- [[Vault as runtime not reference]] -- Why this content lives in the vault
- [[Vault Knowledge Skill]] -- How to interact with the vault
- [[Pre-Compact Checklist]] -- Companion checklist before compaction
- [[Stop Capture Checklist]] -- Companion checklist before stopping

## Changelog

- $TODAY: Seeded from paivot-graph plugin (initial version)
SOM_EOF

cat <<PCL_EOF | write_note "conventions/Pre-Compact Checklist.md"
---
type: convention
scope: system
project: paivot-graph
stack: [claude-code, obsidian]
domain: developer-tools
status: active
created: $TODAY
---

# Pre-Compact Checklist

Context compaction is imminent. Save anything worth remembering NOW.

## 1. DECISIONS made this session

Record any decisions with rationale and alternatives considered:
  vlt vault="Claude" create name="<Decision Title>" path="_inbox/<Decision Title>.md" content="..." silent

Include frontmatter: type: decision, project: <project>, status: active, confidence: high, created: <YYYY-MM-DD>
Include sections: Decision, Rationale, Alternatives considered.

## 2. PATTERNS discovered

Record reusable solutions:
  vlt vault="Claude" create name="<Pattern Name>" path="_inbox/<Pattern Name>.md" content="..." silent

Include frontmatter: type: pattern, project: <project>, stack: [], status: active, created: <YYYY-MM-DD>
Include sections: When to use, Implementation.

## 3. DEBUG INSIGHTS

Record problems solved:
  vlt vault="Claude" create name="<Bug Title>" path="_inbox/<Bug Title>.md" content="..." silent

Include frontmatter: type: debug, project: <project>, status: active, created: <YYYY-MM-DD>
Include sections: Symptoms, Root cause, Fix.

## 4. PROJECT UPDATES

  vlt vault="Claude" append file="<Project>" content="## Session update (<YYYY-MM-DD>)\n- <what was accomplished>"

Do this NOW -- after compaction, the details will be lost.

## Changelog

- $TODAY: Seeded from paivot-graph plugin (initial version)
PCL_EOF

cat <<SCL_EOF | write_note "conventions/Stop Capture Checklist.md"
---
type: convention
scope: system
project: paivot-graph
stack: [claude-code, obsidian]
domain: developer-tools
status: active
created: $TODAY
---

# Stop Capture Checklist

Before ending this session, confirm you have considered each of these:

- [ ] Did you capture any DECISIONS made this session? (chose X over Y, established a convention)
- [ ] Did you capture any PATTERNS discovered? (reusable solutions, idioms, workflows)
- [ ] Did you capture any DEBUG INSIGHTS? (non-obvious bugs, sharp edges, environment issues)
- [ ] Did you update the PROJECT INDEX NOTE with what was accomplished?

If none of the above apply (e.g., quick fix, trivial session), that is fine -- but confirm it was considered, not forgotten.

Use vlt to create notes: vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="..." silent

## Changelog

- $TODAY: Seeded from paivot-graph plugin (initial version)
SCL_EOF

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

echo ""
echo "Done. Created: $(cat "$count_dir/created"), Updated: $(cat "$count_dir/updated"), Skipped: $(cat "$count_dir/skipped")"
