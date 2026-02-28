---
description: Cancel active execution loop
allowed-tools: ["Bash", "Read", "Grep", "Skill"]
---

# piv-cancel-loop -- Cancel Execution Loop

Cancel the active piv-loop execution loop and report backlog state.

## Steps

1. Cancel the loop:
   ```bash
   pvg loop cancel
   ```

2. Report current backlog state:
   ```bash
   nd ready --json | jq length
   nd list --status in_progress --json | jq length
   nd blocked --json | jq length
   ```

3. Summarize:
   - How many iterations completed
   - Stories still ready / in-progress / blocked
   - Suggest next action (resume later with `/piv-loop`, or manual triage)
