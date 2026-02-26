---
description: Show Obsidian vault health -- note counts by folder, recent notes, project vault status, and pending proposals
allowed-tools: ["Bash", "Read", "Glob", "Grep"]
---

# Vault Status

Show the current state and health of both the global Obsidian vault and the project-local vault.

**Global vault:** `vlt vault="Claude"` (resolves path dynamically)
**Project vault path:** `.vault/knowledge/` (relative to project root)

## Steps

1. **Check vault accessibility**:
   ```bash
   vlt vault="Claude" files total
   ```
   If vlt is unavailable, check the directory directly:
   ```bash
   test -d "$HOME/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude"
   ```
   If neither works, report and exit.

2. **Gather global vault statistics** by counting files per folder:

   ```bash
   vlt vault="Claude" files folder="methodology" total
   vlt vault="Claude" files folder="conventions" total
   vlt vault="Claude" files folder="decisions" total
   vlt vault="Claude" files folder="patterns" total
   vlt vault="Claude" files folder="debug" total
   vlt vault="Claude" files folder="concepts" total
   vlt vault="Claude" files folder="projects" total
   vlt vault="Claude" files folder="people" total
   vlt vault="Claude" files folder="_inbox" total
   ```

   Also check vault health:
   ```bash
   vlt vault="Claude" orphans
   vlt vault="Claude" unresolved
   vlt vault="Claude" tags counts
   ```

3. **Search for potential issues**:

   Notes still in inbox (need triage):
   ```bash
   vlt vault="Claude" files folder="_inbox"
   ```

   Orphaned notes (no incoming links):
   ```bash
   vlt vault="Claude" orphans
   ```

   Broken wikilinks:
   ```bash
   vlt vault="Claude" unresolved
   ```

4. **Check project vault status**:

   Check if `.vault/knowledge/` exists in the current project:
   ```bash
   test -d .vault/knowledge && echo "exists" || echo "not initialized"
   ```

   If it exists, count notes per subfolder:
   ```bash
   vlt vault=".vault/knowledge" files folder="decisions" total
   vlt vault=".vault/knowledge" files folder="patterns" total
   vlt vault=".vault/knowledge" files folder="debug" total
   vlt vault=".vault/knowledge" files folder="conventions" total
   ```

   List recent project notes:
   ```bash
   vlt vault=".vault/knowledge" files
   ```

5. **Check for actionable knowledge** (retro insights awaiting incorporation):

   Search the project vault for notes with `actionable: pending`:
   ```bash
   vlt vault=".vault/knowledge" search query="actionable: pending"
   ```

   For each found, read the note and its linked context:
   ```bash
   vlt vault=".vault/knowledge" read file="<note>" follow
   ```

6. **Check for pending proposals**:

   Search the global vault inbox for pending proposals:
   ```bash
   vlt vault="Claude" search query="type: proposal"
   ```

   Or fallback with Grep (resolve vault path first with `vlt vault="Claude" dir`):
   ```
   Grep: pattern="type: proposal" path="<vault-path>/_inbox" glob="*.md"
   ```

   For each found, check if `status: pending`:
   ```
   Grep: pattern="status: pending" in each proposal file
   ```

7. **Present the report**:

   ```
   ## Vault Status

   ### Global Vault (system scope)
   | Folder        | Count | Purpose                              |
   |---------------|-------|--------------------------------------|
   | methodology/  | N     | Paivot methodology (agent prompts)   |
   | conventions/  | N     | Working conventions                  |
   | decisions/    | N     | Architectural decisions              |
   | patterns/     | N     | Reusable solutions                   |
   | debug/        | N     | Problems and resolutions             |
   | concepts/     | N     | Language/framework knowledge         |
   | projects/     | N     | Project index notes                  |
   | people/       | N     | Team preferences                     |
   | _inbox/       | N     | Unsorted (needs triage)              |
   | **Total**     | **N** |                                      |

   ### Project Vault (.vault/knowledge/)
   Status: <exists | not initialized>

   | Subfolder     | Count |
   |---------------|-------|
   | decisions/    | N     |
   | patterns/     | N     |
   | debug/        | N     |
   | conventions/  | N     |
   | **Total**     | **N** |

   (If not initialized: "No project vault. Run /vault-capture to create one.")

   ### Actionable Knowledge (retro insights)
   N notes with actionable: pending (awaiting Sr PM incorporation):
   - <type>: <title> (created <date>)
   - <type>: <title> (created <date>)

   (If none: "No pending retro insights.")

   ### Pending Proposals
   N proposals awaiting review:
   - Proposal for <Target Note A> (from project <X>, created <date>)
   - Proposal for <Target Note B> (from project <Y>, created <date>)

   Run /vault-triage to review and accept/reject.

   (If none: "No pending proposals.")

   ### Health
   - Inbox items: N (triage needed if > 0)
   - Active projects: <list>
   - Most recent notes: <list of last 5>

   ### Recommendations
   - <any actionable suggestions based on the data>
   ```

8. **Provide actionable recommendations** based on what was found:
   - If inbox has items: "N notes in _inbox/ need triage -- move them to proper folders"
   - If pending proposals exist: "N proposals pending -- run /vault-triage to review"
   - If actionable knowledge pending: "N retro insights pending -- Sr PM should incorporate into upcoming stories"
   - If a folder is empty: "No <type> notes yet -- consider capturing <type> knowledge"
   - If project vault not initialized: "Consider initializing .vault/knowledge/ for project-local knowledge"
   - If vault is healthy: "Vault is well-organized. Keep capturing knowledge as you work."

## If vault directory is missing

```
## Vault Status

Vault directory not found at expected path.
Expected: ~/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude

Ensure Obsidian is installed and the "Claude" vault exists.
```
