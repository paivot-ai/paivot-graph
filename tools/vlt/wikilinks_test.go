package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestParseWikilinks(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		wants []wikilink
	}{
		{
			name: "simple link",
			text: "See [[Session Operating Mode]] for details.",
			wants: []wikilink{
				{Title: "Session Operating Mode", Raw: "[[Session Operating Mode]]"},
			},
		},
		{
			name: "link with heading",
			text: "See [[Agent Execution Model#Ephemeral Agents]] here.",
			wants: []wikilink{
				{Title: "Agent Execution Model", Heading: "Ephemeral Agents",
					Raw: "[[Agent Execution Model#Ephemeral Agents]]"},
			},
		},
		{
			name: "link with display text",
			text: "The [[Sr PM Agent|PM]] handles this.",
			wants: []wikilink{
				{Title: "Sr PM Agent", Display: "PM", Raw: "[[Sr PM Agent|PM]]"},
			},
		},
		{
			name: "link with heading and display",
			text: "See [[Developer Agent#TDD|testing section]] for more.",
			wants: []wikilink{
				{Title: "Developer Agent", Heading: "TDD", Display: "testing section",
					Raw: "[[Developer Agent#TDD|testing section]]"},
			},
		},
		{
			name: "multiple links on same line",
			text: "Both [[Anchor Agent]] and [[Retro Agent]] are ephemeral.",
			wants: []wikilink{
				{Title: "Anchor Agent", Raw: "[[Anchor Agent]]"},
				{Title: "Retro Agent", Raw: "[[Retro Agent]]"},
			},
		},
		{
			name: "no links",
			text: "Plain text with no links at all.",
			wants: []wikilink{},
		},
		{
			name: "link with special regex chars in title",
			text: "See [[D&F Sequential (With Alignment)]] for details.",
			wants: []wikilink{
				{Title: "D&F Sequential (With Alignment)",
					Raw: "[[D&F Sequential (With Alignment)]]"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseWikilinks(tt.text)

			if len(got) != len(tt.wants) {
				t.Fatalf("got %d links, want %d", len(got), len(tt.wants))
			}

			for i, want := range tt.wants {
				g := got[i]
				if g.Title != want.Title {
					t.Errorf("link[%d].Title = %q, want %q", i, g.Title, want.Title)
				}
				if g.Heading != want.Heading {
					t.Errorf("link[%d].Heading = %q, want %q", i, g.Heading, want.Heading)
				}
				if g.Display != want.Display {
					t.Errorf("link[%d].Display = %q, want %q", i, g.Display, want.Display)
				}
				if g.Raw != want.Raw {
					t.Errorf("link[%d].Raw = %q, want %q", i, g.Raw, want.Raw)
				}
			}
		})
	}
}

func TestReplaceWikilinks(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		oldTitle string
		newTitle string
		want     string
	}{
		{
			name:     "simple replacement",
			text:     "See [[Old Note]] for details.",
			oldTitle: "Old Note",
			newTitle: "New Note",
			want:     "See [[New Note]] for details.",
		},
		{
			name:     "preserves heading",
			text:     "See [[Old Note#Section]] here.",
			oldTitle: "Old Note",
			newTitle: "New Note",
			want:     "See [[New Note#Section]] here.",
		},
		{
			name:     "preserves display text",
			text:     "The [[Old Note|alias]] is useful.",
			oldTitle: "Old Note",
			newTitle: "New Note",
			want:     "The [[New Note|alias]] is useful.",
		},
		{
			name:     "preserves heading and display",
			text:     "See [[Old Note#Section|alias]] here.",
			oldTitle: "Old Note",
			newTitle: "New Note",
			want:     "See [[New Note#Section|alias]] here.",
		},
		{
			name:     "multiple occurrences",
			text:     "Both [[Old Note]] and later [[Old Note#Heading]] reference it.",
			oldTitle: "Old Note",
			newTitle: "New Note",
			want:     "Both [[New Note]] and later [[New Note#Heading]] reference it.",
		},
		{
			name:     "case insensitive matching",
			text:     "See [[old note]] and [[Old Note]] and [[OLD NOTE]].",
			oldTitle: "Old Note",
			newTitle: "New Note",
			want:     "See [[New Note]] and [[New Note]] and [[New Note]].",
		},
		{
			name:     "no match leaves text unchanged",
			text:     "See [[Other Note]] here.",
			oldTitle: "Old Note",
			newTitle: "New Note",
			want:     "See [[Other Note]] here.",
		},
		{
			name:     "title with special regex characters",
			text:     "See [[D&F (Sequential)]] for the decision.",
			oldTitle: "D&F (Sequential)",
			newTitle: "D&F Sequential With Alignment",
			want:     "See [[D&F Sequential With Alignment]] for the decision.",
		},
		{
			name:     "does not match partial titles",
			text:     "See [[Old Note Extended]] here.",
			oldTitle: "Old Note",
			newTitle: "New Note",
			want:     "See [[Old Note Extended]] here.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceWikilinks(tt.text, tt.oldTitle, tt.newTitle)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUpdateVaultLinks(t *testing.T) {
	vaultDir := t.TempDir()

	// Create vault structure
	os.MkdirAll(filepath.Join(vaultDir, "methodology"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, "conventions"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)

	// File that references the note being renamed
	os.WriteFile(
		filepath.Join(vaultDir, "methodology", "Developer Agent.md"),
		[]byte("# Developer\n\nSee [[Old Name]] and [[Old Name#Section]] for context.\n"),
		0644,
	)

	// File with no references (should be untouched)
	os.WriteFile(
		filepath.Join(vaultDir, "conventions", "Unrelated.md"),
		[]byte("# Unrelated\n\nNo links here.\n"),
		0644,
	)

	// File in .obsidian (should be skipped entirely)
	os.WriteFile(
		filepath.Join(vaultDir, ".obsidian", "config.md"),
		[]byte("[[Old Name]] in hidden dir.\n"),
		0644,
	)

	count, err := updateVaultLinks(vaultDir, "Old Name", "New Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 1 {
		t.Errorf("modified %d files, want 1", count)
	}

	// Verify the file was updated
	data, _ := os.ReadFile(filepath.Join(vaultDir, "methodology", "Developer Agent.md"))
	got := string(data)
	want := "# Developer\n\nSee [[New Name]] and [[New Name#Section]] for context.\n"
	if got != want {
		t.Errorf("updated content:\ngot:  %q\nwant: %q", got, want)
	}

	// Verify unrelated file untouched
	data, _ = os.ReadFile(filepath.Join(vaultDir, "conventions", "Unrelated.md"))
	if string(data) != "# Unrelated\n\nNo links here.\n" {
		t.Error("unrelated file was modified")
	}

	// Verify hidden dir untouched
	data, _ = os.ReadFile(filepath.Join(vaultDir, ".obsidian", "config.md"))
	if string(data) != "[[Old Name]] in hidden dir.\n" {
		t.Error("hidden dir file was modified")
	}
}

func TestFindBacklinks(t *testing.T) {
	vaultDir := t.TempDir()

	os.MkdirAll(filepath.Join(vaultDir, "methodology"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, "conventions"), 0755)
	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0755)

	// Notes that reference "Session Operating Mode"
	os.WriteFile(
		filepath.Join(vaultDir, "methodology", "Developer Agent.md"),
		[]byte("Read [[Session Operating Mode]] first.\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "conventions", "Pre-Compact Checklist.md"),
		[]byte("See [[Session Operating Mode#Protocol]] for steps.\n"),
		0644,
	)

	// Note that does NOT reference it
	os.WriteFile(
		filepath.Join(vaultDir, "methodology", "Retro Agent.md"),
		[]byte("# Retro\n\nNo links to SOM.\n"),
		0644,
	)

	// Hidden dir (should be skipped)
	os.WriteFile(
		filepath.Join(vaultDir, ".obsidian", "hidden.md"),
		[]byte("[[Session Operating Mode]] in hidden dir.\n"),
		0644,
	)

	results, err := findBacklinks(vaultDir, "Session Operating Mode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Strings(results)

	want := []string{
		"conventions/Pre-Compact Checklist.md",
		"methodology/Developer Agent.md",
	}

	if len(results) != len(want) {
		t.Fatalf("got %d results, want %d: %v", len(results), len(want), results)
	}
	for i, w := range want {
		if results[i] != w {
			t.Errorf("results[%d] = %q, want %q", i, results[i], w)
		}
	}
}

func TestFindBacklinks_CaseInsensitive(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "note.md"),
		[]byte("See [[session operating mode]] here.\n"),
		0644,
	)

	results, err := findBacklinks(vaultDir, "Session Operating Mode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("got %d results, want 1 (case-insensitive match)", len(results))
	}
}

func TestParseWikilinks_Embeds(t *testing.T) {
	text := "See ![[Embedded Note]] and ![[Other#Section|alias]] here."

	got := parseWikilinks(text)

	if len(got) != 2 {
		t.Fatalf("got %d links, want 2", len(got))
	}

	if !got[0].Embed || got[0].Title != "Embedded Note" {
		t.Errorf("link[0] = embed:%v title:%q, want embed:true title:\"Embedded Note\"", got[0].Embed, got[0].Title)
	}
	if !got[1].Embed || got[1].Title != "Other" || got[1].Heading != "Section" || got[1].Display != "alias" {
		t.Errorf("link[1] = %+v, want embed with heading and display", got[1])
	}
}

func TestReplaceWikilinks_Embeds(t *testing.T) {
	text := "See ![[Old Note]] and [[Old Note#Heading]] here."
	got := replaceWikilinks(text, "Old Note", "New Note")
	want := "See ![[New Note]] and [[New Note#Heading]] here."

	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindBacklinks_IncludesEmbeds(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "embedder.md"),
		[]byte("Content: ![[Target Note]]\n"),
		0644,
	)

	results, err := findBacklinks(vaultDir, "Target Note")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("got %d results, want 1 (embed as backlink)", len(results))
	}
}
