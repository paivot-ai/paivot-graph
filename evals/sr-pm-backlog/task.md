# Task: Author the initial backlog for CurbSide

Discovery & Framing is complete. The three D&F documents (BUSINESS.md,
DESIGN.md, ARCHITECTURE.md) are in your current working directory. They are
final and mutually consistent: treat them as the source of truth and do not
ask clarifying questions (no user is available in this environment; if a
judgment call is needed, make a reasonable assumption and record it in your
summary).

Author the initial backlog now, with ONE environment change from your normal
workflow: **nd, pvg, and vlt are unavailable in this sandbox.** There is no
issue tracker, no vault, no linter, and no Anchor to submit to. Therefore:

- Do NOT run nd, pvg, or vlt commands, and do not invoke Skills.
- Write the backlog as markdown documents in a `stories/` directory of this
  workspace: one file per epic (`stories/epic-<slug>.md`) and one file per
  story (`stories/story-<slug>.md`).
- Each file uses your normal epic or story body structure exactly as your
  templates define it (Title, plain labels such as `Description:` and
  `Acceptance Criteria:`, and so on).
- Metadata that would normally be nd fields goes at the top of each file as
  plain lines: `Priority: P<n>`, `Parent: <epic file>`,
  `Blocked by: <story files, comma separated>`, `Labels: <labels>` (omit the
  line when empty).
- Skip the mechanical lint gate and the Anchor submission (no tooling), but
  still perform your own final review passes and include the resulting
  per-story verdicts in a closing summary file `stories/SUMMARY.md`.

Everything else in your playbook applies unchanged.
