---
name: nd-agent-integration
description: Use this skill whenever an agent will issue nd commands (the git-native issue tracker) inside a Claude Code session, or will write Bash arguments, wikilink content, commit messages, or vault appends that contain nd subcommand substrings. Covers four predictable friction points -- nd guard false positives (hookify), Bash permission prefix matching, Write-tool-vs-heredoc for large bodies, and the SIGKILL-until-session-restart issue after settings.json changes -- plus correct .vault/ placement. Load before creating stories, implementing stories, or appending vault notes that reference nd operations.
version: 0.1.0
---

# nd + Claude Code Agent Integration

Agents working with `nd` (the git-native issue tracker) inside Claude Code sessions hit four predictable friction points. Each has a known workaround. Read this before issuing any `nd` command or writing Bash arguments that contain `nd` substrings.

## 1. nd guard false positives (hookify)

The nd guard substring-matches the full Bash command string. Any command that contains `nd vault`, `nd issues`, or `nd <subcommand>` in an argument -- even inside `-m`, `content=`, or a for-loop variable list -- triggers the guard as if nd were being invoked directly.

**Patterns that trigger false positives:**

```bash
# BLOCKED: content string contains "nd vault"
vlt vault="Claude" append file="projects/nd" content="...traced nd vault resolution..."

# BLOCKED: path suffix contains "issues/"
git add .vault/issues/

# BLOCKED: commit message contains "nd vault"
git commit -m "feat: track nd vault issues in git"

# BLOCKED: loop body contains "nd pvg"
for d in vlt nd pvg paivot-graph; do git -C $d push; done
```

**Fix:** Write offending content to a temp file first, then reference it.

```bash
# Step 1: Write tool (no Bash)
Write(file_path="/tmp/content.md", content="...traced nd vault resolution...")

# Step 2: Reference in command
vlt vault="Claude" append file="nd" content="$(cat /tmp/content.md)"
```

For git:
- `git add .vault` (not `.vault/issues/`) avoids the path suffix trigger
- `git commit -F /tmp/commit-msg.txt` avoids the `-m` argument trigger
- Run each component separately instead of a for-loop with `nd` in the variable list

## 2. Bash permission prefix matching (git, make, nd)

Claude Code's `Bash(git:*)` permission uses prefix matching. Compound commands starting with `cd` or an env var do not match and require manual approval every time.

```bash
# CORRECT -- starts with "git", matches Bash(git:*)
git -C /path/to/repo log --oneline

# WRONG -- starts with "cd", fails prefix match
cd /path/to/repo && git log --oneline

# CORRECT -- env var after make, not before
make TARGET VAR=val

# WRONG -- env var before make, breaks Bash(make:*)
VAR=val make TARGET
```

For `nd`: always invoke `nd <subcommand>` directly. Do not prepend env vars or `cd`.

## 3. Large file bodies: Write tool, not Bash heredocs

Bash heredocs and `python3 -c "..." <<'EOF'` patterns trigger hookify's "brace with quote character (expansion obfuscation)" security rule.

```bash
# CORRECT: two clean steps
# Step 1: Write tool
Write(file_path="/tmp/story.md", content="## Description\n...")
# Step 2: nd command only
nd create "Title" --type=task --body-file=/tmp/story.md

# WRONG: triggers hookify
python3 -c "..." <<'PAYLOAD'
...content...
PAYLOAD
```

## 4. nd SIGKILL persists until session restart

Adding `Bash(nd:*)` to `settings.json` mid-session does not take effect until Claude Code is fully restarted. `nd` commands continue to exit 137 (SIGKILL) after the file is updated.

**Workaround for the current session:** use the freshly-built `nd` binary from the worktree with an explicit `--vault` flag instead of the installed binary.

```bash
/path/to/worktree/nd --vault /path/to/.vault <subcommand>
```

**Permanent fix:** add the permission before the session starts, or restart Claude Code.

## 5. nd vault placement

Place the `nd` vault at `<repo-root>/.vault/`. `nd` resolves the vault by walking up from cwd looking for `.vault/.nd.yaml`. A misplaced vault (e.g., `~/.vault`) causes `nd` to find the wrong vault at home-directory scope.

Worktrees at `<repo>/.worktrees/<branch>` resolve correctly -- they walk up to `<repo-root>/.vault/` naturally.

## 6. nd capability surface

The `nd` CLI has grown capabilities that simplify agent workflows. Using deprecated flag forms or missing these capabilities leads to extra tool calls and manual steps.

### Priority format (P0-P4)

Create stories with an explicit priority using `--priority=P<N>`:

```bash
nd create "Audit fork commits" --type=task --priority=P1
```

Valid priorities: `P0`, `P1`, `P2`, `P3`, `P4`. Default is `P2` when omitted.

### Watch mode

Add `--watch` to any read command to block until the issue changes state:

```bash
nd show PAI-9oyq --watch
```

Useful when an agent needs to pause until a subagent or CI job updates a story's status before proceeding.

### Force color output

Add `--color=always` when piping `nd list` to preserve color codes:

```bash
nd list --color=always --limit 20
```

Without this flag, color is stripped when piped or when stdout is not a TTY.

### Sort by dependencies

Add `--sort=deps` to `nd list` to produce topological order:

```bash
nd list --sort=deps
```

Stories with blockers appear after their dependencies, making it easier to pick the next ready item.

### Group by parent epic

Add `--group-by=parent` to `nd list` to show a tree under each epic:

```bash
nd list --group-by=parent --limit 30
```

This is the preferred view when reviewing backlog composition rather than picking the next story.

### Comments on updates

Add `--comment` (or use the `comment` subcommand alias) to append a note without editing the body:

```bash
nd update PAI-9oyq --comment "Seed output verified byte-equal via diff -rq"
```

This keeps the issue body clean while recording session milestones in the history.

### Terminal-width formatting

`nd` drops optional columns and compacts tree connectors when the terminal is narrow. Agents running in constrained PTYs may see abbreviated output. If a command looks incomplete, widen the terminal or pipe to a file:

```bash
nd list > /tmp/backlog.txt
```

## Checklist (apply before every nd-adjacent command)

1. Does the Bash argument, commit message, or wikilink content contain an `nd <subcommand>` substring? If yes, write to `/tmp/...` first and reference via `$(cat ...)` or `--body-file`.
2. Is the command starting with `git`, `make`, or `nd` directly (no `cd`, no env-var prefix)? If not, restructure.
3. Is the body longer than a line or two? Use the Write tool to a temp file, then reference. Never use heredocs or `python3 -c`.
4. Is `nd` returning exit 137? Use the locally-built binary with `--vault <path>` until the session restarts.
5. New repo? Verify `<repo-root>/.vault/.nd.yaml` exists before running `nd`.
6. Did you check the nd capability section before composing the command? Some friction historically comes from agents using deprecated flag forms or missing capabilities like `--watch` and `--sort=deps`.

## Source notes (in "Claude" vault)

- `[[Use Write tool for large file bodies, not Bash heredocs]]` (patterns/)
- `[[Use git -C to avoid Bash permission stalls in Claude Code]]` (patterns/)
- `[[nd guard false-positives -- all patterns and workarounds]]` (debug/)
- `[[nd CLI SIGKILL Persists Until Session Restart After settings.local.json Update]]` (debug/)
- `[[Working around nd guard false positives]]` (patterns/)
- `[[nd vault placement in project repo]]` (patterns/)
