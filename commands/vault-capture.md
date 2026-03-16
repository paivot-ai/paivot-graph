---
description: Capture knowledge from the current session to the vault with auto-tagging and link suggestions
allowed-tools: ["Bash", "Read", "Grep", "Glob"]
---

# Vault Capture

Capture knowledge from the current session to the appropriate vault. Auto-derives tags, validates domains, suggests related links, and triages to the correct folder.

## Step 1: Load Context

Load the vault-knowledge skill to understand the controlled domain vocabulary and note template:

```bash
cat ~/workspace/paivot/paivot-graph/skills/vault-knowledge/SKILL.md | head -100
```

Detect the current project:

```bash
project=$(git remote get-url origin 2>/dev/null | xargs basename -s .git || basename "$(pwd)")
```

## Step 2: Review Session for Capturable Knowledge

Scan the conversation for:

- **Decisions**: "chose X", "decided to", "went with", "trade-off"
- **Patterns**: "this approach", "reusable", "pattern", "anti-pattern"
- **Debug insights**: "root cause", "the issue was", "fixed by", "gotcha"
- **Concepts**: "learned that", "turns out", "works by"

For each finding, extract:
1. Title (concise, searchable)
2. Type (decision|pattern|debug|concept)
3. Summary (1-2 sentences)
4. Content (the actual knowledge)
5. Stack (technologies involved)
6. Domain (from controlled vocabulary)

## Step 3: Validate Domain

Check domain against controlled vocabulary:

```
ai-training, ai-inference, ai-agents, ai-nlp
developer-tools (legacy), dev-tools-cli, dev-tools-testing, dev-tools-workflow, dev-tools-knowledge
product-management
security-gateway, security-hardening, security-compliance
finance-quant, finance-fintech
frontend-ui, frontend-performance
calendar-sync
```

If domain doesn't match, suggest closest match or ask user to pick.

## Step 4: Determine Scope

Ask: "Would this knowledge help someone on a DIFFERENT project with a DIFFERENT codebase?"

- **Yes** -> Global vault, triage to folder based on type
- **No** -> Project vault `.vault/knowledge/<type>/`

## Step 5: Suggest Related Links

Before creating, search for related notes:

```bash
vlt vault="Claude" search query="<keywords from title>" --json
```

Present top 5 matches:
```
Related notes you may want to link:
1. [[Existing Note A]] - similar pattern
2. [[Existing Note B]] - same stack
3. [[Project X]] - used this approach
```

ALWAYS include at least the project note as a related link.

## Step 6: Derive Tags

Auto-derive tags based on type + domain:

| Domain | Tag |
|--------|-----|
| ai-training | `#ai/training` |
| ai-inference | `#ai/inference` |
| ai-agents | `#ai/agents` |
| ai-nlp | `#ai/nlp` |
| developer-tools | `#dev-tools/workflow` |
| dev-tools-cli | `#dev-tools/cli` |
| dev-tools-testing | `#dev-tools/testing` |
| dev-tools-workflow | `#dev-tools/workflow` |
| dev-tools-knowledge | `#dev-tools/knowledge` |
| product-management | `#product/management` |
| security-gateway | `#security/gateway` |
| security-hardening | `#security/hardening` |
| security-compliance | `#security/compliance` |
| finance-quant | `#finance/quant` |
| finance-fintech | `#finance/fintech` |
| frontend-ui | `#frontend/ui` |
| frontend-performance | `#frontend/performance` |
| calendar-sync | `#calendar/sync` |

## Step 7: Create Note

Build the note using the template:

```markdown
---
type: <type>
project: <project>
stack: [<stack>]
domain: <domain>
status: active
confidence: <high|medium|low>
created: YYYY-MM-DD
---

# <Title>

<Summary - 1-2 sentences>

## Content

<Main body>

## Related

- [[<Project>]]
- [[<Related Note 1>]]

## Tags

<#derived-tag-1> <#derived-tag-2>
```

Create in `_inbox/` first:

```bash
vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="<full-content>" silent timestamps
```

## Step 8: Triage Immediately

Move to the correct folder based on type:

```bash
vlt vault="Claude" move path="_inbox/<Title>.md" to="<type>s/<Title>.md"
```

Folder mapping:
- decision -> decisions/
- pattern -> patterns/
- debug -> debug/
- concept -> concepts/
- convention -> conventions/

## Step 9: Update Project Note

Append a session update to the project note:

```bash
vlt vault="Claude" append file="<Project>" content="

## Session $(date +%Y-%m-%d)
- <brief summary of what was done>
- Captured: [[<Note 1>]], [[<Note 2>]]"
```

If project note doesn't exist, create it:

```bash
vlt vault="Claude" create name="<Project>" path="projects/<Project>.md" content="---
type: project
project: <project>
stack: [<detected-stack>]
domain: <project-domain>
status: active
confidence: high
created: $(date +%Y-%m-%d)
---

# <Project>

<Brief description>

## Related

- [[<Note 1>]]

## Tags

#<project-domain-tag>" silent timestamps
```

## Step 10: Report Summary

```
## Vault Capture Summary

Project: <name>
Date: <today>

### Captured
- [decision] [[Note Title]] -> decisions/ (#domain-tag)
- [pattern] [[Note Title]] -> patterns/ (#domain-tag)

### Linked To
- [[Project]]
- [[Related Note A]]
- [[Related Note B]]

### Skipped
- <insight> (already exists as [[Existing Note]])

Total: N new notes, M links added
```

## Validation Checklist

Before completing, verify:

- [ ] All domains are from controlled vocabulary
- [ ] Each note has at least one wikilink in "Related"
- [ ] Tags are derived from domain, not manually added
- [ ] Note is triaged to correct folder (not left in _inbox)
- [ ] Project note is updated with session summary

## If Nothing to Capture

```
## Vault Capture Summary

Project: <name>
No new cross-project knowledge to capture.

Session-specific details logged to project execution (not vault-worthy).
```

## If Vault Directory is Missing

If `vlt vault="Claude"` fails:
1. Report that the vault path was not found
2. Output notes as markdown for manual addition
3. Suggest running `vlt vaults` to discover vault paths
