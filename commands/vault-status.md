---
description: Show Obsidian vault health -- note counts by folder, recent notes, and overall state
allowed-tools: ["Bash", "Read", "Glob", "Grep"]
---

# Vault Status

Show the current state and health of the Obsidian knowledge vault.

**Vault path:** `/Users/ramirosalas/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude`

## Steps

1. **Check vault accessibility** (prefer vlt):
   ```bash
   vlt vault="Claude" files total
   ```
   If vlt is unavailable, check the directory directly:
   ```bash
   test -d "$HOME/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude"
   ```
   If neither works, report and exit.

2. **Gather vault statistics** by counting files per folder:

   Preferred (via Bash -- fast counts):
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

   Fallback: use Glob to count notes in each folder.

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

4. **Present the report**:

   ```
   ## Vault Status

   ### Note Inventory
   | Folder        | Count | Purpose                              |
   |---------------|-------|--------------------------------------|
   | methodology/  | N     | Paivot methodology (atomic concepts) |
   | conventions/  | N     | Working conventions                  |
   | decisions/    | N     | Architectural decisions              |
   | patterns/     | N     | Reusable solutions                   |
   | debug/        | N     | Problems and resolutions             |
   | concepts/     | N     | Language/framework knowledge         |
   | projects/     | N     | Project index notes                  |
   | people/       | N     | Team preferences                     |
   | _inbox/       | N     | Unsorted (needs triage)              |
   | **Total**     | **N** |                                      |

   ### Health
   - Inbox items: N (triage needed if > 0)
   - Active projects: <list>
   - Most recent notes: <list of last 5>

   ### Recommendations
   - <any actionable suggestions based on the data>
   ```

5. **Provide actionable recommendations** based on what was found:
   - If inbox has items: "N notes in _inbox/ need triage -- move them to proper folders"
   - If a folder is empty: "No <type> notes yet -- consider capturing <type> knowledge"
   - If vault is healthy: "Vault is well-organized. Keep capturing knowledge as you work."

## If vault directory is missing

```
## Vault Status

Vault directory not found at expected path.
Expected: ~/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude

Ensure Obsidian is installed and the "Claude" vault exists.
```
