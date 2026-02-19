package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// obsidianConfig is the top-level structure of Obsidian's config file.
type obsidianConfig struct {
	Vaults map[string]vaultEntry `json:"vaults"`
}

// vaultEntry represents a single vault in Obsidian's config.
type vaultEntry struct {
	Path string `json:"path"`
	TS   int64  `json:"ts"`
}

// resolveVault turns a vault name (or path) into an absolute directory path.
//
// If name looks like an absolute path, it's used directly.
// Otherwise, it's looked up by directory basename in the Obsidian config.
func resolveVault(name string) (string, error) {
	// Direct path
	if strings.HasPrefix(name, "/") {
		return validateVaultDir(name)
	}
	if strings.HasPrefix(name, "~") {
		home, _ := os.UserHomeDir()
		return validateVaultDir(filepath.Join(home, name[1:]))
	}

	// Look up by name
	vaults, err := discoverVaults()
	if err != nil {
		// Fall back to VLT_VAULT_PATH env var
		if p := os.Getenv("VLT_VAULT_PATH"); p != "" {
			return validateVaultDir(p)
		}
		return "", fmt.Errorf("cannot discover vaults: %w", err)
	}

	path, ok := vaults[name]
	if !ok {
		available := make([]string, 0, len(vaults))
		for k := range vaults {
			available = append(available, k)
		}
		return "", fmt.Errorf("vault %q not found. Available: %s", name, strings.Join(available, ", "))
	}

	return validateVaultDir(path)
}

func validateVaultDir(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("vault directory not found: %s", path)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("vault path is not a directory: %s", path)
	}
	return path, nil
}

// discoverVaults reads the Obsidian config file and returns a map of
// vault name (directory basename) to absolute path.
func discoverVaults() (map[string]string, error) {
	configPath := obsidianConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", configPath, err)
	}

	var config obsidianConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w", configPath, err)
	}

	vaults := make(map[string]string, len(config.Vaults))
	for _, entry := range config.Vaults {
		name := filepath.Base(entry.Path)
		vaults[name] = entry.Path
	}

	return vaults, nil
}

// obsidianConfigPath returns the platform-appropriate path to obsidian.json.
func obsidianConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		// macOS fallback
		configDir = filepath.Join(home, "Library", "Application Support")
	}
	return filepath.Join(configDir, "obsidian", "obsidian.json")
}

// resolveNote finds a note by title within the vault.
// First pass: exact filename match (<title>.md).
// Second pass (if needed): checks frontmatter aliases.
// Skips hidden dirs and .trash.
func resolveNote(vaultDir, title string) (string, error) {
	target := title + ".md"
	var found string

	// First pass: exact filename match (fast, no file reads)
	filepath.WalkDir(vaultDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() && (strings.HasPrefix(name, ".") || name == ".trash") {
			return filepath.SkipDir
		}
		if !d.IsDir() && name == target {
			found = path
			return filepath.SkipAll
		}
		return nil
	})

	if found != "" {
		return found, nil
	}

	// Second pass: check frontmatter aliases
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

		yaml, _, hasFM := extractFrontmatter(string(data))
		if hasFM {
			for _, alias := range frontmatterGetList(yaml, "aliases") {
				if strings.EqualFold(alias, title) {
					found = path
					return filepath.SkipAll
				}
			}
		}
		return nil
	})

	if found != "" {
		return found, nil
	}

	return "", fmt.Errorf("note %q not found in vault", title)
}
