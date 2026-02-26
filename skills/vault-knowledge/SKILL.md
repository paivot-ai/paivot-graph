---
name: vault-knowledge
description: This skill should be used when working on any project to understand how to effectively interact with the Obsidian knowledge vault. It teaches when to capture knowledge, what to capture, how to format vault notes, and how to search effectively. Use when you need to "save to vault", "update vault", "capture a decision", "record a pattern", "log a debug insight", or when starting/ending a significant work session.
version: 0.4.0
---

# Vault Knowledge (Vault-Backed)

The Obsidian vault ("Claude") lives on disk. Interact with it using `vlt` (the fast vault CLI) via Bash, or directly using Read, Write, Grep, and Glob tools. Prefer `vlt` for vault-aware operations (search, create, move with wikilink repair, backlinks, tags). Use Read/Write/Grep/Glob when you need Claude Code tool integration or vlt is unavailable.

**Vault path:** `/Users/ramirosalas/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude`

Read the full skill content from the vault:

    /Users/ramirosalas/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/conventions/Vault Knowledge Skill.md

The vault version is authoritative. Follow it completely.

If the vault is unavailable, use these minimal instructions:

## Three-Tier Knowledge Model

Knowledge lives in three tiers with different governance rules:

### Tier 1: System Vault (global Obsidian "Claude")

Shared across ALL projects. Changes require user approval.

| Folder        | Scope  | Contains                        |
|---------------|--------|---------------------------------|
| methodology/  | system | Agent prompts                   |
| conventions/  | system | Operating mode, checklists, skill |
| decisions/    | system | Cross-project decisions         |
| patterns/     | system | Cross-project patterns          |
| debug/        | system | Cross-project debug insights    |
| concepts/     | system | Language/framework knowledge    |
| projects/     | system | Project index notes             |
| people/       | system | Team preferences                |
| _inbox/       | system | Unsorted capture (triage later) |

**Governance:** System notes are NEVER modified directly during a session. Changes go through a proposal workflow -- `/vault-evolve` creates proposals, `/vault-triage` reviews and applies them.

### Tier 2: Project Vault (`.vault/knowledge/` in each repo)

Scoped to a single project. Changes apply directly, no approval needed.

```
.vault/knowledge/
  decisions/      # Project-specific architectural decisions
  patterns/       # Project-specific reusable patterns
  debug/          # Project-specific debug insights
  conventions/    # Project-specific conventions (override or supplement system)
  changelog.md    # Log of all local knowledge changes
  README.md       # Summary of local knowledge
```

**Governance:** Apply changes directly. Low risk -- only affects this project.

### Tier 3: Session Memory (`~/.claude/projects/*/memory/`)

Ephemeral, per-session. Already exists in Claude Code. No changes needed.

## Scope Convention

Every vault note has a `scope:` frontmatter property:

- `scope: system` -- lives in the global vault, requires approval to change
- `scope: project` -- lives in `.vault/knowledge/`, can be changed directly
- **No `scope:` property** -- defaults to `scope: system` (conservative, protects existing notes)

## Proposal Workflow

When `/vault-evolve` identifies an improvement to a system-scoped note:

1. A **proposal note** is created in `_inbox/` with `type: proposal`, `status: pending`
2. The proposal includes: motivation, before/after diff, full snapshot for rollback
3. The user runs `/vault-triage` to review proposals
4. Each proposal can be: accepted (applied + moved to decisions/), rejected (kept as record in decisions/), or modified
5. Accepted proposals append to the target note's `## Changelog` section

## Promotion Workflow (project -> system)

When project-local knowledge proves universally useful, it can be promoted to the system vault:

1. `/vault-evolve` reviews project notes and identifies promotion candidates
2. Criteria: validated across sessions, applies broadly to the stack/domain, improves cross-project consistency
3. A **promotion proposal** is created in `_inbox/` with `type: proposal`, `promotion_from: project`
4. `/vault-triage` reviews the promotion -- user accepts or rejects
5. On acceptance: note is copied to the system vault's target folder; the project-local copy remains untouched

Promotions are rare. Most project knowledge should stay local. Only promote when there's clear cross-project value.

## Actionable Knowledge Tags

Retro insights and session learnings that need to be incorporated into upcoming work use the `actionable:` frontmatter property:

| Value          | Meaning                                                    |
|----------------|------------------------------------------------------------|
| `pending`      | Written by retro agent, not yet consumed by Sr PM          |
| `incorporated` | Sr PM has read and integrated into upcoming stories         |

### How it works

1. **Retro agent** writes insights to `.vault/knowledge/` (decisions/, patterns/, debug/, conventions/) with `actionable: pending` in frontmatter
2. **Sr PM agent** searches for `actionable: pending` notes, reads them, incorporates relevant feedback into upcoming stories
3. **Sr PM agent** updates the tag to `actionable: incorporated` after consuming the insight
4. `/vault-status` reports the count of pending actionable notes

This replaces the `.learnings/` directory pattern. Knowledge goes directly into the proper vault subfolder with appropriate categorization, making it searchable and linkable from day one.

## Emergency Edits

The scope guard blocks Edit and Write tool calls to protected vault directories. However, **vlt commands via Bash are allowed through** because vlt is the intended mechanism for the proposal workflow.

If a system note has a critical issue that must be fixed immediately (e.g., a typo breaking every session), you can use vlt directly:

```bash
vlt vault="Claude" read file="<Note Title>"          # verify the issue
vlt vault="Claude" patch file="<Note Title>" section="<Section>" content="<fixed content>"
```

This bypasses the proposal workflow. Use it sparingly -- the proposal workflow exists to ensure cross-project awareness. After an emergency edit, append to the note's `## Changelog`:

```bash
vlt vault="Claude" append file="<Note Title>" content="\n- <YYYY-MM-DD>: Emergency fix -- <what was changed and why> (bypassed proposal workflow)"
```

## Fallback: Core Vault Interaction Patterns

The Obsidian vault ("Claude") is your persistent knowledge layer. It survives across sessions, projects, and context compactions.

**Vault path:** `/Users/ramirosalas/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude`

### Vault Structure

```
methodology/  # Agent prompts, paivot methodology
conventions/  # Working conventions (testing, python, communication)
decisions/    # Architectural and design decisions with rationale
patterns/     # Reusable solutions and idioms
debug/        # Problems and their resolutions
concepts/     # Language, framework, and tool knowledge
projects/     # One index note per project
people/       # User preferences and team conventions
_inbox/       # Unsorted capture, triage into proper folders
_templates/   # Note templates
```

### When to Capture

- **Decisions**: chose X over Y, established a convention, made a trade-off
- **Debug insights**: solved a non-obvious bug, found a sharp edge
- **Patterns**: found a reusable solution, identified an anti-pattern
- **Session boundaries**: start (read), before compaction (save), end (update)

### How to Read

Preferred (via Bash):

    vlt vault="Claude" read file="<Note Title>"

Fallback (Read tool):

    Read: /Users/ramirosalas/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/<folder>/<Note Title>.md

### How to Search

Preferred (via Bash -- vault-aware, searches titles and content):

    vlt vault="Claude" search query="<term>"

Fallback (Grep tool):

    Grep: pattern="<term>" path="/Users/ramirosalas/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude"

### How to Create Notes

**First decide the scope:**

- Is this knowledge universal (applies to any project)? -> Global vault `_inbox/`, `scope: system`
- Is this knowledge project-specific (only relevant here)? -> `.vault/knowledge/`, `scope: project`

**Global vault (system scope):**

    vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="---\ntype: decision\nscope: system\nproject: <project>\nstatus: active\ncreated: <YYYY-MM-DD>\n---\n\n# <Title>\n\n<content>" silent

**Project vault (project scope):**

    mkdir -p .vault/knowledge/decisions
    Write: .vault/knowledge/decisions/<Title>.md

Fallback (Write tool):

    Write: /Users/ramirosalas/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/_inbox/<Title>.md

Every note needs frontmatter: type, scope, project, status, created.

### How to Append to Notes

Preferred (via Bash):

    vlt vault="Claude" append file="<Note Title>" content="<text to append>"

Fallback: Read the note with Read tool, then Write it back with additions, or use Edit tool.

### How to Move/Triage Notes

Preferred (via Bash -- updates all wikilinks across the vault):

    vlt vault="Claude" move path="_inbox/<Note>.md" to="decisions/<Note>.md"

Fallback (Bash mv -- wikilinks will NOT be updated):

    mv "<vault-path>/_inbox/<Note>.md" "<vault-path>/decisions/<Note>.md"

### How to Find Related Notes

    vlt vault="Claude" backlinks file="<Note Title>"
    vlt vault="Claude" links file="<Note Title>"
    vlt vault="Claude" tags counts

### Frontmatter Requirements

Every note MUST have: type, scope, project, status, created. Optional: stack, domain, confidence.

### The Rule

Knowledge not captured is knowledge rediscovered at cost. Capture as you go, not at the end.
