---
name: c4
description: Architecture-as-code on the machinery design substrate. Use when the user has explicitly enabled the design.machinery setting (default off; artifact presence alone enables nothing), when the legacy architecture.c4 setting is enabled, or when the user asks about C4 diagrams, Structurizr, architecture boundaries, dependency rules, the Architecture Contract, the event-contract table, boundary baselining, import drift, or the transition architecture of a rebuild. Maps the Paivot roles onto machinery Phase 2: who authors the model, which gates hold it (G2, G4, G5 and the ratchet), and how boundary debt is baselined and burned down.
version: 2.2.0
---

# Architecture with machinery (C4 + contract)

The canonical architecture is machinery Phase 2: `design/workspace.dsl` (the C4 model in
Structurizr DSL), the machine-checkable Architecture Contract inside
`design/ARCHITECTURE.md`, and, for multi-component designs, the event-contract table
(producer, consumer, payload by Modelith attribute reference, delivery guarantee). The
narrative explains why; the DSL, contract, and table define what, and `machinery check`
holds the line deterministically. This skill is the ROLE ADAPTER: it says who does what
in Paivot. The formats are documented once, in the machinery skill's
`references/c4-standalone.md` (Architecture Contract v2, event-contract table, adoption
closure, NFR record); never restate them here or in stories.

## When this applies

```bash
pvg settings design.machinery    # off (default) | on | auto
```

- `off` (default): disabled. The presence of machinery artifacts (`.machinery.json`,
  `design/domain.modelith.yaml`) does NOT enable anything; only the user setting does.
- `on`: promised; a missing design fails loudly.
- `auto`: a deliberate user choice to re-enable artifact detection -- applies exactly
  when the repo carries a `.machinery.json` at the root or
  `design/domain.modelith.yaml`. `pvg gates`, `pvg rtm`, and `pvg story approve-red`
  resolve this themselves; agents do not need to re-derive it.
- Enabling machinery is a user decision with significant token and time cost: agents
  may RECOMMEND enabling it, stating those costs, but must NEVER run
  `pvg settings design.machinery=...` themselves.
- Legacy `architecture.c4=true` projects follow the narrative-twin flow this skill's v1
  described; migrate them by moving `workspace.dsl` and the contract under `design/`,
  then the user chooses whether to set `auto` or `on`.

The `machinery` binary converges from the channel (`pvg update`); `pvg doctor` reports
`machinery-reachable`.

## Role map

| Role | Responsibility |
|---|---|
| Architect | Authors Phase 2: `design/workspace.dsl`, `design/ARCHITECTURE.md` with the Architecture Contract (boundaries with `code:` globs, `exposes`, externals, `ignore`, `dependency_rules`), the per-dependency failure postures, the NFR record, and, on multi-component designs, the event-contract table with named enumeration sources (machine-checkable format when the design decomposes: pack generation and G5 resolve cells by exact component name and fail loudly on any they cannot). Every technology choice gets its adoption closure enumerated, with closure members carried into the mitigation table. Exit gate: `machinery check design --gate g2` green BEFORE handing to the Sr PM. |
| Architect (brownfield) | Runs `machinery baseline design --impl .`, reviews the proposed `baseline:` rules with the user (a baseline edge is tolerated debt, structurally distinct from an intended `allow:`; add a `deny:` for edges that should die), pastes the survivors, commits `design/ratchet.json`. The ratchet makes any NEW offender file on an amnestied edge a blocking finding. |
| Architect (rebuild/hybrid) | Adds the `Transition architecture` section to ARCHITECTURE.md: temporary exporter, replication or dual-write, routing, observability, failure posture. Temporary migration dependencies get the full treatment: detection, mitigation, residual, owner. Gm-transition reports narrative-bridge findings until ARCHITECTURE.md and BUILD.md exist; that is expected, not a defect. The migration contract and surface ledger themselves are in the domain-model skill's role map. |
| Sr PM | References contract boundaries and event-contract rows in stories by id; never restates the contract or a payload. Story ACs include "boundaries respected" only as a pointer to the gate, not as prose to re-check. |
| Developer | Runs `pvg gates` before delivery: the design gate (machinery check, including G4 import boundaries and the ratchet) runs beside the metric gates and blocks on failure. Never edits generated artifacts (`*.oracle.md`, `formal/*.tla|*.cfg|*.als`, `formal/*.oracle.md`, `packs/`, `pack/`, `ratchet.json`); edit sources and regenerate. |
| Anchor | Deterministic pre-pass first (`pvg gates`, `pvg rtm`); attests only what the tools cannot: whether the boundaries are the RIGHT ones, whether every Modelith action has an owning component, whether the event-contract table's enumeration sources are real (a table with no named source is a claim with no evidence), whether the dependency declaration itself is complete, and whether the NFR record is real. At a milestone seal it also writes that milestone's acceptance evidence, `design/acceptance/M<n>.yaml`, which Ga-accept binds. |

## The deterministic split

The full suite is `machinery check <design> [--impl <dir>] [--commit <sha>]
[--gate gm,gs,gp,gi,gn,gc,g2,g3,gx,gk,gb,ga,g4,gt,g5]`; gates activate on the artifacts
that exist, fail on absence rather than silently passing, and print `checked:` counts. The
Phase 2 slice: `--gate g2` (G2-c4) verifies the contract parses, binds to
`workspace.dsl`, no duplicate ids, no edge both allowed and denied (or allowed and
baselined), mitigation coverage for every declared external and every
Database/Queue/External-tagged element, and event-contract presence by the header rule.
G4-import (via `pvg gates` on projects with `impl` configured in `.machinery.json`)
verifies the code's import graph against the contract and holds baselined edges to the
ratchet snapshot. G5-pack (automatic on decomposed designs) regenerates packs in memory,
so a lossy event-contract table fails the gate itself, and prints per-pack
boundary-event counts so an unexpected zero is visible. Everything else about the
architecture is attested by a named reviewer; the gate split in machinery's SKILL.md
says exactly which half is whose.

Ga-accept is the build-side member of that suite: it holds milestone CLOSURE to
committed acceptance evidence (`design/acceptance/M<n>.yaml`) and auto-activates once
that directory exists or any milestone carries `Status: closed`, so from the first
closure it runs inside every `pvg gates` too. `--commit <sha>` binds the evidence to the
commit the review ran on; without it the gate still runs and prints a non-blocking note
that binding was not checked. Who writes the evidence, in what order, and against which
commit: [docs/MILESTONE_ACCEPTANCE.md](../../docs/MILESTONE_ACCEPTANCE.md).

## Boundary debt ceremony (brownfield)

- `machinery baseline` reruns tighten the ratchet after burn-down; "ratchet can tighten"
  notes in gate output are the agenda for the monthly debt review.
- `ratchet.json` diffs are reviewed in PRs like contract changes; unexplained regrowth is
  the tell.
- `ignore:` globs stay unratcheted amnesty; shrinking them is part of the same cadence.

## User-enabled machinery implies hard-TDD

Once the user has enabled `design.machinery` and `impl` is configured, Gt-tests holds
the suite to every committed oracle stable id; the design ships its own test spec. The
Paivot rule follows: any story citing oracle stable ids must carry the `hard-tdd` label
(`pvg lint --backlog` enforces this deterministically as the `hard-tdd-oracle` check).
This fusion holds only because the user opted into the substrate. For the parts the
machines do not cover, the label appears only when the user requested or pre-authorized
hard-TDD; the Sr PM recommends it where it would pay off instead of applying it by
judgment.

## Diagrams

`workspace.dsl` loads into Structurizr tooling; export diagrams to `docs/diagrams/` when
the team wants them (`structurizr-cli export -workspace design/workspace.dsl -format
mermaid -output docs/diagrams/`). Diagrams are derived artifacts; the DSL is the source.
