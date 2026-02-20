// vlt -- fast Obsidian vault CLI (no app required)
//
// Drop-in replacement for the obsidian CLI that operates directly on the
// filesystem. No Obsidian app dependency, no Electron round-trips.
//
// Discovers vaults from the Obsidian config file, resolves notes by title
// or alias, and performs file, property, link, and tag operations.
package main

import (
	"fmt"
	"os"
	"strings"
)

const version = "0.5.0"

var knownCommands = map[string]bool{
	"read": true, "search": true, "create": true,
	"append": true, "prepend": true, "write": true, "patch": true, "move": true, "delete": true,
	"property:set": true, "property:remove": true, "properties": true,
	"backlinks": true, "links": true, "orphans": true, "unresolved": true,
	"tags": true, "tag": true, "files": true,
	"tasks": true, "daily": true,
	"vaults": true, "help": true, "version": true,
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd, params, flags := parseArgs(os.Args[1:])

	if cmd == "help" || flags["--help"] || flags["-h"] {
		usage()
		return
	}
	if cmd == "version" || flags["--version"] {
		fmt.Println("vlt " + version)
		return
	}
	format := outputFormat(flags)

	if cmd == "vaults" {
		if err := cmdVaults(format); err != nil {
			die("%v", err)
		}
		return
	}
	if cmd == "" {
		die("no command specified. Run 'vlt help' for usage.")
	}

	// Resolve vault
	vaultName := params["vault"]
	if vaultName == "" {
		vaultName = os.Getenv("VLT_VAULT")
	}
	if vaultName == "" {
		die("vault not specified. Use vault=\"<name>\" or set VLT_VAULT env var.")
	}

	vaultDir, err := resolveVault(vaultName)
	if err != nil {
		die("%v", err)
	}

	ts := flags["timestamps"]

	// Dispatch
	switch cmd {
	case "read":
		err = cmdRead(vaultDir, params)
	case "search":
		err = cmdSearch(vaultDir, params, format)
	case "create":
		err = cmdCreate(vaultDir, params, flags["silent"], ts)
	case "append":
		err = cmdAppend(vaultDir, params, ts)
	case "prepend":
		err = cmdPrepend(vaultDir, params, ts)
	case "write":
		err = cmdWrite(vaultDir, params, ts)
	case "patch":
		err = cmdPatch(vaultDir, params, flags["delete"], ts)
	case "move":
		err = cmdMove(vaultDir, params)
	case "delete":
		err = cmdDelete(vaultDir, params, flags["permanent"])
	case "property:set":
		err = cmdPropertySet(vaultDir, params)
	case "property:remove":
		err = cmdPropertyRemove(vaultDir, params)
	case "properties":
		err = cmdProperties(vaultDir, params, format)
	case "backlinks":
		err = cmdBacklinks(vaultDir, params, format)
	case "links":
		err = cmdLinks(vaultDir, params, format)
	case "orphans":
		err = cmdOrphans(vaultDir, format)
	case "unresolved":
		err = cmdUnresolved(vaultDir, format)
	case "tags":
		err = cmdTags(vaultDir, params, flags["counts"], format)
	case "tag":
		err = cmdTag(vaultDir, params, format)
	case "files":
		err = cmdFiles(vaultDir, params, flags["total"], format)
	case "tasks":
		err = cmdTasks(vaultDir, params, flags)
	case "daily":
		err = cmdDaily(vaultDir, params)
	default:
		die("unknown command: %s", cmd)
	}

	if err != nil {
		die("%v", err)
	}
}

// parseArgs splits CLI arguments into a command name, key=value parameters,
// and bare-word flags. It preserves the obsidian CLI's key="value" syntax.
func parseArgs(args []string) (string, map[string]string, map[string]bool) {
	params := make(map[string]string)
	flags := make(map[string]bool)
	var cmd string

	for _, arg := range args {
		if i := strings.Index(arg, "="); i > 0 {
			key := arg[:i]
			val := arg[i+1:]
			// Strip surrounding quotes (shouldn't be needed after shell parsing,
			// but handles edge cases like programmatic invocation).
			val = strings.Trim(val, "\"'")
			params[key] = val
		} else if knownCommands[arg] {
			cmd = arg
		} else {
			flags[arg] = true
		}
	}

	return cmd, params, flags
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "vlt: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`vlt -- fast Obsidian vault CLI (no app required)

Usage:
  vlt vault="<name>" <command> [args...]

File commands:
  read           file="<title>" [heading="<heading>"]         Read a note (or a specific section)
  create         name="<title>" path="<path>" [content=...] [silent] [timestamps]  Create a note
  append         file="<title>" [content="<text>"] [timestamps]      Append to end of note
  prepend        file="<title>" [content="<text>"] [timestamps]      Prepend after frontmatter
  write          file="<title>" [content="<text>"] [timestamps]      Replace body (preserve frontmatter)
  patch          file="<title>" heading="<heading>" [content="<text>"] [delete] [timestamps]  Section edit
  patch          file="<title>" line="<N>" [content="<text>"] [delete] [timestamps]           Line edit
  patch          file="<title>" line="<N-M>" [content="<text>"] [delete] [timestamps]         Line range edit
  move           path="<from>" to="<to>"                     Move/rename (updates wiki + md links)
  delete         file="<title>" [permanent]                  Trash (or permanently delete)
  files          [folder="<dir>"] [ext="<ext>"] [total]      List vault files
  daily          [date="YYYY-MM-DD"]                         Create or read daily note

Property commands:
  properties     file="<title>"                              Show all frontmatter
  property:set   file="<title>" name="<key>" value="<val>"   Set a frontmatter property
  property:remove file="<title>" name="<key>"                Remove a frontmatter property

Link commands:
  backlinks      file="<title>"                              Notes linking to this note
  links          file="<title>"                              Outgoing links (flags broken)
  orphans                                                    Notes with no incoming links
  unresolved                                                 Broken links across vault

Tag commands:
  tags           [sort="count"] [counts]                     List all tags in vault
  tag            tag="<tagname>"                             Find notes with tag (+ subtags)

Task commands:
  tasks          [file="<title>"] [path="<dir>"] [done] [pending]  List tasks (checkboxes)

Search:
  search         query="<term> [key:value]" [context="N"]    Search by title, content, properties
  search         regex="<pattern>" [context="N"]              Search by regex (case-insensitive)
                                                              context=N shows N lines before/after each match

Other:
  vaults                                                     List discovered vaults

Options:
  vault="<name>"   Vault name (from Obsidian config), absolute path, or VLT_VAULT env var.
  silent           Suppress output on create.
  permanent        Hard delete instead of .trash.
  delete           Remove heading+content or line(s) instead of replacing (patch).
  timestamps       Auto-manage created_at/updated_at frontmatter (or set VLT_TIMESTAMPS=1).
  counts           Show note counts with tags.
  total            Show count instead of listing files.
  done             Show only completed tasks.
  pending          Show only pending tasks.
  --json           Output in JSON format.
  --yaml           Output in YAML format.
  --csv            Output in CSV format.
  --tsv            Output in TSV (tab-separated values) format.
  --tree           Output file lists as a hierarchical directory tree.

Content from stdin:
  If content= is omitted for create/append/prepend/write, content is read from stdin.

Search filters:
  Property filters can be embedded in search queries: query="term [key:value]"
  Multiple filters: query="architecture [status:active] [type:decision]"
  Filter-only: query="[status:active]"
  Regex search: regex="arch\w+ure" (case-insensitive by default)
  Regex + filters: regex="pattern" query="[status:active]"
  If both query= and regex= provide text, regex takes precedence (with a warning).

Wikilink support:
  [[Note]], [[Note#Heading]], [[Note#^block-id]], [[Note|Display]], ![[Embed]]
  Block references (^block-id) are fully supported in parsing, rename, and backlinks.

Examples:
  vlt vault="Claude" read file="Session Operating Mode"
  vlt vault="Claude" read file="Design Doc" heading="## Architecture"
  vlt vault="Claude" search query="architecture"
  vlt vault="Claude" search query="[status:active] [type:decision]"
  vlt vault="Claude" create name="My Note" path="_inbox/My Note.md" content="# Hello" silent
  echo "## Update" | vlt vault="Claude" append file="My Note"
  vlt vault="Claude" prepend file="My Note" content="New section at top"
  vlt vault="Claude" write file="My Note" content="# Replacement body"
  vlt vault="Claude" patch file="Note" heading="## Section" content="new content"
  vlt vault="Claude" patch file="Note" heading="## Section" delete
  vlt vault="Claude" patch file="Note" line="5" content="replacement line"
  vlt vault="Claude" patch file="Note" line="5-10" content="replacement block"
  vlt vault="Claude" patch file="Note" line="5" delete
  vlt vault="Claude" move path="_inbox/Old.md" to="decisions/New.md"
  vlt vault="Claude" delete file="Old Draft"
  vlt vault="Claude" delete file="Old Draft" permanent
  vlt vault="Claude" properties file="My Decision"
  vlt vault="Claude" property:set file="Note" name="status" value="archived"
  vlt vault="Claude" property:remove file="Note" name="confidence"
  vlt vault="Claude" backlinks file="Session Operating Mode"
  vlt vault="Claude" links file="Developer Agent"
  vlt vault="Claude" orphans
  vlt vault="Claude" unresolved
  vlt vault="Claude" tags counts sort="count"
  vlt vault="Claude" tag tag="project"
  vlt vault="Claude" files folder="methodology"
  vlt vault="Claude" files total
  vlt vault="Claude" tasks
  vlt vault="Claude" tasks file="Project Plan" pending
  vlt vault="Claude" tasks path="projects" --json
  vlt vault="Claude" daily
  vlt vault="Claude" daily date="2025-01-15"
  vlt vault="Claude" orphans --json
  vlt vault="Claude" search query="architecture" --csv
  vlt vault="Claude" search query="architecture" context="2"
  vlt vault="Claude" search query="architecture [status:active]" context="1" --json
  vlt vault="Claude" search regex="arch\w+ure"
  vlt vault="Claude" search regex="\d{4}-\d{2}-\d{2}" context="2"
  vlt vault="Claude" search regex="pattern" query="[status:active]"
  vlt vault="Claude" create name="Note" path="_inbox/Note.md" content="# Note" timestamps
  vlt vault="Claude" append file="Note" content="more" timestamps
  VLT_TIMESTAMPS=1 vlt vault="Claude" write file="Note" content="# New Body"
  vlt vaults
`)
}
