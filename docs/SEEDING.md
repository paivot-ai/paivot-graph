# Vault Seeding: How paivot-graph Plants Notes in the Obsidian Vault

`paivot-graph` operates as a Claude Code plugin whose orchestration layer is the
Obsidian "Claude" vault. Vault seeding is the mechanism that gets agent prompts,
behavioral conventions, and core concepts from the plugin repository into that
vault, while preserving any local edits the user has made.

This document is the source-of-truth reference for what gets seeded, where each
note comes from, how baselines drive 3-way merge, and the exact command sequence
to run after editing seeded content.

> **Important:** there is **no `seed/` directory in the repository**. Earlier
> revisions of this doc claimed agent prompts and behavioral notes lived under
> `paivot-graph/seed/`. They do not. The current authoritative sources are Go
> string constants in `pvg/internal/governance/seed.go`, agent files at
> `paivot-graph/agents/<slug>.md`, and `paivot-graph/skills/vault-knowledge/SKILL.md`.

---

## Overview

Three categories of vault note are seeded by the plugin:

| Category | Vault folder | Source of truth |
|----------|--------------|-----------------|
| Agent prompts (the eight execution agents plus three challenger variants) | `methodology/` | `paivot-graph/agents/<slug>.md` |
| Behavioral conventions (operating mode, checklists, TDD, branching, delivery, etc.) | `conventions/` | Go string constants in `pvg/internal/governance/seed.go` |
| Core concepts (vault-as-runtime, advisory instructions, D&F sequencing) | `concepts/` | Go string constants in `pvg/internal/governance/seed.go` |

The seeded skill content (`Vault Knowledge Skill`) is the single exception: it
lives in `conventions/` but its source is the plugin SKILL file, not a Go string.

The seed routine is implemented in `pvg/internal/governance/seed.go` (the
`Seed` function) and is invoked through the `pvg seed` command.

---

## Source-of-Truth Table

Every note seeded by `pvg seed` is listed below. Paths are relative to the
resolved Obsidian vault directory (typically `~/Claude/`, but discoverable via
`vlt vault="Claude" dir`). Source line numbers are accurate at v1.53.x — verify
against `pvg/internal/governance/seed.go` if the file has shifted.

### Agent prompts (folder: `methodology/`)

These are produced by `seedAgent` (`seed.go:353`), which reads
`${CLAUDE_PLUGIN_ROOT}/agents/<slug>.md` (resolved by `resolveAgentSrc` at
`seed.go:140`), strips frontmatter via `extractBody`, wraps the body in seed
frontmatter, and writes to `methodology/<Vault Name>.md`.

| Rendered note (`methodology/...`) | Source file | Slug |
|-----------------------------------|-------------|------|
| `Sr PM Agent.md` | `paivot-graph/agents/sr-pm.md` | `sr-pm` |
| `PM Acceptor Agent.md` | `paivot-graph/agents/pm.md` | `pm` |
| `Developer Agent.md` | `paivot-graph/agents/developer.md` | `developer` |
| `Architect Agent.md` | `paivot-graph/agents/architect.md` | `architect` |
| `Designer Agent.md` | `paivot-graph/agents/designer.md` | `designer` |
| `Business Analyst Agent.md` | `paivot-graph/agents/business-analyst.md` | `business-analyst` |
| `Anchor Agent.md` | `paivot-graph/agents/anchor.md` | `anchor` |
| `Retro Agent.md` | `paivot-graph/agents/retro.md` | `retro` |
| `BA Challenger Agent.md` | `paivot-graph/agents/ba-challenger.md` | `ba-challenger` |
| `Designer Challenger Agent.md` | `paivot-graph/agents/designer-challenger.md` | `designer-challenger` |
| `Architect Challenger Agent.md` | `paivot-graph/agents/architect-challenger.md` | `architect-challenger` |

### Skill content (folder: `conventions/`)

| Rendered note | Source file | Seed function |
|---------------|-------------|---------------|
| `conventions/Vault Knowledge Skill.md` | `paivot-graph/skills/vault-knowledge/SKILL.md` | `seedSkill` (`seed.go:391`) |

### Behavioral conventions (folder: `conventions/`)

Each note's body is built from a Go string literal inside its dedicated
`seedXxx` function. To change the rendered note you must edit the Go string,
rebuild `pvg`, and reseed.

| Rendered note (`conventions/...`) | Seed function | Approx. line in `seed.go` |
|-----------------------------------|---------------|---------------------------|
| `Session Operating Mode.md` | `seedSessionOperatingMode` | 426 |
| `Pre-Compact Checklist.md` | `seedPreCompactChecklist` | 647 |
| `Stop Capture Checklist.md` | `seedStopCaptureChecklist` | 701 |
| `Hard-TDD.md` | `seedHardTDD` | 734 |
| `Testing Philosophy.md` | `seedTestingPhilosophy` | 804 |
| `Two-Level Branch Model.md` | `seedTwoLevelBranchModel` | 864 |
| `Delivery Workflow.md` | `seedDeliveryWorkflow` | 945 |
| `Subagent question relay via orchestrator.md` | `seedSubagentQuestionRelay` | 1102 |

### Core concepts (folder: `concepts/`)

| Rendered note (`concepts/...`) | Seed function | Approx. line in `seed.go` |
|--------------------------------|---------------|---------------------------|
| `Vault as runtime not reference.md` | `seedVaultAsRuntimeNotReference` | 1026 |
| `Subagents do not follow advisory instructions.md` | `seedSubagentsAdvisoryInstructions` | 1166 |
| `D&F Sequential With Alignment.md` | `seedDFSequentialAlignment` | 1234 |

---

## Plugin Skills vs. Seeded Notes

`paivot-graph` ships three skills under `paivot-graph/skills/`:

| Skill | Path | Seeded into vault? |
|-------|------|--------------------|
| `vault-knowledge` | `paivot-graph/skills/vault-knowledge/SKILL.md` | **Yes** — rendered as `conventions/Vault Knowledge Skill.md` (via `seedSkill`). |
| `nd-agent-integration` | `paivot-graph/skills/nd-agent-integration/SKILL.md` | No — plugin asset only. Loaded at runtime via the `Skill` tool. |
| `c4` | `paivot-graph/skills/c4/SKILL.md` | No — plugin asset only. Loaded at runtime via the `Skill` tool. |

Plugin skills load through Claude Code's Skill mechanism (`Skill(skill="paivot-graph:nd-agent-integration")`). The user vault never sees `nd-agent-integration` or `c4`. Only `vault-knowledge` is duplicated into the vault so that vault-only readers (humans browsing Obsidian, or agents reading vault notes) can find it.

---

## Agents are Self-Contained as of v1.53.0

Prior to `paivot-graph` v1.53.0, agent files in `paivot-graph/agents/` were thin
"vault loaders" containing a single instruction: `vlt vault="Claude" read file="<Agent Name>"`. The actual prompt lived only in the vault.

As of v1.53.0 (commit `f7c0ad1`), every agent file under `paivot-graph/agents/`
contains the full prompt body inline. The Makefile `test` target enforces this
by failing if any of `sr-pm`, `pm`, `developer`, `anchor`, `retro` still
contains the string `Read your full instructions from the vault`. The
consequence:

- **The agent file is the authoritative source for the agent prompt.**
- Vault notes under `methodology/` are still seeded for vault-side discoverability and for the few agents that read each other's notes, but they are derived artifacts.
- Editing only the vault note will be lost on the next `pvg seed --force` run (or worse, will produce diff3 conflict markers).

---

## The `.seed-baselines/` Directory

To support 3-way merge, `pvg` keeps a parallel tree of baseline files inside
the vault. The path is computed by `BaselineDir` (`pvg/internal/governance/baseline.go:10`) as:

```text
<vault>/.seed-baselines/
```

It mirrors the seeded paths exactly. After seeding, the vault looks like:

```text
<vault>/
├── methodology/
│   ├── Sr PM Agent.md
│   ├── Developer Agent.md
│   └── ...
├── conventions/
│   ├── Session Operating Mode.md
│   ├── Hard-TDD.md
│   └── ...
├── concepts/
│   ├── Vault as runtime not reference.md
│   └── ...
└── .seed-baselines/
    ├── methodology/
    │   ├── Sr PM Agent.md           # exact bytes from the last successful seed
    │   ├── Developer Agent.md
    │   └── ...
    ├── conventions/
    │   ├── Session Operating Mode.md
    │   └── ...
    └── concepts/
        └── Vault as runtime not reference.md
```

### When the baseline is written

Baselines are managed by `WriteBaseline` (`baseline.go:31`), called from
`writeNote` (`seed.go:202`):

- **On first creation** of a seeded note: baseline is written immediately after
  the note is created (`seed.go:286`).
- **On a successful clean reseed** (no user edits, force mode): baseline is
  refreshed to match the new content.
- **On a successful 3-way merge** (`Merged` outcome): baseline is refreshed to
  match the new plugin content (`seed.go:245`).
- **On a conflicted 3-way merge** (`Conflicted` outcome): baseline is **NOT**
  updated. This is intentional — the next reseed will use the same ancestor so
  conflict markers can be reproduced or replaced once the user resolves them.

### What baselines are for

Baselines let `pvg seed --force` distinguish three states for any seeded file:

1. **Vault file == baseline** → user has not edited it → safe to overwrite (Updated).
2. **Vault file != baseline** → user has edited it → run 3-way merge (Merged or Conflicted).
3. **No baseline on disk** → first time we have ever seen this file → overwrite and create baseline.

---

## 3-Way Merge via `diff3 -m`

`Merge3` (`pvg/internal/governance/merge.go:13`) shells out to the system
`diff3` binary in merge mode:

```text
diff3 -m <theirs> <base> <ours>
  theirs = current vault file (with potential user edits)
  base   = stored baseline (last seeded content)
  ours   = new plugin content (from Go string or agent file)
```

Exit code semantics:

- `0` → clean merge, no conflict markers in output.
- `1` → conflicts present; output still contains the merged file with
  `<<<<<<<`, `=======`, `>>>>>>>` markers around each conflicted region.
- anything else → execution error (e.g. `diff3` not on PATH); the caller falls
  back to overwriting and emits a `WARN` line.

If `diff3` is missing entirely, `Merge3` returns an error wrapping the original
exec error so the operator gets a clear "install diffutils or verify PATH"
message.

### The five counter outcomes

Every seeded note resolves to exactly one of five outcomes, tracked by the
`Counters` struct (`seed.go:17`). The summary is printed at the end of the
seed run:

```text
Done. Created: N, Updated: N, Merged: N, Conflicted: N, Skipped: N
```

| Counter | When it increments | What happened on disk |
|---------|--------------------|-----------------------|
| `Created` | Note did not exist before this run. | Note written; baseline written. |
| `Updated` | Force mode AND vault content matches baseline (no user edits) OR no baseline existed. | Note overwritten; baseline written. |
| `Merged` | Force mode, user had edits, 3-way merge produced a clean result. | Merged content written; baseline refreshed to plugin content. |
| `Conflicted` | Force mode, user had edits, 3-way merge produced conflict markers. | File written **with** markers; baseline **not** refreshed; path appended to `ConflictedFiles`. |
| `Skipped` | Safe (non-force) mode and the note already exists, OR a source file (agent/skill) was missing. | Nothing written. |

If `Conflicted > 0`, the seeder prints a `CONFLICTS (manual resolution needed):` block listing the affected paths so the operator can resolve them in Obsidian and rerun.

---

## `pvg seed` (safe) vs. `pvg seed --force` (reseed)

The `pvg seed` command takes a single optional flag:

```text
pvg seed             # safe — skip notes that already exist
pvg seed --force     # force — run 3-way merge against the baseline
```

The dispatch is at `pvg/cmd/pvg/main.go:116` (`case "seed":`). The boolean
`force` flag is threaded through `Seed` and ultimately into `writeNote`.

### Safe mode (`pvg seed`)

`writeNote` checks `os.Stat` on the destination. If the file exists, it logs
`SKIP: <relPath> (already exists)` and increments `counters.Skipped`. New notes
are still created, so safe mode is the right choice on first install or after
adding a new seeded note.

### Force mode (`pvg seed --force`)

When the file exists and `force == true`, `writeNote` reads the baseline:

1. **No baseline file** → overwrite the note (`Updated++`) and create the baseline.
2. **Baseline matches current vault content** → user has not modified it; safe overwrite (`Updated++`); baseline refreshed.
3. **Baseline differs from current vault content** → run `Merge3`:
   - clean merge → write merged content, refresh baseline (`Merged++`).
   - conflicted merge → write merged content **with markers**, do **not** refresh baseline (`Conflicted++`).
   - merge tooling failure → log `WARN`, overwrite with plugin content (`Updated++`), refresh baseline.

> **Historical note:** older revisions of this document referred to `make seed`
> and `make reseed` Make targets. Those targets were removed in v1.30.1 (commit
> `2986bdf`, "eliminate all shell scripts, use pvg exclusively"). The canonical
> commands today are `pvg seed` and `pvg seed --force`. Any local wrapper
> aliases (e.g. `make reseed`) should call `pvg seed --force` directly.

---

## Update Flow: 7 Steps to Land a Seeded-Content Change

Whenever you change the rendered output of a seeded note — whether by editing
a Go string, an agent file, or `vault-knowledge/SKILL.md` — follow this exact
sequence:

1. **Edit the source.** Pick the right one based on the table above:
   - Agent prompt → `paivot-graph/agents/<slug>.md`
   - Behavioral convention or core concept → the matching `seedXxx` function in `pvg/internal/governance/seed.go`
   - Vault knowledge skill → `paivot-graph/skills/vault-knowledge/SKILL.md`

2. **Run the governance tests** (only required for changes that touch
   `seed.go`, `merge.go`, or `baseline.go`):

   ```bash
   cd pvg
   go test ./internal/governance/...
   ```

3. **Rebuild and reinstall pvg** (required for any seed-content change because
   the rendered text is baked into the binary):

   ```bash
   cd pvg
   make install
   ```

4. **Sync the new pvg binary into all paivot-graph plugin caches** so running
   Claude Code sessions pick it up without reinstalling:

   ```bash
   cd ../paivot-graph
   make sync-cache
   ```

5. **Reseed the vault** so the new content lands (and baselines stay current):

   ```bash
   pvg seed --force
   ```

   Inspect the summary line. `Conflicted: 0` is the usual goal; if you see
   conflicts, open the listed files in Obsidian, resolve the markers, then
   rerun `pvg seed --force` so the baseline catches up.

6. **Verify the rendered note** in the resolved vault directory:

   ```bash
   vlt vault="Claude" read file="<Vault Note Name>"
   ```

   For agent prompts, the file name is the column 1 value from the agent
   table above (without the `methodology/` prefix or `.md` suffix). For
   conventions/concepts, use the basename you see in `writeNote(... filepath.Join(...))`.

7. **Confirm `pvg doctor` stays green.** The doctor check inspects vault layout
   and seed integrity; failures here mean baselines or seeded notes are not in
   the expected shape.

   ```bash
   pvg doctor
   ```

---

## Troubleshooting

### A reseed produced conflict markers in a methodology note

Open the note in Obsidian. Resolve each `<<<<<<< / ======= / >>>>>>>` block
the same way you'd resolve a git merge conflict. Save the file, then run
`pvg seed --force` again — `Merge3` will see the user-resolved content versus
the baseline and the plugin content, and (assuming you actually resolved the
conflict) will produce a clean merge that increments `Merged++` and refreshes
the baseline.

### `pvg seed` says SKIP for a note I want to update

Safe mode never overwrites. Use `pvg seed --force`.

### `Merge3` keeps failing with "diff3 not found"

Install GNU diffutils (macOS: `brew install diffutils`; most Linux distros ship
it by default). The error message returned by `Merge3` is explicit about this.

### I edited a vault note directly and now it's been merged away

Remember: vault notes under `methodology/`, `conventions/`, and `concepts/`
are derived artifacts. Edit the source (Go string or agent file) and reseed.
Direct vault edits are tolerated by 3-way merge but they are not the system
of record.

### A new seeded note didn't appear

Check the source-of-truth table above. If you added a new `seedXxx` function,
also wire it into the `Seed` function around `seed.go:108-123` so it runs.
Without that call, the function is dead code and produces nothing.

---

## Related

- `paivot-graph/docs/LIVE_SOR.md` — the broader "live system of record" model
  for how vault content evolves.
- `paivot-graph/docs/D_AND_F_GUARD_RAILS.md` — how the dispatcher enforces D&F
  artifact protection (related guard, separate machinery).
- `pvg/internal/governance/seed.go` — the canonical implementation.
- `pvg/internal/governance/merge.go` — the 3-way merge wrapper.
- `pvg/internal/governance/baseline.go` — baseline read/write helpers.
