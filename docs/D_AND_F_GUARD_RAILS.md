# D&F Guard Rails: Old Challenger Model vs Current Anchor Model

## Historical Context: The Challenger Pattern (ns-paivot)

The old system had **phase-specific challenger agents** that reviewed each D&F document immediately after creation:

**The three challengers:**
- **BA Challenger**: Reviewed BUSINESS.md for omissions, hallucinations, misinterpretations, scope creep against user_input.md
- **Designer Challenger**: Reviewed DESIGN.md for unmet user needs, hallucinations, contradictions with BUSINESS.md and user_input.md
- **Architect Challenger**: Reviewed ARCHITECTURE.md for unmet requirements, untraceable decisions, contradictions across all sources (user_input + BUSINESS + DESIGN)
- **Backlog Challenger**: Reviewed Sr PM's backlog for completeness and story quality

**Detection focus:**
- OMISSIONS: Requirements not addressed
- HALLUCINATIONS: Content not traceable to authoritative sources
- DRIFT: Contradictions with prior documents or user input

**Escalation:** Challengers never talked to user. Feedback looped back to the creator through orchestrator.

## Current System: Single Anchor Model (paivot-graph)

The current system has **one adversarial reviewer (Anchor) that reviews the final backlog** after Sr PM creates it:

**Single checkpoint:**
- Anchor reviews backlog for gaps, missing walking skeletons, horizontal layers, integration gaps
- Occurs after Sr PM translates BUSINESS/DESIGN/ARCHITECTURE into stories
- All three document contexts are evaluated through the lens of story quality

**Detection focus:**
- Walking skeleton missing in milestones
- Vertical slices vs horizontal layers
- Missing integration stories
- D&F coverage incomplete
- Test gaps (mocks in integration tests forbidden)
- Story self-containment

## Comparison

| Aspect | Old Challengers | Current Anchor |
|--------|-----------------|-----------------|
| **Timing** | After each document (D&F level) | After backlog (execution level) |
| **Scope** | Specialized (one per BLT member) | Holistic (entire backlog) |
| **Catch Problems** | Early, before cascade | Late, at aggregation point |
| **Expertise** | Document-specific principles | Delivery methodology |
| **Escalation** | To document creator (BA/Designer/Architect) | To Sr PM |
| **Cost** | 8 agents total (BLT + challengers + Sr PM + backlog challenger) | 5 agents total (BLT + Sr PM + Anchor) |
| **User Interaction** | Potential clarifications 3+ times | Clarifications during D&F, none after |
| **Problem Traceability** | Clear: "issue in ARCHITECTURE came from DESIGN miss" | Harder: "issue in story traces to which document?" |

## Structural Trade-Offs

### Old Approach: Early, Specialized Guardrails

Strengths:
- Catches errors before they cascade (BUSINESS error doesn't propagate to ARCHITECTURE)
- Designer Challenger understands composition, information architecture, user journey coherence
- Architect Challenger understands system decomposition, integration patterns, feasibility
- BA Challenger understands scope creep and requirement tracing
- Each document is defensible before moving to next phase
- Phase-specific expertise reduces false positives

Weaknesses:
- More agents to spawn (8 vs 5)
- More expensive
- User could be asked for clarifications 3+ times (fatigue)
- Complex orchestration (BLT → Challengers → Feedback → Refinement loop)
- Orchestrator must route feedback to correct creator

### Current Approach: Late, Holistic Gatekeeping

Strengths:
- Simpler orchestration (fewer agents, linear pipeline)
- Lower cost
- Fewer user interaction rounds (all clarifications during D&F)
- Anchor evaluates real-world deliverability (will stories work?)
- Single decision point (Sr PM → Anchor → execute or iterate)

Weaknesses:
- Problems cascade (BUSINESS miss becomes ARCHITECTURE miss becomes bad story)
- Harder to trace issue origin (is missing feature from BUSINESS or DESIGN?)
- Anchor must understand three document contexts simultaneously
- No specialized expertise (designer principles vs architect principles vs delivery principles)
- One misjudgment in BUSINESS.md ripples through entire backlog

## Current System: BLT Convergence Layer

The current system added **cross-review within BLT** (not between BLT and challengers):

```
1. BA creates BUSINESS.md
2. Designer creates DESIGN.md (reading BUSINESS.md)
3. Architect creates ARCHITECTURE.md (reading BUSINESS.md + DESIGN.md)
   ↓
4. BLT CONVERGENCE: Each member reviews the others' work
   - BA: "Does DESIGN.md and ARCHITECTURE.md reflect BUSINESS.md?"
   - Designer: "Does BUSINESS.md and ARCHITECTURE.md support user needs?"
   - Architect: "Does BUSINESS.md and DESIGN.md lead to feasible architecture?"
   ↓
5. Sr PM creates backlog
6. Anchor reviews backlog
```

This partially replicates the old challenger function but as **peer review** rather than **adversarial review**. Key difference:
- Peer review: "Let's make sure we're aligned" (collaborative)
- Adversarial review: "Here's what you missed" (skeptical)

## Architectural Implications

### Problem Cascade Example (Old vs New)

Scenario: BUSINESS.md omits a compliance requirement.

**Old system (Challenger at document level):**
1. BA Challenger catches: "HIPAA not in BUSINESS.md but user mentioned it"
2. BA fixes BUSINESS.md to include HIPAA
3. Designer sees HIPAA in BUSINESS.md, designs for it
4. Architect sees HIPAA in BUSINESS.md + DESIGN.md, implements for it
5. Sr PM embeds HIPAA context in stories
Result: Problem caught and fixed at source. No cascade.

**Current system (Anchor at backlog level):**
1. BUSINESS.md omits HIPAA
2. Designer doesn't see HIPAA, doesn't design for it
3. Architect doesn't see HIPAA in BUSINESS.md, doesn't implement for it
4. Sr PM doesn't embed HIPAA context (it's not in the documents)
5. Anchor sees backlog and realizes: "There are no HIPAA compliance stories"
6. Anchor rejects backlog
7. Sr PM has to go back and infer HIPAA requirements (expensive, risky)
Result: Problem caught late. Expensive to trace and fix.

### Inherited Problem Example

Scenario: BUSINESS.md adds requirements user didn't ask for (scope creep).

**Old system:**
- BA Challenger catches: "User never mentioned this requirement"
- BA corrects BUSINESS.md before it affects Design/Architecture
- Problem contained

**Current system:**
- BA creates BUSINESS.md with extra requirement
- Designer sees extra requirement, designs for it
- Architect sees extra requirement, builds for it
- Sr PM embeds extra requirement in stories
- Anchor sees stories and might catch over-engineering
- But origin of the problem (BUSINESS.md scope creep) is harder to trace

## Recommendation: Document as Design Decision

This is a legitimate architectural choice. The tradeoff is:

**Old system optimizes for:** Prevention, specialization, early detection
**New system optimizes for:** Cost, simplicity, user experience (fewer rounds)

Neither is objectively better. It depends on project context:
- High-risk projects with strict compliance: old approach better
- Agile, iterative exploration: new approach better
- Large teams: old approach prevents communication overhead
- Small teams: new approach is less overhead

**For paivot-graph, the decision was made:** Single Anchor at backlog level, with BLT peer convergence layer.

This works well when:
1. D&F agents are high quality (less drift to catch)
2. Sr PM is meticulous (embeds context from all sources)
3. Anchor is thorough (reviews for completeness)
4. User is available for clarifications during D&F (not before backlog)

This could break down if:
1. D&F agents skip questionnaire rounds (rich context missing)
2. Sr PM doesn't embed architecture/design details (developers read external files)
3. Anchor focuses on style rather than substance (stories don't match requirements)

## Implemented: Optional Specialist Review Mode

Specialist challengers are now available as an opt-in setting:

**Activation:** `pvg settings dnf.specialist_review=true`

**Pipeline with specialist review enabled:**

```
BA -> BA Challenger -> Designer -> Designer Challenger -> Architect -> Architect Challenger -> Sr PM -> Anchor
```

**Key design decisions:**
- Challengers use Sonnet (cheap, focused review -- not heavy creative work)
- Each challenger loops up to `dnf.max_iterations` times (default 3)
- Challengers never talk to user -- feedback routes to creator via dispatcher
- After max iterations exhausted, dispatcher escalates to user with remaining issues
- Default is still cost-optimized (challengers disabled) -- same as before
- Anchor stays regardless (backlog-level review complements document-level review)

**Differences from old ns-paivot challengers:**
- No Backlog Challenger (Anchor handles this)
- Creator feedback loop instead of user clarification rounds (less user fatigue)
- Opt-in per project, not mandatory
- Max iteration cap prevents infinite loops
- Structured output format (REVIEW_RESULT: APPROVED/REJECTED) for reliable parsing

**Agent definitions:** `agents/ba-challenger.md`, `agents/designer-challenger.md`, `agents/architect-challenger.md`
**Setting docs:** `commands/vault-settings.md` (dnf.specialist_review, dnf.max_iterations)
**Orchestration:** Session Operating Mode vault note (D&F ORCHESTRATION section)

## Related

- [[Session Operating Mode]] — D&F orchestration with specialist review loop
- [[Anchor Agent]] — Backlog-level adversarial review (complements document-level challengers)
- [[BA Challenger Agent]] — Reviews BUSINESS.md
- [[Designer Challenger Agent]] — Reviews DESIGN.md
- [[Architect Challenger Agent]] — Reviews ARCHITECTURE.md
- [[Testing Philosophy]] — Integration test mandate (replaces some Architect Challenger checks)
