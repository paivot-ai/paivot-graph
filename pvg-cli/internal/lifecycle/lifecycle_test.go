package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestFormatSessionEntry_NoLinks(t *testing.T) {
	entry := formatSessionEntry("2026-02-25", nil)
	if !strings.Contains(entry, "Session log (2026-02-25)") {
		t.Error("missing date in session entry")
	}
	if !strings.Contains(entry, "Session ended normally") {
		t.Error("missing status line")
	}
	if strings.Contains(entry, "Notes created") {
		t.Error("should not have Notes created line when no links")
	}
}

func TestFormatSessionEntry_WithLinks(t *testing.T) {
	links := []string{"pvg Go CLI architecture", "Three-Tier Knowledge Model"}
	entry := formatSessionEntry("2026-02-25", links)
	if !strings.Contains(entry, "[[pvg Go CLI architecture]]") {
		t.Error("missing first wikilink")
	}
	if !strings.Contains(entry, "[[Three-Tier Knowledge Model]]") {
		t.Error("missing second wikilink")
	}
	if !strings.Contains(entry, "Notes created: [[pvg Go CLI architecture]], [[Three-Tier Knowledge Model]]") {
		t.Errorf("unexpected format: %s", entry)
	}
}

func TestFormatSessionEntry_SingleLink(t *testing.T) {
	links := []string{"Some Decision"}
	entry := formatSessionEntry("2026-02-25", links)
	if !strings.Contains(entry, "- Notes created: [[Some Decision]]\n") {
		t.Errorf("unexpected format for single link: %s", entry)
	}
	// No comma for single link
	if strings.Contains(entry, ", ") {
		t.Error("should not have comma with single link")
	}
}

func TestCollectSessionLinks_NoVaultDir(t *testing.T) {
	dir := t.TempDir()
	links := collectSessionLinks(dir, "test-project", "2026-02-25")
	// No .vault/knowledge/ exists, no vault available -- should return empty
	if len(links) != 0 {
		t.Errorf("expected empty links, got %v", links)
	}
}

func TestCollectSessionLinks_FindsLocalNotes(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, ".vault", "knowledge", "decisions")
	if err := os.MkdirAll(knowledgeDir, 0755); err != nil {
		t.Fatal(err)
	}

	today := time.Now().Format("2006-01-02")
	note := filepath.Join(knowledgeDir, "Use Go for CLI.md")
	if err := os.WriteFile(note, []byte("# Use Go for CLI\n"), 0644); err != nil {
		t.Fatal(err)
	}

	links := collectSessionLinks(dir, "test-project", today)
	// The note was just created, so its mtime is today
	found := false
	for _, l := range links {
		if l == "Use Go for CLI" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Use Go for CLI' in links, got %v", links)
	}
}

// --- detectStack tests ---

func TestDetectStack_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	stacks := detectStack(dir)
	if len(stacks) != 0 {
		t.Errorf("expected empty stacks, got %v", stacks)
	}
}

func TestDetectStack_GoProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0644)
	stacks := detectStack(dir)
	if len(stacks) != 1 || stacks[0] != "go" {
		t.Errorf("expected [go], got %v", stacks)
	}
}

func TestDetectStack_RustProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\n"), 0644)
	stacks := detectStack(dir)
	if len(stacks) != 1 || stacks[0] != "rust" {
		t.Errorf("expected [rust], got %v", stacks)
	}
}

func TestDetectStack_NodeProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
	stacks := detectStack(dir)
	if len(stacks) != 1 || stacks[0] != "node" {
		t.Errorf("expected [node], got %v", stacks)
	}
}

func TestDetectStack_TypeScriptProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"devDependencies":{"typescript":"^5.0"}}`), 0644)
	stacks := detectStack(dir)
	if len(stacks) != 2 || stacks[0] != "node" || stacks[1] != "typescript" {
		t.Errorf("expected [node, typescript], got %v", stacks)
	}
}

func TestDetectStack_PythonPyproject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\n"), 0644)
	stacks := detectStack(dir)
	if len(stacks) != 1 || stacks[0] != "python" {
		t.Errorf("expected [python], got %v", stacks)
	}
}

func TestDetectStack_PythonRequirements(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask\n"), 0644)
	stacks := detectStack(dir)
	if len(stacks) != 1 || stacks[0] != "python" {
		t.Errorf("expected [python], got %v", stacks)
	}
}

func TestDetectStack_ElixirProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "mix.exs"), []byte("defmodule Test do\nend\n"), 0644)
	stacks := detectStack(dir)
	if len(stacks) != 1 || stacks[0] != "elixir" {
		t.Errorf("expected [elixir], got %v", stacks)
	}
}

func TestDetectStack_CSharpProject_Csproj(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MyApp.csproj"), []byte("<Project />\n"), 0644)
	stacks := detectStack(dir)
	if len(stacks) != 1 || stacks[0] != "csharp" {
		t.Errorf("expected [csharp], got %v", stacks)
	}
}

func TestDetectStack_CSharpProject_Sln(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MyApp.sln"), []byte("Solution\n"), 0644)
	stacks := detectStack(dir)
	if len(stacks) != 1 || stacks[0] != "csharp" {
		t.Errorf("expected [csharp], got %v", stacks)
	}
}

func TestDetectStack_MonorepoMultipleStacks(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"devDependencies":{"typescript":"^5"}}`), 0644)
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\n"), 0644)
	stacks := detectStack(dir)
	if len(stacks) != 4 {
		t.Errorf("expected 4 stacks (go, node, typescript, python), got %v", stacks)
	}
}

func TestDetectStack_JavaProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project />\n"), 0644)
	stacks := detectStack(dir)
	if len(stacks) != 1 || stacks[0] != "java" {
		t.Errorf("expected [java], got %v", stacks)
	}
}

// --- readStackDetectionSetting tests ---

func TestReadStackDetectionSetting_Default(t *testing.T) {
	dir := t.TempDir()
	if readStackDetectionSetting(dir) {
		t.Error("expected false when no settings file exists")
	}
}

func TestReadStackDetectionSetting_Enabled(t *testing.T) {
	dir := t.TempDir()
	settingsDir := filepath.Join(dir, ".vault", "knowledge")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatal(err)
	}
	settingsFile := filepath.Join(settingsDir, ".settings.yaml")
	if err := os.WriteFile(settingsFile, []byte("stack_detection: true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if !readStackDetectionSetting(dir) {
		t.Error("expected true when stack_detection is set to true")
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
