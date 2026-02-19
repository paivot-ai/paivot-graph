package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)


// cmdVaults lists all Obsidian vaults discovered from the config file.
func cmdVaults() error {
	vaults, err := discoverVaults()
	if err != nil {
		return err
	}

	if len(vaults) == 0 {
		fmt.Println("No vaults found.")
		return nil
	}

	// Sort by name for stable output
	names := make([]string, 0, len(vaults))
	for name := range vaults {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Printf("%s\t%s\n", name, vaults[name])
	}
	return nil
}

// cmdRead prints the contents of a note resolved by title.
func cmdRead(vaultDir string, params map[string]string) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("read requires file=\"<title>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fmt.Print(string(data))
	return nil
}

// cmdSearch finds notes whose title or content matches the query (case-insensitive).
func cmdSearch(vaultDir string, params map[string]string) error {
	query := params["query"]
	if query == "" {
		return fmt.Errorf("search requires query=\"<term>\"")
	}

	queryLower := strings.ToLower(query)
	pathFilter := params["path"] // optional: limit to a subdirectory

	searchRoot := vaultDir
	if pathFilter != "" {
		searchRoot = filepath.Join(vaultDir, pathFilter)
		if _, err := os.Stat(searchRoot); os.IsNotExist(err) {
			return fmt.Errorf("path filter %q not found in vault", pathFilter)
		}
	}

	type result struct {
		title   string
		relPath string
	}
	var results []result

	err := filepath.WalkDir(searchRoot, func(path string, d os.DirEntry, err error) error {
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

		title := strings.TrimSuffix(name, ".md")
		relPath, _ := filepath.Rel(vaultDir, path)

		// Check title first (cheap)
		if strings.Contains(strings.ToLower(title), queryLower) {
			results = append(results, result{title, relPath})
			return nil
		}

		// Check content (needs I/O)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(strings.ToLower(string(data)), queryLower) {
			results = append(results, result{title, relPath})
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(results) == 0 {
		return nil // silent on no results, matching grep convention
	}

	for _, r := range results {
		fmt.Printf("%s (%s)\n", r.title, r.relPath)
	}
	return nil
}

// cmdCreate creates a new note at the given path within the vault.
// Content comes from the content= parameter or stdin.
func cmdCreate(vaultDir string, params map[string]string, silent bool) error {
	name := params["name"]
	notePath := params["path"]

	if name == "" || notePath == "" {
		return fmt.Errorf("create requires name=\"<title>\" path=\"<relative-path>\"")
	}

	fullPath := filepath.Join(vaultDir, notePath)

	// Don't overwrite existing notes
	if _, err := os.Stat(fullPath); err == nil {
		if !silent {
			fmt.Fprintf(os.Stderr, "note already exists: %s\n", notePath)
		}
		return nil
	}

	content := params["content"]
	if content == "" {
		content = readStdinIfPiped()
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return err
	}

	if !silent {
		fmt.Printf("created: %s\n", notePath)
	}
	return nil
}

// cmdAppend adds content to the end of an existing note.
// Content comes from the content= parameter or stdin.
func cmdAppend(vaultDir string, params map[string]string) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("append requires file=\"<title>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	content := params["content"]
	if content == "" {
		content = readStdinIfPiped()
	}
	if content == "" {
		return fmt.Errorf("no content provided (use content=\"...\" or pipe to stdin)")
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprint(f, content)
	return err
}

// cmdMove moves a note from one path to another within the vault.
// If the filename changes (rename, not just folder move), all wikilinks
// referencing the old title are updated vault-wide.
func cmdMove(vaultDir string, params map[string]string) error {
	from := params["path"]
	to := params["to"]

	if from == "" || to == "" {
		return fmt.Errorf("move requires path=\"<from>\" to=\"<to>\"")
	}

	fromPath := filepath.Join(vaultDir, from)
	toPath := filepath.Join(vaultDir, to)

	if _, err := os.Stat(fromPath); os.IsNotExist(err) {
		return fmt.Errorf("source not found: %s", from)
	}

	if err := os.MkdirAll(filepath.Dir(toPath), 0755); err != nil {
		return err
	}

	oldTitle := strings.TrimSuffix(filepath.Base(from), ".md")
	newTitle := strings.TrimSuffix(filepath.Base(to), ".md")

	if err := os.Rename(fromPath, toPath); err != nil {
		return err
	}

	fmt.Printf("moved: %s -> %s\n", from, to)

	// If the filename changed, update wikilinks across the vault
	if oldTitle != newTitle {
		count, err := updateVaultLinks(vaultDir, oldTitle, newTitle)
		if err != nil {
			return fmt.Errorf("moved file but failed updating links: %w", err)
		}
		if count > 0 {
			fmt.Printf("updated [[%s]] -> [[%s]] in %d file(s)\n", oldTitle, newTitle, count)
		}
	}

	return nil
}

// cmdBacklinks finds all notes that contain wikilinks to the given title.
func cmdBacklinks(vaultDir string, params map[string]string) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("backlinks requires file=\"<title>\"")
	}

	results, err := findBacklinks(vaultDir, title)
	if err != nil {
		return err
	}

	for _, r := range results {
		fmt.Println(r)
	}
	return nil
}

// cmdLinks lists outgoing wikilinks from a note, reporting which resolve
// and which are broken.
func cmdLinks(vaultDir string, params map[string]string) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("links requires file=\"<title>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	links := parseWikilinks(string(data))
	if len(links) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	for _, link := range links {
		if seen[link.Title] {
			continue
		}
		seen[link.Title] = true

		resolved, resolveErr := resolveNote(vaultDir, link.Title)
		if resolveErr != nil {
			fmt.Printf("  BROKEN: [[%s]]\n", link.Title)
		} else {
			relPath, _ := filepath.Rel(vaultDir, resolved)
			fmt.Printf("  [[%s]] -> %s\n", link.Title, relPath)
		}
	}
	return nil
}

// cmdPropertySet sets or adds a YAML frontmatter property in a note.
func cmdPropertySet(vaultDir string, params map[string]string) error {
	title := params["file"]
	propName := params["name"]
	propValue := params["value"]

	if title == "" || propName == "" {
		return fmt.Errorf("property:set requires file=\"<title>\" name=\"<key>\" value=\"<val>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	// Find frontmatter boundaries (--- ... ---)
	fmStart, fmEnd := -1, -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if fmStart == -1 {
				fmStart = i
			} else {
				fmEnd = i
				break
			}
		}
	}

	if fmStart == -1 || fmEnd == -1 {
		return fmt.Errorf("no frontmatter found in %q", title)
	}

	// Look for existing property line
	found := false
	prefix := propName + ":"
	for i := fmStart + 1; i < fmEnd; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, prefix) {
			lines[i] = fmt.Sprintf("%s: %s", propName, propValue)
			found = true
			break
		}
	}

	// If not found, insert before closing ---
	if !found {
		newLine := fmt.Sprintf("%s: %s", propName, propValue)
		// Insert newLine at position fmEnd
		lines = append(lines[:fmEnd+1], lines[fmEnd:]...)
		lines[fmEnd] = newLine
	}

	result := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(result), 0644); err != nil {
		return err
	}

	fmt.Printf("set %s=%s in %q\n", propName, propValue, title)
	return nil
}

// cmdPrepend inserts content at the top of a note, after frontmatter if present.
func cmdPrepend(vaultDir string, params map[string]string) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("prepend requires file=\"<title>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	content := params["content"]
	if content == "" {
		content = readStdinIfPiped()
	}
	if content == "" {
		return fmt.Errorf("no content provided (use content=\"...\" or pipe to stdin)")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	text := string(data)
	_, bodyStart, hasFM := extractFrontmatter(text)

	lines := strings.Split(text, "\n")
	var result string

	if hasFM && bodyStart <= len(lines) {
		before := strings.Join(lines[:bodyStart], "\n")
		after := strings.Join(lines[bodyStart:], "\n")
		result = before + "\n" + content + after
	} else {
		result = content + text
	}

	return os.WriteFile(path, []byte(result), 0644)
}

// cmdDelete moves a note to .trash/ (or permanently deletes with the permanent flag).
func cmdDelete(vaultDir string, params map[string]string, permanent bool) error {
	title := params["file"]
	notePath := params["path"]

	var fullPath string
	if notePath != "" {
		fullPath = filepath.Join(vaultDir, notePath)
	} else if title != "" {
		resolved, err := resolveNote(vaultDir, title)
		if err != nil {
			return err
		}
		fullPath = resolved
	} else {
		return fmt.Errorf("delete requires file=\"<title>\" or path=\"<path>\"")
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", fullPath)
	}

	relPath, _ := filepath.Rel(vaultDir, fullPath)

	if permanent {
		if err := os.Remove(fullPath); err != nil {
			return err
		}
		fmt.Printf("deleted: %s\n", relPath)
	} else {
		trashDir := filepath.Join(vaultDir, ".trash")
		if err := os.MkdirAll(trashDir, 0755); err != nil {
			return err
		}
		trashPath := filepath.Join(trashDir, filepath.Base(fullPath))
		if err := os.Rename(fullPath, trashPath); err != nil {
			return err
		}
		fmt.Printf("trashed: %s -> .trash/%s\n", relPath, filepath.Base(fullPath))
	}

	return nil
}

// cmdProperties prints the YAML frontmatter block of a note (with --- delimiters).
func cmdProperties(vaultDir string, params map[string]string) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("properties requires file=\"<title>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fm := frontmatterReadAll(string(data))
	if fm == "" {
		return nil
	}

	fmt.Println(fm)
	return nil
}

// cmdPropertyRemove removes a property from a note's frontmatter.
func cmdPropertyRemove(vaultDir string, params map[string]string) error {
	title := params["file"]
	propName := params["name"]

	if title == "" || propName == "" {
		return fmt.Errorf("property:remove requires file=\"<title>\" name=\"<key>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	text := string(data)
	updated := frontmatterRemoveKey(text, propName)

	if updated == text {
		return fmt.Errorf("property %q not found in %q", propName, title)
	}

	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return err
	}

	fmt.Printf("removed %s from %q\n", propName, title)
	return nil
}

// cmdOrphans finds notes that have no incoming wikilinks or embeds.
func cmdOrphans(vaultDir string) error {
	// Collect all note titles
	type noteInfo struct {
		relPath string
		title   string
		aliases []string
	}
	var notes []noteInfo

	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
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

		title := strings.TrimSuffix(name, ".md")
		relPath, _ := filepath.Rel(vaultDir, path)

		info := noteInfo{relPath: relPath, title: title}

		data, err := os.ReadFile(path)
		if err == nil {
			yaml, _, hasFM := extractFrontmatter(string(data))
			if hasFM {
				info.aliases = frontmatterGetList(yaml, "aliases")
			}
		}

		notes = append(notes, info)
		return nil
	})

	// Collect all referenced titles (from wikilinks and embeds)
	referenced := make(map[string]bool)

	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
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

		for _, link := range parseWikilinks(string(data)) {
			referenced[strings.ToLower(link.Title)] = true
		}
		return nil
	})

	// Find orphans: notes whose title AND aliases are all unreferenced
	var orphans []string
	for _, note := range notes {
		if referenced[strings.ToLower(note.title)] {
			continue
		}
		aliasReferenced := false
		for _, a := range note.aliases {
			if referenced[strings.ToLower(a)] {
				aliasReferenced = true
				break
			}
		}
		if !aliasReferenced {
			orphans = append(orphans, note.relPath)
		}
	}

	sort.Strings(orphans)
	for _, o := range orphans {
		fmt.Println(o)
	}
	return nil
}

// cmdUnresolved finds all broken wikilinks across the vault.
func cmdUnresolved(vaultDir string) error {
	// Build sets of resolvable titles and aliases
	titles := make(map[string]bool)
	aliases := make(map[string]bool)

	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
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

		title := strings.TrimSuffix(name, ".md")
		titles[strings.ToLower(title)] = true

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		yaml, _, hasFM := extractFrontmatter(string(data))
		if hasFM {
			for _, alias := range frontmatterGetList(yaml, "aliases") {
				aliases[strings.ToLower(alias)] = true
			}
		}
		return nil
	})

	// Find links that don't resolve
	type unresolvedLink struct {
		source string
		target string
	}
	var results []unresolvedLink
	seenTargets := make(map[string]bool)

	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
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

		relPath, _ := filepath.Rel(vaultDir, path)

		for _, link := range parseWikilinks(string(data)) {
			lower := strings.ToLower(link.Title)
			if seenTargets[lower] {
				continue
			}
			if !titles[lower] && !aliases[lower] {
				seenTargets[lower] = true
				results = append(results, unresolvedLink{relPath, link.Title})
			}
		}
		return nil
	})

	for _, r := range results {
		fmt.Printf("[[%s]] in %s\n", r.target, r.source)
	}
	return nil
}

// cmdFiles lists files in the vault, optionally filtered by folder and extension.
func cmdFiles(vaultDir string, params map[string]string, showTotal bool) error {
	folder := params["folder"]
	ext := params["ext"]
	if ext == "" {
		ext = "md"
	}

	searchRoot := vaultDir
	if folder != "" {
		searchRoot = filepath.Join(vaultDir, folder)
		if _, err := os.Stat(searchRoot); os.IsNotExist(err) {
			return fmt.Errorf("folder not found: %s", folder)
		}
	}

	var files []string

	filepath.WalkDir(searchRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() && (strings.HasPrefix(name, ".") || name == ".trash") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(name, "."+ext) {
			return nil
		}

		relPath, _ := filepath.Rel(vaultDir, path)
		files = append(files, relPath)
		return nil
	})

	sort.Strings(files)

	if showTotal {
		fmt.Println(len(files))
		return nil
	}

	for _, f := range files {
		fmt.Println(f)
	}
	return nil
}

// readStdinIfPiped reads all of stdin if it's being piped (not a terminal).
// Returns empty string if stdin is a terminal.
func readStdinIfPiped() string {
	stat, _ := os.Stdin.Stat()
	if stat.Mode()&os.ModeCharDevice != 0 {
		return "" // stdin is a terminal, not piped
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return ""
	}
	return string(data)
}
