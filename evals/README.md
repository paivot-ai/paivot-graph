# Agent-Prompt Evals (experimental)

A scenario-based regression harness for the agent prompts in `agents/`. The
intent: catch agent-prompt regressions before release, the way TIXX-style
field feedback catches them after. Modeled on the scenario/criteria pattern
from the AI-Unified-Process marketplace (fixture inputs plus a weighted
checklist rubric).

This harness is experimental. It is not wired into CI, and no release gate
depends on it. Runs consume real Claude tokens, so treat it as an on-demand
tool, not an automated check.

## Scenario format

Each scenario is a directory `evals/<name>/` containing:

| File | Purpose |
|------|---------|
| `scenario.json` | `{"agent": "<agent file under agents/>", "description": "...", "include": ["./inputs"]}` |
| `task.md` | The exact task prompt given to the agent under test |
| `criteria.json` | `{"context": "...", "type": "weighted_checklist", "checklist": [{"name", "description", "max_score"}, ...]}` |
| `inputs/` | Fixture files copied into the scratch workspace before the run |

The runner composes the prompt as: contents of the agent `.md` file, a
separator, then `task.md`. The agent runs headlessly (`claude -p` with
`--permission-mode acceptEdits`) from inside a scratch copy of `inputs/`.

## Running

```bash
make evals                       # all scenarios
make evals SCENARIO=<name>       # one scenario
scripts/run-evals.sh [scenario]  # direct invocation
```

Results land in `evals/.results/<scenario>/` (gitignored):

- `workspace/`: the scratch workspace the agent worked in
- `transcript.md`: the agent's stdout
- `grade.json`: per-criterion scores with evidence, plus a total

## Grading

Grading is LLM-judged: a second `claude -p` call receives the rubric from
`criteria.json`, the task, and the workspace contents (each file capped at
200 lines), and returns JSON with a score, max_score, and evidence line per
criterion. Scores are a regression signal, not an absolute measure: compare
a run against a previous run of the same scenario on the prior prompt
version, and read the evidence lines before trusting a delta.

## Guard rails

- If the `claude` CLI is absent the runner prints a SKIP message and exits 0.
- Each agent run and each grading call is bounded by `EVAL_TIMEOUT` seconds
  (default 900), implemented portably (macOS has no `timeout` binary).
- A full run consumes real tokens on your Claude account.
