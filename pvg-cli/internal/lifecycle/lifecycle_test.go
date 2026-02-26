package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectProject_FallsBackToBasename(t *testing.T) {
	// Use a temporary directory (not a git repo)
	dir := t.TempDir()
	project := detectProject(dir)
	expected := filepath.Base(dir)
	if project != expected {
		t.Errorf("expected %q, got %q", expected, project)
	}
}

func TestExtractNoteSummary_ParsesFrontmatter(t *testing.T) {
	dir := t.TempDir()
	note := filepath.Join(dir, "test.md")
	content := `---
type: decision
created: 2026-01-15
status: active
---

# My Decision

We decided to use Go instead of Rust.
`
	if err := os.WriteFile(note, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	date, firstLine := extractNoteSummary(note)
	if date != "2026-01-15" {
		t.Errorf("expected date 2026-01-15, got %q", date)
	}
	if firstLine != "We decided to use Go instead of Rust." {
		t.Errorf("expected first content line, got %q", firstLine)
	}
}

func TestExtractNoteSummary_HandlesNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	note := filepath.Join(dir, "bare.md")
	content := "Just a plain note with no frontmatter."
	if err := os.WriteFile(note, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	date, _ := extractNoteSummary(note)
	if date != "unknown" {
		t.Errorf("expected date unknown, got %q", date)
	}
}

func TestExtractNoteSummary_MissingFile(t *testing.T) {
	date, firstLine := extractNoteSummary("/nonexistent/path.md")
	if date != "unknown" {
		t.Errorf("expected unknown, got %q", date)
	}
	if firstLine != "(no summary)" {
		t.Errorf("expected (no summary), got %q", firstLine)
	}
}

func TestReadMaxNotesSetting_Default(t *testing.T) {
	dir := t.TempDir()
	n := readMaxNotesSetting(dir)
	if n != 10 {
		t.Errorf("expected default 10, got %d", n)
	}
}

func TestReadMaxNotesSetting_CustomValue(t *testing.T) {
	dir := t.TempDir()
	settingsDir := filepath.Join(dir, ".vault", "knowledge")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatal(err)
	}
	settingsFile := filepath.Join(settingsDir, ".settings.yaml")
	if err := os.WriteFile(settingsFile, []byte("session_start_max_notes: 5\n"), 0644); err != nil {
		t.Fatal(err)
	}

	n := readMaxNotesSetting(dir)
	if n != 5 {
		t.Errorf("expected 5, got %d", n)
	}
}

func TestStaticOperatingMode_ContainsKeyContent(t *testing.T) {
	mode := staticOperatingMode()
	checks := []string{
		"CONCURRENCY LIMITS",
		"BEFORE STARTING",
		"WHILE WORKING",
		"BEFORE ENDING",
		"vlt vault=",
	}
	for _, check := range checks {
		if !strings.Contains(mode, check) {
			t.Errorf("static operating mode missing %q", check)
		}
	}
}

func TestStaticPreCompact_ContainsKeyContent(t *testing.T) {
	text := staticPreCompact()
	checks := []string{
		"DECISIONS",
		"PATTERNS",
		"DEBUG INSIGHTS",
		"PROJECT UPDATES",
		"vlt vault=",
	}
	for _, check := range checks {
		if !strings.Contains(text, check) {
			t.Errorf("static pre-compact missing %q", check)
		}
	}
}

func TestStaticStopChecklist_ContainsKeyContent(t *testing.T) {
	text := staticStopChecklist()
	checks := []string{
		"DECISIONS",
		"PATTERNS",
		"DEBUG INSIGHTS",
		"PROJECT INDEX NOTE",
	}
	for _, check := range checks {
		if !strings.Contains(text, check) {
			t.Errorf("static stop checklist missing %q", check)
		}
	}
}

func TestOutputProjectKnowledge_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, ".vault", "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Should not panic on empty directories
	// Capture stdout would require redirecting os.Stdout, which is overkill.
	// Just verify it doesn't panic.
	outputProjectKnowledge(knowledgeDir, dir)
}

func TestOutputProjectKnowledge_WithNotes(t *testing.T) {
	dir := t.TempDir()
	decisionsDir := filepath.Join(dir, ".vault", "knowledge", "decisions")
	if err := os.MkdirAll(decisionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	note := `---
type: decision
created: 2026-02-25
---

# Use Go for CLI

We chose Go for the pvg CLI because it compiles to a single binary.
`
	if err := os.WriteFile(filepath.Join(decisionsDir, "Use Go for CLI.md"), []byte(note), 0644); err != nil {
		t.Fatal(err)
	}

	// Should not panic with actual notes
	knowledgeDir := filepath.Join(dir, ".vault", "knowledge")
	outputProjectKnowledge(knowledgeDir, dir)
}
