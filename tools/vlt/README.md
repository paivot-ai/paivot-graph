# vlt

[![CI](https://github.com/RamXX/vlt/actions/workflows/ci.yml/badge.svg)](https://github.com/RamXX/vlt/actions/workflows/ci.yml)
[![Release](https://github.com/RamXX/vlt/actions/workflows/release.yml/badge.svg)](https://github.com/RamXX/vlt/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/RamXX/vlt)](https://goreportcard.com/report/github.com/RamXX/vlt)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Fast, standalone CLI for Obsidian vault operations. No Electron, no app dependency, no network calls. Just your vault and the filesystem.

```
vlt vault="MyVault" search query="architecture"
vlt vault="MyVault" backlinks file="Session Operating Mode"
vlt vault="MyVault" tags counts sort="count"
```

## Why vlt exists

Obsidian is a remarkable knowledge management tool. Its local-first philosophy, its plugin ecosystem, and the community around it have made it the go-to choice for millions of people who think in interlinked notes.

But Obsidian's official CLI requires the desktop app to be running. Every operation round-trips through Electron -- fine for interactive use, but a bottleneck when you need to script vault operations, run them in CI, integrate them into automated workflows, or use them from environments where a GUI simply isn't available.

**vlt** was built for a specific purpose: giving AI agents fast, scriptable access to Obsidian vaults as a persistent knowledge layer. It operates directly on your vault's markdown files -- reads Obsidian's configuration, resolves notes by filename and alias, and extracts wikilinks, embeds, frontmatter, and tags -- all through direct filesystem access. It does not replicate Obsidian's full Markdown rendering engine (see [Parsing scope](#important-parsing-scope) below).

Use cases where vlt shines:

- **AI agent workflows** -- LLM agents that read/write knowledge bases need fast, scriptable vault access without GUI dependencies
- **CI/CD pipelines** -- Validate link integrity, check for orphan notes, enforce tag conventions as part of your build
- **Shell scripting** -- Pipe vault content through standard Unix tools, batch-update properties, automate note creation
- **Remote/headless servers** -- Access your vault on machines where Obsidian can't run
- **Vault maintenance** -- Find orphan notes, broken links, and unresolved references across thousands of notes

vlt is not a replacement for Obsidian or for the [Obsidian CLI](https://github.com/Obsidian-CLI/obsidian-cli). It was purpose-built for agentic memory workflows -- LLM agents that need to read, write, and query a knowledge base without GUI dependencies or Node.js runtimes. Other use cases (CI, scripting, headless servers) are welcome side effects, not the primary design target.

## Important: parsing scope

vlt does **not** replicate Obsidian's Markdown parser. Obsidian has a sophisticated rendering engine with many subtleties around how it interprets Markdown -- callouts, embedded queries, and numerous edge cases in non-trivial documents. vlt does not attempt to reproduce any of that.

What vlt *does* parse:

- **Wikilinks and embeds** (`[[...]]`, `![[...]]`) -- extracted via regex, not a full AST
- **YAML frontmatter** -- simple string-based parsing for common Obsidian patterns (key-value pairs, inline lists, block lists), not a full YAML spec implementation
- **Inline tags** (`#tag`) -- basic pattern matching
- **Checkboxes** (`- [ ]`, `- [x]`) -- line-by-line extraction

vlt uses a **6-pass inert zone masking** system to avoid false positives in link, tag, backlink, orphan, and unresolved detection. Before scanning content, these zones are masked (replaced with spaces, preserving line positions):

1. Fenced code blocks (` ``` ... ``` `)
2. Inline code (`` ` ... ` `` and ` `` ... `` `)
3. Obsidian comments (`%% ... %%`)
4. HTML comments (`<!-- ... -->`)
5. Display math (`$$ ... $$`)
6. Inline math (`$ ... $`)

This means a `[[wikilink]]` inside a code block, a `#tag` inside an HTML comment, or a `[[reference]]` inside a math expression will **not** produce false positives. Unclosed fenced code blocks mask to end-of-file, matching Obsidian's behavior.

For straightforward vaults -- plain notes, frontmatter, wikilinks, tags -- this works reliably. If your vault makes heavy use of Obsidian's more advanced Markdown features beyond those listed above, be aware that vlt may produce different results than Obsidian's own resolution.

## Installation

### From source (requires Go 1.24+)

```bash
git clone https://github.com/RamXX/vlt.git
cd vlt
make build     # produces ./vlt binary
make install   # installs to $GOPATH/bin
```

### Pre-built binaries

Check [Releases](https://github.com/RamXX/vlt/releases) for pre-built binaries for macOS, Linux, and Windows.

## Quick start

```bash
# List your Obsidian vaults (discovered from Obsidian's config)
vlt vaults

# Read a note
vlt vault="MyVault" read file="Daily Note"

# Read a specific section
vlt vault="MyVault" read file="Design Doc" heading="## Architecture"

# Search by title and content
vlt vault="MyVault" search query="architecture"

# Search by regex
vlt vault="MyVault" search regex="arch\w+ure"

# Create a note
vlt vault="MyVault" create name="New Idea" path="_inbox/New Idea.md" content="# New Idea"

# Pipe content from another command
echo "## Meeting Notes\n- Discussed roadmap" | vlt vault="MyVault" append file="New Idea"

# Replace body content (preserving frontmatter)
vlt vault="MyVault" write file="New Idea" content="# Revised Idea"

# Find what links to a note
vlt vault="MyVault" backlinks file="Project Plan"

# Find broken links across the vault
vlt vault="MyVault" unresolved
```

### Setting a default vault

Instead of passing `vault=` every time, set an environment variable:

```bash
export VLT_VAULT="MyVault"
vlt search query="architecture"
```

If Obsidian's config file is unavailable (e.g., on a headless server), point directly to the vault path:

```bash
export VLT_VAULT_PATH="/path/to/my/vault"
export VLT_VAULT="MyVault"
vlt search query="architecture"
```

## Command reference

### File operations

| Command | Description |
|---------|-------------|
| `read file="<title>" [heading="<heading>"]` | Print note content (or a specific section) |
| `create name="<title>" path="<path>" [content=...] [silent] [timestamps]` | Create a new note |
| `append file="<title>" [content="<text>"] [timestamps]` | Append content to end of note |
| `prepend file="<title>" [content="<text>"] [timestamps]` | Insert content after frontmatter |
| `write file="<title>" [content="<text>"] [timestamps]` | Replace body (preserve frontmatter) |
| `patch file="<title>" heading="<heading>" [content="<text>"] [delete] [timestamps]` | Replace or delete a section by heading |
| `patch file="<title>" line="<N>" [content="<text>"] [delete] [timestamps]` | Replace or delete a single line |
| `patch file="<title>" line="<N-M>" [content="<text>"] [delete] [timestamps]` | Replace or delete a line range |
| `move path="<from>" to="<to>"` | Move/rename note (auto-updates wikilinks and markdown links) |
| `delete file="<title>" [permanent]` | Move to .trash (or hard-delete) |
| `files [folder="<dir>"] [ext="<ext>"] [total]` | List vault files |
| `daily [date="YYYY-MM-DD"]` | Create or read daily note |

### Property (frontmatter) operations

| Command | Description |
|---------|-------------|
| `properties file="<title>"` | Show raw frontmatter block |
| `property:set file="<title>" name="<key>" value="<val>"` | Set or add a YAML property |
| `property:remove file="<title>" name="<key>"` | Remove a YAML property |

### Link operations

| Command | Description |
|---------|-------------|
| `backlinks file="<title>"` | Find notes linking to this note (includes embeds) |
| `links file="<title>"` | Show outgoing links (marks broken ones) |
| `orphans` | Find notes with no incoming links (alias-aware) |
| `unresolved` | Find all broken wikilinks across the vault |

### Tag operations

| Command | Description |
|---------|-------------|
| `tags [sort="count"] [counts]` | List all tags in vault |
| `tag tag="<tagname>"` | Find notes with tag or subtags |

### Task operations

| Command | Description |
|---------|-------------|
| `tasks [file="<title>"] [path="<dir>"] [done] [pending]` | List tasks (checkboxes) from one note or vault-wide |

### Template operations

| Command | Description |
|---------|-------------|
| `templates` | List available templates |
| `templates:apply template="<name>" name="<title>" path="<path>"` | Create note from template with variable substitution |

### Bookmark operations

| Command | Description |
|---------|-------------|
| `bookmarks` | List bookmarked file paths |
| `bookmarks:add file="<title>"` | Add a bookmark for a note |
| `bookmarks:remove file="<title>"` | Remove a bookmark |

### URI generation

| Command | Description |
|---------|-------------|
| `uri file="<title>" [heading="<H>"] [block="<B>"]` | Generate `obsidian://` URI for a note |

### Search

| Command | Description |
|---------|-------------|
| `search query="<term> [key:value]" [context="N"]` | Search by title, content, and frontmatter properties |
| `search regex="<pattern>" [context="N"]` | Search by regex (case-insensitive) |

When `context="N"` is provided, output switches to `file:line:content` format showing N lines before and after each match (similar to `grep -C`).

### Other

| Command | Description |
|---------|-------------|
| `vaults` | List all discovered Obsidian vaults |
| `help` | Show usage information |
| `version` | Print version |

## Features in depth

### Vault discovery

vlt reads Obsidian's configuration to discover your vaults automatically:

| Platform | Config location |
|----------|----------------|
| macOS | `~/Library/Application Support/obsidian/obsidian.json` |
| Linux | `~/.config/obsidian/obsidian.json` |
| Windows | `%APPDATA%\obsidian\obsidian.json` |

You can reference a vault three ways:

```bash
vlt vault="MyVault" ...          # by name (directory basename from config)
vlt vault="/absolute/path" ...   # by absolute path
vlt vault="~/Documents/vault" ...# by home-relative path
```

### Note resolution

Notes are resolved by a two-pass algorithm:

1. **Fast pass** -- exact filename match (`<title>.md`), no file I/O needed
2. **Alias pass** -- if no filename match, scan frontmatter `aliases` for a case-insensitive match

This means you can reference notes by their aliases just like in Obsidian:

```yaml
---
aliases: [PKM, Personal Knowledge Management]
---
```

```bash
vlt vault="MyVault" read file="PKM"  # resolves via alias
```

### Wikilink support

vlt understands all standard Obsidian wikilink formats:

| Format | Example |
|--------|---------|
| Simple link | `[[Note Title]]` |
| Link to heading | `[[Note Title#Section]]` |
| Block reference | `[[Note Title#^block-id]]` |
| Display text | `[[Note Title\|Custom Text]]` |
| Heading + display | `[[Note Title#Section\|Custom Text]]` |
| Block ref + display | `[[Note Title#^block-id\|Custom Text]]` |
| Embed | `![[Note Title]]` |
| Embed with heading + display | `![[Note Title#Section\|Custom Text]]` |

When you rename a note with `move`, vlt automatically updates both wikilinks and markdown-style links across the vault:

```bash
vlt vault="MyVault" move path="drafts/Old Name.md" to="published/New Name.md"
# Output:
# moved: drafts/Old Name.md -> published/New Name.md
# updated [[Old Name]] -> [[New Name]] in 12 file(s)
# updated [...](drafts/Old Name.md) -> [...](published/New Name.md) in 3 file(s)
```

Link updates preserve headings, block references, display text, and embed prefixes. Markdown links have their relative paths recomputed correctly. If only the folder changes (same filename), wikilink updates are skipped since Obsidian resolves by title regardless of path, but markdown links are always updated since they use paths.

### Content manipulation

`write` replaces the entire body of a note while preserving its frontmatter:

```bash
vlt vault="MyVault" write file="My Note" content="# New Body\nAll previous content replaced."
```

`patch` performs targeted edits by heading or line number:

```bash
# Replace a section's content under a heading
vlt vault="MyVault" patch file="Note" heading="## Architecture" content="New content for this section"

# Delete a section entirely
vlt vault="MyVault" patch file="Note" heading="## Old Section" delete

# Replace a single line
vlt vault="MyVault" patch file="Note" line="5" content="replacement line"

# Replace a line range
vlt vault="MyVault" patch file="Note" line="5-10" content="replacement block"

# Delete specific lines
vlt vault="MyVault" patch file="Note" line="5-10" delete
```

Both commands accept content from stdin when `content=` is omitted.

### Tag support

vlt collects tags from two sources, just like Obsidian:

**Frontmatter tags:**
```yaml
---
tags: [project, backend]
---
```

**Inline tags:**
```markdown
This is about #architecture and #design/patterns.
```

Tags are case-insensitive and deduplicated. Hierarchical tags support subtag matching:

```bash
vlt vault="MyVault" tag tag="design"
# Finds notes with #design, #design/patterns, #design/ux, etc.
```

### Regex search

In addition to plain-text search, vlt supports regex patterns:

```bash
# Find date patterns across the vault
vlt vault="MyVault" search regex="\d{4}-\d{2}-\d{2}"

# Regex with surrounding context (like grep -C)
vlt vault="MyVault" search regex="TODO|FIXME" context="2"

# Combine regex with property filters
vlt vault="MyVault" search regex="pattern" query="[status:active]"
```

Regex search is case-insensitive by default.

### Timestamps

Opt-in automatic management of `created_at` and `updated_at` frontmatter properties:

```bash
# Per-command opt-in
vlt vault="MyVault" create name="Note" path="_inbox/Note.md" content="# Note" timestamps
vlt vault="MyVault" append file="Note" content="more" timestamps

# Environment variable (applies to all write operations)
VLT_TIMESTAMPS=1 vlt vault="MyVault" write file="Note" content="# New Body"
```

On `create`, both `created_at` and `updated_at` are set to the current time. On all other write operations (`append`, `prepend`, `write`, `patch`), only `updated_at` is refreshed.

### Templates

vlt discovers template files from `.obsidian/templates.json` (the `folder` key) or falls back to a `templates/` directory in the vault root:

```bash
# List available templates
vlt vault="MyVault" templates

# Create a note from a template
vlt vault="MyVault" templates:apply template="Meeting Notes" name="Q1 Planning" path="meetings/Q1 Planning.md"
```

Template variable substitution supports `{{title}}`, `{{date}}`, `{{time}}`, and formatted variants like `{{date:YYYY-MM-DD}}` and `{{time:HH:mm}}` (Moment.js tokens translated to Go format).

### Bookmarks

Read and manage Obsidian's `.obsidian/bookmarks.json`:

```bash
vlt vault="MyVault" bookmarks              # list bookmarked paths
vlt vault="MyVault" bookmarks:add file="Important Note"
vlt vault="MyVault" bookmarks:remove file="Old Note"
```

Bookmarks are resolved by note title (same alias-aware resolution as all other commands). Groups in the bookmarks file are traversed recursively.

### URI generation

Generate `obsidian://` URIs for opening notes in the Obsidian app:

```bash
vlt vault="MyVault" uri file="Session Operating Mode"
# obsidian://open?vault=MyVault&file=Session%20Operating%20Mode

vlt vault="MyVault" uri file="Design Doc" heading="Architecture"
# obsidian://open?vault=MyVault&file=Design%20Doc&heading=Architecture
```

### Daily notes

Create or read daily notes following Obsidian's daily note conventions:

```bash
# Today's note (creates if missing, prints if exists)
vlt vault="MyVault" daily

# Specific date
vlt vault="MyVault" daily date="2025-01-15"
```

vlt reads configuration from `.obsidian/daily-notes.json` or `.obsidian/plugins/periodic-notes/data.json`, supporting custom folders, date formats (Moment.js tokens translated to Go), and templates with `{{date}}` and `{{title}}` variables.

### Stdin support

`create`, `append`, `prepend`, and `write` accept content from stdin when `content=` is omitted. This makes vlt composable with other Unix tools:

```bash
# Pipe output from another command
date | vlt vault="MyVault" append file="Daily Log"

# Use heredoc for multi-line content
vlt vault="MyVault" create name="Meeting" path="_inbox/Meeting.md" <<'EOF'
---
type: meeting
date: 2025-01-15
---
# Team Sync
- Discussed roadmap priorities
EOF
```

### Output formats

Most listing commands support `--json`, `--yaml`, `--csv`, `--tsv`, and `--tree` output for programmatic consumption:

```bash
# JSON output for scripts
vlt vault="MyVault" orphans --json
# ["_inbox/Stale Note.md","drafts/Abandoned.md"]

# CSV for spreadsheets
vlt vault="MyVault" tags counts --csv
# tag,count
# project,15
# architecture,8

# TSV (tab-separated) for shell pipelines
vlt vault="MyVault" tags counts --tsv

# YAML for config files
vlt vault="MyVault" search query="architecture" --yaml
# - title: System Architecture
#   path: decisions/System Architecture.md

# Tree view for directory structure
vlt vault="MyVault" files --tree
```

### Property-based search

Search queries can include `[key:value]` filters to match frontmatter properties:

```bash
# Find all active decisions
vlt vault="MyVault" search query="[status:active] [type:decision]"

# Text + property filter
vlt vault="MyVault" search query="architecture [status:active]"

# Property filter only (no text search)
vlt vault="MyVault" search query="[type:pattern]"
```

### Task parsing

vlt parses `- [ ]` and `- [x]` checkboxes from notes:

```bash
# All tasks across the vault
vlt vault="MyVault" tasks

# Tasks from a specific note
vlt vault="MyVault" tasks file="Project Plan"

# Only pending tasks in a folder
vlt vault="MyVault" tasks path="projects" pending

# JSON output for programmatic use
vlt vault="MyVault" tasks --json
```

### Output conventions

vlt follows Unix conventions for composability:

- One result per line (easy to pipe to `wc -l`, `grep`, `sort`, etc.)
- Relative paths from vault root
- Silent on empty results (exit code 0, no output -- like `grep`)
- Errors go to stderr with `vlt:` prefix
- Tab-separated fields where applicable (e.g., `tags counts`)

```bash
# Count orphan notes
vlt vault="MyVault" orphans | wc -l

# Find broken links in a specific folder
vlt vault="MyVault" unresolved | grep "^methodology/"

# List top 10 tags by frequency
vlt vault="MyVault" tags counts sort="count" | head -10
```

## Comparison with Obsidian CLI

vlt was built independently for agentic memory use cases, not as a replacement for the official [Obsidian CLI](https://github.com/Obsidian-CLI/obsidian-cli). The parameter syntax is intentionally compatible (`key="value"` style) so that switching between the two is straightforward where their features overlap.

| Capability | vlt | Obsidian CLI |
|------------|-----|--------------|
| read | Yes | Yes |
| read heading= (section extract) | Yes | No |
| search (with property filters) | Yes | Yes (no filters) |
| search regex= | Yes | No |
| search context=N | Yes | No |
| create | Yes | Yes |
| append | Yes | Yes |
| prepend | Yes | Yes |
| write (body replace, preserve frontmatter) | Yes | No |
| patch (heading/line targeted edit) | Yes | No |
| move (wiki + markdown link repair) | Yes | Yes (wiki only) |
| delete (trash + permanent) | Yes | Yes |
| files | Yes | Yes |
| daily notes | Yes | No |
| tasks | Yes | No |
| templates (list + apply with variables) | Yes | No |
| bookmarks (list + add + remove) | Yes | No |
| uri (obsidian:// URI generation) | Yes | No |
| properties | Yes | Yes |
| property:set | Yes | Yes |
| property:remove | Yes | Yes |
| backlinks | Yes | Yes |
| links | Yes | Yes |
| orphans | Yes | Yes |
| unresolved | Yes | Yes |
| tags (list + counts) | Yes | Yes |
| tag (search + hierarchical) | Yes | Yes |
| Alias resolution | Yes | Yes |
| Block references `#^block-id` | Yes | Yes |
| Embed `![[...]]` support | Yes | Yes |
| Inert zone masking (code, comments, math) | Yes | N/A (full parser) |
| Timestamps (created_at/updated_at) | Yes | No |
| Output formats (JSON/CSV/YAML/TSV/Tree) | Yes | No |
| Requires Obsidian running | **No** | Yes |
| External dependencies | **None** | Node.js |

## Architecture

vlt is a single-package Go binary with zero external dependencies. The entire tool runs on Go's standard library.

```
main.go          CLI entry point, argument parsing, command dispatch
vault.go         Vault discovery from Obsidian config, note resolution
commands.go      Command implementations (read, search, create, write, patch, move, etc.)
wikilinks.go     Wikilink/embed parsing, replacement, markdown link repair
frontmatter.go   YAML frontmatter extraction and manipulation
tags.go          Inline tag parsing and tag-based queries
format.go        Output formatting (JSON, CSV, YAML, TSV, tree, plain text)
inert.go         6-pass inert zone masking (code blocks, comments, math)
tasks.go         Task/checkbox parsing and queries
daily.go         Daily note creation and config loading
templates.go     Template discovery, variable substitution, note creation
bookmarks.go     Bookmark management via .obsidian/bookmarks.json
```

**Design choices:**

- **Zero dependencies** -- The `go.mod` has no `require` lines. This eliminates supply chain risk and keeps the binary small and fast to compile.
- **Direct filesystem access** -- All operations read and write files directly. No database, no index, no daemon.
- **Two-pass note resolution** -- Filename match first (no I/O), then alias scan (reads frontmatter). Fast for the common case, correct for the edge case.
- **Case-insensitive link matching** -- Mirrors Obsidian's behavior. `[[my note]]` resolves to `My Note.md`.
- **Simple frontmatter parsing** -- String-based YAML parsing handles Obsidian's common patterns (key-value, inline lists, block lists) without pulling in a full YAML library.
- **Inert zone masking** -- Before scanning for links, tags, or references, content inside code blocks, comments, and math expressions is masked out to prevent false positives. Each pass preserves byte offsets and line numbers so that all downstream operations remain position-accurate.

### Stats

| Metric | Value |
|--------|-------|
| Lines of code | ~4,100 (source) |
| Lines of tests | ~10,000 |
| Test functions | 308 |
| Test coverage | 80% |
| External dependencies | 0 |
| Go version | 1.24+ |

## Development

```bash
make build    # compile
make test     # run tests (verbose)
make install  # install to $GOPATH/bin
make clean    # remove build artifacts
```

### Running tests

```bash
go test -v ./...             # verbose output
go test -cover ./...         # with coverage
go test -run TestCmdMove ./... # run specific test
```

All tests use `t.TempDir()` for isolated vault environments. No mocks -- every test creates real files and exercises real filesystem operations.

### Adding a new command

1. Add the command name to `knownCommands` in `main.go`
2. Implement `cmdYourCommand(vaultDir string, params map[string]string) error` in `commands.go` (or a dedicated file for larger features)
3. Add the dispatch case in `main()` switch
4. Add usage line and examples in `usage()`
5. Write tests in `main_test.go` (or a dedicated `*_test.go` file)

## Contributing

Contributions are welcome. Please:

1. Open an issue describing the feature or bug before submitting a PR
2. Include tests for any new functionality
3. Keep the zero-dependency constraint -- no external modules
4. Follow the existing code style (simple, direct, no abstractions for one-off operations)
5. Run `make test` before submitting

## Roadmap

### Indexed full-text search (tantivy)

The current `search` command is a linear scan -- it reads every `.md` file in the vault on each query. For human-scale vaults (a few thousand notes) this is fast enough thanks to OS page cache. But vlt was built with AI agents in mind, and agents doing proper zettelkasten produce vaults that grow far beyond what a human would maintain by hand.

When demand warrants it, we plan to integrate [tantivy](https://github.com/quickwit-oss/tantivy) (the Rust full-text search engine that powers Quickwit and Meilisearch) to provide:

- Persistent inverted index with incremental updates
- Sub-millisecond search across arbitrarily large vaults
- Relevance-ranked results
- Fuzzy matching and phrase queries

This will be an opt-in feature -- the zero-dependency linear scan remains the default for simplicity. If this matters to you, open an issue or upvote an existing one.

### Recently shipped (v0.5.0)

- **Content manipulation** -- `write` (replace body preserving frontmatter), `patch` (heading-targeted or line-targeted replace/delete), `read heading=` (extract a single section)
- **Regex search** -- `search regex="pattern"` with case-insensitive matching; `context=N` for grep -C style surrounding lines
- **Inert zone masking** -- 6-pass system (fenced code, inline code, `%%` comments, HTML comments, display math, inline math) eliminates false positives in backlinks, links, orphans, unresolved, and tags
- **Templates** -- `templates` (list) and `templates:apply` with `{{title}}`, `{{date}}`, `{{time}}` variable substitution
- **Bookmarks** -- `bookmarks`, `bookmarks:add`, `bookmarks:remove` via `.obsidian/bookmarks.json`
- **URI generation** -- `uri` produces `obsidian://` URIs for opening notes in the app
- **Timestamps** -- opt-in `timestamps` flag (or `VLT_TIMESTAMPS=1`) auto-manages `created_at`/`updated_at` on all write operations
- **Output formats** -- `--tsv` and `--tree` added to existing `--json`/`--yaml`/`--csv`

### Previously shipped (v0.4.0)

- Block references (`[[Note#^block-id]]`) -- full support in parsing, rename, and backlinks
- Markdown link `[text](path.md)` repair on move -- relative paths recomputed correctly
- Property-based search filters (`search query="[status:active] [type:decision]"`)
- Output format flags (`--json`, `--yaml`, `--csv`) for all listing commands
- Daily note commands with Obsidian config support and templates
- Task/checkbox parsing with done/pending filters and vault-wide search

## License

Apache License 2.0. See [LICENSE](LICENSE) for full text.

Copyright 2025 Ramiro Salas.
