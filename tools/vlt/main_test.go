package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
			args:       []string{"vault=Claude", "search", "query=paivot"},
			wantCmd:    "search",
			wantParams: map[string]string{"vault": "Claude", "query": "paivot"},
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
	if err := cmdCreate(vaultDir, params, false); err != nil {
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
	if err := cmdCreate(vaultDir, params, true); err != nil {
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
	if err := cmdAppend(vaultDir, params); err != nil {
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
	if err := cmdBacklinks(vaultDir, params); err != nil {
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
	if err := cmdLinks(vaultDir, params); err != nil {
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
	os.WriteFile(filepath.Join(vaultDir, "decisions", "Paivot Architecture.md"),
		[]byte("# Architecture\nSome content."), 0644)

	// Note with matching content but not title
	os.WriteFile(filepath.Join(vaultDir, "decisions", "Other Decision.md"),
		[]byte("# Other\nThis relates to paivot infrastructure."), 0644)

	// Note that should not match
	os.WriteFile(filepath.Join(vaultDir, "decisions", "Unrelated.md"),
		[]byte("# Unrelated\nNothing here."), 0644)

	// Hidden note that should be skipped
	os.WriteFile(filepath.Join(vaultDir, ".obsidian", "paivot-config.md"),
		[]byte("# Config\npaivot settings."), 0644)

	params := map[string]string{"query": "paivot"}
	// cmdSearch writes to stdout; just verify no error
	if err := cmdSearch(vaultDir, params); err != nil {
		t.Fatalf("search: %v", err)
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
	if err := cmdPrepend(vaultDir, params); err != nil {
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
	if err := cmdPrepend(vaultDir, params); err != nil {
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
	if err := cmdProperties(vaultDir, params); err != nil {
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
	if err := cmdOrphans(vaultDir); err != nil {
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
	if err := cmdOrphans(vaultDir); err != nil {
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
	if err := cmdUnresolved(vaultDir); err != nil {
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
	if err := cmdFiles(vaultDir, params, false); err != nil {
		t.Fatalf("files: %v", err)
	}

	// Total count
	if err := cmdFiles(vaultDir, params, true); err != nil {
		t.Fatalf("files total: %v", err)
	}

	// Filter by folder
	params = map[string]string{"folder": "sub"}
	if err := cmdFiles(vaultDir, params, false); err != nil {
		t.Fatalf("files folder: %v", err)
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
