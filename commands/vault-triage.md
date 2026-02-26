---
description: Review and accept/reject pending proposals for system-scoped vault notes. This is the approval gate -- system knowledge changes only take effect after user review.
allowed-tools: ["Bash", "Read", "Glob", "Grep"]
---

# Vault Triage -- Review System Proposals

Review pending proposals created by `/vault-evolve` or `/vault-capture`. System-scoped vault notes (agent prompts, conventions, methodology) are never modified directly -- changes go through this approval gate.

**Vault:** `vlt vault="Claude"` (resolves path dynamically)

## Step 1: Find Pending Proposals

Search the vault for notes with `type: proposal` and `status: pending`:

```bash
vlt vault="Claude" search query="type: proposal"
```

Fallback: use Grep to find proposal notes (resolve vault path first with `vlt vault="Claude" dir`):
```
Grep: pattern="type: proposal" path="<vault-path>" glob="*.md"
```

For each match, read the note and check that `status: pending` is in the frontmatter. Skip any that are already `accepted` or `rejected`.

**Sort proposals by `created:` date (oldest first).** This ensures chronological processing.

If no pending proposals are found, report:
```
## Vault Triage

No pending proposals found. Nothing to review.

Proposals are created when /vault-evolve identifies improvements to system-scoped notes.
```

## Step 2: Present Each Proposal

For each pending proposal, read its full content and present to the user:

```
### Proposal: <Target Note>

**From project:** <originating project>
**Created:** <date>
**Age:** N days
**Target:** <full path of target note>

**Motivation:**
<what session experience motivated this change>

**Proposed Change:**

Before:
> <relevant section of current note>

After:
> <proposed replacement>

**Impact:** <what this affects>
```

Before presenting, perform two staleness checks:

**Age check:** Calculate the number of days since the `created:` date. If the proposal is older than 30 days, warn:

```
WARNING: This proposal is N days old (created <date>).
It may no longer be relevant. Consider whether the motivation still applies.
```

**Content check:** Read the current target note and compare it to the snapshot in the proposal. If the target has been modified since the proposal was created (content differs from snapshot), warn:

```
WARNING: The target note has been modified since this proposal was created.
The snapshot may be out of date. Review carefully before accepting.
```

## Step 3: User Decision

For each proposal, ask the user to decide:

1. **Accept** -- apply the proposed change to the target note
2. **Reject** -- decline the change with a reason
3. **Modify** -- the user wants to adjust the proposal before applying

Process proposals in chronological order (oldest first). If multiple proposals target the same note, re-read the target between each application.

## Step 4: Apply Decisions

### Accept

1. Read the current content of the target note.
2. Apply the proposed change via vlt:
   - For full body replacement:
     ```bash
     vlt vault="Claude" write file="<target>" content="<new content>"
     ```
   - For section edits:
     ```bash
     vlt vault="Claude" patch file="<target>" heading="<heading>" content="<new section content>"
     ```
3. Append to the target note's `## Changelog` section:
   ```bash
   vlt vault="Claude" append file="<target>" content="
   - <YYYY-MM-DD>: <description of change> (from project: <originating-project>)"
   ```
   If no `## Changelog` section exists, create one at the bottom of the note.
4. Update the proposal note's frontmatter:
   ```bash
   vlt vault="Claude" property:set file="Proposal -- <Target>" name="status" value="accepted"
   vlt vault="Claude" property:set file="Proposal -- <Target>" name="accepted" value="<YYYY-MM-DD>"
   ```
5. Move the proposal to `decisions/`:
   ```bash
   vlt vault="Claude" move path="_inbox/Proposal -- <Target>.md" to="decisions/Proposal -- <Target>.md"
   ```

### Reject

1. Update the proposal note's frontmatter:
   ```bash
   vlt vault="Claude" property:set file="Proposal -- <Target>" name="status" value="rejected"
   vlt vault="Claude" property:set file="Proposal -- <Target>" name="rejected" value="<YYYY-MM-DD>"
   vlt vault="Claude" property:set file="Proposal -- <Target>" name="rejection_reason" value="<user's reason>"
   ```
2. Move the proposal to `decisions/`:
   ```bash
   vlt vault="Claude" move path="_inbox/Proposal -- <Target>.md" to="decisions/Proposal -- <Target>.md"
   ```
   Rejected proposals are kept as decision records -- they document what was considered and why it was declined.

### Modify

1. Present the proposal content to the user for editing.
2. Apply the user's modified version to the target note via vlt write/patch.
3. Update the proposal with the final applied version.
4. Proceed as with Accept (changelog, status update, move).

## Step 5: Report

```
## Vault Triage Summary

Date: <today>

### Accepted
- Proposal for <Note A>: <what was changed>
- Proposal for <Note B>: <what was changed>

### Rejected
- Proposal for <Note C>: <rejection reason>

### Modified and Accepted
- Proposal for <Note D>: <what was changed after modification>

### Remaining
- N proposals still pending (if user chose to skip any)

Total: A accepted, R rejected, M modified, S skipped
```

## Constraints

- Never apply a system-scoped change without explicit user approval
- Always preserve the rollback snapshot in the proposal note (even after acceptance)
- Process proposals in chronological order
- Re-read the target note between proposals that affect the same note
- Do not delete proposal notes -- they are decision records regardless of outcome
