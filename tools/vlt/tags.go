package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// tagPattern matches inline tags: #tag preceded by whitespace or start of line.
// Tags contain Unicode letters, digits, underscores, hyphens, and forward
// slashes (for hierarchical tags like #project/backend).
var tagPattern = regexp.MustCompile(`(?:^|[\s(])#([\p{L}\p{N}_/-]+)`)

// parseInlineTags extracts inline #tags from text.
// Skips pure-numeric tags (Obsidian requires at least one letter).
func parseInlineTags(text string) []string {
	matches := tagPattern.FindAllStringSubmatch(text, -1)
	var tags []string
	for _, m := range matches {
		tag := m[1]
		if hasLetter(tag) {
			tags = append(tags, tag)
		}
	}
	return tags
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// allNoteTags returns all tags from a note (inline body + frontmatter),
// lowercased and deduplicated.
func allNoteTags(text string) []string {
	seen := make(map[string]bool)
	var result []string

	yaml, bodyStart, hasFM := extractFrontmatter(text)
	if hasFM {
		for _, t := range frontmatterGetList(yaml, "tags") {
			lower := strings.ToLower(t)
			if !seen[lower] {
				seen[lower] = true
				result = append(result, lower)
			}
		}
	}

	// Parse inline tags from body only (skip frontmatter)
	body := text
	if hasFM {
		lines := strings.Split(text, "\n")
		if bodyStart < len(lines) {
			body = strings.Join(lines[bodyStart:], "\n")
		}
	}

	for _, t := range parseInlineTags(body) {
		lower := strings.ToLower(t)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, lower)
		}
	}

	return result
}

// cmdTags lists all tags in the vault. With showCounts, includes note counts.
// Supports sort="count" to sort by frequency (default: alphabetical).
func cmdTags(vaultDir string, params map[string]string, showCounts bool) error {
	tagCounts := make(map[string]int)
	sortBy := params["sort"]

	err := filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() && (strings.HasPrefix(name, ".") || name == ".trash") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(name, ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		for _, tag := range allNoteTags(string(data)) {
			tagCounts[tag]++
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(tagCounts) == 0 {
		return nil
	}

	tags := make([]string, 0, len(tagCounts))
	for t := range tagCounts {
		tags = append(tags, t)
	}

	if sortBy == "count" {
		sort.Slice(tags, func(i, j int) bool {
			if tagCounts[tags[i]] != tagCounts[tags[j]] {
				return tagCounts[tags[i]] > tagCounts[tags[j]]
			}
			return tags[i] < tags[j]
		})
	} else {
		sort.Strings(tags)
	}

	for _, tag := range tags {
		if showCounts {
			fmt.Printf("#%s\t%d\n", tag, tagCounts[tag])
		} else {
			fmt.Printf("#%s\n", tag)
		}
	}
	return nil
}

// cmdTag finds notes that have a specific tag or any subtag of it.
// Matches case-insensitively, consistent with Obsidian.
func cmdTag(vaultDir string, params map[string]string) error {
	tag := params["tag"]
	if tag == "" {
		return fmt.Errorf("tag requires tag=\"<tagname>\"")
	}

	tag = strings.TrimPrefix(tag, "#")
	tagLower := strings.ToLower(tag)

	var results []string

	err := filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() && (strings.HasPrefix(name, ".") || name == ".trash") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(name, ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		for _, t := range allNoteTags(string(data)) {
			if t == tagLower || strings.HasPrefix(t, tagLower+"/") {
				relPath, _ := filepath.Rel(vaultDir, path)
				results = append(results, relPath)
				break
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	sort.Strings(results)
	for _, r := range results {
		fmt.Println(r)
	}
	return nil
}
