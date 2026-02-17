---
description: Capture UX/visual/functional feedback and turn it into a prioritized backlog of high-quality stories using the Sr. PM agent
allowed-tools: ["Bash", "Read", "Write", "Edit", "Glob", "Grep", "Skill", "Task", "AskUserQuestion"]
---

# Intake -- Feedback to Backlog

Collect user feedback about the current state of the product, then delegate to the Sr. PM agent to create properly structured stories.

## Phase 1: Collect Raw Feedback

Say: "Ready for feedback. Describe each issue -- include screenshots if you have them. Say 'that's all' when done."

For each issue the user describes:
1. Acknowledge it in your own words to confirm understanding
2. Ask clarifying questions if the desired outcome is ambiguous
3. Record it in a running list (DO NOT create beads issues yet -- the Sr. PM will do that with proper quality)

Keep collecting until the user says "that's all" or equivalent.

## Phase 2: Gather Context Before Delegating

Before spawning the Sr. PM agent, YOU must gather context and pass it in the prompt. The agent cannot be trusted to do this on its own.

### 2a. Fetch vault knowledge

```bash
obsidian vault="Claude" search query="<project-name>"
```

For each relevant note found, read it:
```bash
obsidian vault="Claude" read file="<note-title>"
```

Collect all vault content (decisions, patterns, debug notes) relevant to this project.

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

Work through the approved backlog top-to-bottom. For each story:

1. **Read the full story** from beads (`bd show <id>`)
2. **Load the mandatory skills** listed in the story's MANDATORY SKILLS TO REVIEW section. If the section says "None identified" but the project uses a platform with known skills (macOS, web, mobile), load the relevant platform skills anyway.
3. **Consult the vault** for relevant prior knowledge
4. **Show your approach** before writing code. If the fix touches interaction flow or visual design, describe before/after. Wait for user approval on non-trivial changes.
5. **Implement the fix.** Build and verify.
6. **Close the story** (`bd close <id>`)
7. **Capture learnings** to the vault (decisions, patterns, debug insights)
8. Move to the next story.

## Constraints

- No speculative refactoring. Only fix what is in the backlog.
- Every UI change must follow platform conventions. Load the relevant skills even if the story doesn't explicitly list them.
- If a fix reveals a deeper problem, create a NEW story for it (via the Sr. PM agent if it needs proper decomposition) rather than scope-creeping the current one.
- After completing all stories, run `/vault-capture` for a final knowledge pass.
