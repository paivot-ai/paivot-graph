package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RamXX/paivot-graph/pvg-cli/internal/vaultcfg"
)

// Stop outputs a knowledge capture reminder when Claude tries to stop.
// Reads the Stop Capture Checklist from the vault or uses a static fallback.
func Stop() error {
	v, err := vaultcfg.OpenVault()
	if err == nil {
		result, rerr := v.Read("Stop Capture Checklist", "")
		if rerr == nil && result.Content != "" {
			fmt.Println("[VAULT] Stop capture check (from vault):")
			fmt.Println()
			fmt.Println(result.Content)
			outputTwoTierReminder()
			return nil
		}
	}

	// Try direct file read
	vaultDir, derr := vaultcfg.VaultDir()
	if derr == nil {
		path := filepath.Join(vaultDir, "conventions", "Stop Capture Checklist.md")
		data, ferr := os.ReadFile(path)
		if ferr == nil && len(data) > 0 {
			fmt.Println("[VAULT] Stop capture check (from vault):")
			fmt.Println()
			fmt.Println(string(data))
			outputTwoTierReminder()
			return nil
		}
	}

	// Static fallback
	fmt.Print(staticStopChecklist())
	outputTwoTierReminder()
	return nil
}

func outputTwoTierReminder() {
	cwd, _ := os.Getwd()
	knowledgeDir := filepath.Join(cwd, ".vault", "knowledge")
	if info, err := os.Stat(knowledgeDir); err == nil && info.IsDir() {
		fmt.Print(`
[VAULT] Remember: save to the right tier.
  - Universal insights -> global vault (_inbox/)
  - Project-specific insights -> .vault/knowledge/ (local)
`)
	}
}

func staticStopChecklist() string {
	return `[VAULT] Stop capture check:

Before ending this session, confirm you have considered each of these:

- [ ] Did you capture any DECISIONS made this session?
- [ ] Did you capture any PATTERNS discovered?
- [ ] Did you capture any DEBUG INSIGHTS?
- [ ] Did you update the PROJECT INDEX NOTE?
- [ ] Did you capture project-specific knowledge to .vault/knowledge/?

If none apply (trivial session), that is fine -- but confirm it was considered.

Use: vlt vault="Claude" create name="<Title>" path="_inbox/<Title>.md" content="..." silent
`
}
