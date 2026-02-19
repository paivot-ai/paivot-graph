package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// outputFormat extracts the output format from flags.
// Returns "json", "csv", "yaml", or "" for plain text.
func outputFormat(flags map[string]bool) string {
	if flags["--json"] {
		return "json"
	}
	if flags["--csv"] {
		return "csv"
	}
	if flags["--yaml"] {
		return "yaml"
	}
	return ""
}

// formatList outputs a []string in the requested format.
// For plain text, one item per line.
func formatList(items []string, format string) {
	switch format {
	case "json":
		data, _ := json.Marshal(items)
		fmt.Println(string(data))
	case "csv":
		w := csv.NewWriter(os.Stdout)
		for _, item := range items {
			w.Write([]string{item})
		}
		w.Flush()
	case "yaml":
		for _, item := range items {
			fmt.Printf("- %s\n", item)
		}
	default:
		for _, item := range items {
			fmt.Println(item)
		}
	}
}

// formatTable outputs rows of key-value data in the requested format.
// fields controls column order for CSV and key order for YAML/JSON.
func formatTable(rows []map[string]string, fields []string, format string) {
	switch format {
	case "json":
		data, _ := json.Marshal(rows)
		fmt.Println(string(data))
	case "csv":
		w := csv.NewWriter(os.Stdout)
		w.Write(fields) // header row
		for _, row := range rows {
			record := make([]string, len(fields))
			for i, f := range fields {
				record[i] = row[f]
			}
			w.Write(record)
		}
		w.Flush()
	case "yaml":
		for i, row := range rows {
			if i > 0 {
				fmt.Println("---")
			}
			for _, f := range fields {
				if v, ok := row[f]; ok {
					fmt.Printf("%s: %s\n", f, yamlEscapeValue(v))
				}
			}
		}
	default:
		// Plain text: tab-separated, fields in order
		for _, row := range rows {
			parts := make([]string, 0, len(fields))
			for _, f := range fields {
				if v, ok := row[f]; ok {
					parts = append(parts, v)
				}
			}
			fmt.Println(strings.Join(parts, "\t"))
		}
	}
}

// formatMap outputs a map[string]string (single record) in the requested format.
// keys controls output order.
func formatMap(m map[string]string, keys []string, format string) {
	switch format {
	case "json":
		data, _ := json.Marshal(m)
		fmt.Println(string(data))
	case "csv":
		w := csv.NewWriter(os.Stdout)
		w.Write(keys)
		record := make([]string, len(keys))
		for i, k := range keys {
			record[i] = m[k]
		}
		w.Write(record)
		w.Flush()
	case "yaml":
		for _, k := range keys {
			if v, ok := m[k]; ok {
				fmt.Printf("%s: %s\n", k, yamlEscapeValue(v))
			}
		}
	default:
		for _, k := range keys {
			if v, ok := m[k]; ok {
				fmt.Printf("%s: %s\n", k, v)
			}
		}
	}
}

// formatTagCounts outputs tag-count pairs in the requested format.
func formatTagCounts(tags []string, counts map[string]int, format string) {
	switch format {
	case "json":
		type tagEntry struct {
			Tag   string `json:"tag"`
			Count int    `json:"count"`
		}
		entries := make([]tagEntry, len(tags))
		for i, t := range tags {
			entries[i] = tagEntry{Tag: t, Count: counts[t]}
		}
		data, _ := json.Marshal(entries)
		fmt.Println(string(data))
	case "csv":
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{"tag", "count"})
		for _, t := range tags {
			w.Write([]string{t, fmt.Sprintf("%d", counts[t])})
		}
		w.Flush()
	case "yaml":
		for _, t := range tags {
			fmt.Printf("- tag: %s\n  count: %d\n", t, counts[t])
		}
	default:
		for _, t := range tags {
			fmt.Printf("#%s\t%d\n", t, counts[t])
		}
	}
}

// formatVaults outputs vault name-path pairs in the requested format.
func formatVaults(names []string, vaults map[string]string, format string) {
	switch format {
	case "json":
		type vaultInfo struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		entries := make([]vaultInfo, len(names))
		for i, n := range names {
			entries[i] = vaultInfo{Name: n, Path: vaults[n]}
		}
		data, _ := json.Marshal(entries)
		fmt.Println(string(data))
	case "csv":
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{"name", "path"})
		for _, n := range names {
			w.Write([]string{n, vaults[n]})
		}
		w.Flush()
	case "yaml":
		for _, n := range names {
			fmt.Printf("- name: %s\n  path: %s\n", n, vaults[n])
		}
	default:
		for _, n := range names {
			fmt.Printf("%s\t%s\n", n, vaults[n])
		}
	}
}

// formatSearchResults outputs search results in the requested format.
func formatSearchResults(results []searchResult, format string) {
	switch format {
	case "json":
		type jsonResult struct {
			Title string `json:"title"`
			Path  string `json:"path"`
		}
		entries := make([]jsonResult, len(results))
		for i, r := range results {
			entries[i] = jsonResult{Title: r.title, Path: r.relPath}
		}
		data, _ := json.Marshal(entries)
		fmt.Println(string(data))
	case "csv":
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{"title", "path"})
		for _, r := range results {
			w.Write([]string{r.title, r.relPath})
		}
		w.Flush()
	case "yaml":
		for _, r := range results {
			fmt.Printf("- title: %s\n  path: %s\n", yamlEscapeValue(r.title), r.relPath)
		}
	default:
		for _, r := range results {
			fmt.Printf("%s (%s)\n", r.title, r.relPath)
		}
	}
}

// formatLinks outputs link information in the requested format.
func formatLinks(links []linkInfo, format string) {
	switch format {
	case "json":
		data, _ := json.Marshal(links)
		fmt.Println(string(data))
	case "csv":
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{"target", "path", "broken"})
		for _, l := range links {
			broken := "false"
			if l.Broken {
				broken = "true"
			}
			w.Write([]string{l.Target, l.Path, broken})
		}
		w.Flush()
	case "yaml":
		for _, l := range links {
			fmt.Printf("- target: %s\n  path: %s\n  broken: %v\n", yamlEscapeValue(l.Target), l.Path, l.Broken)
		}
	default:
		for _, l := range links {
			if l.Broken {
				fmt.Printf("  BROKEN: [[%s]]\n", l.Target)
			} else {
				fmt.Printf("  [[%s]] -> %s\n", l.Target, l.Path)
			}
		}
	}
}

// formatUnresolved outputs unresolved link information.
func formatUnresolved(results []unresolvedResult, format string) {
	switch format {
	case "json":
		data, _ := json.Marshal(results)
		fmt.Println(string(data))
	case "csv":
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{"target", "source"})
		for _, r := range results {
			w.Write([]string{r.Target, r.Source})
		}
		w.Flush()
	case "yaml":
		for _, r := range results {
			fmt.Printf("- target: %s\n  source: %s\n", yamlEscapeValue(r.Target), r.Source)
		}
	default:
		for _, r := range results {
			fmt.Printf("[[%s]] in %s\n", r.Target, r.Source)
		}
	}
}

// formatProperties outputs frontmatter properties in the requested format.
func formatProperties(text string, format string) {
	if format == "" {
		fmt.Println(text)
		return
	}

	// Parse the frontmatter into key-value pairs
	lines := strings.Split(text, "\n")
	props := make(map[string]string)
	var keys []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "---" || line == "" {
			continue
		}
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			props[key] = val
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)

	switch format {
	case "json":
		data, _ := json.Marshal(props)
		fmt.Println(string(data))
	case "csv":
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{"key", "value"})
		for _, k := range keys {
			w.Write([]string{k, props[k]})
		}
		w.Flush()
	case "yaml":
		for _, k := range keys {
			fmt.Printf("%s: %s\n", k, props[k])
		}
	}
}

// yamlEscapeValue wraps a value in quotes if it contains characters
// that need escaping in YAML (colons, brackets, etc).
func yamlEscapeValue(s string) string {
	if s == "" {
		return `""`
	}
	needsQuoting := false
	for _, c := range s {
		if c == ':' || c == '#' || c == '[' || c == ']' || c == '{' || c == '}' ||
			c == ',' || c == '&' || c == '*' || c == '!' || c == '|' || c == '>' ||
			c == '\'' || c == '"' || c == '%' || c == '@' || c == '`' {
			needsQuoting = true
			break
		}
	}
	if needsQuoting {
		escaped := strings.ReplaceAll(s, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return s
}
