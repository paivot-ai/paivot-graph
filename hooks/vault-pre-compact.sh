#!/usr/bin/env bash
# vault-pre-compact.sh -- Remind Claude to capture knowledge before context compaction.
#
# This is the last chance to save what was learned before memory is lost.
# Outputs a structured reminder to stdout. Always exits 0.

set -euo pipefail

VAULT="Claude"
TODAY="$(date +%Y-%m-%d)"

cat <<EOF
[VAULT] Context compaction imminent -- capture knowledge now.

Before this context is compacted, save anything worth remembering:

1. DECISIONS made this session (with rationale and alternatives considered):
   obsidian vault="$VAULT" create name="<Decision Title>" path="_inbox/<Decision Title>.md" content="---
type: decision
project: <project>
status: active
confidence: high
created: $TODAY
---

# <Decision Title>

## Decision
<what was decided>

## Rationale
<why>

## Alternatives considered
- <alt 1>
- <alt 2>" silent

2. PATTERNS discovered (reusable solutions):
   obsidian vault="$VAULT" create name="<Pattern Name>" path="_inbox/<Pattern Name>.md" content="---
type: pattern
project: <project>
stack: []
status: active
created: $TODAY
---

# <Pattern Name>

## When to use
<context>

## Implementation
<how>" silent

3. DEBUG INSIGHTS (problems solved):
   obsidian vault="$VAULT" create name="<Bug Title>" path="_inbox/<Bug Title>.md" content="---
type: debug
project: <project>
status: active
created: $TODAY
---

# <Bug Title>

## Symptoms
<what happened>

## Root cause
<why>

## Fix
<how it was resolved>" silent

4. PROJECT UPDATES (progress, state changes):
   obsidian vault="$VAULT" append file="<Project>" content="

## Session update ($TODAY)
<what was accomplished>"

Do this NOW -- after compaction, the details will be lost.
EOF

exit 0
