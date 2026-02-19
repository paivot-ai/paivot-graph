package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// wikilink represents a parsed [[...]] or ![[...]] reference in a note.
type wikilink struct {
	Title   string // note title (e.g., "Session Operating Mode")
	Heading string // optional heading without # (e.g., "Section")
	Display string // optional display text without | (e.g., "alias")
	Embed   bool   // true if ![[...]] (transclusion)
	Raw     string // original matched text including [[ ]]
}

// wikiLinkPattern matches wikilinks and embeds: [[Title]], ![[Title]],
// [[Title#Heading]], [[Title|Display]], [[Title#Heading|Display]].
var wikiLinkPattern = regexp.MustCompile(`(!?)\[\[([^\]#|]+?)(?:#([^\]|]*))?(?:\|([^\]]*))?\]\]`)

// parseWikilinks extracts all wikilinks and embeds from text.
func parseWikilinks(text string) []wikilink {
	matches := wikiLinkPattern.FindAllStringSubmatch(text, -1)
	links := make([]wikilink, 0, len(matches))
	for _, m := range matches {
		wl := wikilink{
			Embed: m[1] == "!",
			Title: strings.TrimSpace(m[2]),
			Raw:   m[0],
		}
		if len(m) > 3 {
			wl.Heading = m[3]
		}
		if len(m) > 4 {
			wl.Display = m[4]
		}
		links = append(links, wl)
	}
	return links
}

// replaceWikilinks replaces all wikilinks and embeds referencing oldTitle
// with newTitle, preserving the !prefix, #heading, and |display text.
// Case-insensitive to match Obsidian's link resolution behavior.
func replaceWikilinks(text, oldTitle, newTitle string) string {
	pattern := regexp.MustCompile(
		`(?i)(!?)\[\[` + regexp.QuoteMeta(oldTitle) +
			`((?:#[^\]|]*)?)` +
			`((?:\|[^\]]*)?)` +
			`\]\]`)
	return pattern.ReplaceAllString(text, `${1}[[`+newTitle+`${2}${3}]]`)
}

// updateVaultLinks scans all .md files in vaultDir and replaces wikilinks
// from oldTitle to newTitle. Returns the number of files modified.
func updateVaultLinks(vaultDir, oldTitle, newTitle string) (int, error) {
	modified := 0

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

		text := string(data)
		updated := replaceWikilinks(text, oldTitle, newTitle)
		if updated != text {
			if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
				return fmt.Errorf("failed to update %s: %w", path, err)
			}
			modified++
		}
		return nil
	})

	return modified, err
}

// findBacklinks returns relative paths of notes that contain wikilinks or
// embeds referencing the given title. Case-insensitive.
func findBacklinks(vaultDir, title string) ([]string, error) {
	pattern := regexp.MustCompile(
		`(?i)!?\[\[` + regexp.QuoteMeta(title) +
			`(?:#[^\]|]*)?(?:\|[^\]]*)?\]\]`)

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

		if pattern.Match(data) {
			relPath, _ := filepath.Rel(vaultDir, path)
			results = append(results, relPath)
		}
		return nil
	})

	return results, err
}
