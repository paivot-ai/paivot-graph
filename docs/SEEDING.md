# Vault Seeding: What `pvg seed` Deploys

## Overview

`pvg seed` writes a set of notes into the system vault (`Claude`) under an
exclusive vlt lock (safe to run while a session is active). It does NOT control
how agents load their instructions: since v1.53.0, agent prompts are
**self-contained in `agents/*.md`** and are read directly by Claude Code when an
agent is spawned. There is no `seed/` directory and no vault-loader fallback.

```bash
pvg seed           # safe mode: create missing notes, skip existing ones
pvg seed --force   # overwrite existing notes with current plugin content
```

The plugin root is resolved from `CLAUDE_PLUGIN_ROOT`, the pvg binary location,
or the newest entry in the Claude Code plugin cache.

## What Gets Seeded

### 1. Behavioral notes (consumed at runtime)

These are the operationally important ones -- the session hooks point the
dispatcher at them:

| Vault note | Purpose |
|---|---|
| `conventions/Session Operating Mode.md` | Concurrency limits, dispatcher mode rules, D&F orchestration, loop priorities |
| `conventions/Pre-Compact Checklist.md` | What to capture (decisions, patterns, debug insights) before context compaction |
| `conventions/Stop Capture Checklist.md` | End-of-session knowledge-capture confirmation list |

### 2. Skill content

`skills/vault-knowledge/SKILL.md` is seeded to
`conventions/Vault Knowledge Skill.md` so the methodology is browsable in
Obsidian.

### 3. Agent prompt reference copies

Each `agents/<slug>.md` prompt (sr-pm, pm, developer, architect, designer,
business-analyst, anchor, retro, and the three challengers) is copied to
`methodology/<Name> Agent.md` with frontmatter normalized to a methodology
note. These are **reference copies for humans browsing the vault** -- agents do
NOT read them at runtime. The files under `agents/` are the single source of
truth; the vault copies go stale between seeds, which is why upgrades should be
followed by `pvg seed --force`.

## Idempotence

- Note missing: created
- Note exists: skipped (safe mode) or overwritten (`--force`)

## When to Re-Seed

After upgrading the plugin, run `pvg seed --force` so the behavioral notes and
reference copies match the installed version.

---

## Related

- [[Session Operating Mode]] -- Dispatcher orchestration (seeded note)
- [[Vault Knowledge Skill]] -- How to interact with the vault from agents
