package main

import (
	"bytes"
	"io"
	"os"
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
