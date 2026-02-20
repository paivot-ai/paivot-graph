package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Unit tests for ensureTimestamps
// ---------------------------------------------------------------------------

// Unit test 1: create mode sets both created_at and updated_at
func TestEnsureTimestampsCreate(t *testing.T) {
	text := "---\ntype: note\n---\n\n# Hello\n"
	now := time.Now().UTC()
	got := ensureTimestamps(text, true, now)

	yaml, _, hasFM := extractFrontmatter(got)
	if !hasFM {
		t.Fatal("frontmatter lost")
	}

	createdAt, ok := frontmatterGetValue(yaml, "created_at")
	if !ok {
		t.Fatal("created_at not set on create")
	}
	if createdAt == "" {
		t.Fatal("created_at is empty")
	}

	updatedAt, ok := frontmatterGetValue(yaml, "updated_at")
	if !ok {
		t.Fatal("updated_at not set on create")
	}
	if updatedAt == "" {
		t.Fatal("updated_at is empty")
	}
}

// Unit test 2: update mode sets only updated_at, preserves existing created_at
func TestEnsureTimestampsUpdate(t *testing.T) {
	text := "---\ntype: note\ncreated_at: 2026-01-01T00:00:00Z\n---\n\n# Hello\n"
	now := time.Now().UTC()
	got := ensureTimestamps(text, false, now)

	yaml, _, hasFM := extractFrontmatter(got)
	if !hasFM {
		t.Fatal("frontmatter lost")
	}

	createdAt, ok := frontmatterGetValue(yaml, "created_at")
	if !ok {
		t.Fatal("created_at missing after update")
	}
	if createdAt != "2026-01-01T00:00:00Z" {
		t.Errorf("created_at changed: got %q, want 2026-01-01T00:00:00Z", createdAt)
	}

	updatedAt, ok := frontmatterGetValue(yaml, "updated_at")
	if !ok {
		t.Fatal("updated_at not set on update")
	}
	if updatedAt == "" {
		t.Fatal("updated_at is empty")
	}
}

// Unit test 3: existing created_at is never overwritten even on create
func TestEnsureTimestampsPreservesCreatedAt(t *testing.T) {
	text := "---\ntype: note\ncreated_at: 2025-06-15T10:30:00Z\n---\n\n# Hello\n"
	now := time.Now().UTC()
	got := ensureTimestamps(text, true, now)

	yaml, _, _ := extractFrontmatter(got)
	createdAt, ok := frontmatterGetValue(yaml, "created_at")
	if !ok {
		t.Fatal("created_at missing")
	}
	if createdAt != "2025-06-15T10:30:00Z" {
		t.Errorf("created_at was overwritten: got %q, want 2025-06-15T10:30:00Z", createdAt)
	}
}

// Unit test 4: frontmatter is added when not present and timestamps requested
func TestEnsureTimestampsNoFrontmatter(t *testing.T) {
	text := "# Just a heading\n\nSome content.\n"
	now := time.Now().UTC()
	got := ensureTimestamps(text, true, now)

	yaml, _, hasFM := extractFrontmatter(got)
	if !hasFM {
		t.Fatal("frontmatter not created for note without frontmatter")
	}

	createdAt, ok := frontmatterGetValue(yaml, "created_at")
	if !ok {
		t.Fatal("created_at not set when frontmatter was added")
	}
	if createdAt == "" {
		t.Fatal("created_at is empty")
	}

	updatedAt, ok := frontmatterGetValue(yaml, "updated_at")
	if !ok {
		t.Fatal("updated_at not set when frontmatter was added")
	}
	if updatedAt == "" {
		t.Fatal("updated_at is empty")
	}

	// Body should still be present
	if !strings.Contains(got, "# Just a heading") {
		t.Error("body content lost when adding frontmatter")
	}
}

// Unit test 5: timestamps use ISO 8601 UTC format (2006-01-02T15:04:05Z)
func TestEnsureTimestampsFormat(t *testing.T) {
	text := "---\ntype: note\n---\n\n# Hello\n"
	now := time.Date(2026, 2, 19, 14, 30, 0, 0, time.UTC)
	got := ensureTimestamps(text, true, now)

	yaml, _, _ := extractFrontmatter(got)

	createdAt, _ := frontmatterGetValue(yaml, "created_at")
	want := "2026-02-19T14:30:00Z"
	if createdAt != want {
		t.Errorf("created_at format: got %q, want %q", createdAt, want)
	}

	updatedAt, _ := frontmatterGetValue(yaml, "updated_at")
	if updatedAt != want {
		t.Errorf("updated_at format: got %q, want %q", updatedAt, want)
	}
}

// ---------------------------------------------------------------------------
// Integration tests (real files, no mocks)
// ---------------------------------------------------------------------------

// Integration test 6: create note with timestamps flag, verify both properties set
func TestCreateWithTimestamps(t *testing.T) {
	vaultDir := t.TempDir()

	params := map[string]string{
		"name":    "Stamped Note",
		"path":    "Stamped Note.md",
		"content": "---\ntype: note\n---\n\n# Stamped\n",
	}
	if err := cmdCreate(vaultDir, params, true, true); err != nil {
		t.Fatalf("create with timestamps: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(vaultDir, "Stamped Note.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got := string(data)

	yaml, _, hasFM := extractFrontmatter(got)
	if !hasFM {
		t.Fatal("no frontmatter")
	}

	if _, ok := frontmatterGetValue(yaml, "created_at"); !ok {
		t.Error("created_at not set on create with timestamps")
	}
	if _, ok := frontmatterGetValue(yaml, "updated_at"); !ok {
		t.Error("updated_at not set on create with timestamps")
	}

	// Original property preserved
	if v, ok := frontmatterGetValue(yaml, "type"); !ok || v != "note" {
		t.Errorf("type property: got %q", v)
	}
}

// Integration test 7: append to existing note, verify updated_at changed, created_at unchanged
func TestAppendWithTimestamps(t *testing.T) {
	vaultDir := t.TempDir()

	original := "---\ntype: note\ncreated_at: 2026-01-01T00:00:00Z\nupdated_at: 2026-01-01T00:00:00Z\n---\n\n# Note\n"
	notePath := filepath.Join(vaultDir, "AppendNote.md")
	os.WriteFile(notePath, []byte(original), 0644)

	params := map[string]string{
		"file":    "AppendNote",
		"content": "\nAppended content.\n",
	}
	if err := cmdAppend(vaultDir, params, true); err != nil {
		t.Fatalf("append with timestamps: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	yaml, _, _ := extractFrontmatter(got)

	// created_at must be unchanged
	createdAt, ok := frontmatterGetValue(yaml, "created_at")
	if !ok {
		t.Fatal("created_at missing after append")
	}
	if createdAt != "2026-01-01T00:00:00Z" {
		t.Errorf("created_at changed: got %q", createdAt)
	}

	// updated_at must be refreshed
	updatedAt, ok := frontmatterGetValue(yaml, "updated_at")
	if !ok {
		t.Fatal("updated_at not set after append")
	}
	if updatedAt == "2026-01-01T00:00:00Z" {
		t.Error("updated_at was not refreshed after append")
	}

	// Content was appended
	if !strings.Contains(got, "Appended content.") {
		t.Error("appended content not found")
	}
}

// Integration test 8: prepend to note, verify updated_at set
func TestPrependWithTimestamps(t *testing.T) {
	vaultDir := t.TempDir()

	original := "---\ntype: note\ncreated_at: 2026-01-01T00:00:00Z\n---\n\n# Existing\n"
	notePath := filepath.Join(vaultDir, "PrependNote.md")
	os.WriteFile(notePath, []byte(original), 0644)

	params := map[string]string{
		"file":    "PrependNote",
		"content": "Prepended line\n",
	}
	if err := cmdPrepend(vaultDir, params, true); err != nil {
		t.Fatalf("prepend with timestamps: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	yaml, _, _ := extractFrontmatter(got)

	updatedAt, ok := frontmatterGetValue(yaml, "updated_at")
	if !ok {
		t.Fatal("updated_at not set after prepend")
	}
	if updatedAt == "" {
		t.Fatal("updated_at is empty after prepend")
	}

	// created_at preserved
	createdAt, _ := frontmatterGetValue(yaml, "created_at")
	if createdAt != "2026-01-01T00:00:00Z" {
		t.Errorf("created_at changed: got %q", createdAt)
	}

	// Content was prepended
	if !strings.Contains(got, "Prepended line") {
		t.Error("prepended content not found")
	}
}

// Integration test 9: write to note, verify updated_at set, created_at preserved
func TestWriteWithTimestamps(t *testing.T) {
	vaultDir := t.TempDir()

	original := "---\ntype: note\ncreated_at: 2026-01-01T00:00:00Z\nupdated_at: 2026-01-01T00:00:00Z\n---\n\n# Old Body\n"
	notePath := filepath.Join(vaultDir, "WriteNote.md")
	os.WriteFile(notePath, []byte(original), 0644)

	params := map[string]string{
		"file":    "WriteNote",
		"content": "# New Body\n\nReplaced.\n",
	}
	if err := cmdWrite(vaultDir, params, true); err != nil {
		t.Fatalf("write with timestamps: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	yaml, _, _ := extractFrontmatter(got)

	// created_at must be unchanged
	createdAt, ok := frontmatterGetValue(yaml, "created_at")
	if !ok {
		t.Fatal("created_at missing after write")
	}
	if createdAt != "2026-01-01T00:00:00Z" {
		t.Errorf("created_at changed: got %q", createdAt)
	}

	// updated_at must be refreshed
	updatedAt, ok := frontmatterGetValue(yaml, "updated_at")
	if !ok {
		t.Fatal("updated_at not set after write")
	}
	if updatedAt == "2026-01-01T00:00:00Z" {
		t.Error("updated_at was not refreshed after write")
	}

	// New body present
	if !strings.Contains(got, "Replaced.") {
		t.Error("new body content not found")
	}
	// Old body gone
	if strings.Contains(got, "Old Body") {
		t.Error("old body still present")
	}
}

// Integration test 10: patch note by heading with timestamps, verify updated_at set, created_at preserved
func TestPatchWithTimestamps(t *testing.T) {
	vaultDir := t.TempDir()

	original := "---\ntype: note\ncreated_at: 2026-01-01T00:00:00Z\nupdated_at: 2026-01-01T00:00:00Z\n---\n\n## Section A\nold content\n\n## Section B\nkeep this\n"
	notePath := filepath.Join(vaultDir, "PatchNote.md")
	os.WriteFile(notePath, []byte(original), 0644)

	params := map[string]string{
		"file":    "PatchNote",
		"heading": "## Section A",
		"content": "new content\n",
	}
	if err := cmdPatch(vaultDir, params, false, true); err != nil {
		t.Fatalf("patch with timestamps: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	yaml, _, _ := extractFrontmatter(got)

	// created_at must be unchanged
	createdAt, ok := frontmatterGetValue(yaml, "created_at")
	if !ok {
		t.Fatal("created_at missing after patch")
	}
	if createdAt != "2026-01-01T00:00:00Z" {
		t.Errorf("created_at changed: got %q", createdAt)
	}

	// updated_at must be refreshed
	updatedAt, ok := frontmatterGetValue(yaml, "updated_at")
	if !ok {
		t.Fatal("updated_at not set after patch")
	}
	if updatedAt == "2026-01-01T00:00:00Z" {
		t.Error("updated_at was not refreshed after patch")
	}

	// Patched content present
	if !strings.Contains(got, "new content") {
		t.Error("patched content not found")
	}
	// Other section intact
	if !strings.Contains(got, "keep this") {
		t.Error("Section B was affected")
	}
}

// Integration test 11: VLT_TIMESTAMPS=1 env var enables timestamps without flag
func TestTimestampsEnvVar(t *testing.T) {
	vaultDir := t.TempDir()

	t.Setenv("VLT_TIMESTAMPS", "1")

	params := map[string]string{
		"name":    "EnvNote",
		"path":    "EnvNote.md",
		"content": "---\ntype: test\n---\n\n# Env Test\n",
	}
	// timestamps=false (flag not set), but env var is set
	if err := cmdCreate(vaultDir, params, true, false); err != nil {
		t.Fatalf("create with env var: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(vaultDir, "EnvNote.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got := string(data)

	yaml, _, hasFM := extractFrontmatter(got)
	if !hasFM {
		t.Fatal("no frontmatter")
	}

	if _, ok := frontmatterGetValue(yaml, "created_at"); !ok {
		t.Error("created_at not set via VLT_TIMESTAMPS env var")
	}
	if _, ok := frontmatterGetValue(yaml, "updated_at"); !ok {
		t.Error("updated_at not set via VLT_TIMESTAMPS env var")
	}
}

// Integration test 12: without flag or env var, no timestamps added
func TestNoTimestampsWithoutFlag(t *testing.T) {
	vaultDir := t.TempDir()

	// Ensure env var is NOT set
	t.Setenv("VLT_TIMESTAMPS", "")

	// Create
	params := map[string]string{
		"name":    "PlainNote",
		"path":    "PlainNote.md",
		"content": "---\ntype: note\n---\n\n# Plain\n",
	}
	if err := cmdCreate(vaultDir, params, true, false); err != nil {
		t.Fatalf("create without timestamps: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(vaultDir, "PlainNote.md"))
	got := string(data)

	yaml, _, _ := extractFrontmatter(got)
	if _, ok := frontmatterGetValue(yaml, "created_at"); ok {
		t.Error("created_at should NOT be set without timestamps flag")
	}
	if _, ok := frontmatterGetValue(yaml, "updated_at"); ok {
		t.Error("updated_at should NOT be set without timestamps flag")
	}

	// Append
	appendParams := map[string]string{
		"file":    "PlainNote",
		"content": "\nMore content.\n",
	}
	if err := cmdAppend(vaultDir, appendParams, false); err != nil {
		t.Fatalf("append without timestamps: %v", err)
	}

	data, _ = os.ReadFile(filepath.Join(vaultDir, "PlainNote.md"))
	got = string(data)

	yaml, _, _ = extractFrontmatter(got)
	if _, ok := frontmatterGetValue(yaml, "updated_at"); ok {
		t.Error("updated_at should NOT be set without timestamps flag on append")
	}
}

// Integration test 13: existing frontmatter properties preserved when timestamps added
func TestTimestampsPreserveExistingFrontmatter(t *testing.T) {
	vaultDir := t.TempDir()

	params := map[string]string{
		"name":    "RichNote",
		"path":    "RichNote.md",
		"content": "---\ntype: decision\nstatus: active\naliases: [Dec1, Alt]\ntags: [project, review]\n---\n\n# Rich Note\n",
	}
	if err := cmdCreate(vaultDir, params, true, true); err != nil {
		t.Fatalf("create: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(vaultDir, "RichNote.md"))
	got := string(data)

	yaml, _, hasFM := extractFrontmatter(got)
	if !hasFM {
		t.Fatal("no frontmatter")
	}

	// All original properties must be intact
	if v, ok := frontmatterGetValue(yaml, "type"); !ok || v != "decision" {
		t.Errorf("type lost or changed: %q", v)
	}
	if v, ok := frontmatterGetValue(yaml, "status"); !ok || v != "active" {
		t.Errorf("status lost or changed: %q", v)
	}
	aliases := frontmatterGetList(yaml, "aliases")
	if len(aliases) != 2 || aliases[0] != "Dec1" || aliases[1] != "Alt" {
		t.Errorf("aliases changed: %v", aliases)
	}
	tags := frontmatterGetList(yaml, "tags")
	if len(tags) != 2 || tags[0] != "project" || tags[1] != "review" {
		t.Errorf("tags changed: %v", tags)
	}

	// Timestamps must be present
	if _, ok := frontmatterGetValue(yaml, "created_at"); !ok {
		t.Error("created_at not added")
	}
	if _, ok := frontmatterGetValue(yaml, "updated_at"); !ok {
		t.Error("updated_at not added")
	}
}

// Integration test 14: patch by line number with timestamps, verify updated_at set
func TestPatchByLineWithTimestamps(t *testing.T) {
	vaultDir := t.TempDir()

	original := "---\ntype: note\ncreated_at: 2026-01-01T00:00:00Z\nupdated_at: 2026-01-01T00:00:00Z\n---\n\nLine A\nLine B\nLine C\n"
	notePath := filepath.Join(vaultDir, "LineNote.md")
	os.WriteFile(notePath, []byte(original), 0644)

	// Line 7 is "Line A" (1=---, 2=type:, 3=created_at:, 4=updated_at:, 5=---, 6=empty, 7=Line A)
	params := map[string]string{
		"file":    "LineNote",
		"line":    "7",
		"content": "PATCHED",
	}
	if err := cmdPatch(vaultDir, params, false, true); err != nil {
		t.Fatalf("patch by line with timestamps: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	yaml, _, _ := extractFrontmatter(got)

	// created_at must be unchanged
	createdAt, ok := frontmatterGetValue(yaml, "created_at")
	if !ok {
		t.Fatal("created_at missing after line patch")
	}
	if createdAt != "2026-01-01T00:00:00Z" {
		t.Errorf("created_at changed: got %q", createdAt)
	}

	// updated_at must be refreshed
	updatedAt, ok := frontmatterGetValue(yaml, "updated_at")
	if !ok {
		t.Fatal("updated_at not set after line patch")
	}
	if updatedAt == "2026-01-01T00:00:00Z" {
		t.Error("updated_at was not refreshed after line patch")
	}

	// Patched content present
	if !strings.Contains(got, "PATCHED") {
		t.Error("patched content not found")
	}
	// Old content gone
	if strings.Contains(got, "Line A") {
		t.Error("old line A still present")
	}
}
