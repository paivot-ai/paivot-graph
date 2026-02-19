---
description: Trigger a deliberate knowledge capture pass -- review the current session and save decisions, patterns, and debug insights to the Obsidian vault
allowed-tools: ["Bash", "Read", "Write", "Edit", "Grep", "Glob"]
---

# Vault Capture

Perform a deliberate knowledge capture pass for the current session. This command reviews what has happened in the conversation and creates/updates vault notes.

**Vault path:** `/Users/ramirosalas/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude`

## Steps

1. **Load the vault-knowledge skill first** to understand vault interaction patterns and note formatting.

2. **Review the current session** for capturable knowledge. Look for:
   - Architectural decisions (chose X over Y, established a convention)
   - Debugging breakthroughs (non-obvious bugs solved, sharp edges found)
   - Pattern discoveries (reusable solutions, anti-patterns identified)
   - Project state changes (features completed, integrations established)

3. **Detect the current project** from git remote or directory name.

4. **Check existing vault knowledge** for this project:

   Search the vault (prefer vlt):
   ```bash
   vlt vault="Claude" search query="<project-name>"
   ```
   Read the project note if it exists to avoid duplicating knowledge:
   ```bash
   vlt vault="Claude" read file="<project-name>"
   ```
   Fallback if vlt unavailable: use Grep/Read tools directly on vault path.

5. **For each piece of capturable knowledge**, create a vault note:

   - Use `_inbox/` as the initial path (triage later)
   - Include proper frontmatter (type, project, status, created date)
   - Add `[[wikilinks]]` to related notes
   - Keep notes atomic -- one idea per note

   Preferred (via Bash):
   ```bash
   vlt vault="Claude" create name="<Note Title>" path="_inbox/<Note Title>.md" content="---
   type: <decision|pattern|debug>
   project: <project>
   status: active
   created: <YYYY-MM-DD>
   ---

   # <Note Title>

   <content>" silent
   ```

   Fallback: use Write tool to create the file directly at the vault path.

6. **Update the project index note** if it exists:

   Preferred (via Bash):
   ```bash
   vlt vault="Claude" append file="<Project>" content="

   ## Session update (<date>)
   - <what was accomplished>
   - New notes: [[<Note 1>]], [[<Note 2>]]"
   ```

   If no project note exists, create one:
   ```bash
   vlt vault="Claude" create name="<Project>" path="projects/<Project>.md" content="..." silent
   ```

   Fallback: use Edit tool to append, or Write tool to create.

7. **Triage inbox notes** to their proper folders (vlt updates wikilinks automatically):
   ```bash
   vlt vault="Claude" move path="_inbox/<Note>.md" to="decisions/<Note>.md"
   ```

8. **Report what was captured** in a summary:

   ```
   ## Vault Capture Summary

   Project: <name>
   Date: <today>

   ### Captured
   - [decision] <Note Title> -> decisions/
   - [pattern] <Note Title> -> patterns/
   - [debug] <Note Title> -> debug/

   ### Updated
   - projects/<Project>.md (session update)

   ### Skipped (already exists)
   - <Note that was already in vault>

   Total: N new notes, M updated notes
   ```

## If vault directory is missing

If the vault path does not exist:
1. Report that the vault directory was not found
2. Offer to output the notes as markdown that the user can manually add to their vault

## If nothing to capture

If the session has no significant decisions, patterns, or debug insights:
```
## Vault Capture Summary

Project: <name>
No new knowledge to capture from this session.

Tip: Knowledge capture is most valuable after:
- Making architectural decisions
- Solving non-obvious bugs
- Discovering reusable patterns
- Completing significant features
```
