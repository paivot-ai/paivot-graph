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
	"path/filepath"

	"github.com/RamXX/paivot-graph/pvg-cli/internal/governance"
	"github.com/RamXX/paivot-graph/pvg-cli/internal/guard"
	"github.com/RamXX/paivot-graph/pvg-cli/internal/lifecycle"
	"github.com/RamXX/paivot-graph/pvg-cli/internal/settings"
	"github.com/RamXX/paivot-graph/pvg-cli/internal/vaultcfg"
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
		err = runHook(args[0])
	case "guard":
		err = runGuard()
	case "seed":
		force := len(args) > 0 && args[0] == "--force"
		err = runSeed(force)
	case "settings":
		err = settings.Run(args)
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

func runHook(name string) error {
	switch name {
	case "session-start":
		return lifecycle.SessionStart()
	case "pre-compact":
		return lifecycle.PreCompact()
	case "stop":
		return lifecycle.Stop()
	case "session-end":
		return lifecycle.SessionEnd()
	default:
		return fmt.Errorf("unknown hook %q", name)
	}
}

func runGuard() error {
	// Parse JSON from stdin
	input, err := guard.ParseInput()
	if err != nil {
		// If we can't parse, allow (don't block on parse failures)
		return nil
	}

	// Determine vault directory
	vaultDir, err := vaultcfg.VaultDir()
	if err != nil {
		// If vault isn't found, nothing to protect
		return nil
	}

	// Check the operation
	result := guard.Check(vaultDir, input)
	if !result.Allowed {
		fmt.Println(result.Reason)
		os.Exit(2)
	}

	return nil
}

func runSeed(force bool) error {
	pluginDir := os.Getenv("CLAUDE_PLUGIN_ROOT")
	if pluginDir == "" {
		// Try to find it relative to the pvg binary
		exe, err := os.Executable()
		if err == nil {
			// bin/pvg -> plugin root is ../
			candidate := filepath.Dir(filepath.Dir(exe))
			if _, serr := os.Stat(filepath.Join(candidate, ".claude-plugin")); serr == nil {
				pluginDir = candidate
			}
		}
	}
	return governance.Seed(force, pluginDir)
}
