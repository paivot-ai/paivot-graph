---
name: vault-knowledge
description: This skill should be used when working on any project to understand how to effectively interact with the Obsidian knowledge vault. It teaches when to capture knowledge, what to capture, how to format vault notes, and how to search effectively. Use when you need to "save to vault", "update vault", "capture a decision", "record a pattern", "log a debug insight", or when starting/ending a significant work session.
version: 0.3.0
---

# Vault Knowledge (Vault-Backed)

The Obsidian vault ("Claude") lives on disk. Interact with it using `vlt` (the fast vault CLI) via Bash, or directly using Read, Write, Grep, and Glob tools. Prefer `vlt` for vault-aware operations (search, create, move with wikilink repair, backlinks, tags). Use Read/Write/Grep/Glob when you need Claude Code tool integration or vlt is unavailable.

**Vault path:** `/Users/ramirosalas/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude`

Read the full skill content from the vault:

    /Users/ramirosalas/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/conventions/Vault Knowledge Skill.md

The vault version is authoritative. Follow it completely.

If the vault is unavailable, use these minimal instructions:

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

Preferred (via Bash):

    vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="---\ntype: decision\nproject: <project>\nstatus: active\ncreated: <YYYY-MM-DD>\n---\n\n# <Title>\n\n<content>" silent

Fallback (Write tool):

    Write: /Users/ramirosalas/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude/_inbox/<Title>.md

Every note needs frontmatter: type, project, status, created.

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

Every note MUST have: type, project, status, created. Optional: stack, domain, confidence.

### The Rule

Knowledge not captured is knowledge rediscovered at cost. Capture as you go, not at the end.
