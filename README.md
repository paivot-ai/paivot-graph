# paivot-graph

A [Claude Code](https://docs.anthropic.com/en/docs/build-with-claude/claude-code) plugin that turns an Obsidian vault into a living runtime for AI agents. Agents read their instructions from the vault, capture knowledge as they work, and refine their own prompts based on experience. The vault evolves with every session.

## Prerequisites

### 1. vlt (required)

**You must install [vlt](https://github.com/RamXX/vlt) before using this plugin.** vlt is the fast, standalone CLI that all hooks, commands, and agents use to interact with your Obsidian vault.

```bash
# From source (requires Go 1.24+)
git clone https://github.com/RamXX/vlt.git
cd vlt
make install

# Verify
vlt version   # should print vlt 0.8.0+
```

Pre-built binaries are available at [vlt releases](https://github.com/RamXX/vlt/releases) if you don't have Go installed.

Without vlt, the plugin falls back to direct filesystem operations (grep, cat) which are slower, lack vault-aware features (alias resolution, wikilink repair, backlink tracking), and miss the inert zone masking that prevents false positives.

### 2. nd (recommended for execution)

**[nd](https://github.com/RamXX/nd) is the git-native issue tracker** that agents use to manage backlogs, stories, dependencies, and execution paths during development. If you plan to use the Paivot execution workflow (developer, PM-Acceptor, Sr PM, anchor, retro agents), nd is required.

```bash
# From source (requires Go 1.22+)
git clone https://github.com/RamXX/nd.git
cd nd
make build
make install    # Installs to ~/.local/bin/nd

# Verify
nd --help
```

Pre-built binaries are available at [nd releases](https://github.com/RamXX/nd/releases).

Without nd, the vault-knowledge and vault-lifecycle features still work (hooks, commands, skills), but the execution agents (developer, PM, Sr PM, anchor, retro) cannot manage work items.

### 3. Obsidian vault

You need an Obsidian vault that vlt can discover. The plugin expects a vault named "Claude" by default. If you don't have one:

1. Open Obsidian and create a new vault named "Claude"
2. Verify vlt can see it: `vlt vaults`

### 4. Claude Code

This is a Claude Code plugin. You need [Claude Code](https://docs.anthropic.com/en/docs/build-with-claude/claude-code) installed.

## Installation

```bash
git clone https://github.com/RamXX/paivot-graph.git
cd paivot-graph
make install
```

This does three things:

1. Checks that vlt and Claude Code are installed
2. Fetches the [vlt skill](https://github.com/RamXX/vlt) from GitHub and installs it to `~/.claude/skills/vlt-skill` (teaches Claude how to use vlt effectively)
3. Registers the plugin with Claude Code's marketplace and installs it

Restart any open Claude Code sessions for hooks to take effect.

### Seed the vault (first time)

Populate the Obsidian vault with the methodology notes, behavioral conventions, and skill content that agents read at runtime:

```bash
make seed
```

This is idempotent -- it creates missing notes and skips existing ones. Use `make reseed` to force-update all notes with the latest plugin content.

## What it does

### Hooks (automatic)

Five hooks fire automatically during Claude Code sessions:

| Hook | Event | What it does |
|------|-------|--------------|
| **Scope Guard** | PreToolUse (Edit/Write/Bash) | Blocks direct writes to system vault and project vault -- enforces vlt-only access |
| **Session Start** | SessionStart | Searches the vault for project context, loads project-local knowledge, loads operating mode |
| **Pre-Compact** | PreCompact | Reminds Claude to capture decisions, patterns, and debug insights before memory is lost |
| **Stop** | Stop | Soft reminder to check the knowledge capture checklist |
| **Session End** | SessionEnd | Appends a session log entry to the project's vault note |

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

### Hard-TDD mode (optional)

For stories where correctness is critical, add the `hard-tdd` label. This activates a two-phase developer workflow:

1. **RED phase** -- A Test Author developer writes tests only (unit + integration). PM-Acceptor reviews tests against acceptance criteria: "If these tests passed, would they prove the story is done?"
2. **GREEN phase** -- A separate Implementer developer writes code to make all tests pass. The Implementer cannot modify test files -- git enforces this structurally via commit diffing.

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
| **vlt-skill** | Complete vlt command reference, agentic patterns, and advanced techniques (fetched from GitHub at install time) |

## Knowledge governance

Knowledge lives in three tiers with different governance rules:

| Tier | Location | Scope | How changes are made |
|------|----------|-------|---------------------|
| **System** | Global Obsidian vault ("Claude") | All projects | Proposal workflow: `/vault-evolve` creates proposals, `/vault-triage` reviews them |
| **Project** | `.vault/knowledge/` in each repo | One project | Via `vlt` commands only (locking enforced) |
| **Session** | `~/.claude/projects/*/memory/` | One session | Ephemeral, managed by Claude Code |

### Why vlt-only access matters

When dozens of agents run concurrently -- developers implementing stories, PM-Acceptors reviewing deliveries, the retro agent extracting learnings -- they all read and write vault notes. vlt uses advisory file locking (`.vlt.lock`) to serialize these operations. Direct file I/O (Edit, Write, `cat >`) bypasses that lock, creating race conditions where one agent's write silently overwrites another's.

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
- **Bash with vlt** -- always allowed (vlt is the intended mechanism, provides locking)

### Graph-aware retrieval with vlt

Vault notes link to each other extensively via `[[wikilinks]]`. vlt 0.8.0 added `follow` and `backlinks` flags to the `read` command, enabling agents to retrieve a note's entire link neighborhood in a single call:

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
