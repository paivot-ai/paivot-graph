---
name: domain-model
description: Canonical domain model (entities, relationships, invariants, scenarios) as a machine-checkable twin of ARCHITECTURE.md, authored with modelith. Use when the project has `dnf.domain_model` enabled in settings, or when the user asks about the domain model, entities, invariants, ubiquitous language, or shared vocabulary during Discovery & Framing. Teaches agents how the Architect owns the model, how the Sr PM turns invariants into acceptance criteria, and how the Anchor checks coverage.
version: 1.0.0
---

# Domain Model in D&F

Maintain a machine-checkable domain model alongside the narrative ARCHITECTURE.md. The model is the single, canonical source of the product's named concepts (entities), how they relate, and the rules that must always hold (invariants). The three D&F documents reference it; they do not each redefine the vocabulary. This is the cure for the single largest D&F failure mode: context divergence (the same concept named differently across BUSINESS.md, DESIGN.md, ARCHITECTURE.md, and the stories).

## When This Applies

Check the project setting before using this skill:

```bash
pvg settings dnf.domain_model
```

- `true` -- the Architect maintains the model; the Sr PM dereferences it into stories; the Anchor checks coverage.
- `false` (default) -- skip entirely, use the narrative ARCHITECTURE.md only.

If the setting is not enabled and the user hasn't asked for a domain model, do not use this skill.

## The tool: modelith

The model is a `*.modelith.yaml` file, linted and rendered by the `modelith` CLI (provisioned by `pvg setup` / `pvg update`; `pvg doctor` reports it). Confirm it is installed:

```bash
modelith --version    # if missing: pvg update
modelith schema       # the authoritative format reference -- read before authoring
```

The YAML is the output of a conversation, not something hand-written. The value is in the questions that pin down each concept, not the typing. If a concept cannot be given a crisp two-to-four-sentence definition, that fuzziness is the signal to resolve, not paper over.

## File Layout

```
domain.modelith.yaml       # Canonical domain model (linted; the machine-checkable twin)
domain.modelith.md         # Generated Markdown + ER diagram (never hand-edited)
ARCHITECTURE.md            # Narrative architecture (always exists; references the model)
```

The `.yaml` is authored; the `.md` is regenerated with `modelith render` and committed alongside it. Keep them in sync. The model lives at the repo root, the machine-checkable twin of ARCHITECTURE.md's "Data architecture" section, exactly as `workspace.dsl` is for the C4 skill.

## Build Order -- Skeleton First

A model is built in passes across the whole model, not field-by-field down one entity. Stop after any pass and you still have something honest.

1. **Skeleton.** Name every entity with a crisp two-to-four-sentence `definition`; declare the `relationships` and `cardinality` (`1:1`, `1:n`, `n:1`, `n:n`) and `ownership` (`owned` = a part that cannot exist without its parent; `referenced`/omitted = an independent entity merely pointed at). This already renders to a real ER diagram and is the minimum useful model.
2. **Behavior.** Add `invariants` (rules that must always hold; each `{id, statement}`, id lowercase kebab-case, statement backticking entity names) and `scenarios` (short narratives exercising every entity, tagged with the `invariants_touched` ids).
3. **Refinement.** Fill in `attributes` (primitive lowercase types or PascalCase enum names), `enums`, `actions`, and `glossary` roles -- only where they add clarity.

Entity keys are PascalCase (`Regulation`, not `regulation`). Backtick entity names in freeform text (`` `Regulation` ``); do not backtick them in structured fields.

## Always Finish by Validating and Rendering

After any edit:

```bash
modelith lint domain.modelith.yaml      # resolve errors; explain remaining warnings
modelith render domain.modelith.yaml    # regenerate the committed Markdown twin
```

`modelith lint` exits non-zero on errors -- that is the model telling you to fix something, not a tool failure.

## Agent Responsibilities

### Architect Agent

Owns `domain.modelith.yaml`. When `dnf.domain_model` is enabled:

1. Author the model by conversation (skeleton first), seeded by the concepts the BA surfaced in BUSINESS.md and the user types in DESIGN.md.
2. Keep the model's entity names, relationships, and invariants consistent with ARCHITECTURE.md's data architecture. The model is canonical; ARCHITECTURE.md references its names rather than redefining them.
3. Run `modelith lint` (must pass) and `modelith render` (commit the `.md` twin) on every change.
4. The domain model is a protected, architect-owned D&F artifact: the guard blocks writes to `*.modelith.yaml` unless the architect agent is active. Only the Architect writes it.

The narrative in ARCHITECTURE.md explains *why*. The model defines *what* and its rules -- machine-checkably.

### Sr PM Agent

When `dnf.domain_model` is enabled, the model is the naming authority. For every story that touches a modeled concept:

- **Dereference, do not reinvent.** Use entity and attribute names verbatim from the model. The Terminology Audit checks story prose against the model's canonical names, not memory.
- **Turn invariants into acceptance criteria.** Each invariant that the story must uphold becomes an EARS **Ubiquitous** AC ("The system shall ..."), referencing the invariant. Invariants map to ACs roughly one-to-one; do not restate rules from memory.
- Add to the story's MANDATORY SKILLS TO REVIEW section:
  - `domain-model` -- for canonical vocabulary and invariant checking.

### Developer Agent

When `dnf.domain_model` is enabled, before coding:

1. Read the domain model for the entities and invariants the story's code paths touch.
2. Use the model's canonical names for types, tables, and fields.
3. Uphold the relevant invariants; if the implementation forces a change to a concept's definition or relationships, that is an architecture change -- raise it, do not silently diverge.

### Anchor Agent

When `dnf.domain_model` is enabled, add to the backlog review checklist:

- Every entity in the model is touched by at least one story.
- Every invariant maps to at least one acceptance criterion in some story.
- No story renames a modeled concept (context divergence from the model is a rejection).
- The `.md` twin is in sync with the `.yaml` (regenerated, not hand-edited).

## What This Skill Does NOT Do

- Does not run when `dnf.domain_model` is `false` (default). Zero behavior change until opted in.
- Does not replace ARCHITECTURE.md -- the narrative always exists and references the model.
- Does not add a CI check; the Architect self-validates with `modelith lint` and the Anchor performs coverage review (mirrors the `c4` skill).
- Does not force a domain model onto thin, infra-heavy, or mechanical work -- it is opt-in per project.
- Does not require modelith at runtime for any other Paivot function; it is only needed while authoring or validating the model.
