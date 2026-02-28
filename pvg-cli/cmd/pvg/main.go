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
//	pvg loop setup [--all|--epic EPIC_ID] [--max-iterations N]
//	pvg loop cancel              # Cancel active loop
//	pvg loop status              # Show loop state
//	pvg version                  # Print version
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"strconv"

	"github.com/RamXX/paivot-graph/pvg-cli/internal/dispatcher"
	"github.com/RamXX/paivot-graph/pvg-cli/internal/governance"
	"github.com/RamXX/paivot-graph/pvg-cli/internal/guard"
	"github.com/RamXX/paivot-graph/pvg-cli/internal/lifecycle"
	"github.com/RamXX/paivot-graph/pvg-cli/internal/loop"
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
	case "loop":
		err = runLoop(args)
	case "dispatcher":
		err = runDispatcher(args)
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
  hook user-prompt       UserPromptSubmit hook (auto-detect dispatcher mode)
  hook subagent-start    SubagentStart hook (BLT agent tracking)
  hook subagent-stop     SubagentStop hook (BLT agent tracking)
  guard                  PreToolUse scope guard (reads JSON from stdin)
  loop setup [flags]     Start an execution loop (--all, --epic ID, --max-iterations N)
  loop cancel            Cancel active execution loop
  loop status            Show execution loop state
  dispatcher on|off|status  Manage dispatcher mode
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
	case "user-prompt":
		return lifecycle.UserPromptSubmit()
	case "subagent-start":
		return lifecycle.SubagentStart()
	case "subagent-stop":
		return lifecycle.SubagentStop()
	default:
		return fmt.Errorf("unknown hook %q", name)
	}
}

func runGuard() error {
	// Parse JSON from stdin -- fail-closed on parse errors to prevent
	// bypasses via malformed input.
	input, err := guard.ParseInput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pvg guard: failed to parse hook input: %v\n", err)
		os.Exit(2)
		return nil // unreachable, for the compiler
	}

	// Determine vault directory -- if vault isn't configured, nothing to
	// protect, so allow. This is intentional: the guard only activates when
	// a vault is present.
	vaultDir, err := vaultcfg.VaultDir()
	if err != nil {
		// No vault configured -- nothing to protect.
		return nil
	}

	// Get project root (CWD) for project vault checks
	cwd, _ := os.Getwd()

	// Check the operation
	result := guard.Check(vaultDir, cwd, input)
	if !result.Allowed {
		fmt.Fprintln(os.Stderr, result.Reason)
		os.Exit(2)
	}

	return nil
}

func runDispatcher(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pvg dispatcher on|off|status")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	switch args[0] {
	case "on":
		if err := dispatcher.On(cwd); err != nil {
			return fmt.Errorf("enable dispatcher mode: %w", err)
		}
		fmt.Println("Dispatcher mode enabled.")
	case "off":
		if err := dispatcher.Off(cwd); err != nil {
			return fmt.Errorf("disable dispatcher mode: %w", err)
		}
		fmt.Println("Dispatcher mode disabled.")
	case "status":
		dispatcher.Status(cwd)
	default:
		return fmt.Errorf("unknown dispatcher subcommand %q (use on|off|status)", args[0])
	}
	return nil
}

func runLoop(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: pvg loop setup|cancel|status")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	switch args[0] {
	case "setup":
		return loopSetup(cwd, args[1:])
	case "cancel":
		return loopCancel(cwd)
	case "status":
		return loopStatus(cwd)
	default:
		return fmt.Errorf("unknown loop subcommand %q (use setup|cancel|status)", args[0])
	}
}

func loopSetup(cwd string, args []string) error {
	// Parse flags manually (consistent with pvg pattern, no cobra)
	var (
		mode      = ""
		epicID    = ""
		maxIter   = 50 // default
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			mode = "all"
		case "--epic":
			if i+1 >= len(args) {
				return fmt.Errorf("--epic requires an argument")
			}
			i++
			epicID = args[i]
			mode = "epic"
		case "--max-iterations":
			if i+1 >= len(args) {
				return fmt.Errorf("--max-iterations requires an argument")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 0 {
				return fmt.Errorf("--max-iterations must be a non-negative integer")
			}
			maxIter = n
		default:
			// Positional argument -- treat as epic ID
			if mode == "" {
				epicID = args[i]
				mode = "epic"
			} else {
				return fmt.Errorf("unexpected argument: %s", args[i])
			}
		}
	}

	if mode == "" {
		return fmt.Errorf("specify --all or --epic EPIC_ID (or pass epic ID as positional arg)")
	}

	// Check not already active
	if loop.IsActive(cwd) {
		return fmt.Errorf("loop already active (use 'pvg loop cancel' first)")
	}

	// Validate epic if specified
	if mode == "epic" {
		if err := loop.ValidateEpic(epicID); err != nil {
			return fmt.Errorf("validate epic: %w", err)
		}
	}

	state := loop.NewState(mode, epicID, maxIter)
	if err := loop.WriteState(cwd, state); err != nil {
		return fmt.Errorf("write loop state: %w", err)
	}

	fmt.Println("[LOOP] Execution loop activated.")
	fmt.Printf("  Mode: %s\n", mode)
	if epicID != "" {
		fmt.Printf("  Target: %s\n", epicID)
	}
	if maxIter > 0 {
		fmt.Printf("  Max iterations: %d\n", maxIter)
	} else {
		fmt.Println("  Max iterations: unlimited")
	}
	return nil
}

func loopCancel(cwd string) error {
	if !loop.IsActive(cwd) {
		fmt.Println("[LOOP] No active loop to cancel.")
		return nil
	}

	state, _ := loop.ReadState(cwd)
	if err := loop.RemoveState(cwd); err != nil {
		return fmt.Errorf("remove loop state: %w", err)
	}

	fmt.Println("[LOOP] Execution loop cancelled.")
	if state != nil {
		fmt.Printf("  Completed iterations: %d\n", state.Iteration)
		fmt.Printf("  Wait iterations: %d\n", state.WaitIterations)
	}
	return nil
}

func loopStatus(cwd string) error {
	state, err := loop.ReadState(cwd)
	if err != nil {
		fmt.Println("[LOOP] No active loop.")
		return nil
	}

	if !state.Active {
		fmt.Println("[LOOP] Loop state exists but is inactive.")
		return nil
	}

	fmt.Println("[LOOP] Execution loop active.")
	fmt.Printf("  Mode: %s\n", state.Mode)
	if state.TargetEpic != "" {
		fmt.Printf("  Target: %s\n", state.TargetEpic)
	}
	fmt.Printf("  Iteration: %d", state.Iteration)
	if state.MaxIterations > 0 {
		fmt.Printf(" / %d", state.MaxIterations)
	}
	fmt.Println()
	fmt.Printf("  Consecutive waits: %d / %d\n", state.ConsecutiveWaits, state.MaxConsecutiveWaits)
	fmt.Printf("  Total wait iterations: %d\n", state.WaitIterations)
	fmt.Printf("  Started: %s\n", state.StartedAt)
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
