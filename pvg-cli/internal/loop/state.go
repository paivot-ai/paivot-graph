// Package loop manages the piv-loop execution loop state.
//
// When a piv-loop is active, the stop hook intercepts session exit and
// evaluates whether to continue (emit continuation JSON) or allow exit.
// State is persisted in .vault/.piv-loop-state.json.
package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const stateFile = ".piv-loop-state.json"

// State represents the persistent loop execution state.
type State struct {
	Active              bool   `json:"active"`
	Mode                string `json:"mode"`                  // "all" or "epic"
	TargetEpic          string `json:"target_epic,omitempty"` // epic ID when mode=epic
	Iteration           int    `json:"iteration"`
	MaxIterations       int    `json:"max_iterations"`        // 0 = unlimited
	ConsecutiveWaits    int    `json:"consecutive_waits"`
	MaxConsecutiveWaits int    `json:"max_consecutive_waits"`
	WaitIterations      int    `json:"wait_iterations"`
	StartedAt           string `json:"started_at"`
}

// NewState creates a new loop state with sensible defaults.
func NewState(mode, epic string, maxIter int) *State {
	return &State{
		Active:              true,
		Mode:                mode,
		TargetEpic:          epic,
		Iteration:           0,
		MaxIterations:       maxIter,
		ConsecutiveWaits:    0,
		MaxConsecutiveWaits: 3,
		WaitIterations:      0,
		StartedAt:           time.Now().UTC().Format(time.RFC3339),
	}
}

// StatePath returns the full path to the loop state file.
func StatePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".vault", stateFile)
}

// StateFileName returns the state file basename (for guard exemption checks).
func StateFileName() string {
	return stateFile
}

// ReadState reads the loop state from disk.
func ReadState(projectRoot string) (*State, error) {
	path := StatePath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse loop state: %w", err)
	}
	return &state, nil
}

// WriteState persists the loop state to disk.
func WriteState(projectRoot string, state *State) error {
	path := StatePath(projectRoot)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal loop state: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// RemoveState deletes the loop state file. No-op if it doesn't exist.
func RemoveState(projectRoot string) error {
	path := StatePath(projectRoot)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// IsActive checks whether a loop is currently active for the project.
func IsActive(projectRoot string) bool {
	state, err := ReadState(projectRoot)
	if err != nil {
		return false
	}
	return state.Active
}
