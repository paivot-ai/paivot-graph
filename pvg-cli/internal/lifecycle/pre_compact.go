package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RamXX/paivot-graph/pvg-cli/internal/vaultcfg"
)

// PreCompact outputs a knowledge capture reminder before context compaction.
// Reads the Pre-Compact Checklist from the vault or uses a static fallback.
func PreCompact() error {
	v, err := vaultcfg.OpenVault()
	if err == nil {
		result, rerr := v.Read("Pre-Compact Checklist", "")
		if rerr == nil && result.Content != "" {
			fmt.Println("[VAULT] Context compaction imminent -- capture knowledge now.")
			fmt.Println()
			fmt.Println(result.Content)
			outputTwoTierGuidance()
			return nil
		}
	}

	// Try direct file read
	vaultDir, derr := vaultcfg.VaultDir()
	if derr == nil {
		path := filepath.Join(vaultDir, "conventions", "Pre-Compact Checklist.md")
		data, ferr := os.ReadFile(path)
		if ferr == nil && len(data) > 0 {
			fmt.Println("[VAULT] Context compaction imminent -- capture knowledge now.")
			fmt.Println()
			fmt.Println(string(data))
			outputTwoTierGuidance()
			return nil
		}
	}

	// Static fallback
	fmt.Print(staticPreCompact())
	outputTwoTierGuidance()
	return nil
}

func outputTwoTierGuidance() {
	cwd, _ := os.Getwd()
	knowledgeDir := filepath.Join(cwd, ".vault", "knowledge")
	if info, err := os.Stat(knowledgeDir); err == nil && info.IsDir() {
		fmt.Print(`
[VAULT] Where to save knowledge:
  - Universal insights (applicable to ANY project) -> global vault _inbox/
      vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="..." silent
  - Project-specific insights (only relevant HERE) -> .vault/knowledge/ locally
      Create files directly in .vault/knowledge/decisions/, patterns/, debug/, or conventions/
`)
	}
}

func staticPreCompact() string {
	return `[VAULT] Context compaction imminent -- capture knowledge now.

Before this context is compacted, save anything worth remembering:

1. DECISIONS made this session (with rationale and alternatives considered):
   vlt vault="Claude" create name="<Decision Title>" path="_inbox/<Decision Title>.md" content="..." silent

2. PATTERNS discovered (reusable solutions):
   vlt vault="Claude" create name="<Pattern Name>" path="_inbox/<Pattern Name>.md" content="..." silent

3. DEBUG INSIGHTS (problems solved):
   vlt vault="Claude" create name="<Bug Title>" path="_inbox/<Bug Title>.md" content="..." silent

4. PROJECT UPDATES (progress, state changes):
   vlt vault="Claude" append file="<Project>" content="## Session update (<date>)\n- <what was accomplished>"

All notes must have frontmatter: type, project, status, created.

Do this NOW -- after compaction, the details will be lost.
`
}
