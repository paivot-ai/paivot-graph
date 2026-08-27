# Milestone acceptance on a machinery design (Ga-accept)

On a machinery-first project the design owns what counts as acceptance evidence and
Paivot is the process that produces the judgment. machinery v0.3.10 added
**Ga-accept**: a build-plan milestone marked `Status: closed` must have committed
acceptance evidence at `<design>/acceptance/M<n>.yaml` whose verdict is `ACCEPTED`,
which names the commit the review ran on and lists every committed oracle id the
milestone's DoD cites. The authoritative contract is machinery's
`docs/acceptance-gate.md` plus the Ga section of its skill; this document says how
Paivot's milestone machinery produces that artifact and how closure is sequenced.

This applies ONLY when the user enabled the machinery substrate
(`pvg settings design.machinery=on`, or `auto` on a machinery-managed repo). With the
substrate off there is no design, no build plan, and no acceptance directory, and
nothing here runs.

## The one-line rule

**No milestone closes anywhere -- not in the build plan, not in the tracker -- without
an ACCEPTED evidence file written by the reviewing Anchor and a bound
`machinery check` run recorded by the dispatcher.**

## What maps to what

| machinery | Paivot |
|---|---|
| A build-plan milestone block, `M<n>`, with its `DoD:` line | The epic that discharges it: the MILESTONE epic in the nested model, or the `milestone`-labeled epic in the flat model when the plan's `M<n>` is that one slice |
| `Status: closed` on the milestone block | The closure act at the Milestone Seal Gate (dispatcher), never a story or epic acceptance |
| `<design>/acceptance/M<n>.yaml` | The written output of the Anchor's milestone (seal) review |
| Ga green with `--commit` | The recorded closure verification the dispatcher runs before it commits |

The evidence is keyed by the BUILD PLAN milestone number, never by the epic id. The
epic that discharges `M3` writes `acceptance/M3.yaml` whatever its own id is, and the
Sr PM records `BUILD PLAN MILESTONE: M3` in that epic's body so the seal gate can key
it without guessing.

## Who writes what (no other role writes any of it)

| Role | Writes | Never |
|---|---|---|
| **Anchor** (milestone / seal review) | `<design>/acceptance/M<n>.yaml`, the whole file, on BOTH verdicts | Anything else under `<design>/`; the build plan; the tracker |
| **Dispatcher** (`/piv-loop` Milestone Seal Gate) | The milestone block's `Status:` line, and only that line; the closure commit; the seal tag; the nd close + `accepted` label | The evidence file's judgment content; any other line of the plan; any other design file |
| **Sr PM, Developer, PM-Acceptor, Retro** | Nothing under `<design>/`, acceptance included | Acceptance evidence, in any form, for any reason |

A developer that "writes the acceptance file for the milestone it finished" has written
its own report card. The reviewer writes the evidence, and the reviewer is not the
delivering role.

## The sequence

Preconditions: every slice of the milestone is closed and merged, the epic completion
gate ran for each of them, and the working tree is `main` at the merge of the last
slice. Nothing is committed between step 1 and step 6: the sha the review ran on must
still be `HEAD` when the gate binds it.

1. **Pin the reviewed commit.**
   ```bash
   git checkout main && git pull --ff-only
   REVIEWED_SHA=$(git rev-parse HEAD)
   ```
2. **Whole-design gate.** `pvg gates --seal` (machinery's Gt-tests and G4-import forced
   in over the configured impl). Red is a failed seal; never `--skip-design` past it.
3. **Milestone DoD.** Every line of the milestone epic's DoD, verbatim from the build
   plan, demonstrably satisfied with evidence.
4. **Anchor seal review at `REVIEWED_SHA`.** The Anchor runs its milestone-review
   ladder over the whole layer and writes `<design>/acceptance/M<n>.yaml` as its
   output, with `commit: <REVIEWED_SHA>`. `REVIEW_RESULT: VALIDATED` writes
   `verdict: ACCEPTED`; `REVIEW_RESULT: GAPS_FOUND` writes `verdict: REJECTED` with the
   gaps as `findings`.
5. **On REJECTED: stop.** Commit the rejected evidence on its own (it is a legal,
   passing state: Ga only rejects a REJECTED verdict on a CLOSED milestone), leave the
   milestone open in the plan and the epic open in the tracker, and send the gaps back
   into delivery. The next attempt overwrites the same file; git history is the record
   of prior attempts, so no `-round2` files are ever created.
6. **On ACCEPTED: the closure act.** One act, in this order, with the evidence file
   already in the working tree:
   ```bash
   # a. Edit ONLY the Status line of the M<n> block in the build plan:
   #      Status: closed
   #    (add the line beside DoD: when the block has none). Nothing else changes.

   # b. Verify the closure BEFORE committing it, bound to the reviewed commit:
   machinery check <design> --impl <impl> --commit "$REVIEWED_SHA"

   # c. Require 0 blocking (ERROR/DRIFT) findings, Ga included. Any finding is a
   #    FAILED closure: fix the evidence (Ga names the missing dod_ids, the bad
   #    field, the unbound commit) and re-run. Never commit a red closure.

   # d. Commit both files as one closure commit, then tag the seal:
   git add <design>/acceptance/M<n>.yaml <design>/BUILD.md   # or the shard holding M<n>
   git commit -m "accept(M<n>): milestone closed on $REVIEWED_SHA, Ga green"
   git tag -a seal/M<n> -m "M<n> sealed"
   ```
7. **Then, and only then, close the epic in the tracker** (`pvg nd close`, then
   `--add-label accepted`). The recorded gate output goes in the close reason and as a
   comment on the milestone epic: the closure verification is evidence, so it is
   written down where an auditor reads, not left in a terminal.

The order in steps 4 to 7 is the whole point. Machinery's rule is that a closed
milestone with no evidence is an ERROR, so the marker can never land first and be
reconciled later; Paivot's rule is that the tracker never records a closure the design
does not carry.

## The commit field, exactly

`commit:` names **the commit the review ran on**, which is `REVIEWED_SHA`, which is
still `HEAD` when the gate runs in step 6b. It is NOT the closure commit (that commit
does not exist yet, and cannot: a file naming the commit that contains it has no fixed
point).

Consequences, all of them intended:

- The bound run is `machinery check <design> --impl <impl> --commit "$REVIEWED_SHA"`
  performed with the evidence and the `Status:` line in the working tree, before the
  closure commit. That is the run whose green output is recorded.
- After the closure commit lands, a re-run binds only when passed the sha recorded in
  the evidence. A run without `--commit` stays green and prints machinery's
  non-blocking "commit binding not checked" note. That is why `pvg gates` (which passes
  no commit) keeps passing for every developer after a milestone closes.
- A CI job that passes its own `HEAD` on every push will not bind against a milestone
  accepted on an earlier commit. Passing the commit is machinery's documented CI recipe
  for the acceptance change itself; Paivot does not add a second one, and does not
  pretend an unbound run is a bound one.

## Ga is armed for everyone once the directory exists

Ga auto-activates as soon as `<design>/acceptance/` exists or any milestone carries the
closed marker. From the first evidence file onward the gate runs inside every
`pvg gates`, which means inside every developer pre-delivery self-check, every
PM-Acceptor Tier 1, and every Anchor pre-pass. Malformed evidence, a closed milestone
whose evidence never landed, or a stray file in the acceptance directory blocks
delivery for everyone until it is fixed. That is the intended blast radius: acceptance
evidence is part of the design's health, not a filing cabinet beside it.

## The guard carve-out

`design.machinery` makes the whole design tree read-only for delivery agents (pvg's
`checkDesignTreeWrite`: every tracked delivery agent, and the coordinator while an
execution loop is active). Milestone acceptance needs exactly two writes inside that
tree and no others:

1. The **reviewing Anchor** may write `<design>/acceptance/M<n>.yaml`, and nothing else
   under `<design>/`.
2. The **coordinator running the Milestone Seal Gate** may write
   `<design>/acceptance/M<n>.yaml` and edit the `Status:` line of the `M<n>` block in
   the build plan, and nothing else under `<design>/`.

Everything else stays exactly as read-only as it is today: sources, generated
artifacts, other milestone blocks, other lines of the milestone block. A design defect
is still a `DESIGN_REVISION_REQUEST` to the user, never an edit.

The carve-out is a pvg-side guard change and ships with the pvg release paired to this
plugin version. Until that pvg release is installed, the guard blocks the write during
an active loop. When it does:

- Do NOT work around it. Never turn the guard off, never `pvg dispatcher off` mid-loop,
  never write the file from a role the carve-out does not name.
- Escalate to the user via `AskUserQuestion` with the exact evidence content and the
  exact `Status:` edit, and hold the seal until the write lands. A blocked write is a
  paused seal, never a skipped one.

## Rejected reviews are normal

A REJECTED evidence file on an OPEN milestone passes Ga: it is the recorded state of a
review that found problems, and recording it is how the next reviewer sees what the
last one found. What fails is a milestone marked closed against a REJECTED file, or
against no file at all. Never delete a rejected file to "clean up" before the next
review; overwrite it when the next review runs.
