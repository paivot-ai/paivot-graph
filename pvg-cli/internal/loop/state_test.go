package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewState_Defaults(t *testing.T) {
	s := NewState("all", "", 50)
	if !s.Active {
		t.Error("expected active=true")
	}
	if s.Mode != "all" {
		t.Errorf("expected mode=all, got %s", s.Mode)
	}
	if s.TargetEpic != "" {
		t.Errorf("expected empty target_epic, got %s", s.TargetEpic)
	}
	if s.Iteration != 0 {
		t.Errorf("expected iteration=0, got %d", s.Iteration)
	}
	if s.MaxIterations != 50 {
		t.Errorf("expected max_iterations=50, got %d", s.MaxIterations)
	}
	if s.ConsecutiveWaits != 0 {
		t.Errorf("expected consecutive_waits=0, got %d", s.ConsecutiveWaits)
	}
	if s.MaxConsecutiveWaits != 3 {
		t.Errorf("expected max_consecutive_waits=3, got %d", s.MaxConsecutiveWaits)
	}
	if s.WaitIterations != 0 {
		t.Errorf("expected wait_iterations=0, got %d", s.WaitIterations)
	}
	if s.StartedAt == "" {
		t.Error("expected non-empty started_at")
	}
}

func TestNewState_EpicMode(t *testing.T) {
	s := NewState("epic", "PROJ-a1b", 0)
	if s.Mode != "epic" {
		t.Errorf("expected mode=epic, got %s", s.Mode)
	}
	if s.TargetEpic != "PROJ-a1b" {
		t.Errorf("expected target_epic=PROJ-a1b, got %s", s.TargetEpic)
	}
	if s.MaxIterations != 0 {
		t.Errorf("expected max_iterations=0 (unlimited), got %d", s.MaxIterations)
	}
}

func TestStateJSON_RoundTrip(t *testing.T) {
	original := &State{
		Active:              true,
		Mode:                "epic",
		TargetEpic:          "TEST-abc",
		Iteration:           5,
		MaxIterations:       50,
		ConsecutiveWaits:    2,
		MaxConsecutiveWaits: 3,
		WaitIterations:      7,
		StartedAt:           "2026-02-27T10:00:00Z",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var restored State
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}

	if restored.Active != original.Active {
		t.Errorf("active mismatch: %v vs %v", restored.Active, original.Active)
	}
	if restored.Mode != original.Mode {
		t.Errorf("mode mismatch: %s vs %s", restored.Mode, original.Mode)
	}
	if restored.TargetEpic != original.TargetEpic {
		t.Errorf("target_epic mismatch: %s vs %s", restored.TargetEpic, original.TargetEpic)
	}
	if restored.Iteration != original.Iteration {
		t.Errorf("iteration mismatch: %d vs %d", restored.Iteration, original.Iteration)
	}
	if restored.MaxIterations != original.MaxIterations {
		t.Errorf("max_iterations mismatch: %d vs %d", restored.MaxIterations, original.MaxIterations)
	}
	if restored.ConsecutiveWaits != original.ConsecutiveWaits {
		t.Errorf("consecutive_waits mismatch: %d vs %d", restored.ConsecutiveWaits, original.ConsecutiveWaits)
	}
	if restored.WaitIterations != original.WaitIterations {
		t.Errorf("wait_iterations mismatch: %d vs %d", restored.WaitIterations, original.WaitIterations)
	}
}

func TestWriteState_ReadState_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".vault"), 0755); err != nil {
		t.Fatal(err)
	}

	original := NewState("all", "", 25)
	original.Iteration = 3

	if err := WriteState(dir, original); err != nil {
		t.Fatalf("WriteState() error: %v", err)
	}

	restored, err := ReadState(dir)
	if err != nil {
		t.Fatalf("ReadState() error: %v", err)
	}

	if restored.Mode != original.Mode {
		t.Errorf("mode mismatch: %s vs %s", restored.Mode, original.Mode)
	}
	if restored.Iteration != original.Iteration {
		t.Errorf("iteration mismatch: %d vs %d", restored.Iteration, original.Iteration)
	}
}

func TestWriteState_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	// Don't pre-create .vault/
	state := NewState("all", "", 10)

	if err := WriteState(dir, state); err != nil {
		t.Fatalf("WriteState() should create directories: %v", err)
	}

	if _, err := os.Stat(StatePath(dir)); err != nil {
		t.Fatalf("state file should exist: %v", err)
	}
}

func TestRemoveState_RemovesFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".vault"), 0755); err != nil {
		t.Fatal(err)
	}

	state := NewState("all", "", 10)
	if err := WriteState(dir, state); err != nil {
		t.Fatal(err)
	}

	if err := RemoveState(dir); err != nil {
		t.Fatalf("RemoveState() error: %v", err)
	}

	if _, err := os.Stat(StatePath(dir)); !os.IsNotExist(err) {
		t.Error("expected state file to be removed")
	}
}

func TestRemoveState_NoopWhenMissing(t *testing.T) {
	dir := t.TempDir()
	if err := RemoveState(dir); err != nil {
		t.Fatalf("RemoveState() with no state file should not error: %v", err)
	}
}

func TestIsActive_TrueWhenActive(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".vault"), 0755); err != nil {
		t.Fatal(err)
	}

	state := NewState("all", "", 10)
	if err := WriteState(dir, state); err != nil {
		t.Fatal(err)
	}

	if !IsActive(dir) {
		t.Error("expected IsActive=true")
	}
}

func TestIsActive_FalseWhenNoFile(t *testing.T) {
	dir := t.TempDir()
	if IsActive(dir) {
		t.Error("expected IsActive=false when no state file")
	}
}

func TestIsActive_FalseWhenInactive(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".vault"), 0755); err != nil {
		t.Fatal(err)
	}

	state := NewState("all", "", 10)
	state.Active = false
	if err := WriteState(dir, state); err != nil {
		t.Fatal(err)
	}

	if IsActive(dir) {
		t.Error("expected IsActive=false when state.Active=false")
	}
}

func TestStatePath(t *testing.T) {
	got := StatePath("/project")
	want := filepath.Join("/project", ".vault", ".piv-loop-state.json")
	if got != want {
		t.Errorf("StatePath() = %s, want %s", got, want)
	}
}

func TestStateFileName_Value(t *testing.T) {
	if StateFileName() != ".piv-loop-state.json" {
		t.Errorf("unexpected state file name: %s", StateFileName())
	}
}
