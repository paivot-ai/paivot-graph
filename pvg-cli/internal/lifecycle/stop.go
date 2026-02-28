package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/RamXX/paivot-graph/pvg-cli/internal/loop"
	"github.com/RamXX/paivot-graph/pvg-cli/internal/vaultcfg"
)

// Stop outputs a knowledge capture reminder when Claude tries to stop.
// If an execution loop is active, it evaluates whether to continue or allow exit.
// Reads the Stop Capture Checklist from the vault or uses a static fallback.
func Stop() error {
	cwd, _ := os.Getwd()

	// Loop check: if active, handle loop logic and return early
	if loop.IsActive(cwd) {
		return checkLoop(cwd)
	}

	v, err := vaultcfg.OpenVault()
	if err == nil {
		result, rerr := v.Read("Stop Capture Checklist", "")
		if rerr == nil && result.Content != "" {
			fmt.Println("[VAULT] Stop capture check (from vault):")
			fmt.Println()
			fmt.Println(result.Content)
			outputTwoTierReminder()
			return nil
		}
	}

	// Try direct file read
	vaultDir, derr := vaultcfg.VaultDir()
	if derr == nil {
		path := filepath.Join(vaultDir, "conventions", "Stop Capture Checklist.md")
		data, ferr := os.ReadFile(path)
		if ferr == nil && len(data) > 0 {
			fmt.Println("[VAULT] Stop capture check (from vault):")
			fmt.Println()
			fmt.Println(string(data))
			outputTwoTierReminder()
			return nil
		}
	}

	// Static fallback
	fmt.Print(staticStopChecklist())
	outputTwoTierReminder()
	return nil
}

func outputTwoTierReminder() {
	cwd, _ := os.Getwd()
	knowledgeDir := filepath.Join(cwd, ".vault", "knowledge")
	if info, err := os.Stat(knowledgeDir); err == nil && info.IsDir() {
		fmt.Print(`
[VAULT] Remember: save to the right tier.
  - Universal insights -> global vault (_inbox/)
  - Project-specific insights -> .vault/knowledge/ (local)
`)
	}
}

func staticStopChecklist() string {
	return `[VAULT] Stop capture check:

Before ending this session, confirm you have considered each of these:

- [ ] Did you capture any DECISIONS made this session?
- [ ] Did you capture any PATTERNS discovered?
- [ ] Did you capture any DEBUG INSIGHTS?
- [ ] Did you update the PROJECT INDEX NOTE?
- [ ] Did you capture project-specific knowledge to .vault/knowledge/?

If none apply (trivial session), that is fine -- but confirm it was considered.

Use: vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="..." silent
`
}

// checkLoop evaluates whether the execution loop should continue or allow exit.
// On block: updates state and emits continuation JSON to stdout.
// On allow: logs reason, removes state if needed.
func checkLoop(cwd string) error {
	state, err := loop.ReadState(cwd)
	if err != nil {
		// State disappeared -- fail open, allow exit
		fmt.Fprintln(os.Stderr, "[LOOP] Could not read loop state, allowing exit")
		return nil
	}

	// Query nd for work counts -- fail open on error
	wc, err := loop.QueryWorkCounts(state.Mode, state.TargetEpic)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[LOOP] Could not query nd: %v -- allowing exit\n", err)
		_ = loop.RemoveState(cwd)
		return nil
	}

	cfg := loop.StopConfig{
		Active:         state.Active,
		Mode:           state.Mode,
		TargetEpic:     state.TargetEpic,
		Iteration:      state.Iteration,
		MaxIterations:  state.MaxIterations,
		ConsecWaits:    state.ConsecutiveWaits,
		MaxConsecWaits: state.MaxConsecutiveWaits,
		WaitIterations: state.WaitIterations,
		Ready:          wc.Ready,
		Delivered:      wc.Delivered,
		InProgress:     wc.InProgress,
		Blocked:        wc.Blocked,
	}

	decision := loop.EvaluateStop(cfg)

	if decision.Allow {
		fmt.Fprintf(os.Stderr, "[LOOP] %s\n", decision.Reason)
		if decision.RemoveState {
			_ = loop.RemoveState(cwd)
		}
		return nil
	}

	// Block exit: update state and emit continuation JSON
	state.Iteration = decision.NewIteration
	state.ConsecutiveWaits = decision.NewConsecWaits
	state.WaitIterations = decision.NewWaitIters
	if err := loop.WriteState(cwd, state); err != nil {
		fmt.Fprintf(os.Stderr, "[LOOP] Could not update state: %v -- allowing exit\n", err)
		return nil
	}

	// Build and emit continuation
	maxIterStr := "unlimited"
	if state.MaxIterations > 0 {
		maxIterStr = strconv.Itoa(state.MaxIterations)
	}
	prompt := buildContinuationPrompt(state, &decision, maxIterStr, &wc)

	continuation := map[string]any{
		"decision": "block",
		"reason":   decision.Reason,
		"options": []map[string]string{
			{"value": prompt},
		},
	}

	data, err := json.Marshal(continuation)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[LOOP] Could not marshal continuation: %v\n", err)
		return nil
	}

	fmt.Println(string(data))
	return nil
}

// buildContinuationPrompt creates the prompt for the next loop iteration.
func buildContinuationPrompt(state *loop.State, decision *loop.StopDecision, maxIterStr string, wc *loop.WorkCounts) string {
	prompt := fmt.Sprintf(
		"[LOOP] Iteration %d/%s | Ready: %d, Delivered: %d, In-progress: %d, Blocked: %d | %s\n\n",
		decision.NewIteration, maxIterStr,
		wc.Ready, wc.Delivered, wc.InProgress, wc.Blocked,
		decision.Reason,
	)

	prompt += "Continue the execution loop. Priority order:\n"
	prompt += "1. PM-Acceptor for delivered stories (nd list --status in_progress --label delivered --json)\n"
	prompt += "2. Developer for rejected stories (nd list --status in_progress --label rejected --json)\n"
	prompt += "3. Developer for ready stories (nd ready --json)\n\n"
	prompt += "Concurrency: max 2 developer agents, max 1 PM agent, max 3 total.\n"
	prompt += "You are dispatcher-only: spawn agents, do not write code or fix errors yourself.\n"

	if state.Mode == "epic" && state.TargetEpic != "" {
		prompt += fmt.Sprintf("Scope: epic %s only.\n", state.TargetEpic)
	}

	return prompt
}
