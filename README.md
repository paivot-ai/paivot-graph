# paivot-graph

A [Claude Code](https://docs.anthropic.com/en/docs/build-with-claude/claude-code) plugin that turns an Obsidian vault into a living runtime for AI agents. Agents read their instructions from the vault, capture knowledge as they work, and refine their own prompts based on experience. The vault evolves with every session.

## Prerequisites

### 1. vlt (required)

**You must install [vlt](https://github.com/paivot-ai/vlt) before using this plugin.** vlt is the fast, standalone CLI that all hooks, commands, and agents use to interact with your Obsidian vault.

```bash
# From source (requires Go 1.24+)
git clone https://github.com/paivot-ai/vlt.git
cd vlt
make install

# Verify
vlt version   # should print vlt 0.9.0+
```

Pre-built binaries are available at [vlt releases](https://github.com/paivot-ai/vlt/releases) if you don't have Go installed.

Without vlt, the plugin falls back to direct filesystem operations (grep, cat) which are slower, lack vault-aware features (alias resolution, wikilink repair, backlink tracking), and miss the inert zone masking that prevents false positives.

### 2. nd (recommended for execution)

**[nd](https://github.com/paivot-ai/nd) is the issue tracker Paivot uses for execution.** The on-disk format is git-native markdown, but for multi-branch execution the live backlog should be branch-independent rather than copied into each story branch checkout. See [docs/LIVE_SOR.md](docs/LIVE_SOR.md).

```bash
# From source (requires Go 1.22+)
git clone https://github.com/paivot-ai/nd.git
cd nd
make build
make install    # Installs to ~/.local/bin/nd

# Verify
nd --help
```

Pre-built binaries are available at [nd releases](https://github.com/paivot-ai/nd/releases).

Without nd, the vault-knowledge and vault-lifecycle features still work (hooks, commands, skills), but the execution agents (developer, PM, Sr PM, anchor, retro) cannot manage work items.

### 3. pvg (required)

**[pvg](https://github.com/paivot-ai/pvg) is the shared control plane Paivot uses for guardrails, live nd routing, loop recovery, and story helpers.** `paivot-graph` shells out to the installed `pvg` binary; `make install` checks that it is already on your `PATH`.

```bash
# Pre-built binaries
gh release download -R paivot-ai/pvg -p '*darwin*arm64*' -D /tmp
tar xzf /tmp/pvg_*.tar.gz -C ~/go/bin

# Or from source (requires Go 1.24+)
git clone https://github.com/paivot-ai/pvg.git
cd pvg
make install

# Verify
pvg version
```

### 4. Obsidian vault

You need an Obsidian vault that vlt can discover. The plugin expects a vault named "Claude" by default. If you don't have one:

1. Open Obsidian and create a new vault named "Claude"
2. Verify vlt can see it: `vlt vaults`

### 5. Claude Code

This is a Claude Code plugin. You need [Claude Code](https://docs.anthropic.com/en/docs/build-with-claude/claude-code) installed.

## Installation

```bash
git clone https://github.com/paivot-ai/paivot-graph.git
cd paivot-graph
make install
```

This does four things:

1. Checks that vlt and Claude Code are installed
2. Verifies that `pvg` is on `PATH`
3. Fetches the [vlt skill](https://github.com/paivot-ai/vlt) from GitHub and installs it to `~/.claude/skills/vlt-skill` (teaches Claude how to use vlt effectively)
4. Registers the plugin with Claude Code's marketplace and installs it

Restart any open Claude Code sessions for hooks to take effect.

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

### Seed the vault (first time)

Populate the Obsidian vault with the methodology notes, behavioral conventions, and skill content that agents read at runtime:

```bash
make seed
```

This is idempotent -- it creates missing notes and skips existing ones. Use `make reseed` to force-update all notes with the latest plugin content.

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
| `/vault-capture` | Deliberate knowledge capture pass -- routes knowledge to the global vault or project vault based on scope |
| `/vault-evolve` | Identifies improvements to vault content; creates proposals for system notes, edits project notes directly |
| `/vault-triage` | Reviews and accepts/rejects pending proposals for system-scoped vault notes |
| `/vault-settings` | View and configure project-level settings (scope defaults, proposal expiry, git tracking) |
| `/vault-status` | Shows vault health -- note counts, project vault status, pending proposals |
| `/intake` | Collects user feedback and delegates to the Sr. PM agent to create a prioritized story backlog |

### Agents

Eight specialized agents that read their full instructions from the vault at runtime:

| Agent | Role |
|-------|------|
| **business-analyst** | Discovery & framing -- asks clarifying questions until requirements are solid |
| **architect** | System architecture, technical feasibility, ARCHITECTURE.md |
| **designer** | UX/API/CLI design for any product type, DESIGN.md |
| **sr-pm** | Creates comprehensive backlogs from D&F documents |
| **anchor** | Adversarial review of backlogs and milestones |
| **developer** | Ephemeral -- implements one story with proof of passing tests |
| **pm** | Ephemeral -- accepts or rejects delivered stories using evidence-based review |
| **retro** | Ephemeral -- extracts learnings from completed epics |

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

## Knowledge governance

Knowledge lives in three tiers with different governance rules:

| Tier | Location | Scope | How changes are made |
|------|----------|-------|---------------------|
| **System** | Global Obsidian vault ("Claude") | All projects | Proposal workflow: `/vault-evolve` creates proposals, `/vault-triage` reviews them |
| **Project** | `.vault/knowledge/` in each repo | One project | Via `vlt` commands only |
| **Session** | `~/.claude/projects/*/memory/` | One session | Ephemeral, managed by Claude Code |

The nd live backlog is a separate execution concern from `.vault/knowledge/`.
For concurrent multi-branch work, keep the mutable nd vault outside branch
checkouts and share it across worktrees. See [docs/LIVE_SOR.md](docs/LIVE_SOR.md).

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
make test              # run all checks (shellcheck + functional)
make lint              # shellcheck on all scripts
make check-deps        # verify vlt and claude are installed
make fetch-vlt-skill   # fetch vlt skill from GitHub (skips if present)
make update-vlt-skill  # force-update vlt skill from GitHub
make seed              # seed vault (idempotent)
make reseed            # force-update vault notes
make install           # check deps + fetch vlt skill + install plugin
make update            # push local changes to installed plugin
make uninstall         # remove plugin
```

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

## License

Apache License 2.0. See [LICENSE](LICENSE) for full text.
