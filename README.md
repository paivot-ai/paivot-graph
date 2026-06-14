# paivot-graph

A [Claude Code](https://docs.anthropic.com/en/docs/build-with-claude/claude-code) plugin that turns an Obsidian vault into a living runtime for AI agents. Agents read their instructions from the vault, capture knowledge as they work, and refine their own prompts based on experience. The vault evolves with every session.

## Installation

One command installs everything:

```bash
curl -fsSL https://raw.githubusercontent.com/paivot-ai/pvg/main/install.sh | sh
```

The installer reads the stable channel manifest ([channel/stable.json](channel/stable.json)) published by this repository and converges your machine onto that pinned, CI-verified combination:

| Component | What it is |
|-----------|------------|
| **[pvg](https://github.com/paivot-ai/pvg)** | The shared control plane for guardrails, live nd routing, loop recovery, story helpers, and updates. All hooks shell out to it. |
| **[vlt](https://github.com/paivot-ai/vlt)** | The fast, standalone CLI that all hooks, commands, and agents use to interact with your Obsidian vault. Without it, agents fall back to grep/cat -- slower, no alias resolution, no concurrent-access locking. |
| **[nd](https://github.com/paivot-ai/nd)** | The issue tracker Paivot uses for execution -- git-native markdown work items. For multi-branch execution see [docs/LIVE_SOR.md](docs/LIVE_SOR.md). |
| **paivot-graph plugin** | This plugin, installed from the GitHub-source marketplace `paivot-ai/paivot-graph`. |
| **nd plugin** | The nd skill and guard hooks, installed from the GitHub-source marketplace `paivot-ai/nd`. |
| **vlt skill** | Complete vlt command reference and agentic patterns, installed to `~/.claude/skills/vlt-skill`. |

Marketplaces are GitHub sources -- no repository clones are needed on consumer machines. Restart any open Claude Code sessions for hooks to take effect.

### Updating

```bash
pvg update
```

`pvg update` re-reads the stable channel at `main` and converges tools, plugins, and skills onto the pinned combo. Updates are **pin + nudge, never silent**: pvg notices when the channel has moved past your installed combo and tells you, but nothing changes until you run `pvg update` yourself.

| Command | What it does |
|---------|--------------|
| `pvg update` | Converge onto the current stable channel (`main` of this repo) |
| `pvg update --to <git-ref>` | Converge onto the channel as of any git ref of this repo (commit SHA, tag, or branch). This is also the rollback mechanism -- every combo that was ever on `main` passed the same CI smoke test. |
| `pvg update --pin <git-ref>` | Hold the machine at a specific combo and suppress update nudges |

See [channel/README.md](channel/README.md) for how the channel works and how a combo gets stamped.

### Development install

For working on the plugin itself, install from a clone:

```bash
git clone https://github.com/paivot-ai/paivot-graph.git
cd paivot-graph
make install
```

This checks that vlt, pvg, and Claude Code are installed, fetches companion tools and skills (`make fetch-tools` -- the development-clone equivalent of what `pvg update` does for consumers), registers this checkout as a local marketplace, and installs the plugin from it.

## Prerequisites

### 1. Claude Code

This is a Claude Code plugin. You need [Claude Code](https://docs.anthropic.com/en/docs/build-with-claude/claude-code) installed.

### 2. Obsidian vault

You need an Obsidian vault that vlt can discover. The plugin expects a vault named "Claude" by default. If you don't have one:

1. Open Obsidian and create a new vault named "Claude"
2. Verify vlt can see it: `vlt vaults`

### 3. Codebase indexing MCP server (strongly recommended)

**A codebase indexing MCP server dramatically improves story quality.** When available, Paivot agents use it for API signature verification, cross-cutting concern discovery, and module count validation instead of grep. This prevents the most common class of Anchor rejections: hallucinated API signatures.

Any MCP server that provides `search_graph`, `get_code_snippet`, and `trace_call_path` works. Two tested options:

- **[codebase-memory-mcp](https://github.com/nicobailon/codebase-memory-mcp)** -- Graph-based indexing with Cypher queries, call path tracing, and architecture summaries
- **[Augment Code](https://www.augmentcode.com/)** (cx) -- Commercial codebase intelligence with similar capabilities

Install via `.mcp.json` in your project or `~/.claude/settings.json`. After indexing, agents automatically prefer MCP tools over grep for codebase queries.

Without a codebase indexing server, agents fall back to grep/ripgrep. This works but is slower, less precise on call graph analysis, and cannot verify module counts as reliably.

### 4. Toolchain containers (if your build runs in a container)

If your project's lint/test toolchain runs inside a container (for example
Elixir/`mix` inside `docker compose`), `pvg` and `nd` must be on `PATH` *inside*
the container too -- vault-backed lints shell out to `pvg nd list --json`, and a
host-only install dies with `:enoent` in the container. Either install the
(static Go) binaries into the toolchain image at build time, or read-only
bind-mount the host binaries when host and container share the same
architecture. See [docs/CONTAINER_TOOLCHAIN.md](docs/CONTAINER_TOOLCHAIN.md) for
the two supported wirings, the architecture caveat, and the `.git/`-in-mount +
committed `.vault/.nd-shared.yaml` vault-resolution requirements.

## If something goes wrong

If a session gets into a bad state, use the smallest escape hatch that solves the problem:

| Situation | What to run | What it does |
|-----------|-------------|--------------|
| You want Claude to stop acting as coordinator-only | `pvg dispatcher off` | Disables dispatcher mode for the current repo |
| An execution loop should stop immediately | `pvg loop cancel` | Cancels the active loop without touching your backlog or vault |
| Claude lost context or a session was interrupted mid-loop | `pvg loop recover` | Rebuilds loop state from git and nd instead of guessing |
| You want the plugin completely out of the way | `make uninstall` | Removes the Claude Code plugin from this checkout |

Practical advice:

- Start with `pvg dispatcher off` or `pvg loop cancel`. Those are the normal operational escape hatches.
- Use `pvg loop recover` after compaction, interruption, or any situation where the loop state looks stale. Do not hand-edit loop state files.
- If hooks are still behaving unexpectedly, uninstall the plugin, restart Claude Code, and reinstall cleanly.
- Your nd backlog and Obsidian vault remain on disk. Turning dispatcher mode off, cancelling a loop, or uninstalling the plugin does not delete your work.

## What it does

### Hooks (automatic)

Multiple hook handlers fire automatically during Claude Code sessions:

| Hook | Event | What it does |
|------|-------|--------------|
| **Scope Guard** | PreToolUse (Edit/Write/Bash) | Blocks direct writes to system vault and project vault -- enforces vlt-only access |
| **Session Start** | SessionStart | Searches the vault for project context, loads project-local knowledge, loads operating mode |
| **Pre-Compact** | PreCompact | Reminds Claude to capture decisions, patterns, and debug insights before memory is lost |
| **Stop** | Stop | Soft reminder to check the knowledge capture checklist |
| **Session End** | SessionEnd | Appends a session log entry to the project's vault note |

Additional hook handlers are also registered for dispatcher tracking, user-prompt detection, and memory-tool interception; see `hooks/hooks.json`. Dispatcher tracking covers both D&F agents (BA, Designer, Architect) and execution agents (Sr PM, Developer, PM) so guarded nd mutations are only permitted from the responsible agent worktree.

### Commands (user-invoked)

| Command | What it does |
|---------|--------------|
| `/piv-loop` | Unattended execution loop -- drains ready stories through developer + PM-Acceptor cycles, handles bugs, and gates epic closure on E2E + Anchor review |
| `/piv-cancel-loop` | Cancels an active execution loop without touching the backlog or vault |
| `/vault-capture` | Deliberate knowledge capture pass -- routes knowledge to the global vault or project vault based on scope |
| `/vault-evolve` | Identifies improvements to vault content; creates proposals for system notes, edits project notes directly |
| `/vault-triage` | Reviews and accepts/rejects pending proposals for system-scoped vault notes |
| `/vault-settings` | View and configure project-level settings (scope defaults, proposal expiry, git tracking) |
| `/vault-status` | Shows vault health -- note counts, project vault status, pending proposals |
| `/intake` | Collects user feedback and delegates to the Sr. PM agent to create a prioritized story backlog |

### Agents

Eleven specialized agents that read their full instructions from the vault at runtime:

| Agent | Role |
|-------|------|
| **business-analyst** | Discovery & framing -- asks clarifying questions until requirements are solid |
| **architect** | System architecture, technical feasibility, ARCHITECTURE.md |
| **designer** | UX/API/CLI design for any product type, DESIGN.md |
| **ba-challenger** | Adversarial review of BUSINESS.md (opt-in via `dnf.specialist_review`) |
| **designer-challenger** | Adversarial review of DESIGN.md (opt-in via `dnf.specialist_review`) |
| **architect-challenger** | Adversarial review of ARCHITECTURE.md (opt-in via `dnf.specialist_review`) |
| **sr-pm** | Creates comprehensive backlogs from D&F documents |
| **anchor** | Adversarial review of backlogs and milestones |
| **developer** | Ephemeral -- implements one story with proof of passing tests |
| **pm** | Ephemeral -- accepts or rejects delivered stories using evidence-based review |
| **retro** | Ephemeral -- extracts learnings from completed epics |

### Model allocation

The plugin assigns models to balance cost and capability:

| Agent | Model | Rationale |
|-------|-------|-----------|
| Dispatcher (`/piv-loop`) | Opus | Worktree lifecycle, CWD safety, and context injection require strong reasoning. Sonnet-class models drift on multi-step git orchestration. |
| Developer | Opus | Code generation requires maximum reasoning. |
| PM-Acceptor | Opus | Quality gate -- false acceptance is the most expensive failure mode. |
| Sr PM | Opus | Story creation needs domain reasoning and precise terminology. |
| Anchor | Opus | Adversarial review needs strongest reasoning to find gaps. |
| BA / Designer / Architect | Opus | D&F requires deep domain reasoning. |
| Challengers (BA/Designer/Architect) | Sonnet | Scoped adversarial critique against a single document. |
| Retro | Sonnet | Pattern extraction from completed work. |

Rate limits are per-model. Challengers and Retro run on Sonnet to preserve Opus headroom for the judgment-heavy agents.

**Per-role model override.** The models above are defaults baked into each
agent's `agents/*.md` frontmatter. You can override any role per project with
`pvg settings model.<role>=<model>` -- no file edits, and the override survives
plugin updates. The dispatcher reads the setting and passes the chosen model at
spawn time; the choice affects only which model runs each agent, never the
structural story/epic gates below. See
[commands/vault-settings.md](commands/vault-settings.md) for the full list of
roles and accepted values.

### Execution workflow

The execution loop (`/piv-loop`) drives stories through development, review, and delivery. Two structural gates enforce quality:

**Story gate:** Every story must have passing integration tests with no mocks before the PM-Acceptor will accept it. Tests gated behind env vars or skipped tests are rejected on sight.

**Epic gate:** After all stories in an epic are accepted and merged to the epic branch, three steps run before the epic reaches main:

1. **E2e verification** -- the full test suite (unit + integration + e2e) runs on the merged epic branch. No epic is done without passing e2e tests.
2. **Anchor milestone review** -- the Anchor agent validates real delivery: no mocks in integration tests, boundary maps satisfied, skills consulted.
3. **Merge to main** -- depends on `workflow.solo_dev` setting:
   - `true` (default): merge directly to main, push, delete epic and story branches
   - `false`: create a PR for team review

Configure with: `pvg settings workflow.solo_dev=false` for team workflows.

### Quality gates

Beyond the structural story/epic gates, the PM-Acceptor runs `pvg gates` on the
delivered diff in Tier 1 of its review -- a deterministic, metric-based gate on
delivered code. It measures **copy-paste duplication**, **cyclomatic
complexity**, and **file size (LOC)** against tunable `gates.*` thresholds and
emits `[BLOCK]`/`[WARN]`/`[SKIP]` lines with a PASS/FAIL summary: a `[BLOCK]`
finding rejects the story, `[WARN]` is noted, and `[SKIP]` (analyzer absent) is
never a failure. Complexity and duplication shell out to external analyzers, so
installing `lizard` (`pip install lizard`) and `jscpd` (`npm install -g jscpd`)
lights up the full gate on virtually any stack -- **apt alone is not enough;
only `radon` ships in the Ubuntu repos.**

See [docs/QUALITY_GATES.md](docs/QUALITY_GATES.md) for the analyzer matrix,
install instructions, default thresholds, the full `gates.*` key reference, and
example output.

### Hard-TDD mode (optional)

For stories where correctness is critical, add the `hard-tdd` label. This activates a two-phase developer workflow:

1. **RED phase** -- A Test Author developer writes tests only (unit + integration). PM-Acceptor reviews tests against acceptance criteria: "If these tests passed, would they prove the story is done?"
2. **GREEN phase** -- A separate Implementer developer writes code to make all tests pass. The Implementer cannot modify test files -- git enforces this structurally via commit diffing.

The active phase is conveyed by the dispatcher prompt (`RED PHASE` / `GREEN PHASE`). No extra nd labels are required beyond `hard-tdd`.

The Sr PM applies the label during backlog creation (user can steer: "hard-tdd on all payment stories"). The label persists on the story as a permanent record, read by every agent:

| Agent | How it uses `hard-tdd` |
|-------|----------------------|
| **sr-pm** | Applies the label (user-directed or by judgment for high-risk stories) |
| **developer** | Reads RED/GREEN phase from prompt, adjusts behavior |
| **pm** | Adjusts review lens per phase (test quality vs implementation correctness) |
| **anchor** | Validates two-commit pattern (test commit before implementation commit) |
| **retro** | Compares outcomes between hard-tdd and normal stories |

No new agent types, hooks, or nd states. The mechanism is dispatcher orchestration + git enforcement.

### Skills

| Skill | What it does |
|-------|--------------|
| **vault-knowledge** | Teaches agents how to interact with the vault -- when to capture, what to capture, how to format notes |
| **c4** | Optional architecture-as-code skill for `workspace.dsl`, diagram exports, and Architecture Contract maintenance when `architecture.c4` is enabled |
| **vlt-skill** | Complete vlt command reference, agentic patterns, and advanced techniques (fetched from GitHub at install time) |

### Provider abstraction (`pvg issues`, `pvg notes`)

Agent prompts no longer call `nd` and `vlt` directly. They use `pvg issues` for backlog operations and `pvg notes` for knowledge-base operations -- a provider-abstracted layer that defaults to nd + vlt but can be reconfigured per project via `.paivot/config.yaml`:

```yaml
backlog:
  primary:
    adapter: nd            # default; or `linear`
notes:
  primary:
    adapter: vlt           # default; placeholders for confluence, jira, notion
```

Reads always go to the primary adapter. Writes go to the primary first and then fan out best-effort to optional mirrors (useful for shadowing into Linear for visibility while keeping nd as the source of truth). Backend-specific operations that have no clean cross-backend abstraction -- nd dependency cycles, `vlt read --follow` graph traversal, heading-anchored `vlt patch` -- remain available via `pvg nd ...` and direct `vlt ...` calls. See [pvg's README](https://github.com/paivot-ai/pvg#provider-configuration) for the full schema.

## Knowledge governance

Knowledge lives in three tiers with different governance rules:

| Tier | Location | Scope | How changes are made |
|------|----------|-------|---------------------|
| **System** | Global Obsidian vault ("Claude") | All projects | Proposal workflow: `/vault-evolve` creates proposals, `/vault-triage` reviews them |
| **Project** | `.vault/knowledge/` in each repo | One project | Via `pvg notes` (or `vlt` for backend-specific operations) |
| **Session** | `~/.claude/projects/*/memory/` | One session | Ephemeral, managed by Claude Code |

The nd live backlog is a separate execution concern from `.vault/knowledge/`.
For concurrent multi-branch work, keep the mutable nd vault outside branch
checkouts and share it across worktrees. See [docs/LIVE_SOR.md](docs/LIVE_SOR.md).

### Convention: Paivot projects do not use a project-level `CLAUDE.md`

A Paivot-managed project (any directory containing `.vault/issues/` or
`.paivot/config.yaml`) deliberately has **no** project-level `CLAUDE.md`. The
project vault and the agent prompts are the single source of truth -- a parallel
`CLAUDE.md` creates two competing sources, drift, and rule duplication.

If you want to record a project-specific hard rule (e.g., "no skip-if-missing
integration tests", "all migrations must be reversible"), write it as a
`scope: project` note under `.vault/knowledge/conventions/`. The Sr PM's
Phase 1 hard-rule ingestion reads those notes automatically (alongside your
user global `~/.claude/CLAUDE.md`) and registers them in the
`lint.quality_gates` setting enforced by `pvg lint --backlog`.

Recommended one-liner to add to your user global `~/.claude/CLAUDE.md` so any
session understands this convention:

> **Paivot project detection.** If the working directory or any ancestor
> contains `.vault/issues/` or `.paivot/config.yaml`, treat it as a
> Paivot-managed project: do not create or expect a project-level
> `CLAUDE.md`. Project-specific conventions live under
> `.vault/knowledge/conventions/`; methodology lives in the Paivot vault;
> workflow is governed by the agent prompts in the `paivot-graph` plugin
> (or `paivot-codex`). Hard rules that would normally live in a project
> `CLAUDE.md` belong as `scope: project` vault notes instead.

### Why vlt-only access matters

When dozens of agents run concurrently -- developers implementing stories, PM-Acceptors reviewing deliveries, the retro agent extracting learnings -- they all read and write vault notes. vlt write paths use advisory file locking (`.vlt.lock`), and pvg hook write paths acquire that lock explicitly before mirroring session state. Direct file I/O (Edit, Write, `cat >`) bypasses that protection, creating race conditions where one agent's write silently overwrites another's.

Advisory instructions ("please use vlt") don't work -- subagents routinely bypass them (see the vault note "Subagents do not follow advisory instructions"). The enforcement must be structural.

### How the scope guard works

The `pvg guard` binary runs as a PreToolUse hook on every Edit, Write, and Bash call. It enforces two layers of protection:

**Layer 1 -- System vault** (global Obsidian vault):

| Directory | Protected? | Rationale |
|-----------|-----------|-----------|
| `methodology/` | Yes | Agent prompts -- changes affect all projects |
| `conventions/` | Yes | Operating mode, checklists -- shared across projects |
| `decisions/` | Yes | Accepted proposals live here |
| `patterns/`, `debug/`, `concepts/`, `projects/`, `people/` | Yes | Curated knowledge |
| `_inbox/` | No | Where proposals and captures land before triage |
| `_templates/` | No | Templates are read-only by convention |

Changes to protected system directories require the proposal workflow: `/vault-evolve` creates a proposal in `_inbox/`, then `/vault-triage` presents it for human review.

**Layer 2 -- Project vault** (`.vault/knowledge/` in the repo):

All files under `.vault/knowledge/` are protected, with one exception: `.settings.yaml` is writable because it's managed by `pvg settings` (our own binary, not an agent).

The guard checks:
- **Edit/Write** -- blocks if `file_path` targets `.vault/knowledge/` (except `.settings.yaml`)
- **Bash** -- blocks shell write patterns (`>`, `>>`, `cat >`, `cp`, `mv`, `mkdir`) targeting `.vault/knowledge/`
- **Bash with vlt** -- always allowed (vlt is the intended mechanism; its write paths provide advisory locking)

### Graph-aware retrieval with vlt

Vault notes link to each other extensively via `[[wikilinks]]`. vlt added `follow` and `backlinks` flags to the `read` command, enabling agents to retrieve a note's entire link neighborhood in a single call:

```bash
# Read a project note + everything it links to (decisions, patterns, debug notes)
vlt vault="Claude" read file="paivot-graph" follow

# Read a note + everything that links TO it (what depends on this note?)
vlt vault="Claude" read file="Testing Philosophy" backlinks
```

Without these flags, an agent would need N+1 calls: read the note, parse links, read each linked note. With `follow`, it's one call. This is especially important for subagents, which tend to skip multi-step link traversal when given advisory instructions.

All vault commands (`/vault-capture`, `/vault-evolve`, `/vault-triage`, `/vault-status`, `/vault-settings`) use vlt exclusively for reads and writes. Edit and Write tools are not in their allowed-tools lists.

## How the vault-as-runtime works

Traditional plugins ship static prompts. paivot-graph ships vault loaders -- thin stubs that point agents to vault notes for their instructions. This means:

1. **Agent prompts live in the vault**, not in plugin files. Edit them with Obsidian or vlt, and the next session picks up changes automatically.
2. **Knowledge compounds** across sessions. Every decision, pattern, and debug insight captured during work is available to future sessions.
3. **The feedback loop closes**. `/vault-evolve` lets agents refine their own instructions based on what worked and what didn't.
4. **Access is structurally enforced**. The scope guard blocks direct file writes; all vault operations go through vlt, which provides concurrent-access locking. This is mechanism, not policy -- subagents can't bypass it.

The vault structure:

```
methodology/  -- Agent prompts (atomic concepts from the Paivot methodology)
conventions/  -- Working conventions (operating mode, checklists, skill content)
decisions/    -- Architectural and design decisions with rationale
patterns/     -- Reusable solutions and idioms
debug/        -- Problems and their resolutions
concepts/     -- Language, framework, and tool knowledge
projects/     -- One index note per project (session logs accumulate here)
people/       -- User preferences and team conventions
_inbox/       -- Unsorted capture, triage into proper folders
_templates/   -- Note templates
```

## Development

```bash
make help              # show all targets
make test              # run all checks (functional)
make check-deps        # verify vlt and claude are installed
make fetch-tools       # install/update vlt + nd binaries and their skills
make fetch-vlt-skill   # fetch vlt skill from GitHub (skips if present)
make update-vlt-skill  # force-update vlt skill from GitHub
make install           # check deps + fetch tools + install plugin
make update            # push local changes to installed plugin
make uninstall         # remove plugin
make bump v=X.Y.Z      # bump VERSION + plugin.json + marketplace.json atomically
make channel-check     # validate channel/stable.json against VERSION before pushing
```

Releases follow the channel discipline: bump and release this plugin first, then update [channel/stable.json](channel/stable.json) to pin the new version. `make channel-check` catches pin/VERSION skew locally; the `channel-verify` and `version-sync` CI workflows enforce it on push. See [docs/DISTRIBUTION.md](docs/DISTRIBUTION.md) for the full distribution design.

## Verifying the installation

After installing, start a new Claude Code session in any git repository. You should see vault context injected at startup:

```
[VAULT] Project: <your-project>
Relevant vault notes:
  ...
[VAULT] Operating mode for this session (from vault):
  ...
```

If you see `[VAULT] Vault directory not found` instead, check that your Obsidian vault exists and that `vlt vaults` lists it.

If you see no vault output at all, check that the plugin installed correctly: `claude plugin list` should show `paivot-graph`.

## Further reading

The README is the hub; the detail docs under [`docs/`](docs/) carry the depth on
a single topic each:

| Doc | What it covers |
|-----|----------------|
| [docs/QUALITY_GATES.md](docs/QUALITY_GATES.md) | `pvg gates` in full -- the analyzer matrix, install instructions, the complete `gates.*` key reference, and example output (see [Quality gates](#quality-gates)) |
| [docs/HARD_TDD_GUARD.md](docs/HARD_TDD_GUARD.md) | The CI structural lock for `hard-tdd` stories -- `pvg story verify-tdd` plus the `scripts/verify-hard-tdd.sh` wrapper, the RED/authorized marker rules, and robust range resolution |
| [docs/LIVE_SOR.md](docs/LIVE_SOR.md) | The live source-of-record: shared nd vault, snapshot-is-export, the dependency-edge lifecycle (`all_blocked_by`), and snapshot-drift (see [Knowledge governance](#knowledge-governance)) |
| [docs/DISTRIBUTION.md](docs/DISTRIBUTION.md) | The channel + one-command install design (see [Installation](#installation)) |
| [docs/PARALLEL_DEV_WORKTREES.md](docs/PARALLEL_DEV_WORKTREES.md) | Why code-writing developers get dispatcher-managed worktrees, and the required developer flow |
| [docs/CONTAINER_TOOLCHAIN.md](docs/CONTAINER_TOOLCHAIN.md) | Running Paivot when the build/lint/test toolchain lives in a container -- installing `pvg`/`nd` in the image vs. bind-mounting, the arch caveat, and in-container vault resolution (see [Toolchain containers](#4-toolchain-containers-if-your-build-runs-in-a-container)) |
| [docs/SEEDING.md](docs/SEEDING.md) | What `pvg seed` deploys into the system vault and how it relates to self-contained agent prompts |
| [docs/BUG_CREATION_EVOLUTION.md](docs/BUG_CREATION_EVOLUTION.md) | Background: the move from distributed bug creation to the centralized Sr PM model (and `bug_fast_track`) |
| [docs/D_AND_F_GUARD_RAILS.md](docs/D_AND_F_GUARD_RAILS.md) | Background: the move from per-document challengers to the Anchor review model (and `dnf.specialist_review`) |

Per-role agent model overrides and the full `gates.*` / `model.*` settings key
list live in [commands/vault-settings.md](commands/vault-settings.md).

## License

Apache License 2.0. See [LICENSE](LICENSE) for full text.
