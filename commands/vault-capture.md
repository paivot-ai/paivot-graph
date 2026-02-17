---
description: Trigger a deliberate knowledge capture pass -- review the current session and save decisions, patterns, and debug insights to the Obsidian vault
allowed-tools: ["Bash", "Read", "Grep", "Glob"]
---

# Vault Capture

Perform a deliberate knowledge capture pass for the current session. This command reviews what has happened in the conversation and creates/updates vault notes.

## Steps

1. **Load the vault-knowledge skill first** to understand vault interaction patterns and note formatting.

2. **Review the current session** for capturable knowledge. Look for:
   - Architectural decisions (chose X over Y, established a convention)
   - Debugging breakthroughs (non-obvious bugs solved, sharp edges found)
   - Pattern discoveries (reusable solutions, anti-patterns identified)
   - Project state changes (features completed, integrations established)

3. **Detect the current project** from git remote or directory name.

4. **Check existing vault knowledge** for this project:
   ```bash
   obsidian vault="Claude" search query="<project-name>"
   ```
   Read the project note if it exists to avoid duplicating knowledge.

5. **For each piece of capturable knowledge**, create a vault note:

   - Use `_inbox/` as the initial path (triage later)
   - Include proper frontmatter (type, project, status, created date)
   - Add `[[wikilinks]]` to related notes
   - Keep notes atomic -- one idea per note

   ```bash
   obsidian vault="Claude" create name="<Note Title>" path="_inbox/<Note Title>.md" content="---
   type: <decision|pattern|debug>
   project: <project>
   status: active
   created: <YYYY-MM-DD>
   ---

   # <Note Title>

   <content>" silent
   ```

6. **Update the project index note** if it exists:
   ```bash
   obsidian vault="Claude" append file="<Project>" content="

   ## Session update (<date>)
   - <what was accomplished>
   - New notes: [[<Note 1>]], [[<Note 2>]]"
   ```

   If no project note exists, create one in `projects/`.

7. **Triage inbox notes** to their proper folders:
   ```bash
   obsidian vault="Claude" move path="_inbox/<Note>.md" to="decisions/<Note>.md"
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

## If Obsidian CLI is unavailable

If `obsidian` is not found:
1. Report that the CLI is not available
2. Suggest installation
3. Offer to output the notes as markdown that the user can manually add to their vault

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
