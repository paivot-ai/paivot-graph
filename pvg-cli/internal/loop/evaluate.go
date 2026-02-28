package loop

// StopConfig holds all inputs needed for the stop decision.
// This is a value struct -- no I/O, no side effects.
type StopConfig struct {
	Active         bool
	Mode           string
	TargetEpic     string
	Iteration      int
	MaxIterations  int // 0 = unlimited
	ConsecWaits    int
	MaxConsecWaits int
	WaitIterations int
	Ready          int
	Delivered      int
	InProgress     int
	Blocked        int
}

// StopDecision is the output of EvaluateStop.
type StopDecision struct {
	Allow         bool   // true = allow session exit
	Reason        string // human-readable explanation
	RemoveState   bool   // true = clean up state file on exit
	NewIteration  int    // updated iteration count
	NewConsecWaits int   // updated consecutive wait count
	NewWaitIters  int    // updated total wait iterations
}

// EvaluateStop is a pure function that decides whether to allow session exit
// or block it (continuing the loop). No I/O -- all context comes from cfg.
func EvaluateStop(cfg StopConfig) StopDecision {
	// Not active -- always allow
	if !cfg.Active {
		return StopDecision{
			Allow:  true,
			Reason: "Loop not active",
		}
	}

	nextIter := cfg.Iteration + 1

	// Max iterations reached
	if cfg.MaxIterations > 0 && nextIter >= cfg.MaxIterations {
		return StopDecision{
			Allow:        true,
			Reason:       "Max iterations reached",
			RemoveState:  true,
			NewIteration: nextIter,
		}
	}

	actionable := cfg.Ready + cfg.Delivered
	total := actionable + cfg.InProgress + cfg.Blocked

	// All work complete (nothing anywhere)
	if total == 0 {
		return StopDecision{
			Allow:        true,
			Reason:       "All work complete",
			RemoveState:  true,
			NewIteration: nextIter,
		}
	}

	// All remaining work is blocked
	if actionable == 0 && cfg.InProgress == 0 && cfg.Blocked > 0 {
		return StopDecision{
			Allow:        true,
			Reason:       "All remaining work is blocked",
			RemoveState:  true,
			NewIteration: nextIter,
		}
	}

	// Wait-like: nothing actionable but work is in progress (agents running)
	if actionable == 0 && cfg.InProgress > 0 {
		newConsec := cfg.ConsecWaits + 1
		newWaitIters := cfg.WaitIterations + 1

		if newConsec >= cfg.MaxConsecWaits {
			return StopDecision{
				Allow:          true,
				Reason:         "Too many consecutive wait iterations",
				RemoveState:    true,
				NewIteration:   nextIter,
				NewConsecWaits: newConsec,
				NewWaitIters:   newWaitIters,
			}
		}

		return StopDecision{
			Allow:          false,
			Reason:         "Waiting for in-progress work to complete",
			NewIteration:   nextIter,
			NewConsecWaits: newConsec,
			NewWaitIters:   newWaitIters,
		}
	}

	// Actionable work exists -- block exit, reset consecutive waits
	return StopDecision{
		Allow:          false,
		Reason:         "Actionable work remains",
		NewIteration:   nextIter,
		NewConsecWaits: 0,
		NewWaitIters:   cfg.WaitIterations,
	}
}
