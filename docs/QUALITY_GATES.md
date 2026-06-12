# Quality gates (`pvg gates`)

`pvg gates` is a deterministic, metric-based quality gate on delivered code. It
complements -- it does not replace -- `pvg verify` (which scans for stubs and
thin files). Where verify asks "is this real?", gates asks "is this
maintainable?" by measuring the delivered code and comparing it against tunable
thresholds.

The PM-Acceptor runs it in Tier 1 of story review. It is also runnable by hand:

```bash
pvg gates [path...]                       # measure the given paths (default ".")
pvg gates --changed <ref> --format text   # scope to the diff against <ref>
pvg gates --json                          # structured output
```

## What the gate checks

| Metric         | What it measures                                | Tool             | Built-in? |
|----------------|-------------------------------------------------|------------------|-----------|
| duplication    | copy-paste / clone percentage and clone size    | jscpd            | no        |
| complexity     | per-function cyclomatic complexity (CCN)         | lizard (fallbacks gocyclo/radon) | no |
| file_loc       | non-blank lines per file                         | built-in         | yes       |

`file_loc` is computed internally and is therefore never skipped. duplication
and complexity shell out to external analyzers; when the analyzer is absent the
gate SKIPs (see below).

## The analyzer matrix

| Tool        | Purpose                       | Languages                                          | Install                                                  | In Ubuntu apt? |
|-------------|-------------------------------|----------------------------------------------------|----------------------------------------------------------|----------------|
| **lizard**  | cyclomatic complexity         | multi (C/C++, Java, JS/TS, Python, Go, Swift, ...) | `pip install lizard`                                     | NO             |
| **jscpd**   | duplication / copy-paste      | multi                                              | `npm install -g jscpd`                                   | NO             |
| gocyclo     | complexity (Go fallback)      | Go only                                            | `go install github.com/fzipp/gocyclo/cmd/gocyclo@latest` | NO             |
| radon       | complexity (Python fallback)  | Python only                                        | `pip install radon`                                      | YES (`apt install python3-radon`) |

**apt alone is not enough.** Only `radon` ships in the Ubuntu repositories. The
two recommended, multi-language tools -- `lizard` (pip) and `jscpd` (npm) --
are not packaged for apt and must come from pip and npm, which are already
present on most dev machines.

Installing just **`lizard` + `jscpd`** lights up the full gate on virtually any
stack. `gocyclo` and `radon` are niche, single-language fallbacks you only need
if you choose not to install `lizard`: when `lizard` is absent, the complexity
gate falls back per language to `gocyclo` (Go files) and `radon` (Python files).

### Install (recommended pair)

```bash
pip install lizard        # cyclomatic complexity (multi-language)
npm install -g jscpd      # duplication detection (multi-language)
```

Run `pvg doctor` to see which analyzers are present, and `pvg setup` nudges you
to install any that are missing.

## SKIP vs WARN vs BLOCK

Each finding has a severity, and each metric a mode (`off` / `warn` / `block`):

- **SKIP** -- the analyzer tool for a metric is absent (e.g. `lizard`/`jscpd`
  not on PATH). The metric is skipped and noted in the report
  (`[SKIP] complexity: lizard not found (pip install lizard) -- or gocyclo for
  Go (...)`). A SKIP is **never a silent pass and never a failure** -- it just
  means the signal was unavailable. Install the tool to turn the SKIP into a
  real measurement.
- **WARN** -- a finding crossed a warn threshold (or its metric is in `warn`
  mode). Reported as `[WARN]`, but does **not** fail the gate.
- **BLOCK** -- a finding crossed a block threshold while its metric is in
  `block` mode. Reported as `[BLOCK]` and **fails** the gate (the run exits
  non-zero and `Report.Blocked` is true). In `warn` mode no BLOCK finding is
  ever produced, even above the block threshold.

## Default severities and thresholds

| Metric        | Default mode | Bands / threshold                                                |
|---------------|--------------|------------------------------------------------------------------|
| duplication   | **block**    | finding when total duplication `>= max_pct` (10%) OR any clone `>= min_lines` (50) |
| complexity    | **block**    | WARN at CCN `>= warn_cc` (15); BLOCK at CCN `>= block_cc` (30)    |
| file_loc      | **warn**     | finding when a file's non-blank LOC `>= max` (400)               |

So out of the box: duplication and complexity can BLOCK; complexity has a warn
band of CC 15-30 and blocks at CC >= 30; file_loc only ever warns. **Every
threshold and mode is tunable** via `pvg settings gates.*`.

## `gates.*` settings reference

| Key                          | Default | Meaning                                                        |
|------------------------------|---------|----------------------------------------------------------------|
| `gates.complexity`           | `block` | Complexity gate mode (`off` / `warn` / `block`)                |
| `gates.complexity.warn_cc`   | `15`    | CCN at/above which a complexity WARN fires                     |
| `gates.complexity.block_cc`  | `30`    | CCN at/above which a complexity BLOCK fires (block mode only)  |
| `gates.duplication`          | `block` | Duplication gate mode (`off` / `warn` / `block`)               |
| `gates.duplication.max_pct`  | `10`    | Total duplication % at/above which a finding fires             |
| `gates.duplication.min_lines`| `50`    | A single clone of >= this many lines fires a finding           |
| `gates.file_loc`             | `warn`  | File-size gate mode (`off` / `warn` / `block`)                 |
| `gates.file_loc.max`         | `400`   | Non-blank lines per file at/above which a finding fires        |
| `gates.exclude`              | `vendor/,node_modules/,*.generated.*,*.pb.go,migrations/,*.lock,*.min.*,dist/,build/` | Comma-separated globs / path-substrings dropped before any metric runs |

Mode keys reject any value other than `off`/`warn`/`block`. Threshold keys take
integers (`max_pct` is a percentage). Set values with, e.g.:

```bash
pvg settings gates.duplication=warn
pvg settings gates.complexity.block_cc=25
pvg settings gates.file_loc.max=500
```

## How the PM-Acceptor uses it (Tier 1)

In Tier 1 of story review (deterministic, before any LLM review), the
PM-Acceptor runs `pvg gates` on the delivered diff and interprets the result:

- **BLOCK => reject.** Any `[BLOCK]` finding is an immediate rejection; the
  PM-Acceptor cites the specific finding (metric, path, symbol, value>threshold)
  in the rejection notes.
- **WARN => noted.** `[WARN]` findings are recorded in the review but are not
  auto-rejections.
- **SKIP => not a failure.** `[SKIP]` lines mean an analyzer tool was absent;
  the gate was skipped, not failed. The PM-Acceptor notes it and moves on.

## Example output

```
$ pvg gates --changed main --format text
[BLOCK] complexity internal/orders/process.go ProcessOrder 41>30  (CCN 41 at line 88)
[BLOCK] duplication (total) 14>10  (14.0% duplicated (max 10%))
[WARN] complexity internal/orders/process.go validate 18>15  (CCN 18 at line 32)
[WARN] file_loc internal/orders/process.go 612>400
[SKIP] duplication: jscpd not found (npm install -g jscpd)
GATES: FAIL (2 block, 2 warn, 1 skipped)
```

A clean run with both analyzers installed and no threshold crossings:

```
$ pvg gates --changed main --format text
GATES: PASS (0 warn, 0 skipped)
```

When an analyzer is missing the corresponding metric SKIPs rather than failing:

```
$ pvg gates --format text
[SKIP] complexity: lizard not found (pip install lizard) -- or gocyclo for Go (go install github.com/fzipp/gocyclo/cmd/gocyclo@latest)
[SKIP] duplication: jscpd not found (npm install -g jscpd)
GATES: PASS (0 warn, 2 skipped)
```
