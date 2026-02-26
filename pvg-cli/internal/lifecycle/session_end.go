package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/RamXX/paivot-graph/pvg-cli/internal/vaultcfg"
)

// SessionEnd appends a session log entry to the project index note.
// Fire-and-forget: always returns nil (never blocks session end).
func SessionEnd() error {
	// 1. Parse hook input
	var input hookInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		input.CWD, _ = os.Getwd()
	}
	if input.CWD == "" {
		input.CWD, _ = os.Getwd()
	}

	// 2. Detect project name
	project := detectProject(input.CWD)
	today := time.Now().Format("2006-01-02")
	entry := fmt.Sprintf("\n\n## Session log (%s)\n- Session ended normally\n", today)

	// 3. Try vlt first
	v, err := vaultcfg.OpenVault()
	if err == nil {
		_ = v.Append(project, entry, false)
		return nil
	}

	// 4. Fallback to direct file ops
	vaultDir, derr := vaultcfg.VaultDir()
	if derr != nil {
		return nil // silently skip
	}

	projectNote := ""
	candidates := []string{
		filepath.Join(vaultDir, "projects", project+".md"),
		filepath.Join(vaultDir, project+".md"),
	}
	for _, c := range candidates {
		if _, serr := os.Stat(c); serr == nil {
			projectNote = c
			break
		}
	}

	if projectNote != "" {
		f, ferr := os.OpenFile(projectNote, os.O_APPEND|os.O_WRONLY, 0644)
		if ferr == nil {
			_, _ = f.WriteString(entry)
			f.Close()
		}
	}

	return nil
}
