// pvg is the paivot-graph CLI -- a deterministic replacement for shell hooks
// and scripts. It uses vlt as a library for all vault operations, encoding
// scope guards, proposal workflow, and session lifecycle in Go.
//
// This replaces: vault-scope-guard.sh, vault-session-start.sh,
// vault-pre-compact.sh, vault-stop.sh, vault-session-end.sh, seed-vault.sh
//
// Usage:
//
//	pvg hook session-start       # SessionStart hook
//	pvg hook pre-compact         # PreCompact hook
//	pvg hook stop                # Stop hook
//	pvg hook session-end         # SessionEnd hook
//	pvg guard                    # PreToolUse scope guard (stdin: JSON)
//	pvg seed [--force]           # Seed vault with agent prompts
//	pvg settings [key=value]     # View/set project settings
//	pvg version                  # Print version
package main

import (
	"fmt"
	"os"
)

// Set at build time via -ldflags
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "hook":
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "pvg hook: missing subcommand (session-start, pre-compact, stop, session-end)")
			os.Exit(1)
		}
		err = runHook(args[0], args[1:])
	case "guard":
		err = runGuard()
	case "seed":
		force := len(args) > 0 && args[0] == "--force"
		err = runSeed(force)
	case "settings":
		err = runSettings(args)
	case "version":
		fmt.Printf("pvg %s\n", version)
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "pvg: unknown command %q\n", cmd)
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "pvg %s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `pvg -- paivot-graph CLI

Commands:
  hook session-start     SessionStart lifecycle hook
  hook pre-compact       PreCompact lifecycle hook
  hook stop              Stop lifecycle hook
  hook session-end       SessionEnd lifecycle hook
  guard                  PreToolUse scope guard (reads JSON from stdin)
  seed [--force]         Seed vault with agent prompts and conventions
  settings [key=value]   View or set project settings
  version                Print version
  help                   Show this help`)
}

// Stubs -- each will be implemented in its own file

func runHook(name string, args []string) error {
	switch name {
	case "session-start":
		return hookSessionStart()
	case "pre-compact":
		return hookPreCompact()
	case "stop":
		return hookStop()
	case "session-end":
		return hookSessionEnd()
	default:
		return fmt.Errorf("unknown hook %q", name)
	}
}

func runGuard() error {
	// TODO: implement -- reads JSON from stdin, checks scope
	return fmt.Errorf("not yet implemented")
}

func runSeed(force bool) error {
	// TODO: implement -- seeds vault notes
	return fmt.Errorf("not yet implemented")
}

func runSettings(args []string) error {
	// TODO: implement -- reads/writes .vault/knowledge/.settings.yaml
	return fmt.Errorf("not yet implemented")
}

func hookSessionStart() error {
	// TODO: implement
	return fmt.Errorf("not yet implemented")
}

func hookPreCompact() error {
	// TODO: implement
	return fmt.Errorf("not yet implemented")
}

func hookStop() error {
	// TODO: implement
	return fmt.Errorf("not yet implemented")
}

func hookSessionEnd() error {
	// TODO: implement
	return fmt.Errorf("not yet implemented")
}
