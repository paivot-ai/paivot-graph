package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout captures stdout output from a function call.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// captureStderr captures stderr output from a function call.
func captureStderr(fn func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestOutputFormat(t *testing.T) {
	tests := []struct {
		flags map[string]bool
		want  string
	}{
		{map[string]bool{}, ""},
		{map[string]bool{"--json": true}, "json"},
		{map[string]bool{"--csv": true}, "csv"},
		{map[string]bool{"--yaml": true}, "yaml"},
		{map[string]bool{"--json": true, "--csv": true}, "json"}, // json wins
	}

	for _, tt := range tests {
		got := outputFormat(tt.flags)
		if got != tt.want {
			t.Errorf("outputFormat(%v) = %q, want %q", tt.flags, got, tt.want)
		}
	}
}

func TestFormatList_JSON(t *testing.T) {
	got := captureStdout(func() {
		formatList([]string{"a.md", "b.md"}, "json")
	})
	want := `["a.md","b.md"]`
	if strings.TrimSpace(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatList_CSV(t *testing.T) {
	got := captureStdout(func() {
		formatList([]string{"a.md", "b.md"}, "csv")
	})
	if !strings.Contains(got, "a.md") || !strings.Contains(got, "b.md") {
		t.Errorf("csv output missing items: %q", got)
	}
}

func TestFormatList_YAML(t *testing.T) {
	got := captureStdout(func() {
		formatList([]string{"a.md", "b.md"}, "yaml")
	})
	if !strings.Contains(got, "- a.md") || !strings.Contains(got, "- b.md") {
		t.Errorf("yaml output missing items: %q", got)
	}
}

func TestFormatList_PlainText(t *testing.T) {
	got := captureStdout(func() {
		formatList([]string{"a.md", "b.md"}, "")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 || lines[0] != "a.md" || lines[1] != "b.md" {
		t.Errorf("plain text output: %q", got)
	}
}

func TestFormatTable_JSON(t *testing.T) {
	rows := []map[string]string{
		{"name": "Alice", "role": "dev"},
		{"name": "Bob", "role": "pm"},
	}
	got := captureStdout(func() {
		formatTable(rows, []string{"name", "role"}, "json")
	})
	if !strings.Contains(got, `"name":"Alice"`) {
		t.Errorf("json table missing data: %q", got)
	}
}

func TestFormatTable_CSV(t *testing.T) {
	rows := []map[string]string{
		{"name": "Alice", "role": "dev"},
	}
	got := captureStdout(func() {
		formatTable(rows, []string{"name", "role"}, "csv")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header + data), got %d: %q", len(lines), got)
	}
	if lines[0] != "name,role" {
		t.Errorf("header = %q, want %q", lines[0], "name,role")
	}
	if lines[1] != "Alice,dev" {
		t.Errorf("data = %q, want %q", lines[1], "Alice,dev")
	}
}

func TestFormatTable_YAML(t *testing.T) {
	rows := []map[string]string{
		{"name": "Alice", "role": "dev"},
	}
	got := captureStdout(func() {
		formatTable(rows, []string{"name", "role"}, "yaml")
	})
	if !strings.Contains(got, "name: Alice") || !strings.Contains(got, "role: dev") {
		t.Errorf("yaml table output: %q", got)
	}
}

func TestFormatSearchResults_JSON(t *testing.T) {
	results := []searchResult{
		{title: "Note A", relPath: "folder/Note A.md"},
	}
	got := captureStdout(func() {
		formatSearchResults(results, "json")
	})
	if !strings.Contains(got, `"title":"Note A"`) || !strings.Contains(got, `"path":"folder/Note A.md"`) {
		t.Errorf("json search results: %q", got)
	}
}

func TestFormatLinks_JSON(t *testing.T) {
	links := []linkInfo{
		{Target: "Note", Path: "Note.md", Broken: false},
		{Target: "Missing", Path: "", Broken: true},
	}
	got := captureStdout(func() {
		formatLinks(links, "json")
	})
	if !strings.Contains(got, `"broken":true`) || !strings.Contains(got, `"broken":false`) {
		t.Errorf("json links: %q", got)
	}
}

func TestFormatTagCounts_JSON(t *testing.T) {
	tags := []string{"project", "review"}
	counts := map[string]int{"project": 5, "review": 2}
	got := captureStdout(func() {
		formatTagCounts(tags, counts, "json")
	})
	if !strings.Contains(got, `"tag":"project"`) || !strings.Contains(got, `"count":5`) {
		t.Errorf("json tag counts: %q", got)
	}
}

func TestFormatVaults_JSON(t *testing.T) {
	names := []string{"Claude"}
	vaults := map[string]string{"Claude": "/path/to/Claude"}
	got := captureStdout(func() {
		formatVaults(names, vaults, "json")
	})
	if !strings.Contains(got, `"name":"Claude"`) || !strings.Contains(got, `"path":"/path/to/Claude"`) {
		t.Errorf("json vaults: %q", got)
	}
}

func TestYamlEscapeValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"has: colon", `"has: colon"`},
		{"has [bracket]", `"has [bracket]"`},
		{"", `""`},
	}
	for _, tt := range tests {
		got := yamlEscapeValue(tt.input)
		if got != tt.want {
			t.Errorf("yamlEscapeValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- TSV and Tree format tests ---

func TestOutputFormatTSV(t *testing.T) {
	flags := map[string]bool{"--tsv": true}
	got := outputFormat(flags)
	if got != "tsv" {
		t.Errorf("outputFormat(--tsv) = %q, want %q", got, "tsv")
	}
}

func TestOutputFormatTree(t *testing.T) {
	flags := map[string]bool{"--tree": true}
	got := outputFormat(flags)
	if got != "tree" {
		t.Errorf("outputFormat(--tree) = %q, want %q", got, "tree")
	}
}

func TestFormatListTSV(t *testing.T) {
	got := captureStdout(func() {
		formatList([]string{"folder/Note A.md", "Note B.md"}, "tsv")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 data), got %d: %q", len(lines), got)
	}
	if lines[0] != "file" {
		t.Errorf("header = %q, want %q", lines[0], "file")
	}
	if lines[1] != "folder/Note A.md" {
		t.Errorf("line 1 = %q, want %q", lines[1], "folder/Note A.md")
	}
	if lines[2] != "Note B.md" {
		t.Errorf("line 2 = %q, want %q", lines[2], "Note B.md")
	}
}

func TestFormatTableTSV(t *testing.T) {
	rows := []map[string]string{
		{"name": "Alice", "role": "dev"},
		{"name": "Bob", "role": "pm"},
	}
	got := captureStdout(func() {
		formatTable(rows, []string{"name", "role"}, "tsv")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 data), got %d: %q", len(lines), got)
	}
	if lines[0] != "name\trole" {
		t.Errorf("header = %q, want %q", lines[0], "name\trole")
	}
	if lines[1] != "Alice\tdev" {
		t.Errorf("row 1 = %q, want %q", lines[1], "Alice\tdev")
	}
	if lines[2] != "Bob\tpm" {
		t.Errorf("row 2 = %q, want %q", lines[2], "Bob\tpm")
	}
}

func TestFormatSearchResultsTSV(t *testing.T) {
	results := []searchResult{
		{title: "Note A", relPath: "folder/Note A.md"},
		{title: "Note B", relPath: "Note B.md"},
	}
	got := captureStdout(func() {
		formatSearchResults(results, "tsv")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 data), got %d: %q", len(lines), got)
	}
	if lines[0] != "title\tpath" {
		t.Errorf("header = %q, want %q", lines[0], "title\tpath")
	}
	if lines[1] != "Note A\tfolder/Note A.md" {
		t.Errorf("row 1 = %q, want %q", lines[1], "Note A\tfolder/Note A.md")
	}
}

func TestFormatLinksTSV(t *testing.T) {
	links := []linkInfo{
		{Target: "Note", Path: "Note.md", Broken: false},
		{Target: "Missing", Path: "", Broken: true},
	}
	got := captureStdout(func() {
		formatLinks(links, "tsv")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 data), got %d: %q", len(lines), got)
	}
	if lines[0] != "target\tpath\tbroken" {
		t.Errorf("header = %q, want %q", lines[0], "target\tpath\tbroken")
	}
	if lines[1] != "Note\tNote.md\tfalse" {
		t.Errorf("row 1 = %q, want %q", lines[1], "Note\tNote.md\tfalse")
	}
	if lines[2] != "Missing\t\ttrue" {
		t.Errorf("row 2 = %q, want %q", lines[2], "Missing\t\ttrue")
	}
}

func TestFormatUnresolvedTSV(t *testing.T) {
	results := []unresolvedResult{
		{Target: "Missing Note", Source: "folder/Ref.md"},
	}
	got := captureStdout(func() {
		formatUnresolved(results, "tsv")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header + 1 data), got %d: %q", len(lines), got)
	}
	if lines[0] != "target\tsource" {
		t.Errorf("header = %q, want %q", lines[0], "target\tsource")
	}
	if lines[1] != "Missing Note\tfolder/Ref.md" {
		t.Errorf("row 1 = %q, want %q", lines[1], "Missing Note\tfolder/Ref.md")
	}
}

func TestFormatTagCountsTSV(t *testing.T) {
	tags := []string{"project", "review"}
	counts := map[string]int{"project": 5, "review": 2}
	got := captureStdout(func() {
		formatTagCounts(tags, counts, "tsv")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 data), got %d: %q", len(lines), got)
	}
	if lines[0] != "tag\tcount" {
		t.Errorf("header = %q, want %q", lines[0], "tag\tcount")
	}
	if lines[1] != "project\t5" {
		t.Errorf("row 1 = %q, want %q", lines[1], "project\t5")
	}
	if lines[2] != "review\t2" {
		t.Errorf("row 2 = %q, want %q", lines[2], "review\t2")
	}
}

func TestFormatVaultsTSV(t *testing.T) {
	names := []string{"Claude", "Work"}
	vaults := map[string]string{"Claude": "/path/to/Claude", "Work": "/path/to/Work"}
	got := captureStdout(func() {
		formatVaults(names, vaults, "tsv")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 data), got %d: %q", len(lines), got)
	}
	if lines[0] != "name\tpath" {
		t.Errorf("header = %q, want %q", lines[0], "name\tpath")
	}
	if lines[1] != "Claude\t/path/to/Claude" {
		t.Errorf("row 1 = %q, want %q", lines[1], "Claude\t/path/to/Claude")
	}
}

func TestFormatPropertiesTSV(t *testing.T) {
	text := "---\nstatus: active\ntype: decision\n---"
	got := captureStdout(func() {
		formatProperties(text, "tsv")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 data), got %d: %q", len(lines), got)
	}
	if lines[0] != "key\tvalue" {
		t.Errorf("header = %q, want %q", lines[0], "key\tvalue")
	}
	// Keys are sorted alphabetically
	if lines[1] != "status\tactive" {
		t.Errorf("row 1 = %q, want %q", lines[1], "status\tactive")
	}
	if lines[2] != "type\tdecision" {
		t.Errorf("row 2 = %q, want %q", lines[2], "type\tdecision")
	}
}

func TestFormatSearchWithContextTSV(t *testing.T) {
	matches := []contextMatch{
		{File: "note.md", Line: 3, Match: "hello world", Context: []string{"line 2", "hello world", "line 4"}},
	}
	got := captureStdout(func() {
		formatSearchWithContext(matches, "tsv")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (header + data), got %d: %q", len(lines), got)
	}
	if lines[0] != "file\tline\tcontent" {
		t.Errorf("header = %q, want %q", lines[0], "file\tline\tcontent")
	}
}

func TestFormatListTree(t *testing.T) {
	items := []string{
		"folder/Note A.md",
		"folder/Note B.md",
		"other/Note C.md",
		"Root Note.md",
	}
	got := captureStdout(func() {
		formatList(items, "tree")
	})

	// Should contain tree characters
	if !strings.Contains(got, "\u251c\u2500\u2500") { // contains branch character
		t.Errorf("tree output missing branch character: %q", got)
	}
	if !strings.Contains(got, "\u2514\u2500\u2500") { // contains last branch character
		t.Errorf("tree output missing last branch character: %q", got)
	}
	// Should contain the directory names and file names
	if !strings.Contains(got, "folder/") {
		t.Errorf("tree output missing 'folder/': %q", got)
	}
	if !strings.Contains(got, "Note A.md") {
		t.Errorf("tree output missing 'Note A.md': %q", got)
	}
	if !strings.Contains(got, "Root Note.md") {
		t.Errorf("tree output missing 'Root Note.md': %q", got)
	}
}

func TestFormatListTreeSingleFile(t *testing.T) {
	items := []string{"Root Note.md"}
	got := captureStdout(func() {
		formatList(items, "tree")
	})
	trimmed := strings.TrimSpace(got)
	// A single file at root level should render with the last-item connector
	if !strings.Contains(trimmed, "Root Note.md") {
		t.Errorf("single file tree output missing file name: %q", trimmed)
	}
}

func TestFormatListTreeNestedFolders(t *testing.T) {
	items := []string{
		"a/b/c/deep.md",
		"a/b/shallow.md",
		"a/top.md",
	}
	got := captureStdout(func() {
		formatList(items, "tree")
	})
	// Should show nested indentation
	if !strings.Contains(got, "a/") {
		t.Errorf("tree output missing 'a/': %q", got)
	}
	if !strings.Contains(got, "b/") {
		t.Errorf("tree output missing 'b/': %q", got)
	}
	if !strings.Contains(got, "c/") {
		t.Errorf("tree output missing 'c/': %q", got)
	}
	if !strings.Contains(got, "deep.md") {
		t.Errorf("tree output missing 'deep.md': %q", got)
	}
	if !strings.Contains(got, "shallow.md") {
		t.Errorf("tree output missing 'shallow.md': %q", got)
	}
	if !strings.Contains(got, "top.md") {
		t.Errorf("tree output missing 'top.md': %q", got)
	}
}

func TestFormatListTreeEmpty(t *testing.T) {
	got := captureStdout(func() {
		formatList([]string{}, "tree")
	})
	if got != "" {
		t.Errorf("empty tree should produce no output, got %q", got)
	}
}

// --- Integration tests: real files, no mocks ---

func TestFilesTSVIntegration(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "folder"), 0755)
	os.WriteFile(filepath.Join(vaultDir, "Root.md"), []byte("# Root"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "folder", "Inner.md"), []byte("# Inner"), 0644)

	got := captureStdout(func() {
		err := cmdFiles(vaultDir, map[string]string{}, false, "tsv")
		if err != nil {
			t.Fatalf("cmdFiles error: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 files), got %d: %q", len(lines), got)
	}
	if lines[0] != "file" {
		t.Errorf("header = %q, want %q", lines[0], "file")
	}
	// Files are sorted: Root.md, folder/Inner.md
	if lines[1] != "Root.md" {
		t.Errorf("line 1 = %q, want %q", lines[1], "Root.md")
	}
	if lines[2] != "folder/Inner.md" {
		t.Errorf("line 2 = %q, want %q", lines[2], "folder/Inner.md")
	}
}

func TestFilesTreeIntegration(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "folder"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, "other"), 0755)
	os.WriteFile(filepath.Join(vaultDir, "Root.md"), []byte("# Root"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "folder", "Note A.md"), []byte("# A"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "folder", "Note B.md"), []byte("# B"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "other", "Note C.md"), []byte("# C"), 0644)

	got := captureStdout(func() {
		err := cmdFiles(vaultDir, map[string]string{}, false, "tree")
		if err != nil {
			t.Fatalf("cmdFiles error: %v", err)
		}
	})

	// Verify tree structure contains expected elements
	if !strings.Contains(got, "folder/") {
		t.Errorf("tree output missing 'folder/': %q", got)
	}
	if !strings.Contains(got, "other/") {
		t.Errorf("tree output missing 'other/': %q", got)
	}
	if !strings.Contains(got, "Note A.md") {
		t.Errorf("tree output missing 'Note A.md': %q", got)
	}
	if !strings.Contains(got, "Root.md") {
		t.Errorf("tree output missing 'Root.md': %q", got)
	}
	// Should contain Unicode box-drawing characters
	if !strings.Contains(got, "\u2514") && !strings.Contains(got, "\u251c") {
		t.Errorf("tree output missing Unicode box-drawing characters: %q", got)
	}
}

func TestSearchTSVIntegration(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "folder"), 0755)
	os.WriteFile(filepath.Join(vaultDir, "folder", "Architecture.md"), []byte("# Architecture\nSome design notes."), 0644)
	os.WriteFile(filepath.Join(vaultDir, "Other.md"), []byte("# Other\nNothing here."), 0644)

	got := captureStdout(func() {
		err := cmdSearch(vaultDir, map[string]string{"query": "Architecture"}, "tsv")
		if err != nil {
			t.Fatalf("cmdSearch error: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (header + result), got %d: %q", len(lines), got)
	}
	if lines[0] != "title\tpath" {
		t.Errorf("header = %q, want %q", lines[0], "title\tpath")
	}
	if !strings.Contains(lines[1], "Architecture") {
		t.Errorf("result should contain 'Architecture': %q", lines[1])
	}
	if !strings.Contains(lines[1], "\t") {
		t.Errorf("result should be tab-separated: %q", lines[1])
	}
}

func TestOrphansTSVIntegration(t *testing.T) {
	vaultDir := t.TempDir()
	os.WriteFile(filepath.Join(vaultDir, "Orphan.md"), []byte("# Orphan\nNo links to me."), 0644)
	os.WriteFile(filepath.Join(vaultDir, "Linked.md"), []byte("# Linked\nSee [[Linked]] here."), 0644)

	got := captureStdout(func() {
		err := cmdOrphans(vaultDir, "tsv")
		if err != nil {
			t.Fatalf("cmdOrphans error: %v", err)
		}
	})

	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (header + orphan), got %d: %q", len(lines), got)
	}
	if lines[0] != "file" {
		t.Errorf("header = %q, want %q", lines[0], "file")
	}
	foundOrphan := false
	for _, line := range lines[1:] {
		if strings.Contains(line, "Orphan.md") {
			foundOrphan = true
		}
	}
	if !foundOrphan {
		t.Errorf("orphans TSV output should contain Orphan.md: %q", got)
	}
}

func TestOrphansTreeIntegration(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "folder"), 0755)
	os.WriteFile(filepath.Join(vaultDir, "folder", "Orphan A.md"), []byte("# Orphan A"), 0644)
	os.WriteFile(filepath.Join(vaultDir, "Orphan B.md"), []byte("# Orphan B"), 0644)
	// This note links to itself, so it is still an orphan (self-links don't count as incoming)
	os.WriteFile(filepath.Join(vaultDir, "folder", "Linked.md"), []byte("# Linked\n[[Linked]]"), 0644)

	got := captureStdout(func() {
		err := cmdOrphans(vaultDir, "tree")
		if err != nil {
			t.Fatalf("cmdOrphans error: %v", err)
		}
	})

	if !strings.Contains(got, "Orphan A.md") {
		t.Errorf("tree orphans output missing 'Orphan A.md': %q", got)
	}
	if !strings.Contains(got, "Orphan B.md") {
		t.Errorf("tree orphans output missing 'Orphan B.md': %q", got)
	}
	// Verify it has tree structure (box-drawing characters)
	if !strings.Contains(got, "\u2514") && !strings.Contains(got, "\u251c") {
		t.Errorf("tree output missing Unicode box-drawing characters: %q", got)
	}
}
