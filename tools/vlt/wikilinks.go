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
	BlockID string // optional block ID without ^ (e.g., "my-block")
	Display string // optional display text without | (e.g., "alias")
	Embed   bool   // true if ![[...]] (transclusion)
	Raw     string // original matched text including [[ ]]
}

// wikiLinkPattern matches wikilinks and embeds: [[Title]], ![[Title]],
// [[Title#Heading]], [[Title#^block-id]], [[Title|Display]],
// [[Title#Heading|Display]], [[Title#^block-id|Display]].
var wikiLinkPattern = regexp.MustCompile(`(!?)\[\[([^\]#|]+?)(?:#(\^?[^\]|]*))?(?:\|([^\]]*))?\]\]`)

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
		if len(m) > 3 && m[3] != "" {
			fragment := m[3]
			if strings.HasPrefix(fragment, "^") {
				wl.BlockID = fragment[1:] // strip the ^ prefix
			} else {
				wl.Heading = fragment
			}
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

// mdLinkPattern matches markdown-style links to .md files: [text](path.md) or [text](path.md#heading)
var mdLinkPattern = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+\.md(?:#[^)]*)?)\)`)

// updateVaultMdLinks scans all .md files in the vault and updates
// markdown-style [text](path.md) links when a file is moved/renamed.
// oldRelPath and newRelPath are vault-relative paths.
// Returns the number of files modified.
func updateVaultMdLinks(vaultDir, oldRelPath, newRelPath string) (int, error) {
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
		fileDir, _ := filepath.Rel(vaultDir, filepath.Dir(path))

		updated := mdLinkPattern.ReplaceAllStringFunc(text, func(match string) string {
			sub := mdLinkPattern.FindStringSubmatch(match)
			if len(sub) < 3 {
				return match
			}

			linkText := sub[1]
			linkTarget := sub[2]

			// Split off fragment (#heading)
			fragment := ""
			if idx := strings.Index(linkTarget, "#"); idx >= 0 {
				fragment = linkTarget[idx:]
				linkTarget = linkTarget[:idx]
			}

			// Resolve the link target relative to the file containing it
			var resolvedTarget string
			if filepath.IsAbs(linkTarget) {
				return match // absolute paths: leave alone
			}
			resolvedTarget = filepath.Join(fileDir, linkTarget)
			resolvedTarget = filepath.Clean(resolvedTarget)

			// Check if this link points to the old path
			if resolvedTarget != filepath.Clean(oldRelPath) {
				return match
			}

			// Compute the new relative path from the referencing file to the new location
			newTarget, err := filepath.Rel(fileDir, newRelPath)
			if err != nil {
				return match
			}
			// filepath.Rel may produce paths without ./ prefix; keep them clean
			newTarget = filepath.Clean(newTarget)

			return "[" + linkText + "](" + newTarget + fragment + ")"
		})

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
