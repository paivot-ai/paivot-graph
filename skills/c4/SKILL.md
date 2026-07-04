---
name: c4
description: Architecture-as-code on the machinery design substrate. Use when the project's design.machinery setting applies (auto detects a machinery-managed repo), when the legacy architecture.c4 setting is enabled, or when the user asks about C4 diagrams, Structurizr, architecture boundaries, dependency rules, the Architecture Contract, boundary baselining, or import drift. Maps the Paivot roles onto machinery Phase 2: who authors the model, which gate holds it, and how boundary debt is baselined and burned down.
version: 2.0.0
---

# Architecture with machinery (C4 + contract)

The canonical architecture is machinery Phase 2: `design/workspace.dsl` (the C4 model in
Structurizr DSL) plus the machine-checkable Architecture Contract inside
`design/ARCHITECTURE.md`. The narrative explains why; the DSL and contract define what,
and `machinery check` holds the line deterministically. This skill is the ROLE ADAPTER:
it says who does what in Paivot. The format itself is documented once, in the machinery
skill's `references/c4-standalone.md`; never restate it here or in stories.

## When this applies

```bash
pvg settings design.machinery    # auto (default) | on | off
```

- `auto`: applies exactly when the repo is machinery-managed (a `.machinery.json` at the
  root, or `design/domain.modelith.yaml`). `pvg gates`, `pvg rtm`, and
  `pvg story approve-red` resolve this themselves; agents do not need to re-derive it.
- `on`: promised; a missing design fails loudly. `off`: disabled.
- Legacy `architecture.c4=true` projects follow the narrative-twin flow this skill's v1
  described; migrate them by moving `workspace.dsl` and the contract under `design/` and
  letting `auto` detection take over.

The `machinery` binary converges from the channel (`pvg update`); `pvg doctor` reports
`machinery-reachable`.

## Role map

| Role | Responsibility |
|---|---|
| Architect | Authors Phase 2: `design/workspace.dsl`, `design/ARCHITECTURE.md` with the Architecture Contract (boundaries with `code:` globs, `exposes`, externals, `ignore`, `dependency_rules`), the per-dependency failure postures, and the NFR record. Exit gate: `machinery check design --gate g2` green BEFORE handing to the Sr PM. |
| Architect (brownfield) | Runs `machinery baseline design --impl .`, reviews the proposed `baseline:` rules with the user (a baseline edge is tolerated debt, structurally distinct from an intended `allow:`; add a `deny:` for edges that should die), pastes the survivors, commits `design/ratchet.json`. The ratchet makes any NEW offender file on an amnestied edge a blocking finding. |
| Sr PM | References contract boundaries in stories by id; never restates the contract. Story ACs include "boundaries respected" only as a pointer to the gate, not as prose to re-check. |
| Developer | Runs `pvg gates` before delivery: the design gate (machinery check, including G4 import boundaries and the ratchet) runs beside the metric gates and blocks on failure. Never edits generated artifacts (`*.oracle.md`, `formal/*.tla|*.cfg`, `packs/`, `pack/`, `ratchet.json`); edit sources and regenerate. |
| Anchor | Deterministic pre-pass first (`pvg gates`, `pvg rtm`); attests only what the tools cannot: whether the boundaries are the RIGHT ones, whether every Modelith action has an owning component, whether the NFR record is real. |

## The deterministic split

`machinery check --gate g2` verifies: the contract parses, binds to `workspace.dsl`,
no duplicate ids, no edge both allowed and denied (or allowed and baselined), every
dependency has a mitigation row. G4 (via `pvg gates` on projects with `impl` configured
in `.machinery.json`) verifies the code's import graph against the contract and holds
baselined edges to the ratchet snapshot. Everything else about the architecture is
attested by a named reviewer; the gate split in machinery's SKILL.md says exactly which
half is whose.

## Boundary debt ceremony (brownfield)

- `machinery baseline` reruns tighten the ratchet after burn-down; "ratchet can tighten"
  notes in gate output are the agenda for the monthly debt review.
- `ratchet.json` diffs are reviewed in PRs like contract changes; unexplained regrowth is
  the tell.
- `ignore:` globs stay unratcheted amnesty; shrinking them is part of the same cadence.

## Diagrams

`workspace.dsl` loads into Structurizr tooling; export diagrams to `docs/diagrams/` when
the team wants them (`structurizr-cli export -workspace design/workspace.dsl -format
mermaid -output docs/diagrams/`). Diagrams are derived artifacts; the DSL is the source.
