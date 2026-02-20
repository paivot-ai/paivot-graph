package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// extractFrontmatter returns the YAML content between --- delimiters,
// the line index where the body starts, and whether frontmatter was found.
func extractFrontmatter(text string) (yaml string, bodyStart int, found bool) {
	lines := strings.Split(text, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return "", 0, false
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), i + 1, true
		}
	}

	return "", 0, false
}

// frontmatterGetList extracts a list value from frontmatter YAML.
// Handles inline format: key: [a, b, c]
// and block format:
//
//	key:
//	  - a
//	  - b
func frontmatterGetList(yaml, key string) []string {
	lines := strings.Split(yaml, "\n")
	prefix := key + ":"

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}

		value := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))

		// Inline list: [a, b, c]
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			inner := value[1 : len(value)-1]
			parts := strings.Split(inner, ",")
			var result []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				p = strings.Trim(p, "\"'")
				if p != "" {
					result = append(result, p)
				}
			}
			return result
		}

		// Non-empty single value
		if value != "" {
			return []string{strings.Trim(value, "\"'")}
		}

		// Block list: subsequent lines starting with "- "
		var result []string
		for j := i + 1; j < len(lines); j++ {
			t := strings.TrimSpace(lines[j])
			if strings.HasPrefix(t, "- ") {
				val := strings.TrimSpace(strings.TrimPrefix(t, "- "))
				val = strings.Trim(val, "\"'")
				if val != "" {
					result = append(result, val)
				}
			} else if t == "" {
				continue
			} else {
				break
			}
		}
		return result
	}

	return nil
}

// frontmatterGetValue extracts a simple string value from frontmatter YAML.
func frontmatterGetValue(yaml, key string) (string, bool) {
	lines := strings.Split(yaml, "\n")
	prefix := key + ":"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			value = strings.Trim(value, "\"'")
			return value, true
		}
	}
	return "", false
}

// frontmatterRemoveKey removes a key and its value (including block lists)
// from text that contains frontmatter. Returns the original text unchanged
// if the key is not found.
func frontmatterRemoveKey(text, key string) string {
	lines := strings.Split(text, "\n")
	prefix := key + ":"

	// Find frontmatter boundaries
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
		return text
	}

	// Find the key line and determine what to remove
	keyLine := -1
	removeEnd := -1

	for i := fmStart + 1; i < fmEnd; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, prefix) {
			keyLine = i
			removeEnd = i + 1

			// Check if followed by a block list
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			if value == "" {
				for j := i + 1; j < fmEnd; j++ {
					t := strings.TrimSpace(lines[j])
					if strings.HasPrefix(t, "- ") || t == "" {
						removeEnd = j + 1
					} else {
						break
					}
				}
			}
			break
		}
	}

	if keyLine == -1 {
		return text
	}

	result := make([]string, 0, len(lines)-(removeEnd-keyLine))
	result = append(result, lines[:keyLine]...)
	result = append(result, lines[removeEnd:]...)

	return strings.Join(result, "\n")
}

// frontmatterReadAll returns the raw frontmatter block including --- delimiters.
// Returns empty string if no frontmatter found.
func frontmatterReadAll(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[:i+1], "\n")
		}
	}
	return ""
}

// timestampsEnabled returns true if timestamps should be applied,
// based on the explicit flag or the VLT_TIMESTAMPS environment variable.
func timestampsEnabled(flag bool) bool {
	if flag {
		return true
	}
	return os.Getenv("VLT_TIMESTAMPS") == "1"
}

// ensureTimestamps adds or updates created_at and updated_at frontmatter properties.
// If isCreate is true, created_at is set (unless it already exists). updated_at is
// always set. If the text has no frontmatter, it is added. The now parameter allows
// callers (and tests) to inject a specific time.
func ensureTimestamps(text string, isCreate bool, now time.Time) string {
	ts := now.UTC().Format(time.RFC3339)

	_, _, hasFM := extractFrontmatter(text)

	if !hasFM {
		// Add frontmatter with timestamps
		var fm strings.Builder
		fm.WriteString("---\n")
		if isCreate {
			fmt.Fprintf(&fm, "created_at: %s\n", ts)
		}
		fmt.Fprintf(&fm, "updated_at: %s\n", ts)
		fm.WriteString("---\n")
		return fm.String() + text
	}

	// Has frontmatter -- operate on lines
	lines := strings.Split(text, "\n")

	// Find frontmatter boundaries
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

	// Set or update properties within frontmatter
	setProperty := func(key, value string, overwrite bool) {
		prefix := key + ":"
		found := false
		for i := fmStart + 1; i < fmEnd; i++ {
			trimmed := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmed, prefix) {
				if overwrite {
					lines[i] = fmt.Sprintf("%s: %s", key, value)
				}
				found = true
				break
			}
		}
		if !found {
			// Insert before closing ---
			newLine := fmt.Sprintf("%s: %s", key, value)
			lines = append(lines[:fmEnd+1], lines[fmEnd:]...)
			lines[fmEnd] = newLine
			fmEnd++ // closing --- moved down by one
		}
	}

	// On create, set created_at only if not already present (overwrite=false)
	if isCreate {
		setProperty("created_at", ts, false)
	}

	// Always set updated_at (overwrite=true)
	setProperty("updated_at", ts, true)

	return strings.Join(lines, "\n")
}
