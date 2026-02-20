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
vlt version   # should print vlt 0.5.0+
```

Without vlt, the plugin falls back to direct filesystem operations (grep, cat) which are slower, lack vault-aware features (alias resolution, wikilink repair, backlink tracking), and miss the inert zone masking that prevents false positives.

### 2. Obsidian vault

You need an Obsidian vault that vlt can discover. The plugin expects a vault named "Claude" by default. If you don't have one:

1. Open Obsidian and create a new vault named "Claude"
2. Verify vlt can see it: `vlt vaults`

### 3. Claude Code

This is a Claude Code plugin. You need [Claude Code](https://docs.anthropic.com/en/docs/build-with-claude/claude-code) installed.

## Installation

```bash
git clone https://github.com/RamXX/paivot-graph.git
cd paivot-graph
make install
```

This registers the plugin with Claude Code's marketplace and installs it. Restart any open Claude Code sessions for hooks to take effect.

### Seed the vault (first time)

Populate the Obsidian vault with the methodology notes, behavioral conventions, and skill content that agents read at runtime:

```bash
make seed
```

This is idempotent -- it creates missing notes and skips existing ones. Use `make reseed` to force-update all notes with the latest plugin content.

## What it does

### Hooks (automatic)

Four lifecycle hooks fire automatically during Claude Code sessions:

| Hook | When | What it does |
|------|------|--------------|
| **SessionStart** | Session begins | Searches the vault for project context, loads the operating mode |
| **PreCompact** | Before context compaction | Reminds Claude to capture decisions, patterns, and debug insights before memory is lost |
| **Stop** | When Claude tries to stop | Soft reminder to check the knowledge capture checklist |
| **SessionEnd** | Session ends | Appends a session log entry to the project's vault note |

### Commands (user-invoked)

| Command | What it does |
|---------|--------------|
| `/vault-capture` | Deliberate knowledge capture pass -- reviews the session and saves decisions, patterns, and debug insights to the vault |
| `/vault-evolve` | Refines vault-backed content (agent prompts, skill content, operating mode) based on session experience |
| `/vault-status` | Shows vault health -- note counts by folder, orphans, broken links, inbox triage needs |
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

### Skills

| Skill | What it does |
|-------|--------------|
| **vault-knowledge** | Teaches agents how to interact with the vault -- when to capture, what to capture, how to format notes |

## How the vault-as-runtime works

Traditional plugins ship static prompts. paivot-graph ships vault loaders -- thin stubs that point agents to vault notes for their instructions. This means:

1. **Agent prompts live in the vault**, not in plugin files. Edit them with Obsidian or vlt, and the next session picks up changes automatically.
2. **Knowledge compounds** across sessions. Every decision, pattern, and debug insight captured during work is available to future sessions.
3. **The feedback loop closes**. `/vault-evolve` lets agents refine their own instructions based on what worked and what didn't.

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
make help       # show all targets
make test       # run all checks (shellcheck + functional)
make lint       # shellcheck on all scripts
make seed       # seed vault (idempotent)
make reseed     # force-update vault notes
make install    # register and install plugin
make update     # push local changes to installed plugin
make uninstall  # remove plugin
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
