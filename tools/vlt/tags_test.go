package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseInlineTags(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "simple tags",
			text: "This is #project and #important work.",
			want: []string{"project", "important"},
		},
		{
			name: "hierarchical tag",
			text: "Working on #project/backend today.",
			want: []string{"project/backend"},
		},
		{
			name: "tag at start of line",
			text: "#meeting notes from today",
			want: []string{"meeting"},
		},
		{
			name: "skip pure numeric",
			text: "Issue #42 and #2024 are not tags but #v2 is.",
			want: []string{"v2"},
		},
		{
			name: "tag with underscore and hyphen",
			text: "See #to_do and #high-priority items.",
			want: []string{"to_do", "high-priority"},
		},
		{
			name: "heading is not a tag",
			text: "## Heading\n\nContent with #real-tag.",
			want: []string{"real-tag"},
		},
		{
			name: "no tags",
			text: "Plain text with no hash marks.",
			want: nil,
		},
		{
			name: "tag in parentheses",
			text: "Some text (#status) here.",
			want: []string{"status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInlineTags(tt.text)

			if tt.want == nil {
				if len(got) != 0 {
					t.Errorf("got %v, want empty", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %d tags %v, want %d %v", len(got), got, len(tt.want), tt.want)
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("tag[%d] = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}

func TestAllNoteTags(t *testing.T) {
	text := "---\ntags: [project, important]\n---\n\n# My Note\n\nSome #inline-tag and #project again.\n"

	got := allNoteTags(text)

	// Should have: project, important, inline-tag (project deduplicated)
	want := map[string]bool{"project": true, "important": true, "inline-tag": true}

	if len(got) != len(want) {
		t.Fatalf("got %d tags %v, want %d", len(got), got, len(want))
	}
	for _, tag := range got {
		if !want[tag] {
			t.Errorf("unexpected tag %q", tag)
		}
	}
}

func TestAllNoteTags_CaseInsensitive(t *testing.T) {
	text := "---\ntags: [Project]\n---\n\n#project again\n"

	got := allNoteTags(text)

	if len(got) != 1 {
		t.Fatalf("got %d tags %v, want 1 (case-insensitive dedup)", len(got), got)
	}
	if got[0] != "project" {
		t.Errorf("tag = %q, want %q", got[0], "project")
	}
}

func TestAllNoteTags_NoFrontmatter(t *testing.T) {
	text := "# My Note\n\nJust #inline tags here.\n"

	got := allNoteTags(text)

	if len(got) != 1 || got[0] != "inline" {
		t.Errorf("got %v, want [inline]", got)
	}
}

func TestCmdTags(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "note1.md"),
		[]byte("---\ntags: [project, important]\n---\n\n# Note 1\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "note2.md"),
		[]byte("# Note 2\n\nSome #project and #review content.\n"),
		0644,
	)
	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)
	os.WriteFile(
		filepath.Join(vaultDir, ".obsidian", "hidden.md"),
		[]byte("#hidden-tag should be skipped\n"),
		0644,
	)

	// Just verify no error
	params := map[string]string{}
	if err := cmdTags(vaultDir, params, true); err != nil {
		t.Fatalf("tags: %v", err)
	}
}

func TestCmdTag(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "methodology"), 0755)

	os.WriteFile(
		filepath.Join(vaultDir, "methodology", "Agent.md"),
		[]byte("---\ntags: [project/backend]\n---\n\n# Agent\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "Other.md"),
		[]byte("# Other\n\nNo relevant tags.\n"),
		0644,
	)

	// Exact match
	params := map[string]string{"tag": "project/backend"}
	if err := cmdTag(vaultDir, params); err != nil {
		t.Fatalf("tag exact: %v", err)
	}

	// Hierarchical match: #project should find #project/backend
	params = map[string]string{"tag": "project"}
	if err := cmdTag(vaultDir, params); err != nil {
		t.Fatalf("tag hierarchical: %v", err)
	}
}

func TestCmdTag_StripHash(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "note.md"),
		[]byte("---\ntags: [meeting]\n---\n\n# Note\n"),
		0644,
	)

	// User passes #meeting with hash prefix -- should still work
	params := map[string]string{"tag": "#meeting"}
	if err := cmdTag(vaultDir, params); err != nil {
		t.Fatalf("tag with hash: %v", err)
	}
}
