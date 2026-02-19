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

const version = "0.3.0"

var knownCommands = map[string]bool{
	"read": true, "search": true, "create": true,
	"append": true, "prepend": true, "move": true, "delete": true,
	"property:set": true, "property:remove": true, "properties": true,
	"backlinks": true, "links": true, "orphans": true, "unresolved": true,
	"tags": true, "tag": true, "files": true,
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
	if cmd == "vaults" {
		if err := cmdVaults(); err != nil {
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

	// Dispatch
	switch cmd {
	case "read":
		err = cmdRead(vaultDir, params)
	case "search":
		err = cmdSearch(vaultDir, params)
	case "create":
		err = cmdCreate(vaultDir, params, flags["silent"])
	case "append":
		err = cmdAppend(vaultDir, params)
	case "prepend":
		err = cmdPrepend(vaultDir, params)
	case "move":
		err = cmdMove(vaultDir, params)
	case "delete":
		err = cmdDelete(vaultDir, params, flags["permanent"])
	case "property:set":
		err = cmdPropertySet(vaultDir, params)
	case "property:remove":
		err = cmdPropertyRemove(vaultDir, params)
	case "properties":
		err = cmdProperties(vaultDir, params)
	case "backlinks":
		err = cmdBacklinks(vaultDir, params)
	case "links":
		err = cmdLinks(vaultDir, params)
	case "orphans":
		err = cmdOrphans(vaultDir)
	case "unresolved":
		err = cmdUnresolved(vaultDir)
	case "tags":
		err = cmdTags(vaultDir, params, flags["counts"])
	case "tag":
		err = cmdTag(vaultDir, params)
	case "files":
		err = cmdFiles(vaultDir, params, flags["total"])
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
  read           file="<title>"                              Read a note by title (or alias)
  create         name="<title>" path="<path>" [content=...] [silent]  Create a note
  append         file="<title>" [content="<text>"]           Append to end of note
  prepend        file="<title>" [content="<text>"]           Prepend after frontmatter
  move           path="<from>" to="<to>"                     Move/rename (updates wikilinks)
  delete         file="<title>" [permanent]                  Trash (or permanently delete)
  files          [folder="<dir>"] [ext="<ext>"] [total]      List vault files

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

Search:
  search         query="<term>"                              Search by title and content

Other:
  vaults                                                     List discovered vaults

Options:
  vault="<name>"   Vault name (from Obsidian config), absolute path, or VLT_VAULT env var.
  silent           Suppress output on create.
  permanent        Hard delete instead of .trash.
  counts           Show note counts with tags.
  total            Show count instead of listing files.

Content from stdin:
  If content= is omitted for create/append/prepend, content is read from stdin.

Examples:
  vlt vault="Claude" read file="Session Operating Mode"
  vlt vault="Claude" search query="paivot"
  vlt vault="Claude" create name="My Note" path="_inbox/My Note.md" content="# Hello" silent
  echo "## Update" | vlt vault="Claude" append file="My Note"
  vlt vault="Claude" prepend file="My Note" content="New section at top"
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
  vlt vaults
`)
}
