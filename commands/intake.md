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

## Phase 2: Delegate to Sr. PM Agent

Once all feedback is collected, use the Task tool to spawn the `sr-pm` agent. Pass it:
- The complete list of raw feedback items (with any screenshots or context the user provided)
- The project name and working directory
- Instructions to explore the codebase and vault before writing stories

The sr-pm agent will:
1. Read the relevant source code to understand current implementation
2. Consult the vault for prior decisions and patterns
3. Create properly structured stories with full context, acceptance criteria, testing requirements, and mandatory skill references
4. Establish dependencies between stories
5. Return the complete backlog

**DO NOT create stories yourself.** The Sr. PM agent produces significantly higher quality stories because it embeds codebase context, platform conventions, and testing requirements into each story.

## Phase 3: Present Backlog for Triage

After the Sr. PM agent returns, present the backlog to the user:

1. Show all stories sorted by priority in a table:
   ```
   | # | Priority | Story | Type | Depends On |
   |---|----------|-------|------|------------|
   ```

2. Ask: "This is the proposed backlog and order. Want to reorder, cut, merge, or add anything before execution begins?"

3. Wait for user approval. Adjust if requested.

## Phase 4: Execute

Work through the approved backlog top-to-bottom. For each story:

1. **Read the full story** from beads (`bd show <id>`)
2. **Load the mandatory skills** listed in the story's MANDATORY SKILLS TO REVIEW section
3. **Consult the vault** for relevant prior knowledge
4. **Show your approach** before writing code. If the fix touches interaction flow or visual design, describe before/after. Wait for user approval on non-trivial changes.
5. **Implement the fix.** Build and verify.
6. **Close the story** (`bd close <id>`)
7. **Capture learnings** to the vault (decisions, patterns, debug insights)
8. Move to the next story.

## Constraints

- No speculative refactoring. Only fix what is in the backlog.
- Every UI change must follow platform conventions. The story's MANDATORY SKILLS section tells you which skills to load.
- If a fix reveals a deeper problem, create a NEW story for it (via the Sr. PM agent if it needs proper decomposition) rather than scope-creeping the current one.
- After completing all stories, run `/vault-capture` for a final knowledge pass.
