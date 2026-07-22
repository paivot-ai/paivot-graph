# Bug Creation Authority: Old Distributed Model vs Current Centralized Model

## Historical Model: Distributed Bug Creation (ns-paivot)

In the old system, **any PM (PM-Dispatcher or PM-Acceptor) could create bugs during story review**.

### Old Workflow

During Phase 4.5 (Discovered Issues Extraction) of the 5-phase review:

```bash
# PM reviews delivered story
# Finds issue during code review

# PM creates bug directly
bd create "OAuth library fails on redirect_uri trailing slash" \
  -t bug \
  -p 2 \                              # Priority: P2 (not P0)
  -d "Discovered during <story-id>: ..." \
  --json

# Link to epic
bd dep add <bug-id> <epic-id> --type parent-child

# Link to discovery story
bd dep add <bug-id> <story-id> --type discovered-from
```

### Key Characteristics

Characteristics of the old approach:

- **Authority**: PM (regular PM, not Sr PM)
- **Timing**: During story review (Phase 4.5)
- **Priority**: P2 (secondary)
- **Linking**: Traced to discovery story via `discovered-from` relationship
- **Mandatory**: REGARDLESS of accept/reject decision
  - If story accepted: bug still filed
  - If story rejected: bug still filed (separate concern)
- **Traceability**: Clear origin story
- **Approval**: PM judgment only (no Sr PM gatekeeper)

### Example from pm_acceptor.yaml

```
Developer notes: "LEARNINGS: Found a bug in the OAuth library - it fails silently
if redirect_uri has trailing slash."

I file this as:
bd create "OAuth library fails silently on redirect_uri trailing slash" \
  -t bug \
  -p 2 \
  -d "Discovered during PV-wb7: OAuth library returns 200 but doesn't redirect
if redirect_uri ends with /. Silent failure makes debugging difficult." \
  --json
```

## Current Model: Centralized Bug Creation (paivot-graph)

In the current system, **only Sr PM can create bugs**. PM-Acceptor cannot.

### Current Workflow

During story review, PM-Acceptor outputs a structured block instead:

```
DISCOVERED_BUG:
  title: OAuth library fails silently on redirect_uri trailing slash
  context: Discovered during story review - redirect_uri with trailing slash
           returns 200 but doesn't redirect. Silent failure.
  affected_files: src/oauth/handlers.py
  discovered_during: story-id
```

The dispatcher routes this to Sr PM for triage:

```bash
# Sr PM creates fully structured bug
nd create "OAuth library fails silently on redirect_uri trailing slash" \
  --type=bug \
  --priority=P0 \                     # Priority: P0 (critical) -- nd accepts P0-P4 natively
  --parent=<epic-id> \
  -d "## Context
...fully detailed context...

## Root Cause (if known)
...analysis...

## Affected Components
...files, modules, services...

## Acceptance Criteria
- [ ] Root cause identified
- [ ] Fix criterion
- [ ] Verification criterion

## Testing Requirements
- Integration tests proving the fix works under real conditions"
```

### Key Characteristics

- **Authority**: Sr PM ONLY (gatekeeper role)
- **Timing**: After PM-Acceptor discovery (async, structured)
- **Priority**: P0 (critical - all bugs are P0)
- **Linking**: Determined by Sr PM during triage
- **Structured**: Full AC, epic placement, dependency chain
- **Gatekeeping**: Sr PM decides: is this a bug? scope? epic placement? dependencies?
- **Approval**: Sr PM expertise + potentially user clarification

## Comparison

| Aspect | Old (Distributed) | Current (Centralized) |
|--------|-------------------|----------------------|
| **Authority** | PM (during review) | Sr PM only (bug triage mode) |
| **Who can create** | PM-Acceptor, PM-Dispatcher | Sr PM (exclusive) |
| **Priority** | P2 | P0 |
| **Timing** | Immediately (inline) | After review (async) |
| **AC Quality** | Simple discovery notes | Full structured AC |
| **Epic Placement** | Automatic (same epic) | Sr PM judgment |
| **Dependencies** | Manual linking | Sr PM establishes chain |
| **Traceability** | `discovered-from` link | Context embedded in AC |
| **Gatekeeping** | None (PM judgment) | Sr PM triage gate |
| **Mandatory** | Yes, regardless of accept/reject | Yes (but only after acceptance) |
| **Validation** | None | Anchor reviews for structure |

## Trade-Offs

### Old Model: Distributed (PM Creates)

Strengths:
- Fast: bugs filed immediately when found
- Low friction: PM doesn't wait for Sr PM
- Clear traceability: `discovered-from` link explicit
- Context fresh: PM captures details right after finding them
- No messaging overhead: PM creates directly in backlog

Weaknesses:
- Quality varies: different PMs have different bug templates
- Weak AC: P2 priority often means bugs are deprioritized
- No epic strategy: bugs go to current epic by default
- Proliferation: every found issue becomes a bug (no filtering)
- No consolidation: duplicate bugs not caught
- Sr PM can't influence scope: Sr PM sees bugs after creation, not before

### Current Model: Centralized (Sr PM Creates)

Strengths:
- Consistent quality: all bugs follow same P0 structure with full AC
- Strategic: Sr PM decides bug scope, epic placement, dependencies
- Consolidated: Sr PM can deduplicate or merge similar issues
- Filtered: Sr PM can clarify with user (is this a bug or a feature request?)
- Gatekept: Anchor validates bug structure as part of backlog review
- Documented: bugs have full acceptance criteria, root cause analysis

Weaknesses:
- Async overhead: PM-Acceptor must wait for Sr PM triage
- Communication required: structured blocks + Sr PM processing
- Context loss: details captured in text block, not direct creation
- Slower feedback: developer doesn't see bug immediately
- Single point of failure: Sr PM becomes bottleneck for bug triage
- User interaction: Sr PM may need to clarify if bug or feature

## Impact on Developer Experience

### Old Model
Developer implements story:
1. Finds bug during coding
2. Notes it in LEARNINGS
3. PM-Acceptor reviews story, sees bug
4. PM creates bug immediately
5. Developer sees new bug in backlog next execution loop

### Current Model
Developer implements story:
1. Finds bug during coding
2. Notes it in LEARNINGS
3. PM-Acceptor reviews story, sees bug
4. PM-Acceptor outputs DISCOVERED_BUG block
5. Dispatcher sends to Sr PM
6. Sr PM creates structured bug (may ask user for clarification)
7. Developer sees new bug in backlog eventually

## When Each Model Works Best

### Old Model Works Well When:
- Teams trust PM judgment
- Bugs are mostly straightforward (simple fixes)
- Quick feedback to developers is critical
- Epic structure is stable (most bugs belong to current epic)
- You want minimal process overhead

### Current Model Works Well When:
- You need bug consistency and quality
- Bugs span multiple epics (Sr PM routes strategically)
- Some "bugs" are actually feature requests (Sr PM can filter)
- User clarification is sometimes needed (Sr PM asks)
- Bugs need full AC and root cause analysis
- You want traceability through Sr PM decisions

## Current Implementation Notes

The current system made this change for a specific reason (from Session Operating Mode):

> "Sr PM is the ONLY agent authorized to create bugs. All bugs are P0. No exceptions."

This suggests the decision was:
1. Raise priority of all bugs to P0 (old: P2 meant they were often ignored)
2. Ensure bugs have full acceptance criteria (Sr PM expertise)
3. Prevent proliferation (Sr PM filters/consolidates)
4. Strategic placement (Sr PM considers epic dependencies)
5. User visibility (Sr PM can clarify ambiguous issues)

## Recommendation: Document as Design Decision

This is a legitimate architectural choice:

**Old system optimizes for:** Speed, low friction, developer feedback
**Current system optimizes for:** Quality, consistency, strategic scope

The centralized approach works well in paivot-graph because:
1. Sr PM is already a gatekeeper (backlog creation + bug triage)
2. All bugs are P0 (no triage levels to manage)
3. Structured AC required (Sr PM expertise)
4. Epic strategy matters (two-level branch model)
5. User interaction is expected (D&F clarifications already happen)

This could break down if:
1. Sr PM becomes bottleneck (bugs pile up waiting for triage)
2. Async communication creates context loss
3. Developers feel disconnected (can't create bugs directly)
4. Simple bugs take too long (full AC for 1-line fix)

## Implemented: Bug Fast-Track (opt-in)

The fast-track model is now available as an opt-in setting. When enabled, PM-Acceptor
can create bugs directly during story review, bypassing the Sr PM routing step.

**Activation (either of these):**
- Project setting: `pvg settings bug_fast_track=true`
- Per-story label: `pm-creates-bugs`

**Mandatory guardrails (non-negotiable, enforced in PM-Acceptor prompt):**
- Priority: ALWAYS P0 (hardcoded)
- Parent: ALWAYS set to story's epic (auto-detected via `nd show`)
- Label: ALWAYS `discovered-by-pm` (origin tracking)
- Deduplication: Check existing bugs in epic before creating

**Default behavior unchanged:** When bug_fast_track is false and no label is present,
PM-Acceptor outputs DISCOVERED_BUG blocks for Sr PM triage (centralized model).

See `agents/pm.md` for the full conditional logic and `commands/vault-settings.md`
for the setting documentation.

## Related

- [[Session Operating Mode]] -- Dispatcher operating mode and bug routing
- [[Sr PM Agent]] -- Bug triage mode section (default path)
- [[PM Acceptor Agent]] -- Reporting Discovered Bugs section (conditional: centralized vs fast-track)
