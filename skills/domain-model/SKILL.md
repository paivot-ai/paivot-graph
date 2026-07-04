---
name: domain-model
description: Canonical domain model and its derivation chain on the machinery design substrate. Use when the project's design.machinery setting applies (auto detects a machinery-managed repo), when the legacy dnf.domain_model setting is enabled, or when the user asks about the domain model, entities, invariants, ubiquitous language, state machines, oracles, stable ids, or how stories derive from the design. Maps the Paivot roles onto machinery Phases 1 and 3: the Architect owns the model and machines, the Sr PM dereferences oracle rows into acceptance criteria, and pvg rtm checks coverage deterministically.
version: 2.0.0
---

# Domain model in D&F (machinery Phases 1 and 3)

The model is `design/domain.modelith.yaml`, authored with modelith and linted by
`modelith lint`: the single canonical source of the product's named concepts (entities),
how they relate, and the rules that must always hold (invariants). The three D&F
documents reference it; they never redefine the vocabulary. This remains the cure for
context divergence, and on the machinery substrate it is only the beginning of a
derivation chain: lifecycle enums become state machines (Phase 3), machines generate
transition oracles, and oracle rows carry content-derived STABLE IDS that stories and
tests key on. Half of each machine is derived from this model, which is why it must lint
clean before anything downstream exists.

## When this applies

```bash
pvg settings design.machinery    # auto (default) | on | off
```

Same resolution as the c4 skill: auto detects a machinery-managed repo. Legacy
`dnf.domain_model=true` projects keep the v1 narrative-twin flow until the model moves
under `design/`. The `modelith` and `machinery` binaries both converge from the channel
(`pvg update`).

## Role map

| Role | Responsibility |
|---|---|
| BA / Designer | Feed the interrogation: what exists, what must always be true, where one word carries two meanings. On brownfield, the question that matters for every tangle: "this looks messy because <specific observation>; what is your desired end state?" The code says what IS; only the user can say what SHOULD BE. |
| Architect | Owns `design/domain.modelith.yaml`. Gate: `modelith lint` clean; every entity with a lifecycle has a status enum; every invariant has an owner. Owns Phase 3 too: one machine per stateful component (the machinery skill and its fsm-author role doc drive the synthesis), then `machinery oracle design/machines` regenerates the committed oracles. A machine edit and its regenerated oracle land in the SAME change; staleness is DRIFT and blocks. |
| Sr PM | Dereferences the model into stories: invariants become acceptance criteria, and for machine-covered slices the ORACLE ROWS become the test spec. Story ACs cite oracle stable ids verbatim (whole tokens, e.g. `DEAL-eb0c40`); those ids are what `pvg rtm` and `pvg story approve-red` key on. Model the stateful core only: CRUD screens and pure transforms get ordinary stories, no machine ceremony. |
| Anchor | Runs `pvg rtm` in the deterministic pre-pass: every oracle stable id must have a covering story ([ORACLE] rows in the report use exact token matching, not keywords). Attests what the tool cannot: whether the invariants are the RIGHT ones; a shallow model gates clean. |
| Developer | Derives hard-TDD RED tests from the oracle rows the story cites, keyed on stable ids. `pvg story approve-red` verifies this deterministically before the suite locks. |
| PM | On design revisions, `pvg story sync-oracle --base <ref>` maps the stable-id diff onto affected stories: added or modified ids need tests re-derived, removed ids mark tests to retire. |

## Codebase archaeology (brownfield)

Excavate the model from the code, the schema, and the production data AS IT IS; record
incoherence as open questions instead of picking winners. When a codebase-graph MCP is
available (codebase-memory-mcp or equivalent), index the repository first and drive the
excavation through its tools (`get_architecture`, `search_graph`, `trace_call_path`)
instead of grep; fall back to plain search only without one. Start this dialog on day
one, in parallel with the boundary baseline: intended boundaries are a domain claim.

## Format and deep dives

The modelith YAML format: `modelith schema`. The four-phase pipeline, the machine
annotations, the oracle contract, and revision mode: the machinery skill (installed with
the machinery plugin) and its references. Do not restate formats in stories or D&F
documents; reference them.
