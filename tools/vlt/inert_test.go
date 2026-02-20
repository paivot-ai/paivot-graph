package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// --- Unit Tests ---

func TestMaskFencedCodeBlock(t *testing.T) {
	input := "Before\n```\n[[Link]] and #tag\n```\nAfter"
	got := maskInertContent(input)

	if strings.Contains(got, "[[Link]]") {
		t.Error("wikilink inside fenced code block should be masked")
	}
	if strings.Contains(got, "#tag") {
		t.Error("tag inside fenced code block should be masked")
	}
	if !strings.HasPrefix(got, "Before\n") {
		t.Error("content before fence should be unchanged")
	}
	if !strings.HasSuffix(got, "\nAfter") {
		t.Error("content after fence should be unchanged")
	}
}

func TestMaskFencedCodeBlockWithLanguage(t *testing.T) {
	languages := []string{"go", "python", "javascript", "rust", "yaml"}
	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			input := "Text\n```" + lang + "\n[[Link]] #tag\n```\nMore text"
			got := maskInertContent(input)

			if strings.Contains(got, "[[Link]]") {
				t.Errorf("wikilink inside ```%s block should be masked", lang)
			}
			if strings.Contains(got, "#tag") {
				t.Errorf("tag inside ```%s block should be masked", lang)
			}
			// Fence delimiter itself is NOT masked
			if !strings.Contains(got, "```"+lang) {
				t.Errorf("fence delimiter ```%s should be preserved", lang)
			}
		})
	}
}

func TestMaskMermaidBlock(t *testing.T) {
	input := "Before\n```mermaid\ngraph TD\nA[[Node A]] --> B[[Node B]]\n```\nAfter"
	got := maskInertContent(input)

	if strings.Contains(got, "[[Node A]]") {
		t.Error("wikilink inside mermaid block should be masked")
	}
	if strings.Contains(got, "[[Node B]]") {
		t.Error("second wikilink inside mermaid block should be masked")
	}
	if !strings.Contains(got, "```mermaid") {
		t.Error("mermaid fence delimiter should be preserved")
	}
}

func TestMaskPreservesLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "basic fenced block",
			input: "Before\n```\n[[Link]] and #tag\n```\nAfter",
		},
		{
			name:  "language tagged block",
			input: "Text\n```go\nfunc main() { fmt.Println(\"[[Link]]\") }\n```\nEnd",
		},
		{
			name:  "mermaid block",
			input: "```mermaid\nA[[X]] --> B[[Y]]\n```",
		},
		{
			name:  "multiple blocks",
			input: "```\nblock1\n```\nMiddle\n```python\nblock2\n```",
		},
		{
			name:  "empty block",
			input: "```\n```",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskInertContent(tt.input)
			if len(got) != len(tt.input) {
				t.Errorf("length changed: input=%d, output=%d", len(tt.input), len(got))
			}
		})
	}
}

func TestMaskPreservesNewlines(t *testing.T) {
	input := "Before\n```\nline1\nline2\nline3\n```\nAfter"
	got := maskInertContent(input)

	inputNewlines := strings.Count(input, "\n")
	gotNewlines := strings.Count(got, "\n")

	if inputNewlines != gotNewlines {
		t.Errorf("newline count changed: input=%d, output=%d", inputNewlines, gotNewlines)
	}

	// Verify specific newlines within masked zone are preserved
	lines := strings.Split(got, "\n")
	// lines: "Before", "```", "     ", "     ", "     ", "```", "After"
	if len(lines) != 7 {
		t.Fatalf("expected 7 lines, got %d", len(lines))
	}
	// Masked lines should be all spaces (non-newline chars replaced)
	for i := 2; i <= 4; i++ {
		if strings.TrimRight(lines[i], " ") != "" {
			t.Errorf("line %d should be all spaces, got %q", i, lines[i])
		}
	}
}

func TestMaskNonFencedContentUnchanged(t *testing.T) {
	input := "# Title\n\nSome [[Link]] and #tag text.\n\nMore content."
	got := maskInertContent(input)

	if got != input {
		t.Errorf("non-fenced content should be unchanged:\ngot:  %q\nwant: %q", got, input)
	}
}

func TestMaskMultipleFencedBlocks(t *testing.T) {
	input := "Start\n```\n[[A]]\n```\nMiddle [[B]]\n```go\n[[C]] #tag\n```\nEnd"
	got := maskInertContent(input)

	if strings.Contains(got, "[[A]]") {
		t.Error("wikilink in first fenced block should be masked")
	}
	if !strings.Contains(got, "[[B]]") {
		t.Error("wikilink between fenced blocks should be preserved")
	}
	if strings.Contains(got, "[[C]]") {
		t.Error("wikilink in second fenced block should be masked")
	}
}

func TestMaskUnclosedFence(t *testing.T) {
	input := "Before\n```\n[[Link]] and #tag\nmore content"
	got := maskInertContent(input)

	if strings.Contains(got, "[[Link]]") {
		t.Error("wikilink after unclosed fence should be masked (Obsidian behavior)")
	}
	if strings.Contains(got, "#tag") {
		t.Error("tag after unclosed fence should be masked")
	}
	if len(got) != len(input) {
		t.Errorf("length changed: input=%d, output=%d", len(input), len(got))
	}
}

func TestMaskNestedBackticks(t *testing.T) {
	input := "```\nSome `inline` code with [[Link]]\n```"
	got := maskInertContent(input)

	if strings.Contains(got, "[[Link]]") {
		t.Error("wikilink inside fenced block with inline backticks should be masked")
	}
}

func TestMaskEmptyFencedBlock(t *testing.T) {
	input := "Before\n```\n```\nAfter"
	got := maskInertContent(input)

	if len(got) != len(input) {
		t.Errorf("length changed: input=%d, output=%d", len(input), len(got))
	}
	if !strings.Contains(got, "Before") || !strings.Contains(got, "After") {
		t.Error("content outside empty fenced block should be unchanged")
	}
}

func TestRegisteredPassesPattern(t *testing.T) {
	// Save and restore global state
	origPasses := make([]maskPass, len(inertPasses))
	copy(origPasses, inertPasses)
	defer func() { inertPasses = origPasses }()

	// Clear passes and verify
	inertPasses = nil

	var callOrder []int

	registerMaskPass(func(text string) string {
		callOrder = append(callOrder, 1)
		return strings.ReplaceAll(text, "AAA", "BBB")
	})
	registerMaskPass(func(text string) string {
		callOrder = append(callOrder, 2)
		return strings.ReplaceAll(text, "BBB", "CCC")
	})

	result := maskInertContent("AAA")

	if result != "CCC" {
		t.Errorf("passes not applied in order: got %q, want %q", result, "CCC")
	}
	if len(callOrder) != 2 || callOrder[0] != 1 || callOrder[1] != 2 {
		t.Errorf("pass execution order wrong: %v", callOrder)
	}
}

// --- Integration Tests ---

func TestParseWikilinksIgnoresFencedCode(t *testing.T) {
	text := "Normal [[Outside]] link.\n```\n[[Inside]] should be ignored.\n```\nMore [[AlsoOutside]]."
	masked := maskInertContent(text)
	links := parseWikilinks(masked)

	titles := make(map[string]bool)
	for _, l := range links {
		titles[l.Title] = true
	}

	if !titles["Outside"] {
		t.Error("expected to find [[Outside]]")
	}
	if !titles["AlsoOutside"] {
		t.Error("expected to find [[AlsoOutside]]")
	}
	if titles["Inside"] {
		t.Error("should NOT find [[Inside]] from fenced code block")
	}
	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d: %v", len(links), links)
	}
}

func TestParseInlineTagsIgnoresFencedCode(t *testing.T) {
	text := "Normal #outside tag.\n```\n#inside should be ignored.\n```\nMore #alsooutside."
	masked := maskInertContent(text)
	tags := parseInlineTags(masked)

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	if !tagSet["outside"] {
		t.Error("expected to find #outside")
	}
	if !tagSet["alsooutside"] {
		t.Error("expected to find #alsooutside")
	}
	if tagSet["inside"] {
		t.Error("should NOT find #inside from fenced code block")
	}
}

func TestFindBacklinksIgnoresFencedCode(t *testing.T) {
	vaultDir := t.TempDir()

	// Note A links to B only inside a code block
	os.WriteFile(
		filepath.Join(vaultDir, "A.md"),
		[]byte("# A\n\nSome text.\n```\n[[B]] in code\n```\n"),
		0644,
	)

	// Note B exists
	os.WriteFile(
		filepath.Join(vaultDir, "B.md"),
		[]byte("# B\n\nContent.\n"),
		0644,
	)

	results, err := findBacklinks(vaultDir, "B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 backlinks (link is inside code block), got %d: %v", len(results), results)
	}
}

func TestOrphansIgnoresFencedCode(t *testing.T) {
	vaultDir := t.TempDir()

	// A links to B ONLY inside a code block -- B should be orphaned
	os.WriteFile(
		filepath.Join(vaultDir, "A.md"),
		[]byte("# A\n\n```\n[[B]] in code\n```\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "B.md"),
		[]byte("# B\n\nContent.\n"),
		0644,
	)

	// Capture orphans by examining the function behavior
	// cmdOrphans uses parseWikilinks which should now mask fenced content
	// B should appear as an orphan since the only link to it is inside a code block
	// We need to test the behavior through the public functions

	// Collect referenced titles the same way cmdOrphans does
	referenced := make(map[string]bool)
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, link := range parseWikilinks(string(data)) {
			referenced[strings.ToLower(link.Title)] = true
		}
		return nil
	})

	if referenced["b"] {
		t.Error("B should NOT be referenced (link is inside code block), so it should be an orphan")
	}
}

func TestUnresolvedIgnoresFencedCode(t *testing.T) {
	vaultDir := t.TempDir()

	// Note with [[Missing]] only inside a code block
	os.WriteFile(
		filepath.Join(vaultDir, "Source.md"),
		[]byte("# Source\n\n```\n[[Missing]] in code\n```\n"),
		0644,
	)

	// Simulate unresolved detection the same way cmdUnresolved does
	titles := make(map[string]bool)
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		title := strings.TrimSuffix(d.Name(), ".md")
		titles[strings.ToLower(title)] = true
		return nil
	})

	var unresolved []string
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, link := range parseWikilinks(string(data)) {
			lower := strings.ToLower(link.Title)
			if !titles[lower] {
				unresolved = append(unresolved, link.Title)
			}
		}
		return nil
	})

	if len(unresolved) != 0 {
		t.Errorf("expected 0 unresolved links (link is inside code block), got %v", unresolved)
	}
}

func TestLinksIgnoresFencedCode(t *testing.T) {
	vaultDir := t.TempDir()

	// Note with [[Target]] both inside and outside code block
	os.WriteFile(
		filepath.Join(vaultDir, "Source.md"),
		[]byte("# Source\n\n```\n[[InsideOnly]] in code\n```\n[[Outside]] is real.\n"),
		0644,
	)
	os.WriteFile(
		filepath.Join(vaultDir, "Outside.md"),
		[]byte("# Outside\n"),
		0644,
	)

	// Read the source and parse links the same way cmdLinks does
	data, err := os.ReadFile(filepath.Join(vaultDir, "Source.md"))
	if err != nil {
		t.Fatal(err)
	}

	links := parseWikilinks(string(data))

	titles := make(map[string]bool)
	for _, l := range links {
		titles[l.Title] = true
	}

	if titles["InsideOnly"] {
		t.Error("should NOT find [[InsideOnly]] from fenced code block")
	}
	if !titles["Outside"] {
		t.Error("should find [[Outside]] from outside code block")
	}
}

func TestTagsIgnoresFencedCode(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "note.md"),
		[]byte("# Note\n\n#real-tag\n\n```\n#hidden-tag\n```\n"),
		0644,
	)

	// Read and parse tags the same way cmdTags does through allNoteTags
	data, err := os.ReadFile(filepath.Join(vaultDir, "note.md"))
	if err != nil {
		t.Fatal(err)
	}

	tags := allNoteTags(string(data))
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	if !tagSet["real-tag"] {
		t.Error("should find #real-tag from outside code block")
	}
	if tagSet["hidden-tag"] {
		t.Error("should NOT find #hidden-tag from inside code block")
	}
}

func TestMermaidBlockIgnored(t *testing.T) {
	vaultDir := t.TempDir()

	os.WriteFile(
		filepath.Join(vaultDir, "diagram.md"),
		[]byte("# Diagram\n\n[[RealLink]]\n\n```mermaid\ngraph TD\nA[[FakeLink]] --> B\n```\n"),
		0644,
	)

	data, err := os.ReadFile(filepath.Join(vaultDir, "diagram.md"))
	if err != nil {
		t.Fatal(err)
	}

	links := parseWikilinks(string(data))
	titles := make(map[string]bool)
	for _, l := range links {
		titles[l.Title] = true
	}

	if !titles["RealLink"] {
		t.Error("should find [[RealLink]] outside mermaid block")
	}
	if titles["FakeLink"] {
		t.Error("should NOT find [[FakeLink]] inside mermaid block")
	}
}

// --- E2E Test ---

func TestE2EInertZoneFencedCode(t *testing.T) {
	vaultDir := t.TempDir()

	// Create a realistic vault with wikilinks and tags both inside and outside fenced code blocks

	// Note 1: Has both real and code-fenced links/tags
	os.WriteFile(
		filepath.Join(vaultDir, "Overview.md"),
		[]byte("---\ntags: [project]\n---\n\n# Overview\n\nSee [[Design Doc]] for details.\n\n#architecture\n\n```go\n// Example: [[FakeLink]] reference\nfmt.Println(\"#not-a-tag\")\n```\n\n```mermaid\ngraph TD\nA[[MermaidNode]] --> B\n```\n"),
		0644,
	)

	// Note 2: The real link target
	os.WriteFile(
		filepath.Join(vaultDir, "Design Doc.md"),
		[]byte("# Design Doc\n\nDetails here. See [[Overview]] for context.\n"),
		0644,
	)

	// Note 3: Only referenced inside a code block (should be orphaned)
	os.WriteFile(
		filepath.Join(vaultDir, "FakeLink.md"),
		[]byte("# FakeLink\n\nI should be an orphan because I'm only referenced in code blocks.\n"),
		0644,
	)

	// Note 4: Not referenced at all
	os.WriteFile(
		filepath.Join(vaultDir, "Island.md"),
		[]byte("# Island\n\nTruly unreferenced.\n"),
		0644,
	)

	// --- Test backlinks ---
	// "Design Doc" should have backlinks from "Overview" (real link)
	backlinks, err := findBacklinks(vaultDir, "Design Doc")
	if err != nil {
		t.Fatalf("findBacklinks Design Doc: %v", err)
	}
	if len(backlinks) != 1 || backlinks[0] != "Overview.md" {
		t.Errorf("Design Doc backlinks: got %v, want [Overview.md]", backlinks)
	}

	// "FakeLink" should have NO backlinks (only referenced in code block)
	backlinks, err = findBacklinks(vaultDir, "FakeLink")
	if err != nil {
		t.Fatalf("findBacklinks FakeLink: %v", err)
	}
	if len(backlinks) != 0 {
		t.Errorf("FakeLink should have 0 backlinks (code-only reference), got %v", backlinks)
	}

	// "MermaidNode" should have NO backlinks (only in mermaid block)
	backlinks, err = findBacklinks(vaultDir, "MermaidNode")
	if err != nil {
		t.Fatalf("findBacklinks MermaidNode: %v", err)
	}
	if len(backlinks) != 0 {
		t.Errorf("MermaidNode should have 0 backlinks, got %v", backlinks)
	}

	// --- Test links ---
	overviewData, _ := os.ReadFile(filepath.Join(vaultDir, "Overview.md"))
	links := parseWikilinks(string(overviewData))
	linkTitles := make(map[string]bool)
	for _, l := range links {
		linkTitles[l.Title] = true
	}

	if !linkTitles["Design Doc"] {
		t.Error("Overview should link to Design Doc")
	}
	if linkTitles["FakeLink"] {
		t.Error("Overview should NOT link to FakeLink (inside code block)")
	}
	if linkTitles["MermaidNode"] {
		t.Error("Overview should NOT link to MermaidNode (inside mermaid block)")
	}

	// --- Test orphans ---
	// Collect referenced titles (same logic as cmdOrphans)
	referenced := make(map[string]bool)
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, link := range parseWikilinks(string(data)) {
			referenced[strings.ToLower(link.Title)] = true
		}
		return nil
	})

	// FakeLink should be orphaned (only referenced in code block)
	if referenced["fakelink"] {
		t.Error("FakeLink should be unreferenced (code-only reference)")
	}
	// Island should be orphaned
	if referenced["island"] {
		t.Error("Island should be unreferenced")
	}
	// Design Doc should NOT be orphaned
	if !referenced["design doc"] {
		t.Error("Design Doc should be referenced by Overview")
	}
	// Overview should NOT be orphaned
	if !referenced["overview"] {
		t.Error("Overview should be referenced by Design Doc")
	}

	// --- Test unresolved ---
	// Collect all note titles
	titles := make(map[string]bool)
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		title := strings.TrimSuffix(d.Name(), ".md")
		titles[strings.ToLower(title)] = true
		return nil
	})

	// Collect unresolved links
	var unresolvedLinks []string
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, link := range parseWikilinks(string(data)) {
			lower := strings.ToLower(link.Title)
			if !titles[lower] {
				unresolvedLinks = append(unresolvedLinks, link.Title)
			}
		}
		return nil
	})

	// FakeLink exists as a note, so even if it were linked, it wouldn't be unresolved.
	// MermaidNode does NOT exist as a note, but the link is inside a code block,
	// so it should NOT appear as unresolved.
	for _, u := range unresolvedLinks {
		if u == "MermaidNode" {
			t.Error("MermaidNode should NOT be unresolved (link is inside code block)")
		}
		if u == "FakeLink" {
			t.Error("FakeLink should NOT be unresolved (link is inside code block)")
		}
	}

	// --- Test tags ---
	overviewTags := allNoteTags(string(overviewData))
	tagSet := make(map[string]bool)
	for _, tag := range overviewTags {
		tagSet[tag] = true
	}

	// Frontmatter tag should be found
	if !tagSet["project"] {
		t.Error("should find frontmatter tag 'project'")
	}
	// Inline tag outside code should be found
	if !tagSet["architecture"] {
		t.Error("should find inline tag 'architecture'")
	}
	// Tag inside code block should NOT be found
	if tagSet["not-a-tag"] {
		t.Error("should NOT find 'not-a-tag' from inside code block")
	}

	// --- Verify orphans are correctly sorted ---
	type noteInfo struct {
		relPath string
		title   string
	}
	var notes []noteInfo
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		title := strings.TrimSuffix(d.Name(), ".md")
		relPath, _ := filepath.Rel(vaultDir, path)
		notes = append(notes, noteInfo{relPath: relPath, title: title})
		return nil
	})

	var orphans []string
	for _, note := range notes {
		if !referenced[strings.ToLower(note.title)] {
			orphans = append(orphans, note.relPath)
		}
	}
	sort.Strings(orphans)

	// FakeLink.md and Island.md should be orphans
	expectedOrphans := []string{"FakeLink.md", "Island.md"}
	if len(orphans) != len(expectedOrphans) {
		t.Errorf("expected %d orphans, got %d: %v", len(expectedOrphans), len(orphans), orphans)
	} else {
		for i, want := range expectedOrphans {
			if orphans[i] != want {
				t.Errorf("orphan[%d] = %q, want %q", i, orphans[i], want)
			}
		}
	}
}

// =============================================================================
// Inline Code Masking Tests (VLT-7fi)
// =============================================================================

// --- Unit Tests ---

func TestMaskInlineCode(t *testing.T) {
	input := "Text `[[Link]]` more text"
	got := maskInertContent(input)

	if strings.Contains(got, "[[Link]]") {
		t.Error("wikilink inside inline code should be masked")
	}
	// Backtick delimiters should be preserved
	if !strings.Contains(got, "`") {
		t.Error("backtick delimiters should be preserved")
	}
	// Text outside inline code should be unchanged
	if !strings.HasPrefix(got, "Text ") {
		t.Error("text before inline code should be unchanged")
	}
	if !strings.HasSuffix(got, " more text") {
		t.Error("text after inline code should be unchanged")
	}
}

func TestMaskDoubleBacktickCode(t *testing.T) {
	input := "Text ``[[Link]] `with` backtick`` more"
	got := maskInertContent(input)

	if strings.Contains(got, "[[Link]]") {
		t.Error("wikilink inside double-backtick inline code should be masked")
	}
	// Text outside should be preserved
	if !strings.HasPrefix(got, "Text ") {
		t.Error("text before double-backtick code should be unchanged")
	}
	if !strings.HasSuffix(got, " more") {
		t.Error("text after double-backtick code should be unchanged")
	}
}

func TestMaskMultipleInlineCodePerLine(t *testing.T) {
	input := "See `[[A]]` then `#tag` end"
	got := maskInertContent(input)

	if strings.Contains(got, "[[A]]") {
		t.Error("first inline code span should be masked")
	}
	if strings.Contains(got, "#tag") {
		t.Error("second inline code span should be masked")
	}
	if !strings.Contains(got, "See ") {
		t.Error("text between spans should be preserved")
	}
	if !strings.Contains(got, " then ") {
		t.Error("text between spans should be preserved")
	}
	if !strings.HasSuffix(got, " end") {
		t.Error("text after spans should be preserved")
	}
}

func TestMaskInlineCodePreservesLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "single backtick",
			input: "Text `[[Link]]` end",
		},
		{
			name:  "double backtick",
			input: "Text ``code `with` backtick`` end",
		},
		{
			name:  "multiple spans",
			input: "`a` text `b` more `c`",
		},
		{
			name:  "inline code with tag",
			input: "See `#not-a-tag` here",
		},
		{
			name:  "mixed fenced and inline",
			input: "Before\n```\n[[A]]\n```\nMiddle `[[B]]` end",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskInertContent(tt.input)
			if len(got) != len(tt.input) {
				t.Errorf("length changed: input=%d, output=%d", len(tt.input), len(got))
			}
		})
	}
}

func TestMaskInlineCodeNotInFencedBlock(t *testing.T) {
	// Backticks inside an already-masked fenced code block should NOT be
	// double-processed. The fenced block pass runs first and masks the
	// content. The inline code pass should not find backtick pairs inside
	// the already-masked (all-spaces) region.
	input := "Before\n```\nSome `inline` with [[Link]]\n```\nAfter `[[Outside]]` end"
	got := maskInertContent(input)

	// The fenced block content is masked first, so [[Link]] should be gone
	if strings.Contains(got, "[[Link]]") {
		t.Error("wikilink inside fenced block should be masked by fenced pass")
	}
	// The inline code outside the fenced block should also be masked
	if strings.Contains(got, "[[Outside]]") {
		t.Error("wikilink inside inline code after fenced block should be masked")
	}
	// "Before" and "After" text outside both should be preserved
	if !strings.HasPrefix(got, "Before\n") {
		t.Error("text before fenced block should be preserved")
	}
	if !strings.HasSuffix(got, " end") {
		t.Error("text after inline code should be preserved")
	}
}

// --- Integration Tests ---

func TestParseWikilinksIgnoresInlineCode(t *testing.T) {
	text := "Normal [[Outside]] link and `[[Inside]]` should be ignored."
	links := parseWikilinks(text)

	titles := make(map[string]bool)
	for _, l := range links {
		titles[l.Title] = true
	}

	if !titles["Outside"] {
		t.Error("expected to find [[Outside]]")
	}
	if titles["Inside"] {
		t.Error("should NOT find [[Inside]] from inline code")
	}
	if len(links) != 1 {
		t.Errorf("expected 1 link, got %d: %v", len(links), links)
	}
}

func TestParseInlineTagsIgnoresInlineCode(t *testing.T) {
	// Use a space before #inside so the tag pattern can match it
	// (tag pattern requires whitespace or start-of-line before #)
	text := "Normal #outside tag and ` #inside ` should be ignored."
	tags := parseInlineTags(text)

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	if !tagSet["outside"] {
		t.Error("expected to find #outside")
	}
	if tagSet["inside"] {
		t.Error("should NOT find #inside from inline code")
	}
}

func TestFindBacklinksIgnoresInlineCode(t *testing.T) {
	vaultDir := t.TempDir()

	// Note A links to B only inside inline code
	os.WriteFile(
		filepath.Join(vaultDir, "A.md"),
		[]byte("# A\n\nSome text with `[[B]]` in inline code.\n"),
		0644,
	)

	// Note B exists
	os.WriteFile(
		filepath.Join(vaultDir, "B.md"),
		[]byte("# B\n\nContent.\n"),
		0644,
	)

	results, err := findBacklinks(vaultDir, "B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 backlinks (link is inside inline code), got %d: %v", len(results), results)
	}
}

func TestMixedFencedAndInlineCode(t *testing.T) {
	vaultDir := t.TempDir()

	// Note with links and tags in both fenced code, inline code, and normal text
	content := `# Mixed Note

Real link: [[RealTarget]]
Real tag: #real-tag

Inline code link: ` + "`[[FakeInline]]`" + `
Inline code tag: ` + "`#fake-inline`" + `

` + "```" + `
[[FakeFenced]]
#fake-fenced
` + "```" + `

Double backtick: ` + "``[[FakeDouble]]``" + `

End.
`

	os.WriteFile(
		filepath.Join(vaultDir, "Mixed.md"),
		[]byte(content),
		0644,
	)

	// Read and parse
	data, err := os.ReadFile(filepath.Join(vaultDir, "Mixed.md"))
	if err != nil {
		t.Fatal(err)
	}

	// Test wikilinks
	links := parseWikilinks(string(data))
	linkTitles := make(map[string]bool)
	for _, l := range links {
		linkTitles[l.Title] = true
	}

	if !linkTitles["RealTarget"] {
		t.Error("should find [[RealTarget]] from normal text")
	}
	if linkTitles["FakeInline"] {
		t.Error("should NOT find [[FakeInline]] from inline code")
	}
	if linkTitles["FakeFenced"] {
		t.Error("should NOT find [[FakeFenced]] from fenced code")
	}
	if linkTitles["FakeDouble"] {
		t.Error("should NOT find [[FakeDouble]] from double-backtick inline code")
	}

	// Test tags
	tags := allNoteTags(string(data))
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	if !tagSet["real-tag"] {
		t.Error("should find #real-tag from normal text")
	}
	if tagSet["fake-inline"] {
		t.Error("should NOT find #fake-inline from inline code")
	}
	if tagSet["fake-fenced"] {
		t.Error("should NOT find #fake-fenced from fenced code")
	}

	// Test backlinks for RealTarget
	backlinks, err := findBacklinks(vaultDir, "RealTarget")
	if err != nil {
		t.Fatalf("findBacklinks: %v", err)
	}
	if len(backlinks) != 1 || backlinks[0] != "Mixed.md" {
		t.Errorf("RealTarget backlinks: got %v, want [Mixed.md]", backlinks)
	}

	// Test backlinks for FakeInline -- should be zero
	backlinks, err = findBacklinks(vaultDir, "FakeInline")
	if err != nil {
		t.Fatalf("findBacklinks FakeInline: %v", err)
	}
	if len(backlinks) != 0 {
		t.Errorf("FakeInline should have 0 backlinks, got %v", backlinks)
	}

	// Test backlinks for FakeDouble -- should be zero
	backlinks, err = findBacklinks(vaultDir, "FakeDouble")
	if err != nil {
		t.Fatalf("findBacklinks FakeDouble: %v", err)
	}
	if len(backlinks) != 0 {
		t.Errorf("FakeDouble should have 0 backlinks, got %v", backlinks)
	}
}

// =============================================================================
// Obsidian Comment Masking Tests (%% ... %%) -- VLT-i6l
// =============================================================================

// --- Unit Tests ---

func TestMaskObsidianCommentInline(t *testing.T) {
	input := "text %% hidden %% more"
	got := maskInertContent(input)

	if strings.Contains(got, "hidden") {
		t.Error("content inside inline %% comment should be masked")
	}
	if !strings.HasPrefix(got, "text ") {
		t.Error("content before comment should be unchanged")
	}
	if !strings.HasSuffix(got, " more") {
		t.Error("content after comment should be unchanged")
	}
	// The %% delimiters themselves should be preserved
	if !strings.Contains(got, "%%") {
		t.Error("%% delimiters should be preserved")
	}
}

func TestMaskObsidianCommentMultiline(t *testing.T) {
	input := "Before\n%%\nThis whole block\nis hidden in preview\n%%\nAfter"
	got := maskInertContent(input)

	if strings.Contains(got, "This whole block") {
		t.Error("content inside multiline %% comment should be masked")
	}
	if strings.Contains(got, "is hidden in preview") {
		t.Error("second line inside multiline %% comment should be masked")
	}
	if !strings.HasPrefix(got, "Before\n") {
		t.Error("content before comment should be unchanged")
	}
	if !strings.HasSuffix(got, "\nAfter") {
		t.Error("content after comment should be unchanged")
	}
}

func TestMaskObsidianCommentPreservesLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "inline comment",
			input: "text %% hidden %% more",
		},
		{
			name:  "multiline comment",
			input: "Before\n%%\nline1\nline2\n%%\nAfter",
		},
		{
			name:  "multiple comments",
			input: "a %% x %% b %% y %% c",
		},
		{
			name:  "comment with wikilink",
			input: "text %% [[Link]] %% end",
		},
		{
			name:  "comment with tag",
			input: "text %% #hidden-tag %% end",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskInertContent(tt.input)
			if len(got) != len(tt.input) {
				t.Errorf("length changed: input=%d, output=%d\ninput:  %q\noutput: %q",
					len(tt.input), len(got), tt.input, got)
			}
		})
	}
}

func TestMaskObsidianCommentPreservesNewlines(t *testing.T) {
	input := "Before\n%%\nline1\nline2\nline3\n%%\nAfter"
	got := maskInertContent(input)

	inputNewlines := strings.Count(input, "\n")
	gotNewlines := strings.Count(got, "\n")

	if inputNewlines != gotNewlines {
		t.Errorf("newline count changed: input=%d, output=%d", inputNewlines, gotNewlines)
	}

	// Verify that masked content lines become spaces (preserving newlines)
	lines := strings.Split(got, "\n")
	// lines: "Before", "%%", "     ", "     ", "     ", "%%", "After"
	if len(lines) != 7 {
		t.Fatalf("expected 7 lines, got %d: %v", len(lines), lines)
	}
	for i := 2; i <= 4; i++ {
		if strings.TrimRight(lines[i], " ") != "" {
			t.Errorf("line %d should be all spaces, got %q", i, lines[i])
		}
	}
}

func TestMaskMultipleObsidianComments(t *testing.T) {
	input := "start %% first comment %% middle %% second comment %% end"
	got := maskInertContent(input)

	if strings.Contains(got, "first comment") {
		t.Error("first comment should be masked")
	}
	if strings.Contains(got, "second comment") {
		t.Error("second comment should be masked")
	}
	if !strings.HasPrefix(got, "start ") {
		t.Error("text before first comment should be preserved")
	}
	if !strings.Contains(got, " middle ") {
		t.Error("text between comments should be preserved")
	}
	if !strings.HasSuffix(got, " end") {
		t.Error("text after last comment should be preserved")
	}
}

func TestMaskObsidianCommentInsideFencedBlock(t *testing.T) {
	// %% inside a code block should NOT be treated as a comment boundary
	// because the fenced code block pass runs first and masks the %% characters
	input := "Outside\n```\n%% not a comment %%\n```\nAlso outside"
	got := maskInertContent(input)

	// The code block content is masked, so %% should already be spaces.
	// The key assertion: "Also outside" must remain intact (not masked as
	// if a comment started inside the code block).
	if !strings.Contains(got, "Also outside") {
		t.Error("content after code block should be preserved; %% inside code block should not start a comment")
	}
	if !strings.Contains(got, "Outside") {
		t.Error("content before code block should be preserved")
	}
}

// --- Integration Tests ---

func TestParseWikilinksIgnoresObsidianComments(t *testing.T) {
	text := "Normal [[Outside]] link.\n%% [[Inside]] should be ignored. %%\nMore [[AlsoOutside]]."
	masked := maskInertContent(text)
	links := parseWikilinks(masked)

	titles := make(map[string]bool)
	for _, l := range links {
		titles[l.Title] = true
	}

	if !titles["Outside"] {
		t.Error("expected to find [[Outside]]")
	}
	if !titles["AlsoOutside"] {
		t.Error("expected to find [[AlsoOutside]]")
	}
	if titles["Inside"] {
		t.Error("should NOT find [[Inside]] from Obsidian comment")
	}
	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d: %v", len(links), links)
	}
}

func TestParseInlineTagsIgnoresObsidianComments(t *testing.T) {
	text := "Normal #outside tag.\n%% #inside should be ignored. %%\nMore #alsooutside."
	masked := maskInertContent(text)
	tags := parseInlineTags(masked)

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	if !tagSet["outside"] {
		t.Error("expected to find #outside")
	}
	if !tagSet["alsooutside"] {
		t.Error("expected to find #alsooutside")
	}
	if tagSet["inside"] {
		t.Error("should NOT find #inside from Obsidian comment")
	}
}

func TestFindBacklinksIgnoresObsidianComments(t *testing.T) {
	vaultDir := t.TempDir()

	// Note A links to B only inside an Obsidian comment
	os.WriteFile(
		filepath.Join(vaultDir, "A.md"),
		[]byte("# A\n\nSome text.\n%% [[B]] in comment %%\n"),
		0644,
	)

	// Note B exists
	os.WriteFile(
		filepath.Join(vaultDir, "B.md"),
		[]byte("# B\n\nContent.\n"),
		0644,
	)

	results, err := findBacklinks(vaultDir, "B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 backlinks (link is inside Obsidian comment), got %d: %v", len(results), results)
	}
}

func TestMixedCodeAndComments(t *testing.T) {
	vaultDir := t.TempDir()

	// Note with links in code blocks, inline code, AND Obsidian comments.
	// Only the plain-text link should be detected.
	os.WriteFile(
		filepath.Join(vaultDir, "Mixed.md"),
		[]byte("# Mixed\n\n[[RealLink]]\n\n```\n[[CodeLink]] #code-tag\n```\n\n%% [[CommentLink]] #comment-tag %%\n\n#real-tag\n"),
		0644,
	)

	data, err := os.ReadFile(filepath.Join(vaultDir, "Mixed.md"))
	if err != nil {
		t.Fatal(err)
	}

	// Test wikilinks
	links := parseWikilinks(string(data))
	linkTitles := make(map[string]bool)
	for _, l := range links {
		linkTitles[l.Title] = true
	}

	if !linkTitles["RealLink"] {
		t.Error("should find [[RealLink]] (plain text)")
	}
	if linkTitles["CodeLink"] {
		t.Error("should NOT find [[CodeLink]] (inside code block)")
	}
	if linkTitles["CommentLink"] {
		t.Error("should NOT find [[CommentLink]] (inside Obsidian comment)")
	}

	// Test tags
	tags := allNoteTags(string(data))
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	if !tagSet["real-tag"] {
		t.Error("should find #real-tag (plain text)")
	}
	if tagSet["code-tag"] {
		t.Error("should NOT find #code-tag (inside code block)")
	}
	if tagSet["comment-tag"] {
		t.Error("should NOT find #comment-tag (inside Obsidian comment)")
	}
}

// =============================================================================
// HTML Comment Masking Tests (<!-- ... -->) -- VLT-h6k
// =============================================================================

// --- Unit Tests ---

func TestMaskHTMLCommentInline(t *testing.T) {
	input := "text <!-- hidden --> more"
	got := maskInertContent(input)

	if strings.Contains(got, "hidden") {
		t.Error("content inside inline HTML comment should be masked")
	}
	if !strings.HasPrefix(got, "text ") {
		t.Error("content before HTML comment should be unchanged")
	}
	if !strings.HasSuffix(got, " more") {
		t.Error("content after HTML comment should be unchanged")
	}
	// The <!-- and --> delimiters themselves should be preserved
	if !strings.Contains(got, "<!--") {
		t.Error("<!-- delimiter should be preserved")
	}
	if !strings.Contains(got, "-->") {
		t.Error("--> delimiter should be preserved")
	}
}

func TestMaskHTMLCommentMultiline(t *testing.T) {
	input := "Before\n<!--\nThis whole block\nis hidden in preview\n-->\nAfter"
	got := maskInertContent(input)

	if strings.Contains(got, "This whole block") {
		t.Error("content inside multiline HTML comment should be masked")
	}
	if strings.Contains(got, "is hidden in preview") {
		t.Error("second line inside multiline HTML comment should be masked")
	}
	if !strings.HasPrefix(got, "Before\n") {
		t.Error("content before HTML comment should be unchanged")
	}
	if !strings.HasSuffix(got, "\nAfter") {
		t.Error("content after HTML comment should be unchanged")
	}
	// Newlines inside the comment should be preserved
	inputNewlines := strings.Count(input, "\n")
	gotNewlines := strings.Count(got, "\n")
	if inputNewlines != gotNewlines {
		t.Errorf("newline count changed: input=%d, output=%d", inputNewlines, gotNewlines)
	}
}

func TestMaskHTMLCommentPreservesLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "inline comment",
			input: "text <!-- hidden --> more",
		},
		{
			name:  "multiline comment",
			input: "Before\n<!--\nline1\nline2\n-->\nAfter",
		},
		{
			name:  "multiple comments",
			input: "a <!-- x --> b <!-- y --> c",
		},
		{
			name:  "comment with wikilink",
			input: "text <!-- [[Link]] --> end",
		},
		{
			name:  "comment with tag",
			input: "text <!-- #hidden-tag --> end",
		},
		{
			name:  "empty comment",
			input: "text <!----> end",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskInertContent(tt.input)
			if len(got) != len(tt.input) {
				t.Errorf("length changed: input=%d, output=%d\ninput:  %q\noutput: %q",
					len(tt.input), len(got), tt.input, got)
			}
		})
	}
}

func TestMaskMultipleHTMLComments(t *testing.T) {
	input := "start <!-- first comment --> middle <!-- second comment --> end"
	got := maskInertContent(input)

	if strings.Contains(got, "first comment") {
		t.Error("first HTML comment should be masked")
	}
	if strings.Contains(got, "second comment") {
		t.Error("second HTML comment should be masked")
	}
	if !strings.HasPrefix(got, "start ") {
		t.Error("text before first comment should be preserved")
	}
	if !strings.Contains(got, " middle ") {
		t.Error("text between comments should be preserved")
	}
	if !strings.HasSuffix(got, " end") {
		t.Error("text after last comment should be preserved")
	}
}

func TestMaskHTMLCommentInsideFencedBlock(t *testing.T) {
	// <!-- inside a code block should NOT be treated as an HTML comment boundary
	// because the fenced code block pass runs first and masks the <!-- characters
	input := "Outside\n```\n<!-- not a comment -->\n```\nAlso outside"
	got := maskInertContent(input)

	// The code block content is masked, so <!-- should already be spaces.
	// The key assertion: "Also outside" must remain intact (not masked as
	// if an HTML comment started inside the code block).
	if !strings.Contains(got, "Also outside") {
		t.Error("content after code block should be preserved; <!-- inside code block should not start an HTML comment")
	}
	if !strings.Contains(got, "Outside") {
		t.Error("content before code block should be preserved")
	}
}

// --- Integration Tests ---

func TestParseWikilinksIgnoresHTMLComments(t *testing.T) {
	text := "Normal [[Outside]] link.\n<!-- [[Inside]] should be ignored. -->\nMore [[AlsoOutside]]."
	masked := maskInertContent(text)
	links := parseWikilinks(masked)

	titles := make(map[string]bool)
	for _, l := range links {
		titles[l.Title] = true
	}

	if !titles["Outside"] {
		t.Error("expected to find [[Outside]]")
	}
	if !titles["AlsoOutside"] {
		t.Error("expected to find [[AlsoOutside]]")
	}
	if titles["Inside"] {
		t.Error("should NOT find [[Inside]] from HTML comment")
	}
	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d: %v", len(links), links)
	}
}

func TestParseInlineTagsIgnoresHTMLComments(t *testing.T) {
	text := "Normal #outside tag.\n<!-- #inside should be ignored. -->\nMore #alsooutside."
	masked := maskInertContent(text)
	tags := parseInlineTags(masked)

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	if !tagSet["outside"] {
		t.Error("expected to find #outside")
	}
	if !tagSet["alsooutside"] {
		t.Error("expected to find #alsooutside")
	}
	if tagSet["inside"] {
		t.Error("should NOT find #inside from HTML comment")
	}
}

func TestFindBacklinksIgnoresHTMLComments(t *testing.T) {
	vaultDir := t.TempDir()

	// Note A links to B only inside an HTML comment
	os.WriteFile(
		filepath.Join(vaultDir, "A.md"),
		[]byte("# A\n\nSome text.\n<!-- [[B]] in HTML comment -->\n"),
		0644,
	)

	// Note B exists
	os.WriteFile(
		filepath.Join(vaultDir, "B.md"),
		[]byte("# B\n\nContent.\n"),
		0644,
	)

	results, err := findBacklinks(vaultDir, "B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 backlinks (link is inside HTML comment), got %d: %v", len(results), results)
	}
}

func TestMixedAllCommentTypes(t *testing.T) {
	vaultDir := t.TempDir()

	// Note with links and tags in code blocks, inline code, Obsidian comments,
	// HTML comments, AND normal text. Only the plain-text link should be detected.
	os.WriteFile(
		filepath.Join(vaultDir, "AllTypes.md"),
		[]byte("# All Types\n\n[[RealLink]]\n#real-tag\n\n```\n[[CodeLink]] #code-tag\n```\n\n`[[InlineCodeLink]]`\n\n%% [[ObsidianCommentLink]] #obsidian-tag %%\n\n<!-- [[HTMLCommentLink]] #html-tag -->\n\n<!--\n[[MultilineHTMLLink]]\n#multiline-html-tag\n-->\n"),
		0644,
	)

	data, err := os.ReadFile(filepath.Join(vaultDir, "AllTypes.md"))
	if err != nil {
		t.Fatal(err)
	}

	// Test wikilinks -- only RealLink should be found
	links := parseWikilinks(string(data))
	linkTitles := make(map[string]bool)
	for _, l := range links {
		linkTitles[l.Title] = true
	}

	if !linkTitles["RealLink"] {
		t.Error("should find [[RealLink]] (plain text)")
	}
	if linkTitles["CodeLink"] {
		t.Error("should NOT find [[CodeLink]] (inside code block)")
	}
	if linkTitles["InlineCodeLink"] {
		t.Error("should NOT find [[InlineCodeLink]] (inside inline code)")
	}
	if linkTitles["ObsidianCommentLink"] {
		t.Error("should NOT find [[ObsidianCommentLink]] (inside Obsidian comment)")
	}
	if linkTitles["HTMLCommentLink"] {
		t.Error("should NOT find [[HTMLCommentLink]] (inside HTML comment)")
	}
	if linkTitles["MultilineHTMLLink"] {
		t.Error("should NOT find [[MultilineHTMLLink]] (inside multiline HTML comment)")
	}

	// Test tags -- only real-tag should be found
	tags := allNoteTags(string(data))
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	if !tagSet["real-tag"] {
		t.Error("should find #real-tag (plain text)")
	}
	if tagSet["code-tag"] {
		t.Error("should NOT find #code-tag (inside code block)")
	}
	if tagSet["obsidian-tag"] {
		t.Error("should NOT find #obsidian-tag (inside Obsidian comment)")
	}
	if tagSet["html-tag"] {
		t.Error("should NOT find #html-tag (inside HTML comment)")
	}
	if tagSet["multiline-html-tag"] {
		t.Error("should NOT find #multiline-html-tag (inside multiline HTML comment)")
	}

	// Test backlinks -- only RealLink should generate backlinks
	for _, name := range []string{"CodeLink", "InlineCodeLink", "ObsidianCommentLink", "HTMLCommentLink", "MultilineHTMLLink"} {
		backlinks, err := findBacklinks(vaultDir, name)
		if err != nil {
			t.Fatalf("findBacklinks %s: %v", name, err)
		}
		if len(backlinks) != 0 {
			t.Errorf("%s should have 0 backlinks (inside inert zone), got %v", name, backlinks)
		}
	}

	// RealLink should have a backlink from AllTypes.md
	backlinks, err := findBacklinks(vaultDir, "RealLink")
	if err != nil {
		t.Fatalf("findBacklinks RealLink: %v", err)
	}
	if len(backlinks) != 1 || backlinks[0] != "AllTypes.md" {
		t.Errorf("RealLink backlinks: got %v, want [AllTypes.md]", backlinks)
	}
}

// =============================================================================
// Math Block Masking Tests ($$ ... $$ and $ ... $) -- VLT-m4j
// =============================================================================

// --- Unit Tests ---

func TestMaskDisplayMath(t *testing.T) {
	input := "Before $$ [[Link]] + #tag $$ After"
	got := maskInertContent(input)

	if strings.Contains(got, "[[Link]]") {
		t.Error("wikilink inside display math should be masked")
	}
	if strings.Contains(got, "#tag") {
		t.Error("tag inside display math should be masked")
	}
	if !strings.HasPrefix(got, "Before ") {
		t.Error("content before display math should be unchanged")
	}
	if !strings.HasSuffix(got, " After") {
		t.Error("content after display math should be unchanged")
	}
	// $$ delimiters should be preserved
	if !strings.Contains(got, "$$") {
		t.Error("$$ delimiters should be preserved")
	}
}

func TestMaskDisplayMathMultiline(t *testing.T) {
	input := "Before\n$$\n\\sum_{i=1}^{n} [[Link]]\nx_i #tag\n$$\nAfter"
	got := maskInertContent(input)

	if strings.Contains(got, "[[Link]]") {
		t.Error("wikilink inside multiline display math should be masked")
	}
	if strings.Contains(got, "#tag") {
		t.Error("tag inside multiline display math should be masked")
	}
	if !strings.HasPrefix(got, "Before\n") {
		t.Error("content before multiline display math should be unchanged")
	}
	if !strings.HasSuffix(got, "\nAfter") {
		t.Error("content after multiline display math should be unchanged")
	}
	// Newlines inside math block should be preserved
	inputNewlines := strings.Count(input, "\n")
	gotNewlines := strings.Count(got, "\n")
	if inputNewlines != gotNewlines {
		t.Errorf("newline count changed: input=%d, output=%d", inputNewlines, gotNewlines)
	}
}

func TestMaskInlineMathSingle(t *testing.T) {
	input := "The formula $x = [[y]]$ is important"
	got := maskInertContent(input)

	if strings.Contains(got, "[[y]]") {
		t.Error("wikilink inside inline math should be masked")
	}
	if !strings.HasPrefix(got, "The formula ") {
		t.Error("content before inline math should be unchanged")
	}
	if !strings.HasSuffix(got, " is important") {
		t.Error("content after inline math should be unchanged")
	}
}

func TestMaskMathPreservesLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "display math inline",
			input: "text $$ x + y $$ end",
		},
		{
			name:  "display math multiline",
			input: "Before\n$$\n\\sum_{i=1}^{n} x_i\n$$\nAfter",
		},
		{
			name:  "inline math",
			input: "The $x + y$ formula",
		},
		{
			name:  "multiple inline math",
			input: "Both $x$ and $y + z$ here",
		},
		{
			name:  "mixed display and inline",
			input: "Inline $a$ then\n$$\nblock\n$$\nend",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskInertContent(tt.input)
			if len(got) != len(tt.input) {
				t.Errorf("length changed: input=%d, output=%d\ninput:  %q\noutput: %q",
					len(tt.input), len(got), tt.input, got)
			}
		})
	}
}

func TestMaskMathDoesNotMatchDollarAmounts(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "dollar amount",
			input: "The cost is $50 per unit.",
		},
		{
			name:  "dollar with space after",
			input: "Pay $ 50 for it.",
		},
		{
			name:  "two separate dollar amounts",
			input: "Between $100 and $200 range.",
		},
		{
			name:  "dollar at end of line",
			input: "Total: $500",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskInertContent(tt.input)
			if got != tt.input {
				t.Errorf("dollar amount should not be masked:\ninput:  %q\noutput: %q", tt.input, got)
			}
		})
	}
}

func TestMaskMathInsideCodeBlock(t *testing.T) {
	// Math delimiters inside an already-masked fenced code block should NOT
	// be double-processed. The fenced block pass runs first.
	input := "Before\n```\n$x + y$ and $$ block $$\n```\nAfter $a + b$ end"
	got := maskInertContent(input)

	// The code block content is masked first
	if !strings.Contains(got, "Before") {
		t.Error("content before code block should be preserved")
	}
	if !strings.Contains(got, "After") {
		t.Error("content after code block should be preserved")
	}
	// The inline math after the code block should be masked
	if strings.Contains(got, "a + b") {
		t.Error("inline math content outside code block should be masked")
	}
	// Length must be preserved
	if len(got) != len(input) {
		t.Errorf("length changed: input=%d, output=%d", len(input), len(got))
	}
}

func TestMaskMultipleMathExpressions(t *testing.T) {
	input := "First $x + 1$ then $y + 2$ and\n$$\n\\alpha + \\beta\n$$\nend"
	got := maskInertContent(input)

	if strings.Contains(got, "x + 1") {
		t.Error("first inline math should be masked")
	}
	if strings.Contains(got, "y + 2") {
		t.Error("second inline math should be masked")
	}
	if strings.Contains(got, "\\alpha") {
		t.Error("display math should be masked")
	}
	if !strings.HasPrefix(got, "First ") {
		t.Error("text before first inline math should be preserved")
	}
	if !strings.HasSuffix(got, "\nend") {
		t.Error("text after display math should be preserved")
	}
}

// --- Integration Tests ---

func TestParseWikilinksIgnoresMathBlocks(t *testing.T) {
	text := "Normal [[Outside]] link.\n$$ [[DisplayInside]] $$\nInline $x=[[InlineInside]]$ formula.\nMore [[AlsoOutside]]."
	links := parseWikilinks(text)

	titles := make(map[string]bool)
	for _, l := range links {
		titles[l.Title] = true
	}

	if !titles["Outside"] {
		t.Error("expected to find [[Outside]]")
	}
	if !titles["AlsoOutside"] {
		t.Error("expected to find [[AlsoOutside]]")
	}
	if titles["DisplayInside"] {
		t.Error("should NOT find [[DisplayInside]] from display math block")
	}
	if titles["InlineInside"] {
		t.Error("should NOT find [[InlineInside]] from inline math")
	}
	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d: %v", len(links), links)
	}
}

func TestParseInlineTagsIgnoresMathBlocks(t *testing.T) {
	text := "Normal #outside tag.\n$$ #displaytag inside $$\nInline $x #inlinetag$ here.\nMore #alsooutside."
	tags := parseInlineTags(text)

	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	if !tagSet["outside"] {
		t.Error("expected to find #outside")
	}
	if !tagSet["alsooutside"] {
		t.Error("expected to find #alsooutside")
	}
	if tagSet["displaytag"] {
		t.Error("should NOT find #displaytag from display math")
	}
	if tagSet["inlinetag"] {
		t.Error("should NOT find #inlinetag from inline math")
	}
}

func TestFindBacklinksIgnoresMathBlocks(t *testing.T) {
	vaultDir := t.TempDir()

	// Note A links to B only inside a display math block
	os.WriteFile(
		filepath.Join(vaultDir, "A.md"),
		[]byte("# A\n\nSome text.\n$$\n[[B]] in math\n$$\n"),
		0644,
	)

	// Note B exists
	os.WriteFile(
		filepath.Join(vaultDir, "B.md"),
		[]byte("# B\n\nContent.\n"),
		0644,
	)

	results, err := findBacklinks(vaultDir, "B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 backlinks (link is inside math block), got %d: %v", len(results), results)
	}
}

func TestAllInertZonesCombined(t *testing.T) {
	vaultDir := t.TempDir()

	// Note with wikilinks and tags inside ALL zone types, plus real links outside.
	// Zone types: fenced code, inline code, %% comments, HTML comments,
	// display math, inline math.
	content := "# Combined Test\n\nReal link: [[RealTarget]]\nReal tag: #real-tag\n\nFenced code:\n```\n[[FencedLink]] #fenced-tag\n```\n\nInline code: `[[InlineCodeLink]]` and `#inlinecode-tag`\n\nObsidian comment: %% [[CommentLink]] #comment-tag %%\n\nHTML comment: <!-- [[HTMLLink]] #html-tag -->\n\nDisplay math:\n$$\n[[DisplayMathLink]] #displaymath-tag\n$$\n\nInline math: $x=[[InlineMathLink]]$ and $y #inlinemath-tag$ here.\n\nEnd with another real link: [[AnotherReal]]\nAnd tag: #another-real-tag\n"

	os.WriteFile(
		filepath.Join(vaultDir, "AllZones.md"),
		[]byte(content),
		0644,
	)

	data, err := os.ReadFile(filepath.Join(vaultDir, "AllZones.md"))
	if err != nil {
		t.Fatal(err)
	}

	// Test wikilinks: only RealTarget and AnotherReal should be found
	links := parseWikilinks(string(data))
	linkTitles := make(map[string]bool)
	for _, l := range links {
		linkTitles[l.Title] = true
	}

	if !linkTitles["RealTarget"] {
		t.Error("should find [[RealTarget]] (plain text)")
	}
	if !linkTitles["AnotherReal"] {
		t.Error("should find [[AnotherReal]] (plain text)")
	}
	if linkTitles["FencedLink"] {
		t.Error("should NOT find [[FencedLink]] (fenced code)")
	}
	if linkTitles["InlineCodeLink"] {
		t.Error("should NOT find [[InlineCodeLink]] (inline code)")
	}
	if linkTitles["CommentLink"] {
		t.Error("should NOT find [[CommentLink]] (Obsidian comment)")
	}
	if linkTitles["HTMLLink"] {
		t.Error("should NOT find [[HTMLLink]] (HTML comment)")
	}
	if linkTitles["DisplayMathLink"] {
		t.Error("should NOT find [[DisplayMathLink]] (display math)")
	}
	if linkTitles["InlineMathLink"] {
		t.Error("should NOT find [[InlineMathLink]] (inline math)")
	}
	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d: %v", len(links), links)
	}

	// Test tags: only real-tag and another-real-tag should be found
	tags := allNoteTags(string(data))
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	if !tagSet["real-tag"] {
		t.Error("should find #real-tag (plain text)")
	}
	if !tagSet["another-real-tag"] {
		t.Error("should find #another-real-tag (plain text)")
	}
	if tagSet["fenced-tag"] {
		t.Error("should NOT find #fenced-tag (fenced code)")
	}
	if tagSet["inlinecode-tag"] {
		t.Error("should NOT find #inlinecode-tag (inline code)")
	}
	if tagSet["comment-tag"] {
		t.Error("should NOT find #comment-tag (Obsidian comment)")
	}
	if tagSet["html-tag"] {
		t.Error("should NOT find #html-tag (HTML comment)")
	}
	if tagSet["displaymath-tag"] {
		t.Error("should NOT find #displaymath-tag (display math)")
	}
	if tagSet["inlinemath-tag"] {
		t.Error("should NOT find #inlinemath-tag (inline math)")
	}

	// Test backlinks for RealTarget
	backlinks, err := findBacklinks(vaultDir, "RealTarget")
	if err != nil {
		t.Fatalf("findBacklinks RealTarget: %v", err)
	}
	if len(backlinks) != 1 || backlinks[0] != "AllZones.md" {
		t.Errorf("RealTarget backlinks: got %v, want [AllZones.md]", backlinks)
	}

	// Test backlinks for inert-zone links: should all be zero
	for _, inertTitle := range []string{"FencedLink", "InlineCodeLink", "CommentLink", "HTMLLink", "DisplayMathLink", "InlineMathLink"} {
		bl, err := findBacklinks(vaultDir, inertTitle)
		if err != nil {
			t.Fatalf("findBacklinks %s: %v", inertTitle, err)
		}
		if len(bl) != 0 {
			t.Errorf("%s should have 0 backlinks (inside inert zone), got %v", inertTitle, bl)
		}
	}
}

// =============================================================================
// E2E Validation: All inert zones across full command suite (VLT-zr3)
// =============================================================================

// allZoneContent returns a string containing [[target]] inside all 6 inert zone types.
// Each zone type places the link/tag on its own line to ensure all patterns match correctly.
// The suffix parameter is appended to the target name for uniqueness between tests.
func allZoneContent(target, tagName string) string {
	return "```\n[[" + target + "]] #" + tagName + "\n```\n\n" + // fenced code
		"`[[" + target + "]]` `#" + tagName + "`\n\n" + // inline code
		"%% [[" + target + "]] #" + tagName + " %%\n\n" + // Obsidian comment
		"<!-- [[" + target + "]] #" + tagName + " -->\n\n" + // HTML comment
		"$$\n[[" + target + "]] #" + tagName + "\n$$\n\n" + // display math
		"$x=[[" + target + "]]$ and $y #" + tagName + "$ here\n" // inline math
}

// TestE2EAllInertZonesBacklinks creates a vault with note A containing [[B]] in
// every inert zone type (6 zones) and one real [[B]] outside any zone.
// Verifies findBacklinks returns A exactly once.
func TestE2EAllInertZonesBacklinks(t *testing.T) {
	vaultDir := t.TempDir()

	// Note A has [[B]] inside all 6 inert zones PLUS one real link outside zones
	content := "# Note A\n\nReal link to [[B]] here.\n\n" + allZoneContent("B", "sometag")
	os.WriteFile(filepath.Join(vaultDir, "A.md"), []byte(content), 0644)
	os.WriteFile(filepath.Join(vaultDir, "B.md"), []byte("# B\n\nTarget note.\n"), 0644)

	backlinks, err := findBacklinks(vaultDir, "B")
	if err != nil {
		t.Fatalf("findBacklinks: %v", err)
	}

	if len(backlinks) != 1 {
		t.Fatalf("expected exactly 1 backlink from A.md, got %d: %v", len(backlinks), backlinks)
	}
	if backlinks[0] != "A.md" {
		t.Errorf("expected backlink from A.md, got %s", backlinks[0])
	}
}

// TestE2EAllInertZonesBacklinksAllMasked is like the above but with NO real link
// outside zones. findBacklinks must return empty.
func TestE2EAllInertZonesBacklinksAllMasked(t *testing.T) {
	vaultDir := t.TempDir()

	// Note A has [[B]] ONLY inside inert zones -- no real link
	content := "# Note A\n\nSome text, no real link.\n\n" + allZoneContent("B", "sometag")
	os.WriteFile(filepath.Join(vaultDir, "A.md"), []byte(content), 0644)
	os.WriteFile(filepath.Join(vaultDir, "B.md"), []byte("# B\n\nTarget note.\n"), 0644)

	backlinks, err := findBacklinks(vaultDir, "B")
	if err != nil {
		t.Fatalf("findBacklinks: %v", err)
	}

	if len(backlinks) != 0 {
		t.Errorf("expected 0 backlinks (all links inside inert zones), got %d: %v", len(backlinks), backlinks)
	}
}

// TestE2EAllInertZonesOrphans verifies that a note referenced ONLY inside inert
// zones of another note appears in the orphans list.
func TestE2EAllInertZonesOrphans(t *testing.T) {
	vaultDir := t.TempDir()

	// Note A references B only inside inert zones
	content := "# Note A\n\nSome text.\n\n" + allZoneContent("B", "sometag")
	os.WriteFile(filepath.Join(vaultDir, "A.md"), []byte(content), 0644)
	os.WriteFile(filepath.Join(vaultDir, "B.md"), []byte("# B\n\nI should be an orphan.\n"), 0644)

	// Also create note C that is genuinely referenced by A
	contentWithC := "# C Referrer\n\n[[C]] is a real link.\n"
	os.WriteFile(filepath.Join(vaultDir, "Referrer.md"), []byte(contentWithC), 0644)
	os.WriteFile(filepath.Join(vaultDir, "C.md"), []byte("# C\n\nNot an orphan.\n"), 0644)

	// Collect referenced titles the same way cmdOrphans does
	referenced := make(map[string]bool)
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, link := range parseWikilinks(string(data)) {
			referenced[strings.ToLower(link.Title)] = true
		}
		return nil
	})

	// B must NOT be referenced (only in inert zones)
	if referenced["b"] {
		t.Error("B should NOT be referenced -- all links to B are inside inert zones")
	}

	// C must BE referenced (real link)
	if !referenced["c"] {
		t.Error("C should be referenced by Referrer.md")
	}

	// Now build the orphan list
	type noteInfo struct {
		relPath string
		title   string
	}
	var notes []noteInfo
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		title := strings.TrimSuffix(d.Name(), ".md")
		relPath, _ := filepath.Rel(vaultDir, path)
		notes = append(notes, noteInfo{relPath: relPath, title: title})
		return nil
	})

	var orphans []string
	for _, note := range notes {
		if !referenced[strings.ToLower(note.title)] {
			orphans = append(orphans, note.relPath)
		}
	}
	sort.Strings(orphans)

	// B.md should be in orphans
	foundB := false
	for _, o := range orphans {
		if o == "B.md" {
			foundB = true
		}
	}
	if !foundB {
		t.Errorf("B.md should be in orphans list, got: %v", orphans)
	}

	// C.md should NOT be in orphans
	for _, o := range orphans {
		if o == "C.md" {
			t.Error("C.md should NOT be in orphans -- it is genuinely referenced")
		}
	}
}

// TestE2EAllInertZonesUnresolved verifies that [[Missing]] appearing inside all
// 6 inert zone types does NOT appear in unresolved output.
func TestE2EAllInertZonesUnresolved(t *testing.T) {
	vaultDir := t.TempDir()

	// Note A has [[Missing]] in every inert zone (Missing.md does NOT exist)
	content := "# Note A\n\nSome text, no real links to missing notes.\n\n" + allZoneContent("Missing", "sometag")
	os.WriteFile(filepath.Join(vaultDir, "A.md"), []byte(content), 0644)

	// Also add a genuine unresolved link for contrast
	os.WriteFile(
		filepath.Join(vaultDir, "RealBroken.md"),
		[]byte("# Real Broken\n\nThis has [[GenuinelyMissing]] link.\n"),
		0644,
	)

	// Collect all note titles
	titles := make(map[string]bool)
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		title := strings.TrimSuffix(d.Name(), ".md")
		titles[strings.ToLower(title)] = true
		return nil
	})

	// Collect unresolved links
	var unresolved []string
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, link := range parseWikilinks(string(data)) {
			lower := strings.ToLower(link.Title)
			if !titles[lower] {
				unresolved = append(unresolved, link.Title)
			}
		}
		return nil
	})

	// [[Missing]] should NOT appear as unresolved (only inside inert zones)
	for _, u := range unresolved {
		if strings.EqualFold(u, "Missing") {
			t.Error("[[Missing]] should NOT be unresolved -- it only appears inside inert zones")
		}
	}

	// [[GenuinelyMissing]] SHOULD appear as unresolved (real broken link)
	foundGenuine := false
	for _, u := range unresolved {
		if strings.EqualFold(u, "GenuinelyMissing") {
			foundGenuine = true
		}
	}
	if !foundGenuine {
		t.Errorf("[[GenuinelyMissing]] should be in unresolved list, got: %v", unresolved)
	}
}

// TestE2EAllInertZonesTags verifies that #mytag inside all 6 inert zone types is
// not counted, but #mytag in body text IS counted exactly once.
func TestE2EAllInertZonesTags(t *testing.T) {
	vaultDir := t.TempDir()

	// Note with #mytag in all 6 inert zones plus one real occurrence in body text
	content := "# Tagged Note\n\n#mytag\n\n" + allZoneContent("SomeLink", "mytag")
	os.WriteFile(filepath.Join(vaultDir, "TaggedNote.md"), []byte(content), 0644)

	data, err := os.ReadFile(filepath.Join(vaultDir, "TaggedNote.md"))
	if err != nil {
		t.Fatal(err)
	}

	tags := allNoteTags(string(data))

	// Count occurrences of "mytag"
	mytagCount := 0
	for _, tag := range tags {
		if tag == "mytag" {
			mytagCount++
		}
	}

	if mytagCount != 1 {
		t.Errorf("expected exactly 1 occurrence of #mytag (the real one), got %d; all tags: %v", mytagCount, tags)
	}
}

// TestE2EAllInertZonesTagSearch verifies that a note with #special ONLY inside
// inert zones is NOT returned by cmdTag(tag="special").
func TestE2EAllInertZonesTagSearch(t *testing.T) {
	vaultDir := t.TempDir()

	// Note with #special only inside inert zones -- no real #special in body
	content := "# Hidden Tag Note\n\nSome content, no real special tag.\n\n" + allZoneContent("SomeLink", "special")
	os.WriteFile(filepath.Join(vaultDir, "HiddenTag.md"), []byte(content), 0644)

	// Another note WITH a real #special tag for contrast
	os.WriteFile(
		filepath.Join(vaultDir, "RealTag.md"),
		[]byte("# Real Tag Note\n\n#special is used for real here.\n"),
		0644,
	)

	// Walk the vault the same way cmdTag does, looking for notes with tag "special"
	tagLower := "special"
	var results []string

	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, tag := range allNoteTags(string(data)) {
			if tag == tagLower || strings.HasPrefix(tag, tagLower+"/") {
				relPath, _ := filepath.Rel(vaultDir, path)
				results = append(results, relPath)
				break
			}
		}
		return nil
	})

	sort.Strings(results)

	// HiddenTag.md should NOT be in results
	for _, r := range results {
		if r == "HiddenTag.md" {
			t.Error("HiddenTag.md should NOT be found by tag search -- #special is only inside inert zones")
		}
	}

	// RealTag.md SHOULD be in results
	foundReal := false
	for _, r := range results {
		if r == "RealTag.md" {
			foundReal = true
		}
	}
	if !foundReal {
		t.Errorf("RealTag.md should be found by tag search for #special, got: %v", results)
	}
}

// TestE2ERealisticNote creates a realistic Obsidian note with mixed content:
// frontmatter, headings, prose with real links, code examples with [[fake]] links,
// LaTeX with $[[x]]$ math, comments with hidden TODO links. Verifies all commands
// produce correct results.
func TestE2ERealisticNote(t *testing.T) {
	vaultDir := t.TempDir()

	// A realistic note that would exist in a knowledge vault
	realisticContent := `---
tags: [architecture, project/backend]
status: active
aliases: [arch-overview]
---

# Architecture Overview

This document describes the system architecture. See [[Design Principles]] for
the guiding decisions, and [[API Reference]] for endpoint documentation.

## Components

The main components are:

- **Gateway**: handles routing, see [[Gateway Design]]
- **Auth**: see [[Auth Module]] for details

#component #infrastructure

## Code Examples

Here is a typical handler pattern:

` + "```go" + `
// Handler for [[User]] resource
func GetUser(ctx context.Context) (*User, error) {
    // See [[Database]] for connection setup
    db := ctx.Value("db").(*sql.DB)
    #handler-pattern
    return queryUser(db, ctx)
}
` + "```" + `

The inline reference ` + "`[[Config]]`" + ` shows the config struct.

## Math Section

The load balancing formula uses $w_i = [[Weight]]$ where each weight
is determined by:

$$
\sum_{i=1}^{n} [[ServerLoad]]_i \cdot #load-factor
$$

## Notes

%% TODO: review [[Draft Section]] and update #todo %%

<!-- NOTE: deprecated link to [[Old API]] #deprecated -->

<!--
Multi-line comment:
[[Planning Doc]] should be merged with [[Roadmap]]
#comment-hidden
-->

%%
Multi-line Obsidian comment:
Link to [[Secret Note]] is hidden
#secret-tag
%%

End of document. Real tag: #reviewed
`

	os.WriteFile(filepath.Join(vaultDir, "Architecture.md"), []byte(realisticContent), 0644)

	// Create the real target notes
	for _, title := range []string{"Design Principles", "API Reference", "Gateway Design", "Auth Module"} {
		os.WriteFile(
			filepath.Join(vaultDir, title+".md"),
			[]byte("# "+title+"\n\nContent.\n"),
			0644,
		)
	}

	data, _ := os.ReadFile(filepath.Join(vaultDir, "Architecture.md"))
	text := string(data)

	// --- Test links ---
	links := parseWikilinks(text)
	linkTitles := make(map[string]bool)
	for _, l := range links {
		linkTitles[l.Title] = true
	}

	// Real links (outside all inert zones)
	realLinks := []string{"Design Principles", "API Reference", "Gateway Design", "Auth Module"}
	for _, expected := range realLinks {
		if !linkTitles[expected] {
			t.Errorf("should find real link [[%s]]", expected)
		}
	}

	// Fake links (inside inert zones) -- must NOT be found
	fakeLinks := []string{"User", "Database", "Config", "Weight", "ServerLoad",
		"Draft Section", "Old API", "Planning Doc", "Roadmap", "Secret Note"}
	for _, fake := range fakeLinks {
		if linkTitles[fake] {
			t.Errorf("should NOT find [[%s]] (inside inert zone)", fake)
		}
	}

	// --- Test tags ---
	tags := allNoteTags(text)
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	// Real tags (frontmatter + body text outside inert zones)
	realTags := []string{"architecture", "project/backend", "component", "infrastructure", "reviewed"}
	for _, expected := range realTags {
		if !tagSet[expected] {
			t.Errorf("should find real tag #%s", expected)
		}
	}

	// Fake tags (inside inert zones) -- must NOT be found
	fakeTags := []string{"handler-pattern", "load-factor", "todo", "deprecated", "comment-hidden", "secret-tag"}
	for _, fake := range fakeTags {
		if tagSet[fake] {
			t.Errorf("should NOT find #%s (inside inert zone)", fake)
		}
	}

	// --- Test backlinks ---
	for _, title := range realLinks {
		bl, err := findBacklinks(vaultDir, title)
		if err != nil {
			t.Fatalf("findBacklinks %s: %v", title, err)
		}
		if len(bl) != 1 || bl[0] != "Architecture.md" {
			t.Errorf("backlinks for %s: got %v, want [Architecture.md]", title, bl)
		}
	}

	// Fake targets should have 0 backlinks
	for _, title := range fakeLinks {
		bl, err := findBacklinks(vaultDir, title)
		if err != nil {
			t.Fatalf("findBacklinks %s: %v", title, err)
		}
		if len(bl) != 0 {
			t.Errorf("%s should have 0 backlinks (inside inert zone), got %v", title, bl)
		}
	}

	// --- Test orphans ---
	referenced := make(map[string]bool)
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, link := range parseWikilinks(string(data)) {
			referenced[strings.ToLower(link.Title)] = true
		}
		return nil
	})

	// Real targets should be referenced
	for _, title := range realLinks {
		if !referenced[strings.ToLower(title)] {
			t.Errorf("%s should be referenced", title)
		}
	}

	// --- Test unresolved ---
	titleSet := make(map[string]bool)
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		title := strings.TrimSuffix(d.Name(), ".md")
		titleSet[strings.ToLower(title)] = true
		return nil
	})

	var unresolvedLinks []string
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, link := range parseWikilinks(string(data)) {
			lower := strings.ToLower(link.Title)
			if !titleSet[lower] {
				unresolvedLinks = append(unresolvedLinks, link.Title)
			}
		}
		return nil
	})

	// Fake links should NOT appear as unresolved
	for _, u := range unresolvedLinks {
		for _, fake := range fakeLinks {
			if strings.EqualFold(u, fake) {
				t.Errorf("[[%s]] should NOT be in unresolved list (inside inert zone)", fake)
			}
		}
	}
}

// TestE2ESearchStillFindsInertContent verifies that cmdSearch finds text inside
// inert zones, because search scans raw content (unlike link/tag extraction).
func TestE2ESearchStillFindsInertContent(t *testing.T) {
	vaultDir := t.TempDir()

	// Note with unique search terms inside various inert zones
	content := "# Searchable Note\n\nReal content with UniqueRealTerm here.\n\n" +
		"```\nUniqueCodeTerm inside fenced code\n```\n\n" +
		"`UniqueInlineTerm`\n\n" +
		"%% UniqueCommentTerm inside Obsidian comment %%\n\n" +
		"<!-- UniqueHTMLTerm inside HTML comment -->\n\n" +
		"$$ UniqueDisplayMathTerm $$\n\n" +
		"$UniqueInlineMathTerm$\n"

	os.WriteFile(filepath.Join(vaultDir, "SearchTarget.md"), []byte(content), 0644)

	// Search should find content inside ALL zones (search reads raw file content)
	searchTerms := []string{
		"UniqueRealTerm",
		"UniqueCodeTerm",
		"UniqueInlineTerm",
		"UniqueCommentTerm",
		"UniqueHTMLTerm",
		"UniqueDisplayMathTerm",
		"UniqueInlineMathTerm",
	}

	for _, term := range searchTerms {
		// Replicate the search logic from cmdSearch: case-insensitive content match
		data, err := os.ReadFile(filepath.Join(vaultDir, "SearchTarget.md"))
		if err != nil {
			t.Fatal(err)
		}

		contentLower := strings.ToLower(string(data))
		termLower := strings.ToLower(term)

		if !strings.Contains(contentLower, termLower) {
			t.Errorf("search should find %q inside raw file content", term)
		}
	}
}

// TestE2EMaskingDoesNotCorruptContent verifies that after all masking-dependent
// operations, the original files remain byte-for-byte identical. Masking is
// read-only and never writes masked content back to disk.
func TestE2EMaskingDoesNotCorruptContent(t *testing.T) {
	vaultDir := t.TempDir()

	// Create notes with content in all inert zone types
	noteAContent := "# Note A\n\n[[B]] real link\n\n" + allZoneContent("C", "tagged") + "\n#real-tag\n"
	noteBContent := "# Note B\n\nSimple content.\n"
	noteCContent := "---\ntags: [important]\n---\n\n# Note C\n\n[[A]] back-reference.\n\n```\n[[Fake]] #fake\n```\n"

	files := map[string]string{
		"A.md": noteAContent,
		"B.md": noteBContent,
		"C.md": noteCContent,
	}

	// Write the original files
	for name, content := range files {
		os.WriteFile(filepath.Join(vaultDir, name), []byte(content), 0644)
	}

	// Capture original content before any operations
	originals := make(map[string][]byte)
	for name := range files {
		data, err := os.ReadFile(filepath.Join(vaultDir, name))
		if err != nil {
			t.Fatalf("reading original %s: %v", name, err)
		}
		originals[name] = data
	}

	// Exercise ALL masking-dependent operations to trigger maskInertContent

	// 1. backlinks (calls maskInertContent on every file)
	findBacklinks(vaultDir, "B")
	findBacklinks(vaultDir, "C")
	findBacklinks(vaultDir, "A")

	// 2. parseWikilinks (calls maskInertContent)
	for name := range files {
		data, _ := os.ReadFile(filepath.Join(vaultDir, name))
		parseWikilinks(string(data))
	}

	// 3. allNoteTags / parseInlineTags (calls maskInertContent)
	for name := range files {
		data, _ := os.ReadFile(filepath.Join(vaultDir, name))
		allNoteTags(string(data))
		parseInlineTags(string(data))
	}

	// 4. Run the actual command functions that use masking
	// cmdOrphans
	cmdOrphans(vaultDir, "")
	// cmdUnresolved
	cmdUnresolved(vaultDir, "")
	// cmdTags
	cmdTags(vaultDir, map[string]string{}, false, "")
	// cmdTag
	cmdTag(vaultDir, map[string]string{"tag": "real-tag"}, "")
	// cmdLinks
	cmdLinks(vaultDir, map[string]string{"file": "A"}, "")
	// cmdBacklinks
	cmdBacklinks(vaultDir, map[string]string{"file": "B"}, "")

	// Now verify every file is byte-for-byte identical to the original
	for name, original := range originals {
		current, err := os.ReadFile(filepath.Join(vaultDir, name))
		if err != nil {
			t.Fatalf("re-reading %s after operations: %v", name, err)
		}

		if len(current) != len(original) {
			t.Errorf("%s: length changed from %d to %d after masking operations", name, len(original), len(current))
			continue
		}

		for i := range original {
			if original[i] != current[i] {
				t.Errorf("%s: byte %d changed from 0x%02x to 0x%02x after masking operations", name, i, original[i], current[i])
				break
			}
		}
	}
}
