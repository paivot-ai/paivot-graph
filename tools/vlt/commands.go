package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// searchResult holds a single search match.
type searchResult struct {
	title   string
	relPath string
}

// contextMatch holds a single line-level match with surrounding context.
type contextMatch struct {
	File    string   // relative path
	Line    int      // 1-based line number of the match
	Match   string   // the matched line text
	Context []string // surrounding lines including the match line
}

// lineRange represents an inclusive range of 0-based line indices.
type lineRange struct {
	start int
	end   int
}

// findMatchLines returns 0-based line indices where query appears (case-insensitive).
func findMatchLines(lines []string, query string) []int {
	queryLower := strings.ToLower(query)
	var matches []int
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {
			matches = append(matches, i)
		}
	}
	return matches
}

// findMatchLinesRegex returns 0-based line indices where the compiled regex matches.
func findMatchLinesRegex(lines []string, re *regexp.Regexp) []int {
	var matches []int
	for i, line := range lines {
		if re.MatchString(line) {
			matches = append(matches, i)
		}
	}
	return matches
}

// expandAndMerge takes match line indices and a context radius, producing merged
// non-overlapping ranges clamped to [0, totalLines).
func expandAndMerge(matchLines []int, contextN int, totalLines int) []lineRange {
	if len(matchLines) == 0 {
		return nil
	}

	var ranges []lineRange
	for _, m := range matchLines {
		start := m - contextN
		if start < 0 {
			start = 0
		}
		end := m + contextN
		if end >= totalLines {
			end = totalLines - 1
		}
		ranges = append(ranges, lineRange{start, end})
	}

	// Merge overlapping or adjacent ranges
	merged := []lineRange{ranges[0]}
	for i := 1; i < len(ranges); i++ {
		last := &merged[len(merged)-1]
		if ranges[i].start <= last.end+1 {
			if ranges[i].end > last.end {
				last.end = ranges[i].end
			}
		} else {
			merged = append(merged, ranges[i])
		}
	}

	return merged
}

// linkInfo holds outgoing link information.
type linkInfo struct {
	Target string `json:"target"`
	Path   string `json:"path"`
	Broken bool   `json:"broken"`
}

// unresolvedResult holds an unresolved link and its source.
type unresolvedResult struct {
	Target string `json:"target"`
	Source string `json:"source"`
}

// cmdVaults lists all Obsidian vaults discovered from the config file.
func cmdVaults(format string) error {
	vaults, err := discoverVaults()
	if err != nil {
		return err
	}

	if len(vaults) == 0 {
		if format == "" {
			fmt.Println("No vaults found.")
		} else {
			formatList(nil, format)
		}
		return nil
	}

	// Sort by name for stable output
	names := make([]string, 0, len(vaults))
	for name := range vaults {
		names = append(names, name)
	}
	sort.Strings(names)

	formatVaults(names, vaults, format)
	return nil
}

// cmdRead prints the contents of a note resolved by title.
// If heading= is provided, only the specified section is returned.
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

	heading := params["heading"]
	if heading == "" {
		// No heading filter: return entire note (backward compatible)
		fmt.Print(string(data))
		return nil
	}

	// Heading-scoped read: find the section and return heading + content
	lines := strings.Split(string(data), "\n")
	bounds, found := findSection(lines, heading)
	if !found {
		return fmt.Errorf("heading %q not found in %q", heading, title)
	}

	// Extract from heading line through end of section.
	// Join with newlines to reconstruct the text. The last element in the
	// slice from Split is typically an empty string when the file ended with
	// a newline, so joining and adding a single trailing newline produces
	// the correct output without doubling.
	section := lines[bounds.HeadingLine:bounds.ContentEnd]
	output := strings.Join(section, "\n")

	// Ensure exactly one trailing newline (matches file convention).
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	fmt.Print(output)
	return nil
}

// searchFilterPattern matches [key:value] property filters in search queries.
var searchFilterPattern = regexp.MustCompile(`\[(\w+):([^\]]+)\]`)

// parseSearchQuery splits a query into text terms and property filters.
// Filters are [key:value] pairs extracted from the query string.
func parseSearchQuery(query string) (text string, filters map[string]string) {
	filters = make(map[string]string)
	matches := searchFilterPattern.FindAllStringSubmatch(query, -1)
	for _, m := range matches {
		filters[m[1]] = m[2]
	}
	text = strings.TrimSpace(searchFilterPattern.ReplaceAllString(query, ""))
	return
}

// cmdSearch finds notes whose title or content matches the query (case-insensitive).
// Supports property filters: query="term [key:value] [key2:value2]"
// Supports regex="pattern" for regexp-based search (case-insensitive by default).
// When both query= and regex= are provided, regex takes precedence (with a warning).
// When context="N" is provided, output switches to file:line:content format
// showing N lines before and after each match (similar to grep -C).
func cmdSearch(vaultDir string, params map[string]string, format string) error {
	query := params["query"]
	regexParam := params["regex"]

	if query == "" && regexParam == "" {
		return fmt.Errorf("search requires query=\"<term>\" or regex=\"<pattern>\"")
	}

	// Compile regex if provided
	var re *regexp.Regexp
	useRegex := regexParam != ""

	if useRegex {
		var compileErr error
		re, compileErr = regexp.Compile("(?i)" + regexParam)
		if compileErr != nil {
			return fmt.Errorf("invalid regex %q: %v", regexParam, compileErr)
		}

		// If both query and regex provided, warn and use regex for text matching
		if query != "" {
			fmt.Fprintf(os.Stderr, "vlt: both query= and regex= provided; regex takes precedence for text matching\n")
		}
	}

	// Parse property filters from query (even when regex is primary text matcher)
	var textQuery string
	var filters map[string]string
	if query != "" {
		textQuery, filters = parseSearchQuery(query)
	} else {
		filters = make(map[string]string)
	}

	// When regex is used, the regex is the text matcher (not the textQuery)
	queryLower := strings.ToLower(textQuery)

	pathFilter := params["path"] // optional: limit to a subdirectory

	// Parse optional context parameter
	contextStr := params["context"]
	contextN := -1 // -1 means no context requested
	if contextStr != "" {
		n, err := parseInt0(contextStr)
		if err != nil {
			return fmt.Errorf("invalid context value: %s", contextStr)
		}
		contextN = n
	}

	searchRoot := vaultDir
	if pathFilter != "" {
		searchRoot = filepath.Join(vaultDir, pathFilter)
		if _, err := os.Stat(searchRoot); os.IsNotExist(err) {
			return fmt.Errorf("path filter %q not found in vault", pathFilter)
		}
	}

	hasTextQuery := useRegex || queryLower != ""
	hasFilters := len(filters) > 0

	if !hasTextQuery && !hasFilters {
		return fmt.Errorf("search requires query=\"<term>\" or regex=\"<pattern>\"")
	}

	var results []searchResult
	var contextResults []contextMatch

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

		// Read file content (needed for both text search and property filters)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := string(data)

		// Check property filters first if present
		if hasFilters {
			yaml, _, hasFM := extractFrontmatter(content)
			if !hasFM {
				return nil // no frontmatter, can't match property filters
			}
			for k, v := range filters {
				got, ok := frontmatterGetValue(yaml, k)
				if !ok || !strings.EqualFold(got, v) {
					return nil // filter doesn't match
				}
			}
		}

		// If no text query, property filters already passed
		if !hasTextQuery {
			results = append(results, searchResult{title, relPath})
			return nil
		}

		// Determine matches based on regex or substring
		var titleMatches, contentMatches bool
		if useRegex {
			titleMatches = re.MatchString(title)
			contentMatches = re.MatchString(content)
		} else {
			titleMatches = strings.Contains(strings.ToLower(title), queryLower)
			contentMatches = strings.Contains(strings.ToLower(content), queryLower)
		}

		if !titleMatches && !contentMatches {
			return nil
		}

		// No context mode: use original behavior
		if contextN < 0 {
			results = append(results, searchResult{title, relPath})
			return nil
		}

		// Context mode: find line-level matches in content
		lines := strings.Split(content, "\n")
		var matchLineIdxs []int
		if useRegex {
			matchLineIdxs = findMatchLinesRegex(lines, re)
		} else {
			matchLineIdxs = findMatchLines(lines, textQuery)
		}

		if len(matchLineIdxs) > 0 {
			// Expand and merge ranges
			ranges := expandAndMerge(matchLineIdxs, contextN, len(lines))
			for _, r := range ranges {
				// For each merged range, output each line
				for i := r.start; i <= r.end; i++ {
					isMatch := false
					for _, m := range matchLineIdxs {
						if m == i {
							isMatch = true
							break
						}
					}
					if isMatch {
						// Collect the full context window around this match
						ctxStart := i - contextN
						if ctxStart < 0 {
							ctxStart = 0
						}
						ctxEnd := i + contextN
						if ctxEnd >= len(lines) {
							ctxEnd = len(lines) - 1
						}
						var ctxLines []string
						for j := ctxStart; j <= ctxEnd; j++ {
							ctxLines = append(ctxLines, lines[j])
						}
						contextResults = append(contextResults, contextMatch{
							File:    relPath,
							Line:    i + 1, // 1-based
							Match:   lines[i],
							Context: ctxLines,
						})
					}
				}
			}
		} else if titleMatches {
			// Title matched but no content match -- still show the file
			// Use a synthetic context match with file info only
			contextResults = append(contextResults, contextMatch{
				File:    relPath,
				Line:    0,
				Match:   title,
				Context: nil,
			})
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Context mode output
	if contextN >= 0 {
		if len(contextResults) == 0 {
			return nil
		}
		formatSearchWithContext(contextResults, format)
		return nil
	}

	// Non-context mode output
	if len(results) == 0 {
		return nil // silent on no results, matching grep convention
	}

	formatSearchResults(results, format)
	return nil
}

// parseInt0 parses a string as a non-negative integer (0 is allowed).
func parseInt0(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}

// cmdCreate creates a new note at the given path within the vault.
// Content comes from the content= parameter or stdin.
// When timestamps is true (or VLT_TIMESTAMPS=1), created_at and updated_at
// are added to frontmatter.
func cmdCreate(vaultDir string, params map[string]string, silent bool, timestamps bool) error {
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

	if timestampsEnabled(timestamps) {
		content = ensureTimestamps(content, true, time.Now())
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
// When timestamps is true (or VLT_TIMESTAMPS=1), updated_at is refreshed.
func cmdAppend(vaultDir string, params map[string]string, timestamps bool) error {
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

	if _, err = fmt.Fprint(f, content); err != nil {
		return err
	}

	if timestampsEnabled(timestamps) {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		updated := ensureTimestamps(string(data), false, time.Now())
		return os.WriteFile(path, []byte(updated), 0644)
	}

	return nil
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

	// Update markdown-style [text](path.md) links across the vault
	mdCount, mdErr := updateVaultMdLinks(vaultDir, from, to)
	if mdErr != nil {
		return fmt.Errorf("moved file but failed updating markdown links: %w", mdErr)
	}
	if mdCount > 0 {
		fmt.Printf("updated [...](%s) -> [...](%s) in %d file(s)\n", from, to, mdCount)
	}

	return nil
}

// cmdBacklinks finds all notes that contain wikilinks to the given title.
func cmdBacklinks(vaultDir string, params map[string]string, format string) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("backlinks requires file=\"<title>\"")
	}

	results, err := findBacklinks(vaultDir, title)
	if err != nil {
		return err
	}

	formatList(results, format)
	return nil
}

// cmdLinks lists outgoing wikilinks from a note, reporting which resolve
// and which are broken.
func cmdLinks(vaultDir string, params map[string]string, format string) error {
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
	var results []linkInfo
	for _, link := range links {
		if seen[link.Title] {
			continue
		}
		seen[link.Title] = true

		resolved, resolveErr := resolveNote(vaultDir, link.Title)
		if resolveErr != nil {
			results = append(results, linkInfo{Target: link.Title, Path: "", Broken: true})
		} else {
			relPath, _ := filepath.Rel(vaultDir, resolved)
			results = append(results, linkInfo{Target: link.Title, Path: relPath, Broken: false})
		}
	}

	formatLinks(results, format)
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

// cmdWrite replaces the body content of an existing note, preserving frontmatter.
// Content comes from the content= parameter or stdin.
// If the note has no frontmatter, the entire file content is replaced.
// When timestamps is true (or VLT_TIMESTAMPS=1), updated_at is refreshed.
func cmdWrite(vaultDir string, params map[string]string, timestamps bool) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("write requires file=\"<title>\"")
	}

	path, err := resolveNote(vaultDir, title)
	if err != nil {
		return err
	}

	content := params["content"]
	if content == "" {
		content = readStdinIfPiped()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	text := string(data)
	_, bodyStart, hasFM := extractFrontmatter(text)

	var result string
	if hasFM {
		lines := strings.Split(text, "\n")
		frontmatter := strings.Join(lines[:bodyStart], "\n")
		result = frontmatter + "\n" + content
	} else {
		result = content
	}

	if timestampsEnabled(timestamps) {
		result = ensureTimestamps(result, false, time.Now())
	}

	return os.WriteFile(path, []byte(result), 0644)
}

// cmdPrepend inserts content at the top of a note, after frontmatter if present.
// When timestamps is true (or VLT_TIMESTAMPS=1), updated_at is refreshed.
func cmdPrepend(vaultDir string, params map[string]string, timestamps bool) error {
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

	if timestampsEnabled(timestamps) {
		result = ensureTimestamps(result, false, time.Now())
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
func cmdProperties(vaultDir string, params map[string]string, format string) error {
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

	formatProperties(fm, format)
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
func cmdOrphans(vaultDir string, format string) error {
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
	formatList(orphans, format)
	return nil
}

// cmdUnresolved finds all broken wikilinks across the vault.
func cmdUnresolved(vaultDir string, format string) error {
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
	var results []unresolvedResult
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
				results = append(results, unresolvedResult{Target: link.Title, Source: relPath})
			}
		}
		return nil
	})

	formatUnresolved(results, format)
	return nil
}

// cmdFiles lists files in the vault, optionally filtered by folder and extension.
func cmdFiles(vaultDir string, params map[string]string, showTotal bool, format string) error {
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

	formatList(files, format)
	return nil
}

// sectionBounds holds the line range of a section identified by findSection.
// HeadingLine is the 0-based index of the heading line itself.
// ContentStart is the 0-based index of the first content line after the heading.
// ContentEnd is the 0-based index one past the last content line (exclusive).
// If the section has no content, ContentStart == ContentEnd.
type sectionBounds struct {
	HeadingLine  int
	ContentStart int
	ContentEnd   int
}

// headingLevel returns the Markdown heading level (number of leading # chars).
// Returns 0 if the line is not a heading.
func headingLevel(line string) int {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return 0
	}
	level := 0
	for _, ch := range trimmed {
		if ch == '#' {
			level++
		} else {
			break
		}
	}
	// Must be followed by a space or end of line to be a valid heading
	if level >= len(trimmed) || trimmed[level] == ' ' {
		return level
	}
	return 0
}

// findSection locates a heading in the given lines and returns its bounds.
// The heading parameter should include the # prefix (e.g., "## Section A").
// Heading match is case-insensitive and trims whitespace.
// The section extends from the heading to the line before the next heading of
// equal or higher level (or EOF). This operates on RAW content, not masked.
func findSection(lines []string, heading string) (sectionBounds, bool) {
	heading = strings.TrimSpace(heading)
	targetLevel := headingLevel(heading)
	if targetLevel == 0 {
		return sectionBounds{}, false
	}

	headingTextLower := strings.ToLower(heading)

	for i, line := range lines {
		lineTrimmed := strings.TrimSpace(line)
		if strings.ToLower(lineTrimmed) == headingTextLower {
			// Found the heading. Now find the end of the section.
			contentStart := i + 1
			contentEnd := len(lines) // default: extends to EOF

			for j := contentStart; j < len(lines); j++ {
				lvl := headingLevel(lines[j])
				if lvl > 0 && lvl <= targetLevel {
					contentEnd = j
					break
				}
			}

			return sectionBounds{
				HeadingLine:  i,
				ContentStart: contentStart,
				ContentEnd:   contentEnd,
			}, true
		}
	}

	return sectionBounds{}, false
}

// cmdPatch performs surgical edits to a note: heading-targeted or line-targeted
// replace/delete. The delete parameter controls whether content is removed
// (true) or replaced with new content (false).
// When timestamps is true (or VLT_TIMESTAMPS=1), updated_at is refreshed.
func cmdPatch(vaultDir string, params map[string]string, delete bool, timestamps bool) error {
	title := params["file"]
	if title == "" {
		return fmt.Errorf("patch requires file=\"<title>\"")
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
	lines := strings.Split(text, "\n")

	heading := params["heading"]
	lineSpec := params["line"]

	if heading == "" && lineSpec == "" {
		return fmt.Errorf("patch requires heading=\"<heading>\" or line=\"<N>\" (or line=\"<N-M>\")")
	}

	content := params["content"]

	var result []string

	if heading != "" {
		// Heading-targeted patch
		bounds, found := findSection(lines, heading)
		if !found {
			return fmt.Errorf("heading %q not found in %q", heading, title)
		}

		if delete {
			// Delete mode: remove heading + content
			result = append(result, lines[:bounds.HeadingLine]...)
			result = append(result, lines[bounds.ContentEnd:]...)
		} else {
			// Replace mode: keep heading, replace content
			result = append(result, lines[:bounds.ContentStart]...)
			// Add new content (split into lines if multiline)
			if content != "" {
				contentLines := strings.Split(content, "\n")
				result = append(result, contentLines...)
			}
			result = append(result, lines[bounds.ContentEnd:]...)
		}
	} else {
		// Line-targeted patch
		startLine, endLine, err := parseLineSpec(lineSpec)
		if err != nil {
			return err
		}

		// Validate range (1-based to 0-based)
		if startLine < 1 || endLine < startLine {
			return fmt.Errorf("invalid line specification: %s", lineSpec)
		}
		if startLine > len(lines) {
			return fmt.Errorf("line %d is beyond file length (%d lines); out of range", startLine, len(lines))
		}
		if endLine > len(lines) {
			return fmt.Errorf("line %d is beyond file length (%d lines); out of range", endLine, len(lines))
		}

		// Convert to 0-based
		start := startLine - 1
		end := endLine // exclusive (endLine is 1-based, so endLine = 0-based + 1)

		if delete {
			result = append(result, lines[:start]...)
			result = append(result, lines[end:]...)
		} else {
			result = append(result, lines[:start]...)
			result = append(result, content)
			result = append(result, lines[end:]...)
		}
	}

	output := strings.Join(result, "\n")

	if timestampsEnabled(timestamps) {
		output = ensureTimestamps(output, false, time.Now())
	}

	return os.WriteFile(path, []byte(output), 0644)
}

// parseLineSpec parses a line specification like "5" or "5-10" into start and end
// line numbers (1-based, inclusive).
func parseLineSpec(spec string) (start, end int, err error) {
	if idx := strings.Index(spec, "-"); idx >= 0 {
		startStr := spec[:idx]
		endStr := spec[idx+1:]
		start, err = parseInt(startStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid line range start: %s", startStr)
		}
		end, err = parseInt(endStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid line range end: %s", endStr)
		}
		return start, end, nil
	}

	start, err = parseInt(spec)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid line number: %s", spec)
	}
	return start, start, nil
}

// parseInt parses a string as a positive integer.
func parseInt(s string) (int, error) {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int(ch-'0')
	}
	if n == 0 {
		return 0, fmt.Errorf("not a positive number: %s", s)
	}
	return n, nil
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
