---
description: Show Obsidian vault health -- note counts by folder, recent notes, and overall state
allowed-tools: ["Bash"]
---

# Vault Status

Show the current state and health of the Obsidian knowledge vault.

## Steps

1. **Check if Obsidian CLI is available**:
   ```bash
   command -v obsidian
   ```
   If not available, report and suggest installation.

2. **Gather vault statistics** by running these commands:

   Count notes by folder:
   ```bash
   for folder in methodology conventions decisions patterns debug concepts projects people _inbox; do
     count=$(obsidian vault="Claude" search query="path:$folder/" 2>/dev/null | grep -c "." || echo "0")
     echo "$folder: $count"
   done
   ```

   List recently modified notes (search broadly):
   ```bash
   obsidian vault="Claude" search query="status:active" 2>/dev/null || echo "No results"
   ```

3. **Search for potential issues**:

   Notes still in inbox (need triage):
   ```bash
   obsidian vault="Claude" search query="path:_inbox/" 2>/dev/null || echo "None"
   ```

   Notes with missing frontmatter (search for notes without type):
   ```bash
   obsidian vault="Claude" search query="type:" 2>/dev/null || echo "Unable to check"
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

## If Obsidian CLI is unavailable

```
## Vault Status

Obsidian CLI is not available. Cannot query vault.

To install: https://github.com/Acylation/obsidian-cli
Ensure Obsidian is running for the CLI to connect.
```
