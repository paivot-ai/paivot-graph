// Package settings manages project-local vault settings (.vault/knowledge/.settings.yaml).
package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const settingsFile = ".vault/knowledge/.settings.yaml"

// defaults for all known settings.
var defaults = map[string]string{
	"session_start_max_notes": "10",
	"auto_capture":            "true",
	"staleness_days":          "30",
	"stack_detection":         "false",
}

// Run handles the `pvg settings` command.
// With no args: display current settings.
// With key=value args: set settings.
func Run(args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	path := filepath.Join(cwd, settingsFile)

	if len(args) == 0 {
		return showSettings(path)
	}

	return setSettings(path, args)
}

func showSettings(path string) error {
	settings := loadSettings(path)

	fmt.Println("Project vault settings (.vault/knowledge/.settings.yaml):")
	fmt.Println()

	// Sort keys for stable output
	keys := make([]string, 0, len(defaults))
	for k := range defaults {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		val, ok := settings[k]
		if !ok {
			val = defaults[k] + " (default)"
		}
		fmt.Printf("  %s: %s\n", k, val)
	}

	// Show any extra settings not in defaults
	for k, v := range settings {
		if _, ok := defaults[k]; !ok {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	return nil
}

func setSettings(path string, args []string) error {
	settings := loadSettings(path)

	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid setting %q (expected key=value)", arg)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return fmt.Errorf("empty key in %q", arg)
		}

		settings[key] = value
		fmt.Printf("  set %s = %s\n", key, value)
	}

	return writeSettings(path, settings)
}

func loadSettings(path string) map[string]string {
	settings := make(map[string]string)
	data, err := os.ReadFile(path)
	if err != nil {
		return settings
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			settings[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return settings
}

func writeSettings(path string, settings map[string]string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create settings directory: %w", err)
	}

	var lines []string
	lines = append(lines, "# paivot-graph project vault settings")
	lines = append(lines, "# Managed by: pvg settings key=value")
	lines = append(lines, "")

	// Sort keys for stable output
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s: %s", k, settings[k]))
	}
	lines = append(lines, "")

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}
