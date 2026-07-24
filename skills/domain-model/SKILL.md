---
name: domain-model
description: Canonical domain model and its derivation chain on the machinery design substrate. Use when the user has explicitly enabled the design.machinery setting (default off; artifact presence alone enables nothing), when the legacy dnf.domain_model setting is enabled, or when the user asks about the domain model, entities, invariants, ubiquitous language, state machines, oracles, stable ids, the relational layers (policy, integrity, isolation), rebuild or hybrid migration, the surface ledger, or how stories derive from the design. Maps the Paivot roles onto machinery Phases 1, 1.5, and 3 plus the Phase 4 backlog handoff: the Architect owns the model, the annotations, and the machines; the Sr PM dereferences oracle rows and the build plan into stories; pvg rtm checks coverage deterministically.
version: 2.1.0
---

# Domain model in D&F (machinery Phases 1 and 3)

The model is `design/domain.modelith.yaml`, authored with modelith and linted by
`modelith lint`: the single canonical source of the product's named concepts (entities),
how they relate, and the rules that must always hold (invariants). The three D&F
documents reference it; they never redefine the vocabulary. This remains the cure for
context divergence, and on the machinery substrate it is only the beginning of a
derivation chain: lifecycle enums become state machines (Phase 3), machines generate
transition oracles, static relational invariants compile into Alloy models with their
own decision-table oracles (Phase 1.5), and every oracle row carries a content-derived
STABLE ID that stories and tests key on. Half of each machine is derived from this
model, which is why it must lint clean before anything downstream exists.

## When this applies

```bash
pvg settings design.machinery    # off (default) | on | auto
```

Same resolution as the c4 skill: the substrate applies ONLY when the user has
explicitly enabled it (`on`, or `auto` as a deliberate user choice to re-enable
artifact detection). The presence of machinery artifacts (`.machinery.json`,
`design/domain.modelith.yaml`) does NOT enable it. Enabling machinery is a user
decision with significant token and time cost: agents may RECOMMEND it, stating those
costs, but must NEVER set it themselves. Legacy `dnf.domain_model=true` projects keep
the v1 narrative-twin flow until the model moves under `design/`. The `modelith` and
`machinery` binaries both converge from the channel (`pvg update`).

## Role map

| Role | Responsibility |
|---|---|
| BA / Designer | Feed the interrogation, sweeping the five EARS behavior categories by name (ubiquitous, event-driven, state-driven, optional, unwanted); unwanted and state-driven are the chronically under-specified two, and they become negative invariants now and lifecycle states and guards in Phase 3. On brownfield, the question that matters for every tangle: "this looks messy because <specific observation>; what is your desired end state?" The code says what IS; only the user can say what SHOULD BE. |
| Architect | Owns `design/domain.modelith.yaml`. Gate: `modelith lint` clean; every entity with a lifecycle has a status enum; every invariant has an owner. When the model carries static relational invariants, also owns the opt-in Phase 1.5 annotations, `design/formal/{policy,integrity,isolation}.relational.yaml`: `machinery alloy design/` compiles each present layer, `machinery verify-formal` solver-checks them, and gates gp/gi/gn hold binding, coverage, and freshness deterministically. Owns Phase 3 too: one machine per stateful component (the machinery skill and its fsm-author role doc drive the synthesis), then `machinery oracle design/machines` regenerates the committed oracles. An annotation or machine edit and its regenerated artifacts land in the SAME change; staleness is DRIFT and blocks. |
| Architect (rebuild/hybrid) | Owns BOTH domain truths: `design/legacy/domain.modelith.yaml` (the system as it is) and `design/domain.modelith.yaml` (the normative target), never merged into one compromise model. Owns the strict transition contract `design/migration.yaml` (gate Gm-transition: every legacy entity disposed, every enum value covered, ordered phases with entry/exit, rollback, and cutover criteria) and the surface ledger `design/legacy/surface.yaml` (gate Gs-surface), authored by the opening sweep and settled by the closing sweep after Gate 4. |
| Sr PM | Dereferences the design into stories: invariants become acceptance criteria, and for machine-covered slices the ORACLE ROWS become the test spec. Story ACs cite oracle stable ids verbatim (whole tokens, e.g. `DEAL-eb0c40`); those ids are what `pvg rtm` and `pvg story approve-red` key on. Backlog derivation starts from BUILD.md's Build plan section, whose shape Gb-plan holds deterministically (milestones with `DoD:` lines; the walking-skeleton milestone's DoD cites a committed oracle id). On rebuild/hybrid, also derives migration-step stories from `migration.yaml`'s ordered phases (entry/exit criteria and rollback become ACs) and surface-parity stories from ledger rows. Model the stateful core only: CRUD screens and pure transforms get ordinary stories, no machine ceremony. |
| Anchor | Runs `pvg rtm` in the deterministic pre-pass: every oracle stable id must have a covering story ([ORACLE] rows in the report use exact token matching, not keywords). On rebuild/hybrid, also verifies backlog coverage of the transition: every migration phase and replace-mapping has a story, and every surface ledger row is story-covered or carries a deliberate dropped/deferred rationale (no opening placeholders at handoff; the Gs `checked:` line prints the disposition counts). Attests what the tools cannot: whether the invariants are the RIGHT ones (a shallow model gates clean), and the judgment half of the relational layers: gp/gi/gn prove binding, coverage, and freshness, not that an annotation captures the real policy or that its residuals are honest. |
| Developer | Derives hard-TDD RED tests from the oracle rows the story cites, keyed on stable ids. Gt-tests is the RED-exit check made deterministic: with `impl` configured, every committed oracle stable id (machine, policy, isolation) must appear whole-token in the suite, or a test file must earn the strict conformance-parse citation. `pvg story approve-red` verifies this before the suite locks. |
| PM | On design revisions, `pvg story sync-oracle --base <ref>` maps the stable-id diff onto affected stories: added or modified ids need tests re-derived, removed ids mark tests to retire. |

## User-enabled machinery implies hard-TDD

On a project where the user enabled `design.machinery`, the design ships a test oracle
and Gt-tests holds the implementation to it. The Paivot rule: any story whose ACs cite
oracle stable ids MUST carry the `hard-tdd` label; `pvg lint --backlog` enforces this
deterministically as the `hard-tdd-oracle` check. This fusion holds only because the
user opted into the substrate. Stories that touch no oracle (CRUD screens, pure
transforms) get the label only when the user requested or pre-authorized hard-TDD; the
Sr PM recommends it where it would pay off instead of applying it by judgment.

## Codebase archaeology (brownfield)

Excavate the model from the code, the schema, and the production data AS IT IS; record
incoherence as open questions instead of picking winners. When a codebase-graph MCP is
available (codebase-memory-mcp or equivalent), index the repository first and drive the
excavation through its tools (`get_architecture`, `search_graph`, `trace_call_path`)
instead of grep; fall back to plain search only without one. Start this dialog on day
one, in parallel with the boundary baseline: intended boundaries are a domain claim.
When the system has role- or ownership-based access control, author the policy
annotation as the code BEHAVES today and let the meta-checks and the generated authz
oracle arbitrate; a failing row is a discovered incoherence, adjudicated like any other
archaeology finding.

## Format and deep dives

The modelith YAML format: `modelith schema`. The four-phase pipeline, the machine
annotations, the oracle contract, the relational layers, rebuild/hybrid mode, the
surface ledger, and revision mode: the machinery skill (installed with the machinery
plugin) and its references (`rebuild-guide.md`, `surface-ledger.md`,
`xstate-format.md`, `c4-standalone.md`). Do not restate formats in stories or D&F
documents; reference them.
