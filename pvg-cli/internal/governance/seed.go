// Package governance implements vault seeding and knowledge governance operations.
package governance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/paivot-graph/pvg-cli/internal/vaultcfg"
)

// Counters tracks seed operations.
type Counters struct {
	Created int
	Updated int
	Skipped int
}

// Seed writes vault notes directly to disk. Obsidian picks them up via iCloud sync.
func Seed(force bool, pluginDir string) error {
	today := time.Now().Format("2006-01-02")

	// Open vault
	v, err := vaultcfg.OpenVault()
	if err != nil {
		return fmt.Errorf("cannot open vault: %w", err)
	}
	vaultDir := v.Dir()

	// Resolve agent source
	agentSrc, err := resolveAgentSrc()
	if err != nil {
		return err
	}

	// Resolve plugin dir if not provided
	if pluginDir == "" {
		exe, eerr := os.Executable()
		if eerr == nil {
			// Assume pvg is at bin/pvg or pvg-cli/pvg, walk up to plugin root
			pluginDir = filepath.Dir(filepath.Dir(exe))
		}
	}
	// If pluginDir still empty, try CLAUDE_PLUGIN_ROOT
	if pluginDir == "" {
		pluginDir = os.Getenv("CLAUDE_PLUGIN_ROOT")
	}

	counters := &Counters{}

	fmt.Println("paivot-graph vault seeder")
	fmt.Println("=========================")
	if force {
		fmt.Println("Mode: force (overwriting existing notes)")
	} else {
		fmt.Println("Mode: safe (skipping existing notes)")
	}
	fmt.Println()

	// 1. Seed agent prompts
	fmt.Println("Seeding agent prompts...")
	agents := []struct {
		slug      string
		vaultName string
	}{
		{"paivot-sr-pm", "Sr PM Agent"},
		{"paivot-pm", "PM Acceptor Agent"},
		{"paivot-developer", "Developer Agent"},
		{"paivot-architect", "Architect Agent"},
		{"paivot-designer", "Designer Agent"},
		{"paivot-business-analyst", "Business Analyst Agent"},
		{"paivot-anchor", "Anchor Agent"},
		{"paivot-retro", "Retro Agent"},
	}

	for _, agent := range agents {
		seedAgent(vaultDir, agentSrc, agent.slug, agent.vaultName, today, force, counters)
	}

	// 2. Seed skill content
	fmt.Println()
	fmt.Println("Seeding skill content...")
	seedSkill(vaultDir, pluginDir, today, force, counters)

	// 3. Seed behavioral notes
	fmt.Println()
	fmt.Println("Seeding behavioral notes...")
	seedSessionOperatingMode(vaultDir, today, force, counters)
	seedPreCompactChecklist(vaultDir, today, force, counters)
	seedStopCaptureChecklist(vaultDir, today, force, counters)

	fmt.Println()
	fmt.Printf("Done. Created: %d, Updated: %d, Skipped: %d\n",
		counters.Created, counters.Updated, counters.Skipped)

	return nil
}

func resolveAgentSrc() (string, error) {
	src := os.Getenv("AGENT_SRC")
	if src != "" {
		return src, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	// Walk the paivot-claude cache to find the agents/ directory
	cacheBase := filepath.Join(home, ".claude", "plugins", "cache", "paivot-claude")
	var candidates []string

	_ = filepath.WalkDir(cacheBase, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if d.IsDir() && d.Name() == "agents" {
			candidates = append(candidates, path)
		}
		return nil
	})

	if len(candidates) == 0 {
		return "", fmt.Errorf("could not find paivot-claude agents directory in plugin cache.\nSet AGENT_SRC=/path/to/agents manually, or install paivot-claude first")
	}

	// Return the last one (sort order matches the newest version)
	return candidates[len(candidates)-1], nil
}

func writeNote(vaultDir, relPath, content string, force bool, counters *Counters) {
	fullPath := filepath.Join(vaultDir, relPath)

	if _, err := os.Stat(fullPath); err == nil {
		if force {
			if werr := os.WriteFile(fullPath, []byte(content), 0644); werr != nil {
				fmt.Printf("  ERROR: %s: %v\n", relPath, werr)
				return
			}
			fmt.Printf("  UPDATED: %s\n", relPath)
			counters.Updated++
		} else {
			fmt.Printf("  SKIP: %s (already exists)\n", relPath)
			counters.Skipped++
		}
		return
	}

	// Ensure parent directory
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		fmt.Printf("  ERROR: cannot create directory for %s: %v\n", relPath, err)
		return
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		fmt.Printf("  ERROR: %s: %v\n", relPath, err)
		return
	}
	fmt.Printf("  CREATED: %s\n", relPath)
	counters.Created++
}

func extractBody(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	fmCount := 0
	bodyStart := 0

	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			fmCount++
			if fmCount >= 2 {
				bodyStart = i + 1
				break
			}
		}
	}

	if bodyStart == 0 {
		return string(data), nil
	}

	return strings.Join(lines[bodyStart:], "\n"), nil
}

func seedAgent(vaultDir, agentSrc, slug, vaultName, today string, force bool, counters *Counters) {
	srcFile := filepath.Join(agentSrc, slug+".md")
	if _, err := os.Stat(srcFile); err != nil {
		fmt.Printf("  WARN: %s not found, skipping %s\n", srcFile, vaultName)
		counters.Skipped++
		return
	}

	body, err := extractBody(srcFile)
	if err != nil {
		fmt.Printf("  WARN: cannot read %s: %v\n", srcFile, err)
		counters.Skipped++
		return
	}

	content := fmt.Sprintf(`---
type: methodology
scope: system
project: paivot
stack: [claude-code]
domain: developer-tools
status: active
created: %s
---

%s

## Changelog

- %s: Seeded from paivot-graph plugin (initial version)
`, today, strings.TrimSpace(body), today)

	writeNote(vaultDir, filepath.Join("methodology", vaultName+".md"), content, force, counters)
}

func seedSkill(vaultDir, pluginDir, today string, force bool, counters *Counters) {
	skillSrc := filepath.Join(pluginDir, "skills", "vault-knowledge", "SKILL.md")
	if _, err := os.Stat(skillSrc); err != nil {
		fmt.Printf("  WARN: %s not found\n", skillSrc)
		counters.Skipped++
		return
	}

	body, err := extractBody(skillSrc)
	if err != nil {
		fmt.Printf("  WARN: cannot read %s: %v\n", skillSrc, err)
		counters.Skipped++
		return
	}

	content := fmt.Sprintf(`---
type: convention
scope: system
project: paivot-graph
stack: [claude-code, obsidian]
domain: developer-tools
status: active
created: %s
---

%s

## Changelog

- %s: Seeded from paivot-graph plugin (initial version)
`, today, strings.TrimSpace(body), today)

	writeNote(vaultDir, filepath.Join("conventions", "Vault Knowledge Skill.md"), content, force, counters)
}

func seedSessionOperatingMode(vaultDir, today string, force bool, counters *Counters) {
	content := fmt.Sprintf(`---
type: convention
scope: system
project: paivot-graph
stack: [claude-code, obsidian]
domain: developer-tools
status: active
created: %s
---

# Session Operating Mode

CONCURRENCY LIMITS (HARD RULE -- unless user explicitly overrides):
  - Maximum 2 developer agents running simultaneously
  - Maximum 1 PM-Acceptor agent running simultaneously
  - Total active subagents (all types) must not exceed 3
  These limits prevent context exhaustion. Violating them risks losing the entire session.

BEFORE STARTING: Read the vault notes listed above. Do not rediscover what is already known.
  vlt vault="Claude" read file="<note>"

WHILE WORKING: Capture knowledge as it emerges -- do not wait for the end.
  - After making a decision (chose X over Y): create a decision note
  - After solving a non-obvious bug: create a debug note
  - After discovering a reusable pattern: create a pattern note
  Use: vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="..." silent

BEFORE ENDING: Update the project index note with what was accomplished.
  vlt vault="Claude" append file="<Project>" content="## Session update (<date>)\n- <what was done>"

This is not optional. Knowledge that is not captured is knowledge that will be rediscovered at cost.

## Related

- [[paivot-graph]] -- Plugin that reads this note at session start
- [[Vault as runtime not reference]] -- Why this content lives in the vault
- [[Vault Knowledge Skill]] -- How to interact with the vault
- [[Pre-Compact Checklist]] -- Companion checklist before compaction
- [[Stop Capture Checklist]] -- Companion checklist before stopping

## Changelog

- %s: Seeded from paivot-graph plugin (initial version)
`, today, today)

	writeNote(vaultDir, filepath.Join("conventions", "Session Operating Mode.md"), content, force, counters)
}

func seedPreCompactChecklist(vaultDir, today string, force bool, counters *Counters) {
	content := fmt.Sprintf(`---
type: convention
scope: system
project: paivot-graph
stack: [claude-code, obsidian]
domain: developer-tools
status: active
created: %s
---

# Pre-Compact Checklist

Context compaction is imminent. Save anything worth remembering NOW.

## 1. DECISIONS made this session

Record any decisions with rationale and alternatives considered:
  vlt vault="Claude" create name="<Decision Title>" path="_inbox/<Decision Title>.md" content="..." silent

Include frontmatter: type: decision, project: <project>, status: active, confidence: high, created: <YYYY-MM-DD>
Include sections: Decision, Rationale, Alternatives considered.

## 2. PATTERNS discovered

Record reusable solutions:
  vlt vault="Claude" create name="<Pattern Name>" path="_inbox/<Pattern Name>.md" content="..." silent

Include frontmatter: type: pattern, project: <project>, stack: [], status: active, created: <YYYY-MM-DD>
Include sections: When to use, Implementation.

## 3. DEBUG INSIGHTS

Record problems solved:
  vlt vault="Claude" create name="<Bug Title>" path="_inbox/<Bug Title>.md" content="..." silent

Include frontmatter: type: debug, project: <project>, status: active, created: <YYYY-MM-DD>
Include sections: Symptoms, Root cause, Fix.

## 4. PROJECT UPDATES

  vlt vault="Claude" append file="<Project>" content="## Session update (<YYYY-MM-DD>)\n- <what was accomplished>"

Do this NOW -- after compaction, the details will be lost.

## Changelog

- %s: Seeded from paivot-graph plugin (initial version)
`, today, today)

	writeNote(vaultDir, filepath.Join("conventions", "Pre-Compact Checklist.md"), content, force, counters)
}

func seedStopCaptureChecklist(vaultDir, today string, force bool, counters *Counters) {
	content := fmt.Sprintf(`---
type: convention
scope: system
project: paivot-graph
stack: [claude-code, obsidian]
domain: developer-tools
status: active
created: %s
---

# Stop Capture Checklist

Before ending this session, confirm you have considered each of these:

- [ ] Did you capture any DECISIONS made this session? (chose X over Y, established a convention)
- [ ] Did you capture any PATTERNS discovered? (reusable solutions, idioms, workflows)
- [ ] Did you capture any DEBUG INSIGHTS? (non-obvious bugs, sharp edges, environment issues)
- [ ] Did you update the PROJECT INDEX NOTE with what was accomplished?

If none of the above apply (e.g., quick fix, trivial session), that is fine -- but confirm it was considered, not forgotten.

Use vlt to create notes: vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="..." silent

## Changelog

- %s: Seeded from paivot-graph plugin (initial version)
`, today, today)

	writeNote(vaultDir, filepath.Join("conventions", "Stop Capture Checklist.md"), content, force, counters)
}
