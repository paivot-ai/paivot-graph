---
description: Refine vault-backed content based on session experience. Review what happened, identify improvements to agent prompts, skill content, or operating mode, and update the relevant vault notes. System-scoped notes get proposals; project-scoped notes get direct edits.
allowed-tools: ["Bash", "Read", "Glob", "Grep"]
---

# Vault Evolve -- Refine Vault Content from Experience

Review the current session's work and refine the vault notes that power paivot-graph. This closes the feedback loop: work produces experience, experience refines the vault, refined vault improves future work.

**Vault:** `vlt vault="Claude"` (resolves path dynamically)

**Scope rules:**
- `scope: system` (or no scope property) -- propose changes only; user must approve via `/vault-triage`
- `scope: project` -- apply changes directly to `.vault/knowledge/` in the project repo

## Step 1: Assess What Happened

Review the conversation so far. Identify:
- What tasks were completed
- What friction was encountered (agent prompts that were unclear, missing context, wrong defaults)
- What patterns emerged that should be codified
- What decisions were made that should be recorded

## Step 2: Identify Vault Notes to Update

Check which vault-backed content could be improved:

### Agent prompts (methodology/)

Find agent notes:
```bash
vlt vault="Claude" files folder="methodology"
```

Read any that need review, including all notes they link to:
```bash
vlt vault="Claude" read file="<Agent Name>" follow
```

Look for:
- Instructions that were unclear or missing (agent got confused or went off-track)
- Workflow steps that should be reordered
- Quality checks that should be added
- Mode descriptions that need refinement

### Skill content (conventions/)

```bash
vlt vault="Claude" read file="Vault Knowledge Skill" follow
```

Look for:
- Capture patterns that should be updated
- Search strategies that worked well
- Frontmatter conventions that evolved

### Behavioral notes (conventions/)

```bash
vlt vault="Claude" read file="Session Operating Mode" follow
```
This returns Session Operating Mode plus all linked notes (Pre-Compact Checklist, Stop Capture Checklist, etc.) in a single call.

Look for:
- Operating mode instructions that were ignored (make them more explicit)
- Checklist items that were not useful (remove or rephrase)
- Missing checklist items (add them)

### Project-local knowledge (.vault/knowledge/)

Check if the project has local knowledge:
```bash
vlt vault=".vault/knowledge" files
```

Look for project-specific conventions, patterns, or decisions that need updating.

### Promotion candidates (project -> system)

Review project-local notes for knowledge that has proven **universally useful** -- patterns, decisions, or debug insights that would benefit other projects too. Only promote when the knowledge is genuinely cross-project; most project knowledge should stay local.

Criteria for promotion:
- The pattern has been validated across multiple sessions or use cases within this project
- The insight applies to the technology/stack broadly, not just this project's specific setup
- The convention would improve consistency across all projects, not just this one

To find candidates:
```bash
vlt vault=".vault/knowledge" search query="scope: project"
```

Read each candidate and evaluate whether it should be promoted. If yes, create a **promotion proposal** in Step 3.

### Skill synthesis candidates

Scan `patterns/`, `decisions/`, and `debug/` in both the project vault (`.vault/knowledge/`) and the system vault for clusters of related notes that could be distilled into a reusable skill.

Criteria:
- 3+ related notes on the same topic within a single vault, OR
- 2+ notes on the same topic from different projects
- The notes must describe HOW to do something (process, technique, workflow), not just WHAT was decided

To find candidates in the system vault:
```bash
vlt vault="Claude" files folder="patterns"
vlt vault="Claude" files folder="decisions"
vlt vault="Claude" files folder="debug"
```

To find candidates in the project vault:
```bash
vlt vault=".vault/knowledge" files folder="patterns"
vlt vault=".vault/knowledge" files folder="decisions"
vlt vault=".vault/knowledge" files folder="debug"
```

Read the notes and look for clusters -- multiple notes that share a common theme and together describe a repeatable process.

## Step 3: Determine Scope and Apply

For each improvement identified, **read the target note's frontmatter first** and check the `scope:` property.

### If `scope: system` (or no scope -- defaults to system):

**DO NOT modify the note directly.** Instead, create a proposal:

1. Read the full current content of the target note (this becomes the rollback snapshot).
2. Create a proposal note in the vault `_inbox/`:

```bash
vlt vault="Claude" create name="Proposal -- <Target Note>" path="_inbox/Proposal -- <Target Note>.md" content="---
type: proposal
scope: system
target: \"<full vault path of target note>\"
project: <originating-project>
status: pending
created: <YYYY-MM-DD>
---

# Proposed change: <Target Note>

## Motivation
<what session experience revealed the need for this change>

## Change
### Before
<relevant section of the current note>

### After
<proposed replacement>

## Snapshot (for rollback)
<full content of the target note at time of proposal>

## Impact
Affects all projects using <Target Note>." silent
```

3. Tell the user: "Created proposal for <note>. Run /vault-triage to review and apply."

### If `scope: project`:

Apply changes directly to `.vault/knowledge/` via vlt:

1. For targeted section updates:
```bash
vlt vault=".vault/knowledge" patch file="<Note>" heading="<heading>" content="<new section content>"
```

2. For full note replacement:
```bash
vlt vault=".vault/knowledge" write file="<Note>" content="<full new content>"
```

3. For new notes:
```bash
vlt vault=".vault/knowledge" create name="<Note Title>" path="<subfolder>/<Note Title>.md" content="..." silent
```

4. Append to changelog:
```bash
vlt vault=".vault/knowledge" append file="changelog" content="
- <YYYY-MM-DD>: Updated <note> -- <what changed and why>"
```

When updating any note, be conservative:
- Add clarifying instructions, do not remove existing ones without good reason
- Add examples of what went wrong and how to avoid it
- Preserve the overall structure

### Promotion proposals (project -> system)

For project-local notes identified as promotion candidates in Step 2, create a promotion proposal in the global vault's `_inbox/`:

```bash
vlt vault="Claude" create name="Promotion -- <Note Title>" path="_inbox/Promotion -- <Note Title>.md" content="---
type: proposal
scope: system
promotion_from: project
source_project: <originating-project>
source_path: \".vault/knowledge/<subfolder>/<Note>.md\"
target_path: \"<target folder>/<Note>.md\"
status: pending
created: <YYYY-MM-DD>
---

# Promotion: <Note Title>

## Source
Project: <project-name>
Path: .vault/knowledge/<subfolder>/<Note>.md

## Rationale
<why this knowledge is universally useful, not just project-specific>

## Content
<full content of the project-local note>

## Suggested target
<target folder>/<Note>.md (e.g., patterns/, decisions/, debug/)

## Impact
Would benefit all projects working with <relevant stack/domain>." silent
```

Tell the user: "Created promotion proposal for <note>. Run /vault-triage to review."

**Do NOT delete the project-local note.** It stays in the project vault regardless of whether the promotion is accepted. The system vault gets its own copy.

### Skill synthesis

For each cluster of related notes identified in Step 2 as a skill synthesis candidate:

**Project-scope skills** (all source notes are from this project):

Create the skill via vlt:
```bash
vlt vault=".vault/knowledge" create name="<Skill Name>" path="skills/<skill-name>.md" content="---
type: skill
scope: project
project: <project-name>
source_notes:
  - \"<subfolder>/<Note A>.md\"
  - \"<subfolder>/<Note B>.md\"
  - \"<subfolder>/<Note C>.md\"
status: active
created: <YYYY-MM-DD>
---

# <Skill Name>

<Synthesized how-to content distilled from the source notes. Should be actionable and self-contained.>

## When to use

<Conditions under which this skill applies>

## Steps

<Step-by-step process>

## Source

Synthesized from:
- [[<Note A>]]
- [[<Note B>]]
- [[<Note C>]]" silent
```

**System-scope skills** (source notes span multiple projects):

Create a proposal in the system vault `_inbox/`:
```bash
vlt vault="Claude" create name="Skill Proposal -- <Name>" path="_inbox/Skill Proposal -- <Name>.md" content="---
type: skill-proposal
scope: system
source_notes:
  - \"<project-a>/<subfolder>/<Note>.md\"
  - \"<project-b>/<subfolder>/<Note>.md\"
source_projects:
  - <project-a>
  - <project-b>
status: pending
created: <YYYY-MM-DD>
---

# Skill Proposal: <Name>

## Motivation
<Why these notes should become a skill>

## Synthesized Skill Content
<The distilled how-to content>

## When to use
<Conditions under which this skill applies>

## Steps
<Step-by-step process>

## Source Notes
<Full list of notes that informed this skill, with project context>" silent
```

Tell the user: "Created skill proposal for <Name>. Run /vault-triage to review and apply."

## Step 4: Report Changes

Separate the report into three sections:

```
## Vault Evolve Summary

### Proposals Created (system scope -- requires /vault-triage)
- Proposal for <Note A>: <what would change and why>
- Proposal for <Note B>: <what would change and why>

### Promotions Proposed (project -> system -- requires /vault-triage)
- Promotion for <Note C>: <why it's universally useful>

### Skills Proposed (system scope -- requires /vault-triage)
- Skill Proposal for <Name>: <what it synthesizes and from which notes>

### Skills Created (project scope -- applied directly)
- Created .vault/knowledge/skills/<name>.md: <what it synthesizes>

### Changes Applied (project scope -- applied directly)
- Updated .vault/knowledge/<path>: <what changed>
- Created .vault/knowledge/<path>: <why>

### No Changes Needed
- <Notes reviewed but found adequate>
```

## Constraints

- Only modify vault notes, never modify the plugin's static files (those are fallbacks)
- Keep changes grounded in actual session experience, not hypothetical improvements
- If unsure whether a change is warranted, describe it to the user and ask
- NEVER directly modify a system-scoped note -- always create a proposal
