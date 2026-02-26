---
description: Trigger a deliberate knowledge capture pass -- review the current session and save decisions, patterns, and debug insights to the appropriate vault tier (global or project-local)
allowed-tools: ["Bash", "Read", "Grep", "Glob"]
---

# Vault Capture

Perform a deliberate knowledge capture pass for the current session. This command reviews what has happened in the conversation and creates/updates vault notes, routing each piece of knowledge to the correct tier.

**Global vault:** `vlt vault="Claude"` (resolves path dynamically)
**Project vault path:** `.vault/knowledge/` (relative to project root)

## Steps

1. **Load the vault-knowledge skill first** to understand vault interaction patterns and note formatting.

2. **Review the current session** for capturable knowledge. Look for:
   - Architectural decisions (chose X over Y, established a convention)
   - Debugging breakthroughs (non-obvious bugs solved, sharp edges found)
   - Pattern discoveries (reusable solutions, anti-patterns identified)
   - Project state changes (features completed, integrations established)

3. **Detect the current project** from git remote or directory name.

4. **Check existing vault knowledge** for this project:

   Read the project note and all linked knowledge in one call:
   ```bash
   vlt vault="Claude" read file="<project-name>" follow
   ```
   This returns the project note plus every note it links to (decisions, patterns, debug insights). Use it to avoid duplicating knowledge that already exists.

5. **For each piece of capturable knowledge**, decide which tier it belongs to:

   ### Universal knowledge (applies to ANY project using this stack/methodology)
   Examples: methodology refinements, cross-project patterns, tool insights, convention updates.

   Route to global vault `_inbox/` with `scope: system`:
   ```bash
   vlt vault="Claude" create name="<Note Title>" path="_inbox/<Note Title>.md" content="---
   type: <decision|pattern|debug>
   scope: system
   project: <project>
   status: active
   created: <YYYY-MM-DD>
   ---

   # <Note Title>

   <content>" silent
   ```

   ### Project-specific knowledge (only relevant to THIS project)
   Examples: project architecture decisions, project-specific patterns, local debug insights, project conventions.

   Route to `.vault/knowledge/` with `scope: project`:
   ```bash
   vlt vault=".vault/knowledge" create name="<Note Title>" path="<subfolder>/<Note Title>.md" content="---
   type: <decision|pattern|debug|convention>
   scope: project
   project: <project>
   status: active
   created: <YYYY-MM-DD>
   ---

   # <Note Title>

   <content>" silent
   ```

   Subfolder mapping: decisions/ for decisions, patterns/ for patterns, debug/ for debug insights, conventions/ for conventions.

   **If `.vault/` does not exist** (nd not initialized): fall back to the global vault with a `scope: project` tag so it can be moved later. Tell the user: "No .vault/ directory found. Saved to global vault with scope: project. Initialize nd to enable project-local storage."

6. **Update the project index note** if it exists:

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

   Also create/update `.vault/knowledge/README.md` if project-local notes were created:
   ```bash
   vlt vault=".vault/knowledge" write file="README" content="# Project Knowledge

   Local knowledge for <project>. See also the global vault for cross-project knowledge.

   ## Contents
   - decisions/: N notes
   - patterns/: N notes
   - debug/: N notes
   - conventions/: N notes

   Last updated: <YYYY-MM-DD>"
   ```

7. **Triage inbox notes** to their proper folders (vlt updates wikilinks automatically):
   ```bash
   vlt vault="Claude" move path="_inbox/<Note>.md" to="decisions/<Note>.md"
   ```

8. **Report what was captured** in a summary:

   ```
   ## Vault Capture Summary

   Project: <name>
   Date: <today>

   ### Captured to Global Vault
   - [decision] <Note Title> -> decisions/
   - [pattern] <Note Title> -> patterns/

   ### Captured to Project Vault (.vault/knowledge/)
   - [decision] <Note Title> -> .vault/knowledge/decisions/
   - [debug] <Note Title> -> .vault/knowledge/debug/

   ### Updated
   - projects/<Project>.md (session update)
   - .vault/knowledge/README.md

   ### Skipped (already exists)
   - <Note that was already in vault>

   Total: N new notes (G global, P project-local), M updated notes
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
