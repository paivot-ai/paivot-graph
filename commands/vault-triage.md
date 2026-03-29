---
description: Triage notes in _inbox to their proper folders and check vault health
allowed-tools: ["Bash", "Read", "Glob", "Grep"]
---

# Vault Triage

Review notes in `_inbox/` and move them to the correct folders. Also check for orphan notes and broken wikilinks.

**Vault:** `vlt vault="Claude"` (resolves path dynamically)

## Step 1: Check Inbox

List all notes in `_inbox/`:

```bash
vlt vault="Claude" files folder="_inbox" --json
```

If empty, report:
```
## Vault Triage

_inbox/ is empty. Nothing to triage.
```

## Step 2: For Each Inbox Note

Read each note and determine:

1. **Type** (from frontmatter): decision, pattern, debug, concept, convention, methodology
2. **Target folder** based on type:
   - decision -> decisions/
   - pattern -> patterns/
   - debug -> debug/
   - concept -> concepts/
   - convention -> conventions/
   - methodology -> methodology/
3. **Domain validation**: Check domain is from controlled vocabulary
4. **Missing links**: Check if "Related" section exists with at least one wikilink

### Present to User

```
### Inbox Note: [[Note Title]]

Type: <type>
Domain: <domain> (valid/invalid)
Target: <folder>/
Links: <has Related section?> <count> links

Summary: <first paragraph>

Issues:
- <domain not in vocabulary>
- <no Related section>
- <no Tags section>
```

### User Decision

1. **Move** -> triage to correct folder
2. **Edit** -> fix issues before moving
3. **Delete** -> remove if not vault-worthy
4. **Skip** -> leave for later

## Step 3: Apply Triage

### Move Note

```bash
vlt vault="Claude" move path="_inbox/<Note>.md" to="<folder>/<Note>.md"
```

### Fix Missing Links

If note lacks Related section, search for related notes:

```bash
vlt vault="Claude" search query="<keywords>" --json
```

Suggest top 3 matches, then add:

```bash
vlt vault="Claude" append file="<Note>" content="

## Related

- [[<Suggested Link 1>]]
- [[<Suggested Link 2>]]"
```

### Fix Missing Tags

Derive tags from domain and add to body:

```bash
vlt vault="Claude" append file="<Note>" content="

## Tags

#<derived-tag>"
```

### Delete Note

```bash
vlt vault="Claude" delete file="<Note>" permanent
```

## Step 4: Check Vault Health

After triaging inbox, check overall vault health:

```bash
# Orphans (notes with no incoming links)
orphans=$(vlt vault="Claude" orphans --json)

# Broken wikilinks
broken=$(vlt vault="Claude" unresolved --json)
```

Report:
```
## Vault Health

| Metric | Count | Status |
|--------|-------|--------|
| _inbox | N | (should be 0) |
| Orphans | N | (warn if > 10) |
| Broken links | N | (should be 0) |
```

### Fix Orphans

For each orphan, suggest links:

```bash
vlt vault="Claude" search query="<keywords from note>" --json
```

Add suggested link to the related note (creating an incoming link):

```bash
vlt vault="Claude" append file="<Related Note>" content="
- [[<Orphan Note>]]"
```

### Fix Broken Links

For each broken wikilink:

1. Check if target exists with different name (alias)
2. If yes, add alias to existing note
3. If no, create stub note or remove the link

## Step 5: Report Summary

```
## Vault Triage Summary

Date: <today>

### Moved
- [[Note A]] -> decisions/
- [[Note B]] -> patterns/

### Fixed
- Added Related section to [[Note C]]
- Added tags to [[Note D]]
- Linked orphan [[Note E]] to [[Project X]]

### Deleted
- [[Note F]] (not vault-worthy)

### Remaining Issues
- N orphans still need linking
- N broken wikilinks need resolution

### Vault Health
- Total notes: N
- _inbox: 0
- Orphans: N
- Broken links: 0
```

## Controlled Domain Vocabulary

Valid domains:
- ai-training, ai-inference, ai-agents, ai-nlp
- developer-tools (legacy), dev-tools-cli, dev-tools-testing, dev-tools-workflow, dev-tools-knowledge
- platform-engineering (Kubernetes, Flux, Helm, Talos, Proxmox, Velero, CI/CD, IaC)
- product-management
- security-gateway, security-hardening, security-compliance
- finance-quant, finance-fintech
- frontend-ui, frontend-performance
- calendar-sync

## Constraints

- Every note must have at least one wikilink (no orphans)
- Every note must have domain from controlled vocabulary
- Every note should have Tags section at bottom (not frontmatter)
- _inbox/ should be empty after triage
