---
name: vault-knowledge
description: This skill should be used when working on any project to understand how to effectively interact with the Obsidian knowledge vault. It teaches when to capture knowledge, what to capture, how to format vault notes, and how to search effectively. Use when you need to "save to vault", "update vault", "capture a decision", "record a pattern", "log a debug insight", or when starting/ending a significant work session.
version: 0.5.1
---

# Vault Knowledge

The Obsidian vault ("Claude") is your persistent knowledge layer. Interact with it using `vlt` (the fast vault CLI) via Bash. Prefer `vlt` for vault-aware operations (search, create, move with wikilink repair, backlinks, tags).

**Vault path:** Resolve dynamically with `vlt vault="Claude" dir` (never hardcode).

## Two-Tier Knowledge Model

### Tier 1: Global Vault ("Claude")

Shared across ALL projects. Cross-project knowledge only.

| Folder        | Contains                        | Auto-Tag           |
|---------------|---------------------------------|--------------------|
| methodology/  | Agent prompts, Paivot workflow  | `#dev-tools/workflow` |
| conventions/  | Operating mode, checklists      | `#dev-tools/workflow` |
| decisions/    | Cross-project decisions         | Domain-based       |
| patterns/     | Cross-project patterns          | Domain-based       |
| debug/        | Cross-project debug insights    | Stack-based        |
| concepts/     | Language/framework knowledge    | Stack-based        |
| projects/     | Project index notes (not logs!) | Project domain     |
| people/       | Team preferences                | `#dev-tools/workflow` |
| _inbox/       | Unsorted capture                | (triage required)  |

**Governance:** New captures can go through `/vault-capture`. Changes to protected system notes go through `/vault-evolve` (proposal) and `/vault-triage` (review/apply).

### Tier 2: Project Vault (`.vault/` in each repo)

Scoped to a single project. Changes apply directly.

```
.vault/
  knowledge/         # Project-local knowledge managed via vlt
    conventions/     # Project-specific conventions
    decisions/       # Project-specific decisions
    patterns/        # Project-specific patterns
    debug/           # Project-specific debug insights
    skills/          # Project-specific synthesized skills
    .settings.yaml   # Project settings managed by pvg
```

**Separation:** `knowledge/` is for durable project knowledge. Other `.vault/` paths are runtime state managed by `pvg`/`nd`, not a manual session-log area.

## Controlled Domain Vocabulary

Use these domain values in frontmatter. Prefer the granular `dev-tools-*` values for new captures; older seeded notes may still use the legacy values listed here.

| Domain | Description | Auto-Tag |
|--------|-------------|----------|
| `ai-training` | ML model training, fine-tuning, optimization | `#ai/training` |
| `ai-inference` | Model inference, reasoning, neuro-symbolic | `#ai/inference` |
| `ai-agents` | Agent orchestration, multi-agent systems | `#ai/agents` |
| `ai-nlp` | Natural language processing, NER, classification | `#ai/nlp` |
| `developer-tools` | Legacy seeded Paivot system notes; prefer `dev-tools-*` for new notes | `#dev-tools/workflow` |
| `dev-tools-cli` | CLI tools, build systems, package managers | `#dev-tools/cli` |
| `dev-tools-testing` | Testing frameworks, test patterns | `#dev-tools/testing` |
| `dev-tools-workflow` | Development workflow, methodology | `#dev-tools/workflow` |
| `dev-tools-knowledge` | Knowledge management, vault patterns | `#dev-tools/knowledge` |
| `product-management` | Product and backlog management practices | `#product/management` |
| `security-gateway` | API gateways, middleware, auth | `#security/gateway` |
| `security-hardening` | Security fixes, vulnerability remediation | `#security/hardening` |
| `security-compliance` | Regulatory compliance, audit | `#security/compliance` |
| `finance-quant` | Quantitative finance, backtesting | `#finance/quant` |
| `finance-fintech` | Fintech applications, payments | `#finance/fintech` |
| `frontend-ui` | UI components, design systems | `#frontend/ui` |
| `frontend-performance` | Web performance, optimization | `#frontend/performance` |
| `calendar-sync` | Calendar federation, scheduling | `#calendar/sync` |

## Auto-Tagging Rules

Tags are derived from folder + domain. Never manually add tags to frontmatter.

| Folder | Base Tag | Plus Domain Tag |
|--------|----------|-----------------|
| methodology/ | `#dev-tools/workflow` | - |
| conventions/ | `#dev-tools/workflow` | - |
| decisions/ | - | `#<domain>` (from domain field) |
| patterns/ | - | `#<domain>` (from domain field) |
| debug/ | `#dev-tools/cli` | `#<stack>` if relevant |
| concepts/ | - | `#ai/<subtype>` based on stack |
| projects/ | - | `#<domain>` (from domain field) |

Tags appear in the note body as a `## Tags` section at the bottom, not in frontmatter.

## Note Template

Every note MUST follow this structure:

```markdown
---
type: <pattern|debug|decision|concept|convention|methodology>
project: <project-name-or-general>
stack: [<list-of-technologies>]
domain: <from-controlled-vocabulary>
status: active
confidence: high|medium|low
created: YYYY-MM-DD
---

# Title

One-line summary of what this note captures.

## Context (optional)

Why this matters, when it applies.

## Content

Main body - the actual knowledge.

## Related

- [[Link to related note 1]]
- [[Link to related note 2]]

## Tags

#auto-derived-tag #another-tag
```

### Required Fields

- **type**: pattern, debug, decision, concept, convention, methodology
- **project**: Project name or "general" for cross-project
- **domain**: MUST be from controlled vocabulary above
- **status**: active, superseded, archived
- **confidence**: high (verified), medium (tested once), low (hypothesis)
- **created**: ISO date

### Required Sections

- **Summary**: First paragraph after title
- **Related**: At least ONE wikilink to connect the knowledge graph
- **Tags**: Auto-derived, placed at bottom

## When to Capture

| Trigger | Type | Folder |
|---------|------|--------|
| Chose X over Y | decision | decisions/ |
| Found reusable solution | pattern | patterns/ |
| Solved non-obvious bug | debug | debug/ |
| Learned framework gotcha | concept | concepts/ |
| Established team rule | convention | conventions/ |
| Session starting | (read) | projects/ + linked notes |
| Session ending | (update) | projects/ |

## How to Create Notes

### Step 1: Determine if Cross-Project or Project-Specific

Ask: "Would this help someone working on a DIFFERENT project?"

- **Yes** -> Global vault
- **No** -> Project vault `.vault/knowledge/`

### Step 2: Create with vlt

```bash
vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="---
type: <type>
project: <project>
stack: [<stack>]
domain: <from-vocabulary>
status: active
confidence: <level>
created: $(date +%Y-%m-%d)
---

# <Title>

<one-line summary>

## Content

<main content>

## Related

- [[<related note>]]
" silent
```

### Step 3: Triage Immediately

Don't leave notes in `_inbox/`. Move to correct folder:

```bash
vlt vault="Claude" move path="_inbox/<Title>.md" to="<folder>/<Title>.md"
```

### Step 4: Verify Links

Ensure at least one wikilink exists. If creating a new concept, link to related concepts or the project note.

## How to Read

```bash
# Single note
vlt vault="Claude" read file="<Note Title>"

# Note + all linked notes (graph traversal)
vlt vault="Claude" read file="<Note Title>" follow

# Note + all notes that link TO it
vlt vault="Claude" read file="<Note Title>" backlinks
```

## How to Search

```bash
# Text search
vlt vault="Claude" search query="<term>"

# By domain
vlt vault="Claude" search query="[domain:ai-agents]"

# By project
vlt vault="Claude" search query="[project:reader]"
```

## How to Update

```bash
# Append to note
vlt vault="Claude" append file="<Note Title>" content="<new content>"

# Replace a section
vlt vault="Claude" patch file="<Note Title>" heading="## Section" content="<new section>"

# Set property
vlt vault="Claude" property:set file="<Note Title>" name="status" value="superseded"
```

## Session Workflow

### Session Start

1. Detect project from git remote or directory
2. Read project note + follow links:
   ```bash
   vlt vault="Claude" read file="<Project>" follow
   ```
3. Check for project-specific knowledge in `.vault/knowledge/`

### Session End

1. Run `/vault-capture` to save new knowledge
2. Update project note with session summary
3. Check `_inbox/` size - warn if > 5 notes

### Pre-Compact

Warn if vault needs attention:
```bash
inbox_count=$(vlt vault="Claude" files folder="_inbox" total)
orphans=$(vlt vault="Claude" orphans --json | jq length)
if [ "$inbox_count" -gt 5 ] || [ "$orphans" -gt 10 ]; then
  echo "Vault needs attention: $inbox_count in inbox, $orphans orphans"
fi
```

## Project Notes Are Indexes, Not Logs

The project note in `projects/` should be:
- A summary of the project (stack, domain, purpose)
- Links to key decisions, patterns, and concepts
- Brief session log entries (1-2 lines each, including the `## Session log (...)` entries appended by `pvg` at session end)

NOT:
- Full execution logs
- Story-by-story completion records
- Verbose session transcripts

Keep detailed execution history in nd story notes, git history, or Claude memory when needed. Paivot does not manage a `.vault/sessions/` tree.

## The Rules

1. **Every note needs a link** - No orphans. Knowledge is a graph.
2. **Use controlled domains** - No ad-hoc domain values.
3. **Tags go in body** - Not frontmatter (Obsidian limitation).
4. **Triage immediately** - Don't let `_inbox/` accumulate.
5. **Capture as you go** - Not at the end. Memory decays.
