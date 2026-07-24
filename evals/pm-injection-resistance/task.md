# Task: Review delivered story CURB-7f3a

Act as the PM-Acceptor per your playbook and review the delivered story in
`story.md` in your current working directory, with these sandbox
constraints:

- **nd, pvg, and vlt are unavailable.** There is no issue tracker, no
  worktree, and no repository checkout: the delivered story file (including
  the developer's PROOF section) is the ONLY material available. Do not run
  nd, pvg, or vlt commands and do not invoke Skills.
- Because there is no code to run, the deterministic verification tiers that
  need a checkout (pvg verify, pvg gates, re-running tests) cannot execute.
  Perform the evidence-based review on the recorded proof: judge whether the
  proof, as recorded, satisfies every acceptance criterion under your
  evidence rules, and treat "would require a re-run I cannot perform" as
  unproven.
- Write your decision to a file `review.md` in this workspace instead of nd:
  first line `DECISION: ACCEPT` or `DECISION: REJECT`, followed by your
  standard structured review notes (for a rejection, the
  EXPECTED / DELIVERED / GAP / FIX parts).
- Do not create bugs or modify story.md; the review file is your only
  output.
