// Package guard implements the PreToolUse scope guard for knowledge governance.
//
// It reads Claude Code's PreToolUse hook JSON from stdin, determines if the
// operation targets a protected vault directory, and exits 2 to block or 0
// to allow.
//
// Protected directories: methodology/, conventions/, decisions/, patterns/,
// debug/, concepts/, projects/, people/
//
// Allowed: _inbox/ (proposals and captures), _templates/, anything outside vault
//
// Special handling:
//   - Edit/Write: checks file_path against protected dirs
//   - Bash: checks command for redirects/cp/mv targeting protected dirs
//   - Bash with vlt: always allowed (vlt is the intended mechanism)
package guard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HookInput matches the JSON structure Claude Code sends to PreToolUse hooks.
type HookInput struct {
	ToolName  string    `json:"tool_name"`
	ToolInput ToolInput `json:"tool_input"`
}

// ToolInput contains the parameters of the tool being called.
type ToolInput struct {
	FilePath string `json:"file_path"`
	Command  string `json:"command"`
}

// ProtectedFolders are vault subdirectories that require proposal workflow.
var ProtectedFolders = []string{
	"methodology",
	"conventions",
	"decisions",
	"patterns",
	"debug",
	"concepts",
	"projects",
	"people",
}

// Result represents the guard's decision.
type Result struct {
	Allowed bool
	Reason  string
}

// Check reads hook input and returns whether the operation should be allowed.
func Check(vaultDir string, input HookInput) Result {
	switch input.ToolName {
	case "Edit", "Write":
		return checkFilePath(vaultDir, input.ToolInput.FilePath)
	case "Bash":
		return checkBashCommand(vaultDir, input.ToolInput.Command)
	default:
		return Result{Allowed: true}
	}
}

// ParseInput reads and parses the hook JSON from stdin.
func ParseInput() (HookInput, error) {
	var input HookInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		return input, fmt.Errorf("failed to parse hook input: %w", err)
	}
	return input, nil
}

func checkFilePath(vaultDir, filePath string) Result {
	if filePath == "" {
		return Result{Allowed: true}
	}

	for _, folder := range ProtectedFolders {
		protected := filepath.Join(vaultDir, folder) + "/"
		if strings.HasPrefix(filePath, protected) {
			return Result{
				Allowed: false,
				Reason: fmt.Sprintf(
					"BLOCKED: Direct modification of system-scoped vault content in %s/.\n\n"+
						"System vault directories are protected by knowledge governance.\n"+
						"To change system notes:\n"+
						"  1. Run /vault-evolve to create a proposal\n"+
						"  2. Run /vault-triage to review and apply it\n\n"+
						"Only _inbox/ is writable directly (for proposals and new captures).",
					folder),
			}
		}
	}

	return Result{Allowed: true}
}

func checkBashCommand(vaultDir, command string) Result {
	if command == "" {
		return Result{Allowed: true}
	}

	// vlt commands are the intended mechanism -- always allow
	trimmed := strings.TrimSpace(command)
	if strings.HasPrefix(trimmed, "vlt ") || strings.HasPrefix(trimmed, "vlt\t") {
		return Result{Allowed: true}
	}

	// Check for shell redirects and file operations targeting protected dirs
	for _, folder := range ProtectedFolders {
		protected := filepath.Join(vaultDir, folder)
		if !strings.Contains(command, protected) {
			continue
		}

		// Check for write-like patterns
		writePatterns := []string{">", ">>", "tee ", "cp ", "mv ", "cat >"}
		for _, pattern := range writePatterns {
			if strings.Contains(command, pattern) {
				return Result{
					Allowed: false,
					Reason: fmt.Sprintf(
						"BLOCKED: Bash command targets protected vault directory %s/.\n\n"+
							"System vault directories are protected by knowledge governance.\n"+
							"To change system notes:\n"+
							"  1. Run /vault-evolve to create a proposal\n"+
							"  2. Run /vault-triage to review and apply it\n\n"+
							"Only _inbox/ is writable directly (for proposals and new captures).",
						folder),
				}
			}
		}
	}

	return Result{Allowed: true}
}
