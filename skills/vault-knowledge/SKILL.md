---
name: vault-knowledge
description: This skill should be used when working on any project to understand how to effectively interact with the Obsidian knowledge vault. It teaches when to capture knowledge, what to capture, how to format vault notes, and how to search effectively. Use when you need to "save to vault", "update vault", "capture a decision", "record a pattern", "log a debug insight", or when starting/ending a significant work session.
version: 0.1.0
---

# Vault Knowledge -- Effective Obsidian Vault Interaction

## Overview

The Obsidian vault ("Claude") is your persistent knowledge layer. It survives across sessions, across projects, and across context compactions. Every interaction with a project is an opportunity to compound knowledge.

The vault CLI: `obsidian vault="Claude" <command>`

## When to Capture Knowledge

Capture immediately after any of these events:

### Architectural Decisions
- Chose one approach over another (e.g., REST vs GraphQL, monorepo vs polyrepo)
- Established a convention (naming, file structure, error handling)
- Made a trade-off (performance vs readability, simplicity vs flexibility)

### Debugging Breakthroughs
- Solved a non-obvious bug (especially if it took multiple attempts)
- Found a sharp edge in a library or framework
- Discovered an environment-specific issue

### Pattern Discoveries
- Found a reusable solution that could apply to other projects
- Identified an anti-pattern worth avoiding
- Developed a workflow that improved productivity

### Significant Feature Completion
- Completed a major feature or milestone
- Established a new integration
- Changed the project architecture

### Session Boundaries
- At the start of a session: consult the vault for project context
- Before context compaction: save everything learned (this is your last chance)
- At the end of significant work: distill and capture

## What to Capture (by Note Type)

### Decisions (`decisions/`)

Record the decision, the reasoning, and what was NOT chosen:

```
obsidian vault="Claude" create name="Use SQLite for local state" path="_inbox/Use SQLite for local state.md" content="---
type: decision
project: my-project
status: active
confidence: high
created: 2025-01-15
---

# Use SQLite for local state

## Decision
Use SQLite via sqlx for persisting local application state.

## Rationale
- Single-file database, no server needed
- Excellent concurrent read performance
- Built-in WAL mode for write performance
- sqlx provides compile-time query verification

## Alternatives considered
- JSON files: too fragile for concurrent access
- PostgreSQL: overkill for local-only state
- In-memory only: loses state on restart

## Consequences
- Must include sqlite3 as dependency
- Migration story needed for schema changes
- [[sqlx compile-time checks]] require DATABASE_URL at build time" silent
```

### Patterns (`patterns/`)

Record when to use it, how to implement it, and where it has been applied:

```
obsidian vault="Claude" create name="Graceful degradation in CLI hooks" path="_inbox/Graceful degradation in CLI hooks.md" content="---
type: pattern
project: paivot-graph
stack: [bash, claude-code]
status: active
created: 2025-01-15
---

# Graceful degradation in CLI hooks

## When to use
Any CLI hook that depends on an external tool (obsidian, git, etc.).

## Implementation
1. Check if the dependency exists: command -v <tool>
2. If missing, output a soft warning (not an error)
3. Always exit 0 -- never block the parent process
4. Provide install/setup instructions in the warning

## Applied in
- [[paivot-graph]] SessionStart hook
- [[paivot-graph]] PreCompact hook" silent
```

### Debug Notes (`debug/`)

Record symptoms, root cause, and the fix:

```
obsidian vault="Claude" create name="obsidian CLI hangs on large search" path="_inbox/obsidian CLI hangs on large search.md" content="---
type: debug
project: paivot-graph
stack: [bash, obsidian]
status: active
created: 2025-01-15
---

# obsidian CLI hangs on large search

## Symptoms
obsidian search with broad query hangs for 10+ seconds, sometimes times out.

## Root cause
Obsidian must be running for the CLI to work. When Obsidian is closed,
the CLI blocks waiting for a connection.

## Fix
Add a timeout wrapper and check for Obsidian process before calling CLI.
timeout 5 obsidian vault=Claude search query=term || echo 'Obsidian not responding'" silent
```

### Project Notes (`projects/`)

One index note per project, linking to all related knowledge:

```
obsidian vault="Claude" create name="paivot-graph" path="projects/paivot-graph.md" content="---
type: project
project: paivot-graph
stack: [bash, claude-code, obsidian]
domain: developer-tools
status: active
created: 2025-01-15
---

# paivot-graph

Claude Code plugin for Obsidian vault knowledge integration.

## Architecture
- SessionStart hook for automatic vault consultation
- PreCompact hook for knowledge capture reminders
- vault-knowledge skill for teaching interaction patterns
- /vault-capture and /vault-status commands

## Related
- [[Graceful degradation in CLI hooks]]
- [[Use obsidian CLI for vault access]]
- [[Paivot cognitive architecture]]" silent
```

## How to Search the Vault

### Finding existing knowledge
```bash
obsidian vault="Claude" search query="<term>"
```
Use specific terms: project names, technology names, error messages.

### Reading a specific note
```bash
obsidian vault="Claude" read file="<Note Title>"
```

### Updating an existing note
```bash
obsidian vault="Claude" append file="<Note Title>" content="

## New section
Additional content here"
```

### Moving notes from inbox
After creating notes in `_inbox/`, triage them to proper folders:
```bash
obsidian vault="Claude" move path="_inbox/My Note.md" to="decisions/My Note.md"
```

## Frontmatter Requirements

Every note MUST have frontmatter with these properties:

```yaml
type: methodology | convention | decision | pattern | debug | concept | project | person
project: <project-name>
stack: [<languages-and-frameworks>]    # optional but recommended
domain: <business-domain>              # optional
status: active | superseded | archived
confidence: high | medium | low        # for decisions
created: <YYYY-MM-DD>
```

## Cross-linking

Use `[[wikilinks]]` to connect related notes:
- Decisions that led to patterns: `"This pattern emerged from [[Decision Name]]"`
- Debug notes that informed decisions: `"See [[Bug Title]] for why we chose this approach"`
- Project notes that reference all related knowledge

## The Knowledge Compounding Principle

Every project should leave the vault richer than it found it. This means:

1. **Before starting**: Read what exists. Do not rediscover what is already known.
2. **While working**: Capture as you go. Do not wait for the end.
3. **Before compaction**: Save everything. This is the last chance before memory loss.
4. **After completing**: Update the project index note with what was accomplished.

Knowledge that is not captured is knowledge that will be rediscovered (at cost). Knowledge that is captured compounds -- it makes every future session faster and more informed.

## Integration with Paivot Methodology

When running Paivot execution cycles:
- Retro learnings flow into `methodology/` notes, updating the methodology itself
- Sprint decisions go into `decisions/` with sprint context
- Patterns discovered during execution go into `patterns/`
- The methodology evolves through practice, not just theory
