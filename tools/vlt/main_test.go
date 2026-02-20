package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Ensure json import is used (referenced in TestSearchContextWithJSONFormat).
var _ = json.Unmarshal

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCmd    string
		wantParams map[string]string
		wantFlags  map[string]bool
	}{
		{
			name:       "read command",
			args:       []string{"vault=Claude", "read", "file=Session Operating Mode"},
			wantCmd:    "read",
			wantParams: map[string]string{"vault": "Claude", "file": "Session Operating Mode"},
			wantFlags:  map[string]bool{},
		},
		{
			name:       "create with silent flag",
			args:       []string{"vault=Claude", "create", "name=My Note", "path=_inbox/My Note.md", "content=# Hello", "silent"},
			wantCmd:    "create",
			wantParams: map[string]string{"vault": "Claude", "name": "My Note", "path": "_inbox/My Note.md", "content": "# Hello"},
			wantFlags:  map[string]bool{"silent": true},
		},
		{
			name:       "search command",
			args:       []string{"vault=Claude", "search", "query=architecture"},
			wantCmd:    "search",
			wantParams: map[string]string{"vault": "Claude", "query": "architecture"},
			wantFlags:  map[string]bool{},
		},
		{
			name:       "move command",
			args:       []string{"vault=Claude", "move", "path=_inbox/Note.md", "to=decisions/Note.md"},
			wantCmd:    "move",
			wantParams: map[string]string{"vault": "Claude", "path": "_inbox/Note.md", "to": "decisions/Note.md"},
			wantFlags:  map[string]bool{},
		},
		{
			name:       "property:set command",
			args:       []string{"vault=Claude", "property:set", "file=Note", "name=status", "value=archived"},
			wantCmd:    "property:set",
			wantParams: map[string]string{"vault": "Claude", "file": "Note", "name": "status", "value": "archived"},
			wantFlags:  map[string]bool{},
		},
		{
			name:       "content with equals sign",
			args:       []string{"vault=Claude", "create", "name=Note", "path=_inbox/Note.md", "content=key=value"},
			wantCmd:    "create",
			wantParams: map[string]string{"vault": "Claude", "name": "Note", "path": "_inbox/Note.md", "content": "key=value"},
			wantFlags:  map[string]bool{},
		},
		{
			name:       "quoted value stripping",
			args:       []string{`vault="Claude"`, "read", `file="My Note"`},
			wantCmd:    "read",
			wantParams: map[string]string{"vault": "Claude", "file": "My Note"},
			wantFlags:  map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, params, flags := parseArgs(tt.args)

			if cmd != tt.wantCmd {
				t.Errorf("cmd = %q, want %q", cmd, tt.wantCmd)
			}

			for k, want := range tt.wantParams {
				got, ok := params[k]
				if !ok {
					t.Errorf("missing param %q", k)
				} else if got != want {
					t.Errorf("param[%q] = %q, want %q", k, got, want)
				}
			}
			if len(params) != len(tt.wantParams) {
				t.Errorf("got %d params, want %d", len(params), len(tt.wantParams))
			}

			for k := range tt.wantFlags {
				if !flags[k] {
					t.Errorf("missing flag %q", k)
				}
			}
			if len(flags) != len(tt.wantFlags) {
				t.Errorf("got %d flags, want %d", len(flags), len(tt.wantFlags))
			}
		})
	}
}

func TestResolveNote(t *testing.T) {
	// Create a temporary vault
	vaultDir := t.TempDir()

	// Create directory structure
	os.MkdirAll(filepath.Join(vaultDir, "methodology"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, "conventions"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, "_inbox"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)

	// Create test notes
	os.WriteFile(filepath.Join(vaultDir, "methodology", "Sr PM Agent.md"), []byte("# Sr PM"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "conventions", "Session Operating Mode.md"), []byte("# SOM"), 0644)
	os.WriteFile(filepath.Join(vaultDir, ".obsidian", "hidden.md"), []byte("# Hidden"), 0644)

	tests := []struct {
		title   string
		wantRel string
		wantErr bool
	}{
		{"Sr PM Agent", "methodology/Sr PM Agent.md", false},
		{"Session Operating Mode", "conventions/Session Operating Mode.md", false},
		{"Nonexistent Note", "", true},
		{"hidden", "", true}, // should not find notes in .obsidian
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			path, err := resolveNote(vaultDir, tt.title)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got path %q", path)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			relPath, _ := filepath.Rel(vaultDir, path)
			if relPath != tt.wantRel {
				t.Errorf("got %q, want %q", relPath, tt.wantRel)
			}
		})
	}
}

func TestResolveNote_Alias(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "methodology"), 0755)
	os.WriteFile(
		filepath.Join(vaultDir, "methodology", "Sr PM Agent.md"),
		[]byte("---\naliases: [PM, Senior PM]\n---\n\n# Sr PM Agent\n"),
		0644,
	)

	// Resolve by alias
	path, err := resolveNote(vaultDir, "PM")
	if err != nil {
		t.Fatalf("alias resolution failed: %v", err)
	}
	relPath, _ := filepath.Rel(vaultDir, path)
	if relPath != "methodology/Sr PM Agent.md" {
		t.Errorf("got %q, want methodology/Sr PM Agent.md", relPath)
	}

	// Resolve by alias (case insensitive)
	path, err = resolveNote(vaultDir, "senior pm")
	if err != nil {
		t.Fatalf("case-insensitive alias failed: %v", err)
	}
	relPath, _ = filepath.Rel(vaultDir, path)
	if relPath != "methodology/Sr PM Agent.md" {
		t.Errorf("got %q, want methodology/Sr PM Agent.md", relPath)
	}

	// Filename match still takes priority
	path, err = resolveNote(vaultDir, "Sr PM Agent")
	if err != nil {
		t.Fatalf("filename resolution failed: %v", err)
	}
	relPath, _ = filepath.Rel(vaultDir, path)
	if relPath != "methodology/Sr PM Agent.md" {
		t.Errorf("got %q, want methodology/Sr PM Agent.md", relPath)
	}
}

func TestCmdCreateAndRead(t *testing.T) {
	vaultDir := t.TempDir()

	// Create a note
	params := map[string]string{
		"name":    "Test Note",
		"path":    "_inbox/Test Note.md",
		"content": "---\ntype: test\n---\n\n# Test Note\n\nHello world.\n",
	}
	if err := cmdCreate(vaultDir, params, false, false); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Verify file exists
	fullPath := filepath.Join(vaultDir, "_inbox", "Test Note.md")
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if string(data) != params["content"] {
		t.Errorf("content mismatch:\ngot:  %q\nwant: %q", string(data), params["content"])
	}

	// Create again (should be a no-op, not overwrite)
	params["content"] = "overwritten"
	if err := cmdCreate(vaultDir, params, true, false); err != nil {
		t.Fatalf("create (duplicate): %v", err)
	}
	data, _ = os.ReadFile(fullPath)
	if string(data) == "overwritten" {
		t.Error("create overwrote existing note")
	}
}

func TestCmdAppend(t *testing.T) {
	vaultDir := t.TempDir()

	// Create a note to append to
	notePath := filepath.Join(vaultDir, "Test Append.md")
	os.WriteFile(notePath, []byte("# Test\n"), 0644)

	params := map[string]string{
		"file":    "Test Append",
		"content": "\n## Added section\n",
	}
	if err := cmdAppend(vaultDir, params, false); err != nil {
		t.Fatalf("append: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	want := "# Test\n\n## Added section\n"
	if string(data) != want {
		t.Errorf("got %q, want %q", string(data), want)
	}
}

func TestCmdMove(t *testing.T) {
	vaultDir := t.TempDir()

	// Create source
	os.MkdirAll(filepath.Join(vaultDir, "_inbox"), 0755)
	srcPath := filepath.Join(vaultDir, "_inbox", "Note.md")
	os.WriteFile(srcPath, []byte("# Note"), 0644)

	params := map[string]string{
		"path": "_inbox/Note.md",
		"to":   "decisions/Note.md",
	}
	if err := cmdMove(vaultDir, params); err != nil {
		t.Fatalf("move: %v", err)
	}

	// Source should be gone
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("source file still exists after move")
	}

	// Destination should exist
	dstPath := filepath.Join(vaultDir, "decisions", "Note.md")
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("destination not found: %v", err)
	}
	if string(data) != "# Note" {
		t.Errorf("content mismatch after move: %q", string(data))
	}
}

func TestCmdMove_RenameUpdatesLinks(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "_inbox"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, "methodology"), 0755)

	// The note being renamed
	os.WriteFile(
		filepath.Join(vaultDir, "_inbox", "Old Name.md"),
		[]byte("# Old Name\n\nContent here.\n"),
		0644,
	)

	// Another note that references it
	os.WriteFile(
		filepath.Join(vaultDir, "methodology", "Developer Agent.md"),
		[]byte("# Developer\n\nSee [[Old Name]] and [[Old Name#Section|details]].\n"),
		0644,
	)

	params := map[string]string{
		"path": "_inbox/Old Name.md",
		"to":   "decisions/New Name.md",
	}
	if err := cmdMove(vaultDir, params); err != nil {
		t.Fatalf("move: %v", err)
	}

	// Verify the referencing file was updated
	data, _ := os.ReadFile(filepath.Join(vaultDir, "methodology", "Developer Agent.md"))
	got := string(data)

	if contains(got, "[[Old Name]]") {
		t.Error("old wikilink [[Old Name]] still present")
	}
	if !contains(got, "[[New Name]]") {
		t.Error("new wikilink [[New Name]] not found")
	}
	if !contains(got, "[[New Name#Section|details]]") {
		t.Error("new wikilink [[New Name#Section|details]] not found")
	}
}

func TestCmdMove_FolderOnlyNoLinkUpdate(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "_inbox"), 0755)

	// The note being moved (same filename, different folder)
	os.WriteFile(
		filepath.Join(vaultDir, "_inbox", "Note.md"),
		[]byte("# Note\n"),
		0644,
	)

	// Another note referencing it
	os.WriteFile(
		filepath.Join(vaultDir, "Referrer.md"),
		[]byte("See [[Note]] here.\n"),
		0644,
	)

	params := map[string]string{
		"path": "_inbox/Note.md",
		"to":   "decisions/Note.md",
	}
	if err := cmdMove(vaultDir, params); err != nil {
		t.Fatalf("move: %v", err)
	}

	// Link should remain unchanged (title didn't change)
	data, _ := os.ReadFile(filepath.Join(vaultDir, "Referrer.md"))
	if string(data) != "See [[Note]] here.\n" {
		t.Errorf("referrer was unexpectedly modified: %q", string(data))
	}
}

func TestCmdMove_UpdatesMdLinks(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "_inbox"), 0755)

	// The note being moved
	os.WriteFile(
		filepath.Join(vaultDir, "_inbox", "Note.md"),
		[]byte("# Note\n"),
		0644,
	)

	// Another note referencing it via markdown link
	os.WriteFile(
		filepath.Join(vaultDir, "Referrer.md"),
		[]byte("See [note](_inbox/Note.md) and [heading](_inbox/Note.md#section) here.\n"),
		0644,
	)

	params := map[string]string{
		"path": "_inbox/Note.md",
		"to":   "decisions/Note.md",
	}
	if err := cmdMove(vaultDir, params); err != nil {
		t.Fatalf("move: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(vaultDir, "Referrer.md"))
	got := string(data)

	if strings.Contains(got, "_inbox/Note.md") {
		t.Error("old markdown link path still present")
	}
	if !strings.Contains(got, "decisions/Note.md") {
		t.Error("new markdown link path not found")
	}
	if !strings.Contains(got, "decisions/Note.md#section") {
		t.Error("markdown link fragment not preserved")
	}
}

func TestCmdBacklinks(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "methodology"), 0755)

	os.WriteFile(
		filepath.Join(vaultDir, "methodology", "Developer Agent.md"),
		[]byte("Read [[Session Operating Mode]] first.\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "methodology", "Retro Agent.md"),
		[]byte("# Retro\n\nNo links to SOM.\n"),
		0644,
	)

	// Just verify no error (output goes to stdout)
	params := map[string]string{"file": "Session Operating Mode"}
	if err := cmdBacklinks(vaultDir, params, ""); err != nil {
		t.Fatalf("backlinks: %v", err)
	}
}

func TestCmdLinks(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "methodology"), 0755)

	// Target note with outgoing links
	os.WriteFile(
		filepath.Join(vaultDir, "methodology", "Developer Agent.md"),
		[]byte("# Developer\n\nSee [[Session Operating Mode]] and [[Nonexistent Note]].\n"),
		0644,
	)

	// One of the linked notes exists
	os.WriteFile(
		filepath.Join(vaultDir, "Session Operating Mode.md"),
		[]byte("# SOM\n"),
		0644,
	)

	// Just verify no error (output goes to stdout)
	params := map[string]string{"file": "Developer Agent"}
	if err := cmdLinks(vaultDir, params, ""); err != nil {
		t.Fatalf("links: %v", err)
	}
}

func TestCmdPropertySet(t *testing.T) {
	vaultDir := t.TempDir()

	content := "---\ntype: decision\nstatus: active\ncreated: 2024-01-15\n---\n\n# My Decision\n"
	notePath := filepath.Join(vaultDir, "My Decision.md")
	os.WriteFile(notePath, []byte(content), 0644)

	// Update existing property
	params := map[string]string{
		"file":  "My Decision",
		"name":  "status",
		"value": "archived",
	}
	if err := cmdPropertySet(vaultDir, params); err != nil {
		t.Fatalf("property:set: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	if got := string(data); !contains(got, "status: archived") {
		t.Errorf("property not updated: %s", got)
	}
	if got := string(data); contains(got, "status: active") {
		t.Errorf("old property value still present: %s", got)
	}

	// Add new property
	params = map[string]string{
		"file":  "My Decision",
		"name":  "confidence",
		"value": "high",
	}
	if err := cmdPropertySet(vaultDir, params); err != nil {
		t.Fatalf("property:set (add): %v", err)
	}

	data, _ = os.ReadFile(notePath)
	if got := string(data); !contains(got, "confidence: high") {
		t.Errorf("new property not added: %s", got)
	}
}

func TestCmdSearch(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "decisions"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)

	// Note with matching title
	os.WriteFile(filepath.Join(vaultDir, "decisions", "System Architecture.md"),
		[]byte("# Architecture\nSome content."), 0644)

	// Note with matching content but not title
	os.WriteFile(filepath.Join(vaultDir, "decisions", "Other Decision.md"),
		[]byte("# Other\nThis relates to system infrastructure."), 0644)

	// Note that should not match
	os.WriteFile(filepath.Join(vaultDir, "decisions", "Unrelated.md"),
		[]byte("# Unrelated\nNothing here."), 0644)

	// Hidden note that should be skipped
	os.WriteFile(filepath.Join(vaultDir, ".obsidian", "system-config.md"),
		[]byte("# Config\nsystem settings."), 0644)

	params := map[string]string{"query": "system"}
	// cmdSearch writes to stdout; just verify no error
	if err := cmdSearch(vaultDir, params, ""); err != nil {
		t.Fatalf("search: %v", err)
	}
}

func TestParseSearchQuery(t *testing.T) {
	tests := []struct {
		query       string
		wantText    string
		wantFilters map[string]string
	}{
		{
			query:       "architecture",
			wantText:    "architecture",
			wantFilters: map[string]string{},
		},
		{
			query:       "architecture [status:active]",
			wantText:    "architecture",
			wantFilters: map[string]string{"status": "active"},
		},
		{
			query:       "[status:active] [type:decision]",
			wantText:    "",
			wantFilters: map[string]string{"status": "active", "type": "decision"},
		},
		{
			query:       "search term [status:active] more text",
			wantText:    "search term  more text",
			wantFilters: map[string]string{"status": "active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			text, filters := parseSearchQuery(tt.query)
			if text != tt.wantText {
				t.Errorf("text = %q, want %q", text, tt.wantText)
			}
			if len(filters) != len(tt.wantFilters) {
				t.Errorf("got %d filters, want %d", len(filters), len(tt.wantFilters))
			}
			for k, v := range tt.wantFilters {
				if filters[k] != v {
					t.Errorf("filter[%q] = %q, want %q", k, filters[k], v)
				}
			}
		})
	}
}

func TestCmdSearch_PropertyFilter(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "decisions"), 0755)

	os.WriteFile(filepath.Join(vaultDir, "decisions", "Active Decision.md"),
		[]byte("---\ntype: decision\nstatus: active\n---\n\n# Active\nSome content."), 0644)

	os.WriteFile(filepath.Join(vaultDir, "decisions", "Archived Decision.md"),
		[]byte("---\ntype: decision\nstatus: archived\n---\n\n# Archived\nOther content."), 0644)

	os.WriteFile(filepath.Join(vaultDir, "decisions", "No Frontmatter.md"),
		[]byte("# No FM\nPlain note."), 0644)

	// Filter by status:active should find only the active note
	params := map[string]string{"query": "[status:active]"}
	// Just verify no error; output goes to stdout
	if err := cmdSearch(vaultDir, params, ""); err != nil {
		t.Fatalf("search with property filter: %v", err)
	}
}

func TestCmdSearch_PropertyFilterWithText(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(filepath.Join(vaultDir, "Match.md"),
		[]byte("---\nstatus: active\n---\n\n# Match\narchitecture discussion."), 0644)

	os.WriteFile(filepath.Join(vaultDir, "NoMatch.md"),
		[]byte("---\nstatus: archived\n---\n\n# NoMatch\narchitecture discussion."), 0644)

	params := map[string]string{"query": "architecture [status:active]"}
	if err := cmdSearch(vaultDir, params, ""); err != nil {
		t.Fatalf("search with text + filter: %v", err)
	}
}

func TestCmdSearch_MultipleFilters(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(filepath.Join(vaultDir, "Both.md"),
		[]byte("---\ntype: decision\nstatus: active\n---\n\n# Both\nContent."), 0644)

	os.WriteFile(filepath.Join(vaultDir, "OneOnly.md"),
		[]byte("---\ntype: pattern\nstatus: active\n---\n\n# OneOnly\nContent."), 0644)

	params := map[string]string{"query": "[type:decision] [status:active]"}
	if err := cmdSearch(vaultDir, params, ""); err != nil {
		t.Fatalf("search with multiple filters: %v", err)
	}
}

func TestCmdPrepend(t *testing.T) {
	vaultDir := t.TempDir()

	// With frontmatter: should insert after ---
	os.WriteFile(
		filepath.Join(vaultDir, "WithFM.md"),
		[]byte("---\ntype: note\n---\n\n# Existing Content\n"),
		0644,
	)

	params := map[string]string{"file": "WithFM", "content": "PREPENDED\n"}
	if err := cmdPrepend(vaultDir, params, false); err != nil {
		t.Fatalf("prepend with FM: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(vaultDir, "WithFM.md"))
	got := string(data)
	want := "---\ntype: note\n---\nPREPENDED\n\n# Existing Content\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Without frontmatter: should insert at top
	os.WriteFile(
		filepath.Join(vaultDir, "NoFM.md"),
		[]byte("# Existing Content\n"),
		0644,
	)

	params = map[string]string{"file": "NoFM", "content": "TOP\n"}
	if err := cmdPrepend(vaultDir, params, false); err != nil {
		t.Fatalf("prepend without FM: %v", err)
	}

	data, _ = os.ReadFile(filepath.Join(vaultDir, "NoFM.md"))
	got = string(data)
	want = "TOP\n# Existing Content\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCmdDelete_Trash(t *testing.T) {
	vaultDir := t.TempDir()

	notePath := filepath.Join(vaultDir, "ToTrash.md")
	os.WriteFile(notePath, []byte("# Delete me\n"), 0644)

	params := map[string]string{"file": "ToTrash"}
	if err := cmdDelete(vaultDir, params, false); err != nil {
		t.Fatalf("delete (trash): %v", err)
	}

	// Original should be gone
	if _, err := os.Stat(notePath); !os.IsNotExist(err) {
		t.Error("original file still exists after trash")
	}

	// Should exist in .trash
	trashPath := filepath.Join(vaultDir, ".trash", "ToTrash.md")
	if _, err := os.Stat(trashPath); os.IsNotExist(err) {
		t.Error("file not found in .trash")
	}
}

func TestCmdDelete_Permanent(t *testing.T) {
	vaultDir := t.TempDir()

	notePath := filepath.Join(vaultDir, "ToDelete.md")
	os.WriteFile(notePath, []byte("# Delete me\n"), 0644)

	params := map[string]string{"file": "ToDelete"}
	if err := cmdDelete(vaultDir, params, true); err != nil {
		t.Fatalf("delete (permanent): %v", err)
	}

	if _, err := os.Stat(notePath); !os.IsNotExist(err) {
		t.Error("file still exists after permanent delete")
	}

	// Should NOT exist in .trash
	trashPath := filepath.Join(vaultDir, ".trash", "ToDelete.md")
	if _, err := os.Stat(trashPath); !os.IsNotExist(err) {
		t.Error("file unexpectedly found in .trash after permanent delete")
	}
}

func TestCmdProperties(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "Props.md"),
		[]byte("---\ntype: decision\nstatus: active\n---\n\n# Note\n"),
		0644,
	)

	// Just verify no error (output goes to stdout)
	params := map[string]string{"file": "Props"}
	if err := cmdProperties(vaultDir, params, ""); err != nil {
		t.Fatalf("properties: %v", err)
	}
}

func TestCmdPropertyRemove(t *testing.T) {
	vaultDir := t.TempDir()

	notePath := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(notePath, []byte("---\ntype: decision\nstatus: active\ncreated: 2024-01-15\n---\n\n# Note\n"), 0644)

	params := map[string]string{"file": "Note", "name": "status"}
	if err := cmdPropertyRemove(vaultDir, params); err != nil {
		t.Fatalf("property:remove: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if contains(got, "status:") {
		t.Error("property 'status' still present after removal")
	}
	if !contains(got, "type: decision") || !contains(got, "created: 2024-01-15") {
		t.Error("other properties were affected by removal")
	}
}

func TestCmdOrphans(t *testing.T) {
	vaultDir := t.TempDir()

	// A references B; C is orphaned
	os.WriteFile(
		filepath.Join(vaultDir, "A.md"),
		[]byte("# A\n\nSee [[B]] for details.\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "B.md"),
		[]byte("# B\n\nReferenced by A.\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "C.md"),
		[]byte("# C\n\nNobody links to me.\n"),
		0644,
	)

	// Just verify no error
	if err := cmdOrphans(vaultDir, ""); err != nil {
		t.Fatalf("orphans: %v", err)
	}
}

func TestCmdOrphans_AliasAware(t *testing.T) {
	vaultDir := t.TempDir()

	// A references "Alt Name" which is an alias of B
	os.WriteFile(
		filepath.Join(vaultDir, "A.md"),
		[]byte("# A\n\nSee [[Alt Name]].\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "B.md"),
		[]byte("---\naliases: [Alt Name]\n---\n\n# B\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "C.md"),
		[]byte("# C\n\nOrphan.\n"),
		0644,
	)

	// Just verify no error (A is orphaned since nothing links to it,
	// B is NOT orphaned due to alias, C is orphaned)
	if err := cmdOrphans(vaultDir, ""); err != nil {
		t.Fatalf("orphans: %v", err)
	}
}

func TestCmdUnresolved(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "Existing.md"),
		[]byte("# Existing\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "Referrer.md"),
		[]byte("# Referrer\n\n[[Existing]] and [[Ghost Note]] and ![[Missing Embed]].\n"),
		0644,
	)

	// Just verify no error
	if err := cmdUnresolved(vaultDir, ""); err != nil {
		t.Fatalf("unresolved: %v", err)
	}
}

func TestCmdFiles(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "sub"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)
	os.WriteFile(filepath.Join(vaultDir, "root.md"), []byte("# Root\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "sub", "child.md"), []byte("# Child\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, ".obsidian", "config.md"), []byte("hidden\n"), 0644)

	// List all
	params := map[string]string{}
	if err := cmdFiles(vaultDir, params, false, ""); err != nil {
		t.Fatalf("files: %v", err)
	}

	// Total count
	if err := cmdFiles(vaultDir, params, true, ""); err != nil {
		t.Fatalf("files total: %v", err)
	}

	// Filter by folder
	params = map[string]string{"folder": "sub"}
	if err := cmdFiles(vaultDir, params, false, ""); err != nil {
		t.Fatalf("files folder: %v", err)
	}
}

// ---------------------------------------------------------------------------
// write command tests
// ---------------------------------------------------------------------------

// Unit test 1: write replaces body while preserving frontmatter
func TestCmdWriteReplacesBody(t *testing.T) {
	vaultDir := t.TempDir()

	original := "---\ntype: decision\nstatus: active\n---\n\n# Old Body\n\nOld content here.\n"
	notePath := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(notePath, []byte(original), 0644)

	params := map[string]string{
		"file":    "Note",
		"content": "# New Body\n\nCompletely replaced.\n",
	}
	if err := cmdWrite(vaultDir, params, false); err != nil {
		t.Fatalf("write: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	// Frontmatter must be preserved
	if !strings.Contains(got, "type: decision") {
		t.Error("frontmatter property 'type' lost after write")
	}
	if !strings.Contains(got, "status: active") {
		t.Error("frontmatter property 'status' lost after write")
	}

	// Body must be replaced
	if strings.Contains(got, "Old Body") {
		t.Error("old body content still present after write")
	}
	if !strings.Contains(got, "Completely replaced.") {
		t.Error("new body content not found after write")
	}
}

// Unit test 2: write to note without frontmatter replaces entire content
func TestCmdWriteNoFrontmatter(t *testing.T) {
	vaultDir := t.TempDir()

	original := "# Old Title\n\nSome old content.\n"
	notePath := filepath.Join(vaultDir, "Plain.md")
	os.WriteFile(notePath, []byte(original), 0644)

	params := map[string]string{
		"file":    "Plain",
		"content": "# New Title\n\nNew content.\n",
	}
	if err := cmdWrite(vaultDir, params, false); err != nil {
		t.Fatalf("write: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if strings.Contains(got, "Old Title") {
		t.Error("old content still present in note without frontmatter")
	}
	if got != "# New Title\n\nNew content.\n" {
		t.Errorf("unexpected content: %q", got)
	}
}

// Unit test 3: write empty content results in frontmatter-only note
func TestCmdWriteEmptyBody(t *testing.T) {
	vaultDir := t.TempDir()

	original := "---\ntype: note\n---\n\n# Content\n"
	notePath := filepath.Join(vaultDir, "EmptyBody.md")
	os.WriteFile(notePath, []byte(original), 0644)

	params := map[string]string{
		"file":    "EmptyBody",
		"content": "",
	}
	if err := cmdWrite(vaultDir, params, false); err != nil {
		t.Fatalf("write: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	// Should have frontmatter but no body content
	if !strings.Contains(got, "---\ntype: note\n---") {
		t.Error("frontmatter lost when writing empty body")
	}
	if strings.Contains(got, "# Content") {
		t.Error("old body still present after writing empty content")
	}
}

// Unit test 4: write without file= returns error
func TestCmdWriteRequiresFile(t *testing.T) {
	vaultDir := t.TempDir()

	params := map[string]string{
		"content": "some content",
	}
	err := cmdWrite(vaultDir, params, false)
	if err == nil {
		t.Fatal("expected error when file= not provided")
	}
	if !strings.Contains(err.Error(), "file") {
		t.Errorf("error should mention 'file', got: %v", err)
	}
}

// Unit test 5: write to nonexistent note returns error
func TestCmdWriteNoteNotFound(t *testing.T) {
	vaultDir := t.TempDir()

	params := map[string]string{
		"file":    "Nonexistent",
		"content": "some content",
	}
	err := cmdWrite(vaultDir, params, false)
	if err == nil {
		t.Fatal("expected error for nonexistent note")
	}
}

// ---------------------------------------------------------------------------
// Integration tests (real files, no mocks)
// ---------------------------------------------------------------------------

// Integration test 6: create real note with frontmatter + body, write new body, verify frontmatter intact
func TestWritePreservesFrontmatter(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "decisions"), 0755)

	original := "---\ntype: decision\nstatus: active\ncreated: 2026-02-19\naliases: [Dec1, First Decision]\n---\n\n# Original Decision\n\nOriginal body with [[wikilinks]] and content.\n"
	notePath := filepath.Join(vaultDir, "decisions", "My Decision.md")
	os.WriteFile(notePath, []byte(original), 0644)

	params := map[string]string{
		"file":    "My Decision",
		"content": "# Updated Decision\n\nNew body with different content.\n",
	}
	if err := cmdWrite(vaultDir, params, false); err != nil {
		t.Fatalf("write: %v", err)
	}

	data, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("failed to read back note: %v", err)
	}
	got := string(data)

	// All frontmatter properties must be intact
	if !strings.Contains(got, "type: decision") {
		t.Error("frontmatter 'type' lost")
	}
	if !strings.Contains(got, "status: active") {
		t.Error("frontmatter 'status' lost")
	}
	if !strings.Contains(got, "created: 2026-02-19") {
		t.Error("frontmatter 'created' lost")
	}
	if !strings.Contains(got, "aliases: [Dec1, First Decision]") {
		t.Error("frontmatter 'aliases' lost")
	}

	// New body must be present
	if !strings.Contains(got, "# Updated Decision") {
		t.Error("new body not found")
	}
	if !strings.Contains(got, "New body with different content.") {
		t.Error("new body content not found")
	}

	// Old body must be gone
	if strings.Contains(got, "Original Decision") {
		t.Error("old body content still present")
	}
	if strings.Contains(got, "[[wikilinks]]") {
		t.Error("old wikilinks still present in body")
	}
}

// Integration test 7: write content piped from stdin (test the stdin fallback path)
// Note: We cannot truly pipe stdin in a test, but we can test the code path
// by passing content="" and verifying behavior. The actual stdin path is tested
// by verifying the function signature accepts empty content gracefully when
// there's no piped input. Instead, we test that content= takes priority.
func TestWriteViaContentParam(t *testing.T) {
	vaultDir := t.TempDir()

	notePath := filepath.Join(vaultDir, "StdinNote.md")
	os.WriteFile(notePath, []byte("---\ntitle: stdin test\n---\n\nOld body.\n"), 0644)

	params := map[string]string{
		"file":    "StdinNote",
		"content": "Body from content param.\n",
	}
	if err := cmdWrite(vaultDir, params, false); err != nil {
		t.Fatalf("write: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if !strings.Contains(got, "Body from content param.") {
		t.Error("content= param not applied")
	}
	if strings.Contains(got, "Old body.") {
		t.Error("old body still present")
	}
}

// Integration test 8: write content then read back with cmdRead to verify round-trip
func TestWriteThenRead(t *testing.T) {
	vaultDir := t.TempDir()

	notePath := filepath.Join(vaultDir, "RoundTrip.md")
	os.WriteFile(notePath, []byte("---\ntype: test\n---\n\n# Before\n"), 0644)

	newBody := "# After Write\n\nThis is the new content.\n"
	writeParams := map[string]string{
		"file":    "RoundTrip",
		"content": newBody,
	}
	if err := cmdWrite(vaultDir, writeParams, false); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read back with resolveNote (same path cmdRead uses)
	path, err := resolveNote(vaultDir, "RoundTrip")
	if err != nil {
		t.Fatalf("resolveNote: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile: %v", err)
	}

	got := string(data)
	if !strings.Contains(got, "type: test") {
		t.Error("frontmatter not preserved on read-back")
	}
	if !strings.Contains(got, "# After Write") {
		t.Error("new body not found on read-back")
	}
	if !strings.Contains(got, "This is the new content.") {
		t.Error("new body content not found on read-back")
	}
}

// Integration test 9: write to nonexistent file returns error, file does not appear
func TestWriteDoesNotCreateFile(t *testing.T) {
	vaultDir := t.TempDir()

	params := map[string]string{
		"file":    "Ghost Note",
		"content": "Should not be created",
	}
	err := cmdWrite(vaultDir, params, false)
	if err == nil {
		t.Fatal("expected error for nonexistent note")
	}

	// Verify no file was created
	matches, _ := filepath.Glob(filepath.Join(vaultDir, "*.md"))
	if len(matches) > 0 {
		t.Errorf("unexpected files created: %v", matches)
	}
}

// E2E test 10: full workflow -- create vault, create note with frontmatter and body,
// run cmdWrite with new content, verify with cmdRead and cmdProperties
func TestE2EWriteCommand(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "methodology"), 0755)

	// Step 1: Create a note with frontmatter and body
	originalContent := "---\ntype: methodology\nstatus: active\ncreated: 2026-02-19\n---\n\n# Original Heading\n\nOriginal body paragraph.\n\n## Section 2\n\nMore original content.\n"
	notePath := filepath.Join(vaultDir, "methodology", "Test Method.md")
	os.WriteFile(notePath, []byte(originalContent), 0644)

	// Step 2: Write new body content
	newBody := "# Revised Heading\n\nCompletely new body.\n\n## New Section\n\nAll new content here.\n"
	writeParams := map[string]string{
		"file":    "Test Method",
		"content": newBody,
	}
	if err := cmdWrite(vaultDir, writeParams, false); err != nil {
		t.Fatalf("E2E write: %v", err)
	}

	// Step 3: Verify with direct file read (simulates cmdRead)
	data, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("E2E read: %v", err)
	}
	got := string(data)

	// Frontmatter must be fully preserved
	if !strings.Contains(got, "type: methodology") {
		t.Error("E2E: frontmatter 'type' missing")
	}
	if !strings.Contains(got, "status: active") {
		t.Error("E2E: frontmatter 'status' missing")
	}
	if !strings.Contains(got, "created: 2026-02-19") {
		t.Error("E2E: frontmatter 'created' missing")
	}

	// New body must be present
	if !strings.Contains(got, "# Revised Heading") {
		t.Error("E2E: new heading not found")
	}
	if !strings.Contains(got, "All new content here.") {
		t.Error("E2E: new body content not found")
	}

	// Old body must be gone
	if strings.Contains(got, "Original Heading") {
		t.Error("E2E: old heading still present")
	}
	if strings.Contains(got, "Original body paragraph") {
		t.Error("E2E: old body still present")
	}

	// Step 4: Verify properties are intact via extractFrontmatter
	yaml, _, hasFM := extractFrontmatter(got)
	if !hasFM {
		t.Fatal("E2E: no frontmatter found after write")
	}
	typeVal, ok := frontmatterGetValue(yaml, "type")
	if !ok || typeVal != "methodology" {
		t.Errorf("E2E: type property = %q, want 'methodology'", typeVal)
	}
	statusVal, ok := frontmatterGetValue(yaml, "status")
	if !ok || statusVal != "active" {
		t.Errorf("E2E: status property = %q, want 'active'", statusVal)
	}
	createdVal, ok := frontmatterGetValue(yaml, "created")
	if !ok || createdVal != "2026-02-19" {
		t.Errorf("E2E: created property = %q, want '2026-02-19'", createdVal)
	}

	// Step 5: Verify the complete structure (frontmatter + separator + body)
	expectedPrefix := "---\ntype: methodology\nstatus: active\ncreated: 2026-02-19\n---\n"
	if !strings.HasPrefix(got, expectedPrefix) {
		t.Errorf("E2E: file does not start with expected frontmatter block.\nGot prefix: %q", got[:min(len(got), len(expectedPrefix)+20)])
	}
}

// ---------------------------------------------------------------------------
// patch command tests (VLT-54o)
// ---------------------------------------------------------------------------

// Unit test 1: replace section content under ## heading
func TestPatchByHeadingReplace(t *testing.T) {
	vaultDir := t.TempDir()

	content := "# Title\n\n## Section A\ncontent a\nmore a\n\n## Section B\ncontent b\n"
	notePath := filepath.Join(vaultDir, "Note.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "Note",
		"heading": "## Section A",
		"content": "replaced content\n",
	}
	if err := cmdPatch(vaultDir, params, false, false); err != nil {
		t.Fatalf("patch: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if !strings.Contains(got, "## Section A\nreplaced content\n") {
		t.Errorf("section not replaced correctly.\ngot: %q", got)
	}
	if !strings.Contains(got, "## Section B\ncontent b\n") {
		t.Error("Section B was affected by patching Section A")
	}
	if strings.Contains(got, "content a") {
		t.Error("old section A content still present")
	}
}

// Unit test 2: other sections remain unchanged after heading patch
func TestPatchByHeadingPreservesOtherSections(t *testing.T) {
	vaultDir := t.TempDir()

	content := "## First\nfirst content\n## Second\nsecond content\n## Third\nthird content\n"
	notePath := filepath.Join(vaultDir, "Multi.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "Multi",
		"heading": "## Second",
		"content": "new second\n",
	}
	if err := cmdPatch(vaultDir, params, false, false); err != nil {
		t.Fatalf("patch: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if !strings.Contains(got, "## First\nfirst content\n") {
		t.Error("First section was modified")
	}
	if !strings.Contains(got, "## Third\nthird content\n") {
		t.Error("Third section was modified")
	}
	if !strings.Contains(got, "## Second\nnew second\n") {
		t.Errorf("Second section not correctly replaced. got: %q", got)
	}
}

// Unit test 3: heading match is case-insensitive
func TestPatchByHeadingCaseInsensitive(t *testing.T) {
	vaultDir := t.TempDir()

	content := "## My Section\noriginal\n"
	notePath := filepath.Join(vaultDir, "Case.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "Case",
		"heading": "## my section",
		"content": "patched\n",
	}
	if err := cmdPatch(vaultDir, params, false, false); err != nil {
		t.Fatalf("patch: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if strings.Contains(got, "original") {
		t.Error("case-insensitive heading match failed, old content still present")
	}
	if !strings.Contains(got, "patched") {
		t.Error("patched content not found")
	}
}

// Unit test 4: subsections included in scope (section extends to next equal-or-higher heading)
func TestPatchByHeadingScopeToNextEqualLevel(t *testing.T) {
	vaultDir := t.TempDir()

	content := "## Section A\ncontent a\n### Subsection\nsub content\n## Section B\ncontent b\n"
	notePath := filepath.Join(vaultDir, "Scope.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "Scope",
		"heading": "## Section A",
		"content": "all new\n",
	}
	if err := cmdPatch(vaultDir, params, false, false); err != nil {
		t.Fatalf("patch: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	// Subsection and its content should be replaced
	if strings.Contains(got, "### Subsection") {
		t.Error("subsection heading should have been replaced")
	}
	if strings.Contains(got, "sub content") {
		t.Error("subsection content should have been replaced")
	}
	if !strings.Contains(got, "## Section A\nall new\n") {
		t.Errorf("section A not correctly replaced. got: %q", got)
	}
	if !strings.Contains(got, "## Section B\ncontent b\n") {
		t.Error("Section B was affected")
	}
}

// Unit test 5: section extends to end of file when at EOF
func TestPatchByHeadingAtEOF(t *testing.T) {
	vaultDir := t.TempDir()

	content := "## Earlier\nearlier content\n## Last Section\nlast content\nmore last\n"
	notePath := filepath.Join(vaultDir, "EOF.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "EOF",
		"heading": "## Last Section",
		"content": "replaced last\n",
	}
	if err := cmdPatch(vaultDir, params, false, false); err != nil {
		t.Fatalf("patch: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if strings.Contains(got, "last content") {
		t.Error("old EOF section content still present")
	}
	if !strings.Contains(got, "## Last Section\nreplaced last\n") {
		t.Errorf("EOF section not replaced. got: %q", got)
	}
}

// Unit test 6: delete heading + content
func TestPatchByHeadingDelete(t *testing.T) {
	vaultDir := t.TempDir()

	content := "## Keep\nkeep content\n## Remove\nremove content\n## Also Keep\nalso keep\n"
	notePath := filepath.Join(vaultDir, "Del.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "Del",
		"heading": "## Remove",
	}
	if err := cmdPatch(vaultDir, params, true, false); err != nil {
		t.Fatalf("patch delete: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if strings.Contains(got, "## Remove") {
		t.Error("deleted heading still present")
	}
	if strings.Contains(got, "remove content") {
		t.Error("deleted section content still present")
	}
	if !strings.Contains(got, "## Keep\nkeep content\n") {
		t.Error("Keep section was affected")
	}
	if !strings.Contains(got, "## Also Keep\nalso keep\n") {
		t.Error("Also Keep section was affected")
	}
}

// Unit test 7: single line replacement
func TestPatchByLineReplace(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line one\nline two\nline three\nline four\n"
	notePath := filepath.Join(vaultDir, "Lines.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "Lines",
		"line":    "2",
		"content": "REPLACED",
	}
	if err := cmdPatch(vaultDir, params, false, false); err != nil {
		t.Fatalf("patch line: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if strings.Contains(got, "line two") {
		t.Error("old line 2 still present")
	}
	if !strings.Contains(got, "REPLACED") {
		t.Error("replacement content not found")
	}
	// Check structure: line 1 and 3-4 should be intact
	lines := strings.Split(got, "\n")
	if lines[0] != "line one" {
		t.Errorf("line 1 changed: %q", lines[0])
	}
	if lines[2] != "line three" {
		t.Errorf("line 3 changed: %q", lines[2])
	}
}

// Unit test 8: line range replacement
func TestPatchByLineRangeReplace(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\n"
	notePath := filepath.Join(vaultDir, "Range.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "Range",
		"line":    "3-5",
		"content": "REPLACED BLOCK",
	}
	if err := cmdPatch(vaultDir, params, false, false); err != nil {
		t.Fatalf("patch line range: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if strings.Contains(got, "line 3") || strings.Contains(got, "line 4") || strings.Contains(got, "line 5") {
		t.Error("replaced lines still present")
	}
	if !strings.Contains(got, "REPLACED BLOCK") {
		t.Error("replacement content not found")
	}
	if !strings.Contains(got, "line 1") || !strings.Contains(got, "line 2") {
		t.Error("lines before range were affected")
	}
	if !strings.Contains(got, "line 6") {
		t.Error("line after range was affected")
	}
}

// Unit test 9: single line deletion
func TestPatchByLineDelete(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line 1\nline 2\nline 3\nline 4\n"
	notePath := filepath.Join(vaultDir, "DelLine.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file": "DelLine",
		"line": "3",
	}
	if err := cmdPatch(vaultDir, params, true, false); err != nil {
		t.Fatalf("patch delete line: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if strings.Contains(got, "line 3") {
		t.Error("deleted line still present")
	}
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %v", len(lines), lines)
	}
}

// Unit test 10: line range deletion
func TestPatchByLineRangeDelete(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	notePath := filepath.Join(vaultDir, "DelRange.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file": "DelRange",
		"line": "2-4",
	}
	if err := cmdPatch(vaultDir, params, true, false); err != nil {
		t.Fatalf("patch delete range: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	if strings.Contains(got, "line 2") || strings.Contains(got, "line 3") || strings.Contains(got, "line 4") {
		t.Error("deleted lines still present")
	}
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "line 1" || lines[1] != "line 5" {
		t.Errorf("remaining lines wrong: %v", lines)
	}
}

// Unit test 11: error for line number beyond file length
func TestPatchLineOutOfRange(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line 1\nline 2\n"
	notePath := filepath.Join(vaultDir, "Short.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "Short",
		"line":    "10",
		"content": "nope",
	}
	err := cmdPatch(vaultDir, params, false, false)
	if err == nil {
		t.Fatal("expected error for out-of-range line")
	}
	if !strings.Contains(err.Error(), "out of range") && !strings.Contains(err.Error(), "beyond") {
		t.Errorf("error should mention range issue, got: %v", err)
	}
}

// Unit test 12: error for nonexistent heading
func TestPatchHeadingNotFound(t *testing.T) {
	vaultDir := t.TempDir()

	content := "## Existing\ncontent\n"
	notePath := filepath.Join(vaultDir, "NoHead.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "NoHead",
		"heading": "## Nonexistent",
		"content": "nope",
	}
	err := cmdPatch(vaultDir, params, false, false)
	if err == nil {
		t.Fatal("expected error for nonexistent heading")
	}
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "heading") {
		t.Errorf("error should mention heading not found, got: %v", err)
	}
}

// Unit test 13: error without file=
func TestPatchRequiresFile(t *testing.T) {
	vaultDir := t.TempDir()

	params := map[string]string{
		"heading": "## Heading",
		"content": "content",
	}
	err := cmdPatch(vaultDir, params, false, false)
	if err == nil {
		t.Fatal("expected error when file= not provided")
	}
	if !strings.Contains(err.Error(), "file") {
		t.Errorf("error should mention 'file', got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Integration tests (real files, no mocks)
// ---------------------------------------------------------------------------

// Integration test 14: create real note with multiple sections, patch one, read back
func TestPatchByHeadingIntegration(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "methodology"), 0755)

	content := "---\ntype: methodology\nstatus: active\n---\n\n# Main Title\n\nIntro paragraph.\n\n## Architecture\n\nOriginal architecture description.\nMore details.\n\n## Implementation\n\nImpl details.\n"
	notePath := filepath.Join(vaultDir, "methodology", "Design Doc.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "Design Doc",
		"heading": "## Architecture",
		"content": "Completely revised architecture.\nNew approach.\n",
	}
	if err := cmdPatch(vaultDir, params, false, false); err != nil {
		t.Fatalf("integration patch: %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	got := string(data)

	// Heading preserved
	if !strings.Contains(got, "## Architecture") {
		t.Error("heading was removed")
	}
	// New content present
	if !strings.Contains(got, "Completely revised architecture.") {
		t.Error("new content not found")
	}
	// Old content gone
	if strings.Contains(got, "Original architecture description.") {
		t.Error("old content still present")
	}
	// Other section intact
	if !strings.Contains(got, "## Implementation\n\nImpl details.") {
		t.Error("Implementation section was affected")
	}
	// Frontmatter intact
	if !strings.Contains(got, "type: methodology") {
		t.Error("frontmatter lost")
	}
}

// Integration test 15: create real note, patch specific line, verify with file read
func TestPatchByLineIntegration(t *testing.T) {
	vaultDir := t.TempDir()

	content := "---\nstatus: draft\n---\n\n# Title\n\nLine A\nLine B\nLine C\n"
	notePath := filepath.Join(vaultDir, "LineNote.md")
	os.WriteFile(notePath, []byte(content), 0644)

	// Line 7 is "Line A" (1-based: 1=---, 2=status:draft, 3=---, 4=empty, 5=# Title, 6=empty, 7=Line A)
	params := map[string]string{
		"file":    "LineNote",
		"line":    "7",
		"content": "PATCHED A",
	}
	if err := cmdPatch(vaultDir, params, false, false); err != nil {
		t.Fatalf("integration line patch: %v", err)
	}

	data, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	got := string(data)

	if strings.Contains(got, "Line A") {
		t.Error("old line A still present")
	}
	if !strings.Contains(got, "PATCHED A") {
		t.Error("patched content not found")
	}
	// Frontmatter intact
	if !strings.Contains(got, "status: draft") {
		t.Error("frontmatter affected")
	}
}

// Integration test 16: delete a section, verify remaining content intact
func TestPatchDeleteSectionIntegration(t *testing.T) {
	vaultDir := t.TempDir()

	content := "---\ntype: note\n---\n\n## Keep This\n\nKeep content.\n\n## Delete This\n\nDelete content.\n\n## Also Keep\n\nAlso keep content.\n"
	notePath := filepath.Join(vaultDir, "Sections.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "Sections",
		"heading": "## Delete This",
	}
	if err := cmdPatch(vaultDir, params, true, false); err != nil {
		t.Fatalf("integration delete: %v", err)
	}

	data, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	got := string(data)

	if strings.Contains(got, "## Delete This") {
		t.Error("deleted heading still present")
	}
	if strings.Contains(got, "Delete content.") {
		t.Error("deleted content still present")
	}
	if !strings.Contains(got, "## Keep This") || !strings.Contains(got, "Keep content.") {
		t.Error("Keep This section affected")
	}
	if !strings.Contains(got, "## Also Keep") || !strings.Contains(got, "Also keep content.") {
		t.Error("Also Keep section affected")
	}
	// Frontmatter intact
	if !strings.Contains(got, "type: note") {
		t.Error("frontmatter affected")
	}
}

// Integration test 17: patch does not corrupt frontmatter
func TestPatchPreservesFrontmatter(t *testing.T) {
	vaultDir := t.TempDir()

	content := "---\ntype: decision\nstatus: active\ncreated: 2026-02-19\naliases: [Dec1, First]\n---\n\n## Summary\n\nSummary content.\n\n## Details\n\nDetail content.\n"
	notePath := filepath.Join(vaultDir, "FMTest.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "FMTest",
		"heading": "## Summary",
		"content": "New summary.\n",
	}
	if err := cmdPatch(vaultDir, params, false, false); err != nil {
		t.Fatalf("patch: %v", err)
	}

	data, _ := os.ReadFile(notePath)
	got := string(data)

	// Verify all frontmatter properties
	yaml, _, hasFM := extractFrontmatter(got)
	if !hasFM {
		t.Fatal("frontmatter lost after patch")
	}
	if v, ok := frontmatterGetValue(yaml, "type"); !ok || v != "decision" {
		t.Errorf("type = %q, want 'decision'", v)
	}
	if v, ok := frontmatterGetValue(yaml, "status"); !ok || v != "active" {
		t.Errorf("status = %q, want 'active'", v)
	}
	if v, ok := frontmatterGetValue(yaml, "created"); !ok || v != "2026-02-19" {
		t.Errorf("created = %q, want '2026-02-19'", v)
	}
	aliases := frontmatterGetList(yaml, "aliases")
	if len(aliases) != 2 || aliases[0] != "Dec1" || aliases[1] != "First" {
		t.Errorf("aliases = %v, want [Dec1, First]", aliases)
	}
}

// Integration test 18: patch a section that contained wikilinks, verify backlinks updated
func TestPatchThenBacklinks(t *testing.T) {
	vaultDir := t.TempDir()

	// Note with wikilinks in a section
	content := "## Links\n\nSee [[Target]] for details.\n\n## Other\n\nOther stuff.\n"
	os.WriteFile(filepath.Join(vaultDir, "Linker.md"), []byte(content), 0644)

	// The target note
	os.WriteFile(filepath.Join(vaultDir, "Target.md"), []byte("# Target\n"), 0644)

	// Verify backlink exists before patch
	results, err := findBacklinks(vaultDir, "Target")
	if err != nil {
		t.Fatalf("backlinks before patch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected backlink to Target before patch")
	}

	// Patch the Links section, removing the wikilink
	params := map[string]string{
		"file":    "Linker",
		"heading": "## Links",
		"content": "No links here anymore.\n",
	}
	if err := cmdPatch(vaultDir, params, false, false); err != nil {
		t.Fatalf("patch: %v", err)
	}

	// Verify backlink is gone after patch
	results, err = findBacklinks(vaultDir, "Target")
	if err != nil {
		t.Fatalf("backlinks after patch: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no backlinks to Target after patch, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// read heading extraction tests (VLT-4lr)
// ---------------------------------------------------------------------------

// Unit test 1: findSection helper finds correct section boundaries
func TestFindSection(t *testing.T) {
	lines := strings.Split("## Section A\ncontent a\nmore a\n## Section B\ncontent b\n", "\n")

	bounds, found := findSection(lines, "## Section A")
	if !found {
		t.Fatal("section not found")
	}
	if bounds.HeadingLine != 0 {
		t.Errorf("HeadingLine = %d, want 0", bounds.HeadingLine)
	}
	if bounds.ContentStart != 1 {
		t.Errorf("ContentStart = %d, want 1", bounds.ContentStart)
	}
	if bounds.ContentEnd != 3 {
		t.Errorf("ContentEnd = %d, want 3", bounds.ContentEnd)
	}
}

// Unit test 2: section extends to EOF when it is the last section
func TestFindSectionAtEOF(t *testing.T) {
	lines := strings.Split("## First\nfirst content\n## Last\nlast content\nmore last\n", "\n")

	bounds, found := findSection(lines, "## Last")
	if !found {
		t.Fatal("section not found")
	}
	if bounds.ContentEnd != len(lines) {
		t.Errorf("ContentEnd = %d, want %d (EOF)", bounds.ContentEnd, len(lines))
	}
}

// Unit test 3: heading match is case-insensitive
func TestFindSectionCaseInsensitive(t *testing.T) {
	lines := strings.Split("## My Section\ncontent here\n", "\n")

	bounds, found := findSection(lines, "## my section")
	if !found {
		t.Fatal("case-insensitive match failed")
	}
	if bounds.HeadingLine != 0 {
		t.Errorf("HeadingLine = %d, want 0", bounds.HeadingLine)
	}
}

// Unit test 4: returns false for nonexistent heading
func TestFindSectionNotFound(t *testing.T) {
	lines := strings.Split("## Existing\ncontent\n", "\n")

	_, found := findSection(lines, "## Nonexistent")
	if found {
		t.Error("expected section not to be found")
	}
}

// Unit test 5: read with heading= returns heading + section content
func TestReadWithHeadingBasic(t *testing.T) {
	vaultDir := t.TempDir()

	content := "## Section A\ncontent a\nmore a\n## Section B\ncontent b\n"
	os.WriteFile(filepath.Join(vaultDir, "Note.md"), []byte(content), 0644)

	params := map[string]string{
		"file":    "Note",
		"heading": "## Section A",
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmdRead(vaultDir, params)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("cmdRead with heading: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	got := string(buf[:n])

	want := "## Section A\ncontent a\nmore a\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Unit test 6: subsections are included in the extracted section
func TestReadWithHeadingIncludesSubheadings(t *testing.T) {
	vaultDir := t.TempDir()

	content := "## Section A\ncontent a\n### Sub\nsub content\n## Section B\ncontent b\n"
	os.WriteFile(filepath.Join(vaultDir, "Note.md"), []byte(content), 0644)

	params := map[string]string{
		"file":    "Note",
		"heading": "## Section A",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmdRead(vaultDir, params)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("cmdRead with heading: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	got := string(buf[:n])

	// Must include subsection
	if !strings.Contains(got, "### Sub") {
		t.Error("subsection heading not included")
	}
	if !strings.Contains(got, "sub content") {
		t.Error("subsection content not included")
	}
	// Must NOT include Section B
	if strings.Contains(got, "## Section B") {
		t.Error("next section incorrectly included")
	}
}

// Unit test 7: read without heading= returns full note (backward compat)
func TestReadWithoutHeading(t *testing.T) {
	vaultDir := t.TempDir()

	content := "---\ntype: note\n---\n\n# Title\n\nBody content.\n"
	os.WriteFile(filepath.Join(vaultDir, "Note.md"), []byte(content), 0644)

	params := map[string]string{
		"file": "Note",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmdRead(vaultDir, params)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("cmdRead without heading: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	got := string(buf[:n])

	if got != content {
		t.Errorf("full note not returned.\ngot:  %q\nwant: %q", got, content)
	}
}

// ---------------------------------------------------------------------------
// Integration tests for read heading (real files, no mocks)
// ---------------------------------------------------------------------------

// Integration test 8: create note with multiple sections, read specific heading
func TestReadHeadingIntegration(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "methodology"), 0755)

	content := "# Design Doc\n\nIntro paragraph.\n\n## Architecture\n\nArch description.\nMore details.\n\n## Implementation\n\nImpl details.\n\n## Testing\n\nTest plan.\n"
	notePath := filepath.Join(vaultDir, "methodology", "Design Doc.md")
	os.WriteFile(notePath, []byte(content), 0644)

	params := map[string]string{
		"file":    "Design Doc",
		"heading": "## Implementation",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmdRead(vaultDir, params)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("integration read heading: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	got := string(buf[:n])

	// Must contain the heading and its content
	if !strings.Contains(got, "## Implementation") {
		t.Error("heading not in output")
	}
	if !strings.Contains(got, "Impl details.") {
		t.Error("section content not in output")
	}
	// Must NOT contain other sections
	if strings.Contains(got, "## Architecture") {
		t.Error("Architecture section leaked into output")
	}
	if strings.Contains(got, "## Testing") {
		t.Error("Testing section leaked into output")
	}
	if strings.Contains(got, "Intro paragraph") {
		t.Error("intro paragraph leaked into output")
	}
}

// Integration test 9: note with frontmatter, heading section does not include frontmatter
func TestReadHeadingWithFrontmatter(t *testing.T) {
	vaultDir := t.TempDir()

	content := "---\ntype: decision\nstatus: active\n---\n\n# Title\n\nIntro.\n\n## Details\n\nDetail content.\n\n## Conclusion\n\nConclusion content.\n"
	os.WriteFile(filepath.Join(vaultDir, "Note.md"), []byte(content), 0644)

	params := map[string]string{
		"file":    "Note",
		"heading": "## Details",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmdRead(vaultDir, params)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("read heading with frontmatter: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	got := string(buf[:n])

	// Must contain section content
	if !strings.Contains(got, "## Details") {
		t.Error("heading not in output")
	}
	if !strings.Contains(got, "Detail content.") {
		t.Error("section content not in output")
	}
	// Must NOT contain frontmatter or other sections
	if strings.Contains(got, "---") {
		t.Error("frontmatter delimiter leaked into output")
	}
	if strings.Contains(got, "type: decision") {
		t.Error("frontmatter content leaked into output")
	}
	if strings.Contains(got, "## Conclusion") {
		t.Error("Conclusion section leaked into output")
	}
}

// Integration test 10: read the last section (extends to EOF)
func TestReadHeadingLastSection(t *testing.T) {
	vaultDir := t.TempDir()

	content := "## First\n\nFirst content.\n\n## Second\n\nSecond content.\n\n## Last\n\nLast content.\nMore last.\n"
	os.WriteFile(filepath.Join(vaultDir, "Note.md"), []byte(content), 0644)

	params := map[string]string{
		"file":    "Note",
		"heading": "## Last",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmdRead(vaultDir, params)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("read last section: %v", err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	got := string(buf[:n])

	want := "## Last\n\nLast content.\nMore last.\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Integration test 11: nonexistent heading returns error
func TestReadHeadingNotFoundIntegration(t *testing.T) {
	vaultDir := t.TempDir()

	content := "## Existing\n\nSome content.\n"
	os.WriteFile(filepath.Join(vaultDir, "Note.md"), []byte(content), 0644)

	params := map[string]string{
		"file":    "Note",
		"heading": "## Nonexistent",
	}

	err := cmdRead(vaultDir, params)
	if err == nil {
		t.Fatal("expected error for nonexistent heading")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
	if !strings.Contains(err.Error(), "## Nonexistent") {
		t.Errorf("error should mention the heading, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// search context tests (VLT-hha)
// ---------------------------------------------------------------------------

// Unit test 1: search with context=1 shows 1 line before and after
func TestSearchContextBasic(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line one\nline two\nthe architecture is key\nline four\nline five\n"
	os.WriteFile(filepath.Join(vaultDir, "Note.md"), []byte(content), 0644)

	params := map[string]string{"query": "architecture", "context": "1"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("search with context: %v", err)
		}
	})

	// Should contain file:line:content format
	if !strings.Contains(out, "Note.md:") {
		t.Errorf("output should contain file:line format, got: %q", out)
	}
	// Should contain the match line
	if !strings.Contains(out, "the architecture is key") {
		t.Errorf("output should contain match line, got: %q", out)
	}
	// Should contain context lines
	if !strings.Contains(out, "line two") {
		t.Errorf("output should contain line before match, got: %q", out)
	}
	if !strings.Contains(out, "line four") {
		t.Errorf("output should contain line after match, got: %q", out)
	}
	// Should NOT contain lines outside context
	if strings.Contains(out, "line one") {
		t.Errorf("output should not contain lines outside context range, got: %q", out)
	}
	if strings.Contains(out, "line five") {
		t.Errorf("output should not contain lines outside context range, got: %q", out)
	}
}

// Unit test 2: match on line 1, no lines before (no error/panic)
func TestSearchContextAtFileStart(t *testing.T) {
	vaultDir := t.TempDir()

	content := "architecture first line\nline two\nline three\n"
	os.WriteFile(filepath.Join(vaultDir, "Start.md"), []byte(content), 0644)

	params := map[string]string{"query": "architecture", "context": "2"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("search context at start: %v", err)
		}
	})

	if !strings.Contains(out, "architecture first line") {
		t.Errorf("match line missing, got: %q", out)
	}
	// Context after should be present (up to 2 lines)
	if !strings.Contains(out, "line two") {
		t.Errorf("context line after match missing, got: %q", out)
	}
	if !strings.Contains(out, "line three") {
		t.Errorf("second context line after match missing, got: %q", out)
	}
}

// Unit test 3: match on last line, no lines after
func TestSearchContextAtFileEnd(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line one\nline two\narchitecture at the end"
	os.WriteFile(filepath.Join(vaultDir, "End.md"), []byte(content), 0644)

	params := map[string]string{"query": "architecture", "context": "2"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("search context at end: %v", err)
		}
	})

	if !strings.Contains(out, "architecture at the end") {
		t.Errorf("match line missing, got: %q", out)
	}
	if !strings.Contains(out, "line one") {
		t.Errorf("context line before match missing, got: %q", out)
	}
	if !strings.Contains(out, "line two") {
		t.Errorf("context line before match missing, got: %q", out)
	}
}

// Unit test 4: multiple matches with overlapping context are merged
func TestSearchContextMultipleMatches(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line one\narchitecture here\nmiddle\narchitecture again\nline five\n"
	os.WriteFile(filepath.Join(vaultDir, "Multi.md"), []byte(content), 0644)

	params := map[string]string{"query": "architecture", "context": "1"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("search context multiple: %v", err)
		}
	})

	// All lines from the merged range should appear
	if !strings.Contains(out, "line one") {
		t.Errorf("merged context should contain 'line one', got: %q", out)
	}
	if !strings.Contains(out, "architecture here") {
		t.Errorf("first match missing, got: %q", out)
	}
	if !strings.Contains(out, "middle") {
		t.Errorf("middle context line missing, got: %q", out)
	}
	if !strings.Contains(out, "architecture again") {
		t.Errorf("second match missing, got: %q", out)
	}
	if !strings.Contains(out, "line five") {
		t.Errorf("trailing context line missing, got: %q", out)
	}

	// Verify no duplicate lines
	count := strings.Count(out, "middle")
	if count != 1 {
		t.Errorf("'middle' should appear exactly once in merged output, appeared %d times", count)
	}
}

// Unit test 5: context=0 shows only the match line with file:line:content format
func TestSearchContextZero(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line one\narchitecture here\nline three\n"
	os.WriteFile(filepath.Join(vaultDir, "Zero.md"), []byte(content), 0644)

	params := map[string]string{"query": "architecture", "context": "0"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("search context=0: %v", err)
		}
	})

	// Should contain match line in file:line:content format
	if !strings.Contains(out, "Zero.md:2:") {
		t.Errorf("expected file:line format for context=0, got: %q", out)
	}
	if !strings.Contains(out, "architecture here") {
		t.Errorf("match line missing, got: %q", out)
	}
	// Should NOT contain surrounding lines
	if strings.Contains(out, "line one") {
		t.Errorf("context=0 should not show surrounding lines, got: %q", out)
	}
	if strings.Contains(out, "line three") {
		t.Errorf("context=0 should not show surrounding lines, got: %q", out)
	}
}

// Unit test 6: existing query= behavior unchanged when no context used
func TestSearchQueryUnchangedWithoutContext(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(filepath.Join(vaultDir, "TestNote.md"),
		[]byte("# Test\nSome architecture content."), 0644)

	params := map[string]string{"query": "architecture"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("search without context: %v", err)
		}
	})

	// Should use old title (path) format
	if !strings.Contains(out, "TestNote") {
		t.Errorf("expected title in output, got: %q", out)
	}
	// Old format is "Title (path)" -- should NOT have file:line:content colons
	if strings.Contains(out, "TestNote.md:2:") {
		t.Errorf("without context, should use title+path format, not file:line:content, got: %q", out)
	}
}

// Integration test 7: create notes in t.TempDir(), search with context, verify output
func TestSearchContextIntegration(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "decisions"), 0755)

	content := "---\ntype: decision\nstatus: active\n---\n\n# Architecture Decision Record\n\n## Context\n\nThe previous approach used monolithic design.\nAfter reviewing the options,\nthe architecture decision was made\nto use a layered pattern\nwith clear boundaries.\n\n## Decision\n\nWe chose microservices.\n"
	os.WriteFile(filepath.Join(vaultDir, "decisions", "ADR-001.md"), []byte(content), 0644)

	os.WriteFile(filepath.Join(vaultDir, "decisions", "Unrelated.md"),
		[]byte("# Unrelated\nNothing relevant here.\n"), 0644)

	params := map[string]string{"query": "architecture", "context": "2"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("integration search context: %v", err)
		}
	})

	if !strings.Contains(out, "architecture") {
		t.Errorf("match line missing in output: %q", out)
	}
	if !strings.Contains(out, "decisions/ADR-001.md:") {
		t.Errorf("output should use relpath:line format, got: %q", out)
	}
	if strings.Contains(out, "Unrelated.md") {
		t.Errorf("unrelated file should not appear in context output, got: %q", out)
	}
}

// Integration test 8: verify --json output includes context array structure
func TestSearchContextWithJSONFormat(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line one\nline two\narchitecture here\nline four\nline five\n"
	os.WriteFile(filepath.Join(vaultDir, "JSON.md"), []byte(content), 0644)

	params := map[string]string{"query": "architecture", "context": "1"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, "json"); err != nil {
			t.Fatalf("search context json: %v", err)
		}
	})

	if !strings.Contains(out, `"file"`) {
		t.Errorf("JSON output missing 'file' field, got: %q", out)
	}
	if !strings.Contains(out, `"line"`) {
		t.Errorf("JSON output missing 'line' field, got: %q", out)
	}
	if !strings.Contains(out, `"match"`) {
		t.Errorf("JSON output missing 'match' field, got: %q", out)
	}
	if !strings.Contains(out, `"context"`) {
		t.Errorf("JSON output missing 'context' field, got: %q", out)
	}

	type contextResult struct {
		File    string   `json:"file"`
		Line    int      `json:"line"`
		Match   string   `json:"match"`
		Context []string `json:"context"`
	}
	var results []contextResult
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %q", err, out)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one JSON result")
	}
	r := results[0]
	if r.File != "JSON.md" {
		t.Errorf("file = %q, want JSON.md", r.File)
	}
	if r.Line != 3 {
		t.Errorf("line = %d, want 3", r.Line)
	}
	if r.Match != "architecture here" {
		t.Errorf("match = %q, want 'architecture here'", r.Match)
	}
	if len(r.Context) != 3 {
		t.Errorf("context length = %d, want 3, context: %v", len(r.Context), r.Context)
	}
}

// Integration test 9: verify --csv output has file,line,content columns
func TestSearchContextWithCSVFormat(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line one\narchitecture here\nline three\n"
	os.WriteFile(filepath.Join(vaultDir, "CSV.md"), []byte(content), 0644)

	params := map[string]string{"query": "architecture", "context": "1"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, "csv"); err != nil {
			t.Fatalf("search context csv: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 CSV lines (header + data), got %d: %q", len(lines), out)
	}

	header := lines[0]
	if header != "file,line,content" {
		t.Errorf("CSV header = %q, want 'file,line,content'", header)
	}

	foundMatch := false
	for _, line := range lines[1:] {
		if strings.Contains(line, "architecture here") {
			foundMatch = true
		}
	}
	if !foundMatch {
		t.Errorf("CSV output missing match line, got: %q", out)
	}
}

// Integration test 10: context works alongside [key:value] property filters
func TestSearchContextWithPropertyFilter(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(filepath.Join(vaultDir, "Active.md"),
		[]byte("---\nstatus: active\n---\n\nline one\nthe architecture here\nline three\n"), 0644)

	os.WriteFile(filepath.Join(vaultDir, "Archived.md"),
		[]byte("---\nstatus: archived\n---\n\nthe architecture is also here\n"), 0644)

	params := map[string]string{"query": "architecture [status:active]", "context": "1"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("search context with filter: %v", err)
		}
	})

	if !strings.Contains(out, "Active.md:") {
		t.Errorf("expected Active.md in output, got: %q", out)
	}
	if strings.Contains(out, "Archived.md") {
		t.Errorf("Archived.md should be filtered out, got: %q", out)
	}
	if !strings.Contains(out, "the architecture here") {
		t.Errorf("match line missing, got: %q", out)
	}
}

// Unit test 11: context with title-only match outputs title info
func TestSearchContextTitleMatch(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line one\nline two\nline three\n"
	os.WriteFile(filepath.Join(vaultDir, "Architecture Overview.md"), []byte(content), 0644)

	params := map[string]string{"query": "architecture", "context": "1"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("search context title match: %v", err)
		}
	})

	if !strings.Contains(out, "Architecture Overview.md") {
		t.Errorf("title-matched file should appear, got: %q", out)
	}
}

// Integration test 12: YAML format with context
func TestSearchContextWithYAMLFormat(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line one\narchitecture here\nline three\n"
	os.WriteFile(filepath.Join(vaultDir, "YAML.md"), []byte(content), 0644)

	params := map[string]string{"query": "architecture", "context": "1"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, "yaml"); err != nil {
			t.Fatalf("search context yaml: %v", err)
		}
	})

	if !strings.Contains(out, "file:") {
		t.Errorf("YAML output missing 'file:' field, got: %q", out)
	}
	if !strings.Contains(out, "line:") {
		t.Errorf("YAML output missing 'line:' field, got: %q", out)
	}
	if !strings.Contains(out, "match:") {
		t.Errorf("YAML output missing 'match:' field, got: %q", out)
	}
	if !strings.Contains(out, "architecture here") {
		t.Errorf("YAML output missing match text, got: %q", out)
	}
}

// === Regex search tests (VLT-aca) ===

// Unit test 1: regex search finds matches with a pattern
func TestSearchRegexBasic(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(filepath.Join(vaultDir, "Note.md"),
		[]byte("The architecture is modular.\nDesign uses microservices.\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "Other.md"),
		[]byte("Nothing relevant here.\n"), 0644)

	params := map[string]string{"regex": `arch\w+ure`}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("regex basic search: %v", err)
		}
	})

	if !strings.Contains(out, "Note") {
		t.Errorf("expected Note in output, got: %q", out)
	}
	if strings.Contains(out, "Other") {
		t.Errorf("Other should not appear, got: %q", out)
	}
}

// Unit test 2: invalid regex returns clear error with compilation message
func TestSearchRegexInvalid(t *testing.T) {
	vaultDir := t.TempDir()
	os.WriteFile(filepath.Join(vaultDir, "Note.md"), []byte("content"), 0644)

	params := map[string]string{"regex": `[invalid`}
	err := cmdSearch(vaultDir, params, "")

	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
	if !strings.Contains(err.Error(), "invalid regex") {
		t.Errorf("error should mention 'invalid regex', got: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "[invalid") {
		t.Errorf("error should include the bad pattern, got: %q", err.Error())
	}
}

// Unit test 3: regex matching is case-insensitive by default
func TestSearchRegexCaseInsensitive(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(filepath.Join(vaultDir, "Upper.md"),
		[]byte("The ARCHITECTURE is here.\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "Lower.md"),
		[]byte("The architecture is here.\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "Mixed.md"),
		[]byte("The Architecture Is Here.\n"), 0644)

	params := map[string]string{"regex": `architecture`}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("regex case insensitive: %v", err)
		}
	})

	if !strings.Contains(out, "Upper") {
		t.Errorf("should match uppercase, got: %q", out)
	}
	if !strings.Contains(out, "Lower") {
		t.Errorf("should match lowercase, got: %q", out)
	}
	if !strings.Contains(out, "Mixed") {
		t.Errorf("should match mixed case, got: %q", out)
	}
}

// Unit test 4: regex + [key:value] property filter works together
func TestSearchRegexWithPropertyFilter(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(filepath.Join(vaultDir, "Active.md"),
		[]byte("---\nstatus: active\n---\n\nThe architecture is modular.\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "Archived.md"),
		[]byte("---\nstatus: archived\n---\n\nThe architecture is legacy.\n"), 0644)

	// regex with property filter embedded in query -- but regex uses a separate param
	// Property filters come from the query= param, so we pass query with filter only
	params := map[string]string{"regex": `arch\w+ure`, "query": "[status:active]"}
	out := captureStdout(func() {
		// When both regex and query are provided, regex takes precedence for text matching
		// but property filters from query should still apply
		stderr := captureStderr(func() {
			if err := cmdSearch(vaultDir, params, ""); err != nil {
				t.Fatalf("regex with property filter: %v", err)
			}
		})
		_ = stderr // warning expected
	})

	if !strings.Contains(out, "Active") {
		t.Errorf("expected Active in output, got: %q", out)
	}
	if strings.Contains(out, "Archived") {
		t.Errorf("Archived should be filtered out, got: %q", out)
	}
}

// Unit test 5: both query= and regex= provided: regex takes precedence with warning
func TestSearchRegexAndQueryPrecedence(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(filepath.Join(vaultDir, "Note.md"),
		[]byte("The architecture is modular.\nDesign pattern here.\n"), 0644)

	params := map[string]string{"query": "design", "regex": `arch\w+ure`}
	var stderr string
	out := captureStdout(func() {
		stderr = captureStderr(func() {
			if err := cmdSearch(vaultDir, params, ""); err != nil {
				t.Fatalf("regex and query precedence: %v", err)
			}
		})
	})

	// Regex should match "architecture", not query "design"
	if !strings.Contains(out, "Note") {
		t.Errorf("expected Note in output, got: %q", out)
	}
	// Should have a warning on stderr about regex taking precedence
	if !strings.Contains(stderr, "regex") {
		t.Errorf("expected warning about regex on stderr, got: %q", stderr)
	}
}

// Unit test 6: regex matches against note title too
func TestSearchRegexMatchesTitle(t *testing.T) {
	vaultDir := t.TempDir()

	// Title matches, content does not
	os.WriteFile(filepath.Join(vaultDir, "Architecture Overview.md"),
		[]byte("Nothing matching in content.\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "Unrelated.md"),
		[]byte("Also nothing matching.\n"), 0644)

	params := map[string]string{"regex": `arch\w+ure`}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("regex title match: %v", err)
		}
	})

	if !strings.Contains(out, "Architecture Overview") {
		t.Errorf("should match title, got: %q", out)
	}
	if strings.Contains(out, "Unrelated") {
		t.Errorf("unrelated should not appear, got: %q", out)
	}
}

// Unit test 7: regex with no matches returns empty/silent
func TestSearchRegexNoMatch(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(filepath.Join(vaultDir, "Note.md"),
		[]byte("Nothing interesting here.\n"), 0644)

	params := map[string]string{"regex": `zzz\d{4}qqq`}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("regex no match: %v", err)
		}
	})

	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty output for no matches, got: %q", out)
	}
}

// Integration test 8: create notes in t.TempDir(), search with regex, verify correct matches
func TestSearchRegexIntegration(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "decisions"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, "patterns"), 0755)

	os.WriteFile(filepath.Join(vaultDir, "decisions", "ADR-001.md"),
		[]byte("---\ntype: decision\nstatus: active\n---\n\n# Architecture Decision\n\nWe chose microservices architecture.\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "decisions", "ADR-002.md"),
		[]byte("---\ntype: decision\nstatus: active\n---\n\n# Database Choice\n\nWe selected PostgreSQL for storage.\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "patterns", "Observer.md"),
		[]byte("# Observer Pattern\n\nThe observer pattern is used for event-driven architecture.\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "patterns", "Singleton.md"),
		[]byte("# Singleton Pattern\n\nSingleton ensures one instance.\n"), 0644)

	// Search for architecture using regex
	params := map[string]string{"regex": `architect\w+`}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("regex integration: %v", err)
		}
	})

	if !strings.Contains(out, "ADR-001") {
		t.Errorf("ADR-001 should match, got: %q", out)
	}
	if !strings.Contains(out, "Observer") {
		t.Errorf("Observer should match, got: %q", out)
	}
	if strings.Contains(out, "ADR-002") {
		t.Errorf("ADR-002 should not match, got: %q", out)
	}
	if strings.Contains(out, "Singleton") {
		t.Errorf("Singleton should not match, got: %q", out)
	}
}

// Integration test 9: complex regex pattern (date pattern)
func TestSearchRegexComplexPattern(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(filepath.Join(vaultDir, "WithDate.md"),
		[]byte("Created on 2025-01-15 for the project.\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "WithBadDate.md"),
		[]byte("Created on 25-1-5 for the project.\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "NoDate.md"),
		[]byte("No date information here.\n"), 0644)

	params := map[string]string{"regex": `\d{4}-\d{2}-\d{2}`}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("regex complex pattern: %v", err)
		}
	})

	if !strings.Contains(out, "WithDate") {
		t.Errorf("WithDate should match date pattern, got: %q", out)
	}
	if strings.Contains(out, "WithBadDate") {
		t.Errorf("WithBadDate should not match strict date pattern, got: %q", out)
	}
	if strings.Contains(out, "NoDate") {
		t.Errorf("NoDate should not match, got: %q", out)
	}
}

// Integration test 10: regex search combined with context= parameter
func TestSearchRegexWithContext(t *testing.T) {
	vaultDir := t.TempDir()

	content := "line one\nline two\nthe architecture is key\nline four\nline five\n"
	os.WriteFile(filepath.Join(vaultDir, "Note.md"), []byte(content), 0644)

	params := map[string]string{"regex": `arch\w+ure`, "context": "1"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("regex with context: %v", err)
		}
	})

	// Should contain file:line format
	if !strings.Contains(out, "Note.md:") {
		t.Errorf("output should contain file:line format, got: %q", out)
	}
	// Should contain the match line
	if !strings.Contains(out, "the architecture is key") {
		t.Errorf("output should contain match line, got: %q", out)
	}
	// Should contain context lines
	if !strings.Contains(out, "line two") {
		t.Errorf("output should contain line before match, got: %q", out)
	}
	if !strings.Contains(out, "line four") {
		t.Errorf("output should contain line after match, got: %q", out)
	}
	// Should NOT contain lines outside context
	if strings.Contains(out, "line one") {
		t.Errorf("output should not contain lines outside context range, got: %q", out)
	}
	if strings.Contains(out, "line five") {
		t.Errorf("output should not contain lines outside context range, got: %q", out)
	}
}

// Unit test 11: regex with neither query nor regex errors
func TestSearchRegexRequiresParam(t *testing.T) {
	vaultDir := t.TempDir()
	os.WriteFile(filepath.Join(vaultDir, "Note.md"), []byte("content"), 0644)

	params := map[string]string{}
	err := cmdSearch(vaultDir, params, "")

	if err == nil {
		t.Fatal("expected error when neither query nor regex is provided")
	}
}

// Unit test 12: existing search behavior unchanged when regex= not provided
func TestSearchRegexBackwardCompatible(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(filepath.Join(vaultDir, "TestNote.md"),
		[]byte("# Test\nSome architecture content.\n"), 0644)

	params := map[string]string{"query": "architecture"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("backward compat: %v", err)
		}
	})

	if !strings.Contains(out, "TestNote") {
		t.Errorf("expected title in output, got: %q", out)
	}
}

// Unit test 13: regex with path filter
func TestSearchRegexWithPathFilter(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "decisions"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, "patterns"), 0755)

	os.WriteFile(filepath.Join(vaultDir, "decisions", "ADR.md"),
		[]byte("Architecture here.\n"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "patterns", "Pattern.md"),
		[]byte("Architecture there.\n"), 0644)

	params := map[string]string{"regex": `architecture`, "path": "decisions"}
	out := captureStdout(func() {
		if err := cmdSearch(vaultDir, params, ""); err != nil {
			t.Fatalf("regex with path filter: %v", err)
		}
	})

	if !strings.Contains(out, "ADR") {
		t.Errorf("ADR should match, got: %q", out)
	}
	if strings.Contains(out, "Pattern") {
		t.Errorf("Pattern should be filtered by path, got: %q", out)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
