---
description: Capture UX/visual/functional feedback and turn it into a prioritized backlog of high-quality stories using the Sr. PM agent
allowed-tools: ["Bash", "Read", "Write", "Edit", "Glob", "Grep", "Skill", "Agent", "Task", "AskUserQuestion"]
---

# Intake -- Feedback to Backlog

Collect user feedback about the current state of the product, then delegate to the Sr. PM agent to create properly structured stories.

**Vault:** `vlt vault="Claude"` (resolves path dynamically)

## Phase 1: Collect Raw Feedback

Say: "Ready for feedback. Describe each issue -- include screenshots if you have them. Say 'that's all' when done."

For each issue the user describes:
1. Acknowledge it in your own words to confirm understanding
2. Ask clarifying questions if the desired outcome is ambiguous
3. Record it in a running list (DO NOT create nd issues yet -- the Sr. PM will do that with proper quality)

Keep collecting until the user says "that's all" or equivalent.

## Phase 2: Gather Context Before Delegating

Before spawning the Sr. PM agent, YOU must gather context and pass it in the prompt. The agent cannot be trusted to do this on its own.

### 2a. Fetch vault knowledge

First, read the vault-backed operating mode and everything it links to:
```bash
pvg notes read "Session Operating Mode"
# Note: `pvg notes read` addresses by full path; if the note is not at vault root,
# pass the full path (e.g., "methodology/Session Operating Mode"). For transitive
# expansion (auto-include linked notes), use the vlt escape hatch:
#   vlt vault="Claude" read file="Session Operating Mode" follow
# This is a vlt-specific operation and assumes the global Claude vault.
```

Then read the project note with all linked context (decisions, patterns, session logs):
```bash
pvg notes read "<project-name>"
# For transitive expansion (recommended here -- pulls in all linked decisions,
# patterns, debug insights, prior session logs):
#   vlt vault="Claude" read file="<project-name>" follow
```

This gives you the project note plus the full content of every note it references -- typically decisions, patterns, debug insights, and prior session logs -- in a single call.

If you need additional context not linked from the project note:
```bash
pvg notes search "<project-name>"
```

Fallback if vlt unavailable: use Read/Grep tools directly on the vault path.

### 2b. Detect the project's tech stack

Identify the language, framework, and platform from the codebase (e.g., SwiftUI + macOS, React + web, Flutter + mobile). This determines which skills the stories must reference.

### 2c. Build the skill mapping

Based on the detected stack, determine which skills apply:
- macOS/SwiftUI: `macos-design-guidelines`, `swiftui-skills`
- Web React: `ui-ux-pro-max`, `tailwind-design-system`
- Mobile: `mobile-design`
- Other: identify from available skills

## Phase 3: Delegate to Sr. PM Agent

Use the Task tool to spawn the `sr-pm` agent. The prompt MUST include:

1. **The complete list of raw feedback items** (with any screenshots or context the user provided)
2. **The project name and working directory**
3. **All vault knowledge fetched in Phase 2a** -- paste the actual content, not "consult the vault"
4. **The tech stack and applicable skills** -- explicitly state: "Every story's MANDATORY SKILLS TO REVIEW section must include: `<skill-1>`, `<skill-2>`" based on the mapping from Phase 2c
5. **Any DESIGN.md, ARCHITECTURE.md, or similar doc paths** if they exist in the project

The sr-pm agent will:
1. Read the relevant source code to understand current implementation
2. Use the vault knowledge you provided to avoid rediscovering known patterns
3. Create properly structured stories with full context, acceptance criteria, testing requirements, and mandatory skill references
4. Establish dependencies between stories
5. Return the complete backlog

**DO NOT create stories yourself.** The Sr. PM agent produces significantly higher quality stories because it embeds codebase context, platform conventions, and testing requirements into each story.

## Phase 4: Present Backlog for Triage

After the Sr. PM agent returns, present the backlog to the user:

1. Show all stories sorted by priority in a table:
   ```
   | # | Priority | Story | Type | Depends On |
   |---|----------|-------|------|------------|
   ```

2. Ask: "This is the proposed backlog and order. Want to reorder, cut, merge, or add anything before execution begins?"

3. Wait for user approval. Adjust if requested.

## Phase 5: Execute

### Concurrency Limits (HARD RULE)

Limits are stack-dependent -- see the Concurrency Limits table in `/piv-loop`, which is authoritative.
Summary: heavy stacks (Rust, iOS/Swift, C#, CloudFlare Workers) allow 2 developers / 1 PM-Acceptor / 3 total;
light stacks (Python, non-CF TypeScript/JavaScript) allow 4 developers / 2 PM-Acceptors / 6 total. Mixed stacks use the most restrictive limit.

### Execution Loop

Work through the approved backlog top-to-bottom. For each story:

1. **Spawn a developer agent** to implement the story. The developer will:
   - Read the full story (`pvg nd show <id>`) and claim it (`pvg nd update <id> --status in_progress`)
   - Load mandatory skills from the story's MANDATORY SKILLS TO REVIEW section
   - Implement the change, write tests, run CI locally
   - Leave breadcrumb notes: `pvg nd update <id> --append-notes "COMPLETED: ... IN PROGRESS: ... NEXT: ..."` (nd-specific)
   - Mark as delivered with proof (`pvg nd update <id> --add-label delivered`)
   - The developer does NOT close stories

2. **Spawn a PM-Acceptor agent** to review the delivered story. The PM-Acceptor will:
   - Review evidence (CI results, coverage, test output)
   - Verify outcomes match acceptance criteria
   - Accept: `pvg nd close <id> --reason="Accepted: <summary>" --start=<next-id>` (--start is nd-specific)
   - Or reject: return the story to `open`, remove `delivered`, add `rejected`, and leave detailed notes via `pvg issues comment`

3. **Capture learnings** to the vault via `pvg notes create "_inbox/<Title>.md" --title "<Title>" --body "..."` (decisions, patterns, debug insights)

4. If a discovered issue arises during implementation, route it through the documented bug flow: Developer/PM-Acceptor emits `DISCOVERED_BUG`, then Sr PM triages it (or PM fast-track if enabled). Do NOT quick-capture ad-hoc bugs with `nd q`.

5. Move to the next story.

## Constraints

- No speculative refactoring. Only fix what is in the backlog.
- Every UI change must follow platform conventions. Load the relevant skills even if the story doesn't explicitly list them.
- If a fix reveals a deeper problem, create a NEW story for it (via the Sr. PM agent if it needs proper decomposition) rather than scope-creeping the current one.
- After completing all stories, run `/vault-evolve` to refine vault content from session experience.
