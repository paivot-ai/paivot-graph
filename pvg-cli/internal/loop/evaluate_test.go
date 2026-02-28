package loop

import "testing"

func TestEvaluateStop_NotActive(t *testing.T) {
	d := EvaluateStop(StopConfig{Active: false})
	if !d.Allow {
		t.Error("expected allow when not active")
	}
	if d.RemoveState {
		t.Error("should not remove state when not active")
	}
}

func TestEvaluateStop_MaxIterations(t *testing.T) {
	d := EvaluateStop(StopConfig{
		Active:        true,
		Iteration:     49,
		MaxIterations: 50,
		Ready:         5, // actionable work exists
	})
	if !d.Allow {
		t.Error("expected allow at max iterations")
	}
	if !d.RemoveState {
		t.Error("expected remove state at max iterations")
	}
	if d.Reason != "Max iterations reached" {
		t.Errorf("unexpected reason: %s", d.Reason)
	}
}

func TestEvaluateStop_UnlimitedIterations(t *testing.T) {
	d := EvaluateStop(StopConfig{
		Active:        true,
		Iteration:     999,
		MaxIterations: 0, // unlimited
		Ready:         1,
	})
	if d.Allow {
		t.Error("expected block with unlimited iterations and ready work")
	}
}

func TestEvaluateStop_AllComplete(t *testing.T) {
	d := EvaluateStop(StopConfig{
		Active:        true,
		Iteration:     5,
		MaxIterations: 50,
	})
	if !d.Allow {
		t.Error("expected allow when all complete")
	}
	if !d.RemoveState {
		t.Error("expected remove state when all complete")
	}
	if d.Reason != "All work complete" {
		t.Errorf("unexpected reason: %s", d.Reason)
	}
}

func TestEvaluateStop_AllBlocked(t *testing.T) {
	d := EvaluateStop(StopConfig{
		Active:        true,
		Iteration:     3,
		MaxIterations: 50,
		Blocked:       4,
	})
	if !d.Allow {
		t.Error("expected allow when all blocked")
	}
	if !d.RemoveState {
		t.Error("expected remove state when all blocked")
	}
	if d.Reason != "All remaining work is blocked" {
		t.Errorf("unexpected reason: %s", d.Reason)
	}
}

func TestEvaluateStop_ActionableReady(t *testing.T) {
	d := EvaluateStop(StopConfig{
		Active:        true,
		Iteration:     2,
		MaxIterations: 50,
		Ready:         3,
	})
	if d.Allow {
		t.Error("expected block with ready work")
	}
	if d.NewConsecWaits != 0 {
		t.Errorf("expected consec waits reset to 0, got %d", d.NewConsecWaits)
	}
	if d.Reason != "Actionable work remains" {
		t.Errorf("unexpected reason: %s", d.Reason)
	}
}

func TestEvaluateStop_ActionableDelivered(t *testing.T) {
	d := EvaluateStop(StopConfig{
		Active:        true,
		Iteration:     2,
		MaxIterations: 50,
		Delivered:     2,
	})
	if d.Allow {
		t.Error("expected block with delivered work")
	}
}

func TestEvaluateStop_ActionableMixed(t *testing.T) {
	d := EvaluateStop(StopConfig{
		Active:        true,
		Iteration:     2,
		MaxIterations: 50,
		Ready:         1,
		Delivered:     1,
		InProgress:    2,
		Blocked:       1,
	})
	if d.Allow {
		t.Error("expected block with mixed actionable work")
	}
	if d.NewConsecWaits != 0 {
		t.Errorf("expected consec waits reset, got %d", d.NewConsecWaits)
	}
}

func TestEvaluateStop_WaitLike_FirstWait(t *testing.T) {
	d := EvaluateStop(StopConfig{
		Active:         true,
		Iteration:      5,
		MaxIterations:  50,
		ConsecWaits:    0,
		MaxConsecWaits: 3,
		WaitIterations: 0,
		InProgress:     2,
	})
	if d.Allow {
		t.Error("expected block on first wait")
	}
	if d.NewConsecWaits != 1 {
		t.Errorf("expected consec waits=1, got %d", d.NewConsecWaits)
	}
	if d.NewWaitIters != 1 {
		t.Errorf("expected wait iters=1, got %d", d.NewWaitIters)
	}
}

func TestEvaluateStop_WaitLike_SecondWait(t *testing.T) {
	d := EvaluateStop(StopConfig{
		Active:         true,
		Iteration:      6,
		MaxIterations:  50,
		ConsecWaits:    1,
		MaxConsecWaits: 3,
		WaitIterations: 1,
		InProgress:     2,
	})
	if d.Allow {
		t.Error("expected block on second wait")
	}
	if d.NewConsecWaits != 2 {
		t.Errorf("expected consec waits=2, got %d", d.NewConsecWaits)
	}
	if d.NewWaitIters != 2 {
		t.Errorf("expected wait iters=2, got %d", d.NewWaitIters)
	}
}

func TestEvaluateStop_WaitLike_ThresholdReached(t *testing.T) {
	d := EvaluateStop(StopConfig{
		Active:         true,
		Iteration:      7,
		MaxIterations:  50,
		ConsecWaits:    2,
		MaxConsecWaits: 3,
		WaitIterations: 2,
		InProgress:     2,
	})
	if !d.Allow {
		t.Error("expected allow at wait threshold")
	}
	if !d.RemoveState {
		t.Error("expected remove state at wait threshold")
	}
	if d.NewConsecWaits != 3 {
		t.Errorf("expected consec waits=3, got %d", d.NewConsecWaits)
	}
}

func TestEvaluateStop_ConsecWaitsReset_AfterActionable(t *testing.T) {
	// First: wait (in-progress only)
	d1 := EvaluateStop(StopConfig{
		Active:         true,
		Iteration:      5,
		MaxIterations:  50,
		ConsecWaits:    0,
		MaxConsecWaits: 3,
		InProgress:     2,
	})
	if d1.NewConsecWaits != 1 {
		t.Fatalf("setup: expected consec waits=1, got %d", d1.NewConsecWaits)
	}

	// Then: actionable work appears -- should reset
	d2 := EvaluateStop(StopConfig{
		Active:         true,
		Iteration:      d1.NewIteration,
		MaxIterations:  50,
		ConsecWaits:    d1.NewConsecWaits,
		MaxConsecWaits: 3,
		WaitIterations: d1.NewWaitIters,
		Ready:          1,
		InProgress:     1,
	})
	if d2.NewConsecWaits != 0 {
		t.Errorf("expected consec waits reset to 0, got %d", d2.NewConsecWaits)
	}
	// Total wait iterations should NOT reset
	if d2.NewWaitIters != d1.NewWaitIters {
		t.Errorf("expected wait iters preserved at %d, got %d", d1.NewWaitIters, d2.NewWaitIters)
	}
}

func TestEvaluateStop_IterationIncrement(t *testing.T) {
	d := EvaluateStop(StopConfig{
		Active:        true,
		Iteration:     10,
		MaxIterations: 50,
		Ready:         1,
	})
	if d.NewIteration != 11 {
		t.Errorf("expected iteration=11, got %d", d.NewIteration)
	}
}

func TestEvaluateStop_BlockedPlusInProgress(t *testing.T) {
	// Blocked + in-progress but no actionable = wait-like
	d := EvaluateStop(StopConfig{
		Active:         true,
		Iteration:      3,
		MaxIterations:  50,
		ConsecWaits:    0,
		MaxConsecWaits: 3,
		InProgress:     1,
		Blocked:        2,
	})
	if d.Allow {
		t.Error("expected block: in-progress work exists even with blocked items")
	}
	if d.NewConsecWaits != 1 {
		t.Errorf("expected consec waits=1, got %d", d.NewConsecWaits)
	}
}
