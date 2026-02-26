// Package lifecycle implements the SessionStart, PreCompact, Stop, and SessionEnd hooks.
package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/RamXX/vlt"

	"github.com/RamXX/paivot-graph/pvg-cli/internal/vaultcfg"
)

// hookInput matches the JSON Claude Code sends to lifecycle hooks.
type hookInput struct {
	CWD string `json:"cwd"`
}

// SessionStart loads vault context and project-local knowledge at session start.
// Reads JSON from stdin, outputs structured context to stdout. Always exits 0.
func SessionStart() error {
	// 1. Parse hook input
	var input hookInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		// If parsing fails, use cwd
		input.CWD, _ = os.Getwd()
	}
	if input.CWD == "" {
		input.CWD, _ = os.Getwd()
	}

	// 2. Detect project name
	project := detectProject(input.CWD)

	// 2b. Stack detection (opt-in)
	if readStackDetectionSetting(input.CWD) {
		if stacks := detectStack(input.CWD); len(stacks) > 0 {
			fmt.Printf("[STACK] Detected: %s\n", strings.Join(stacks, ", "))
			fmt.Printf("  Suggested vault search: vlt vault=\"Claude\" search query=\"%s\"\n\n", stacks[0])
		}
	}

	// 3. Open vault
	v, err := vaultcfg.OpenVault()
	if err != nil {
		fmt.Printf("[VAULT] Vault not available -- vault consultation skipped. (%v)\n", err)
		return nil // never block session start
	}

	// 4. Search vault for project context
	results, err := v.Search(vlt.SearchOptions{Query: project})
	searchOutput := ""
	if err != nil || len(results) == 0 {
		searchOutput = "(none found -- this is a new project to the vault)"
	} else {
		var lines []string
		for _, r := range results {
			lines = append(lines, fmt.Sprintf("%s (%s)", r.Title, r.RelPath))
		}
		searchOutput = strings.Join(lines, "\n")
	}

	fmt.Printf("[VAULT] Project: %s\nRelevant vault notes:\n\n%s\n\n", project, searchOutput)

	// 4b. Check for project-local knowledge
	projectVaultDir := filepath.Join(input.CWD, ".vault", "knowledge")
	if info, serr := os.Stat(projectVaultDir); serr == nil && info.IsDir() {
		outputProjectKnowledge(projectVaultDir, input.CWD)
	}

	// 5. Read operating mode
	content, err := v.Read("Session Operating Mode", "")
	if err != nil || content == "" {
		// Static fallback
		fmt.Print(staticOperatingMode())
	} else {
		fmt.Printf("[VAULT] Operating mode for this session (from vault):\n\n%s\n", content)
	}

	return nil
}

func detectProject(cwd string) string {
	// Try git remote first
	cmd := exec.Command("git", "-C", cwd, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err == nil {
		url := strings.TrimSpace(string(out))
		if url != "" {
			base := filepath.Base(url)
			return strings.TrimSuffix(base, ".git")
		}
	}
	return filepath.Base(cwd)
}

// outputProjectKnowledge prints summaries of project-local knowledge notes.
func outputProjectKnowledge(projectVaultDir, cwd string) {
	maxNotes := readMaxNotesSetting(cwd)
	subfolders := []string{"conventions", "decisions", "patterns", "debug", "skills"}

	fmt.Println("Project-local knowledge (.vault/knowledge/):")
	fmt.Println()

	found := false
	for _, sub := range subfolders {
		dir := filepath.Join(projectVaultDir, sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		var mdFiles []os.DirEntry
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				mdFiles = append(mdFiles, e)
			}
		}
		if len(mdFiles) == 0 {
			continue
		}
		found = true

		count := 0
		for _, e := range mdFiles {
			if count >= maxNotes {
				break
			}
			filePath := filepath.Join(dir, e.Name())
			date, firstLine := extractNoteSummary(filePath)
			title := strings.TrimSuffix(e.Name(), ".md")
			fmt.Printf("  %s/%s [%s] %s\n", sub, title, date, firstLine)
			count++
		}
		if len(mdFiles) > maxNotes {
			fmt.Printf("  ... and %d more in %s/\n", len(mdFiles)-maxNotes, sub)
		}
	}

	if found {
		fmt.Println()
		fmt.Println("To read a project note in full, use: Read .vault/knowledge/<subfolder>/<note>.md")
		fmt.Println("For deeper assessment, spawn an Explore agent to review project knowledge.")
		fmt.Println()
	}
}

func readMaxNotesSetting(cwd string) int {
	settingsFile := filepath.Join(cwd, ".vault", "knowledge", ".settings.yaml")
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		return 10
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "session_start_max_notes:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "session_start_max_notes:"))
			n := 10
			fmt.Sscanf(val, "%d", &n)
			if n > 0 {
				return n
			}
		}
	}
	return 10
}

func extractNoteSummary(filePath string) (date, firstLine string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "unknown", "(no summary)"
	}

	lines := strings.Split(string(data), "\n")
	date = "unknown"
	firstLine = "(no summary)"
	inFrontmatter := false
	frontmatterEnd := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			frontmatterEnd = true
			continue
		}

		if inFrontmatter && !frontmatterEnd {
			if strings.HasPrefix(trimmed, "created:") {
				date = strings.TrimSpace(strings.TrimPrefix(trimmed, "created:"))
			}
			continue
		}

		if frontmatterEnd && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			firstLine = trimmed
			break
		}
	}

	return date, firstLine
}

// detectStack checks for common project marker files and returns detected stacks.
func detectStack(cwd string) []string {
	type marker struct {
		file  string
		stack string
	}
	markers := []marker{
		{"go.mod", "go"},
		{"Cargo.toml", "rust"},
		{"Gemfile", "ruby"},
		{"pom.xml", "java"},
		{"build.gradle", "java"},
		{"package.json", "node"},
		{"pyproject.toml", "python"},
		{"requirements.txt", "python"},
		{"mix.exs", "elixir"},
	}

	seen := make(map[string]bool)
	var stacks []string

	for _, m := range markers {
		path := filepath.Join(cwd, m.file)
		if _, err := os.Stat(path); err == nil {
			if !seen[m.stack] {
				seen[m.stack] = true
				stacks = append(stacks, m.stack)
			}
			// Check for typescript in package.json
			if m.file == "package.json" {
				data, err := os.ReadFile(path)
				if err == nil && strings.Contains(string(data), "typescript") {
					if !seen["typescript"] {
						seen["typescript"] = true
						stacks = append(stacks, "typescript")
					}
				}
			}
		}
	}

	// Check for C# projects (glob patterns, no single canonical filename)
	csMatches, _ := filepath.Glob(filepath.Join(cwd, "*.csproj"))
	if len(csMatches) == 0 {
		csMatches, _ = filepath.Glob(filepath.Join(cwd, "*.sln"))
	}
	if len(csMatches) > 0 && !seen["csharp"] {
		seen["csharp"] = true
		stacks = append(stacks, "csharp")
	}

	return stacks
}

// readStackDetectionSetting checks .vault/knowledge/.settings.yaml for stack_detection: true.
func readStackDetectionSetting(cwd string) bool {
	settingsFile := filepath.Join(cwd, ".vault", "knowledge", ".settings.yaml")
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "stack_detection:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "stack_detection:"))
			return val == "true"
		}
	}
	return false
}

func staticOperatingMode() string {
	return `[VAULT] Operating mode for this session:

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
`
}

