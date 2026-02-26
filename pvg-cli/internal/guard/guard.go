// Package guard implements the PreToolUse scope guard for knowledge governance.
//
// It reads Claude Code's PreToolUse hook JSON from stdin, determines if the
// operation targets a protected vault directory, and exits 2 to block or 0
// to allow. Two layers of protection:
//
// Layer 1 -- System vault (global Obsidian vault "Claude"):
//
//	Protected: methodology/, conventions/, decisions/, patterns/,
//	           debug/, concepts/, projects/, people/
//	Allowed:   _inbox/ (proposals and captures), _templates/
//	Mechanism: checkFilePath blocks Edit/Write, checkBashCommand blocks
//	           shell redirects/cp/mv. vlt commands are always allowed.
//
// Layer 2 -- Project vault (.vault/knowledge/ in project root):
//
//	Protected: all files under .vault/knowledge/
//	Exception: .settings.yaml (managed by pvg settings binary)
//	Mechanism: checkProjectVault blocks Edit/Write, checkBashProjectVault
//	           blocks shell writes. vlt commands are always allowed.
//
// Why vlt is the required mechanism: vlt uses advisory file locking
// (.vlt.lock) to serialize concurrent agent writes. Direct file I/O
// bypasses this lock, risking data loss when multiple agents run
// simultaneously.
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
// projectRoot is the CWD of the invoking process (used to resolve .vault/knowledge/).
func Check(vaultDir, projectRoot string, input HookInput) Result {
	switch input.ToolName {
	case "Edit", "Write":
		if r := checkFilePath(vaultDir, input.ToolInput.FilePath); !r.Allowed {
			return r
		}
		return checkProjectVault(projectRoot, input.ToolInput.FilePath)
	case "Bash":
		if r := checkBashCommand(vaultDir, input.ToolInput.Command); !r.Allowed {
			return r
		}
		return checkBashProjectVault(projectRoot, input.ToolInput.Command)
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

// normalizePath resolves symlinks and cleans a file path for reliable prefix
// comparison. Falls back to filepath.Clean if symlink resolution fails (e.g.
// the path does not exist yet).
func normalizePath(p string) string {
	if p == "" {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		// Path may not exist yet (Write to a new file). Clean it at least.
		return filepath.Clean(p)
	}
	return resolved
}

func systemBlockMsg(folder string) string {
	return fmt.Sprintf(
		"BLOCKED: Direct modification of system-scoped vault content in %s/.\n\n"+
			"System vault directories are protected by knowledge governance.\n"+
			"To change system notes:\n"+
			"  1. Run /vault-evolve to create a proposal\n"+
			"  2. Run /vault-triage to review and apply it\n\n"+
			"Only _inbox/ is writable directly (for proposals and new captures).",
		folder)
}

func checkFilePath(vaultDir, filePath string) Result {
	if filePath == "" {
		return Result{Allowed: true}
	}

	// Normalize both paths so symlinks and case tricks don't bypass the guard.
	normVault := normalizePath(vaultDir)
	normFile := normalizePath(filePath)

	for _, folder := range ProtectedFolders {
		protected := filepath.Join(normVault, folder) + "/"
		if strings.HasPrefix(normFile, protected) {
			return Result{Allowed: false, Reason: systemBlockMsg(folder)}
		}
	}

	// Also check the raw (non-resolved) path in case the file doesn't exist yet
	// and EvalSymlinks fell back to Clean -- the vault dir itself may resolve.
	if normFile != filePath {
		cleanFile := filepath.Clean(filePath)
		for _, folder := range ProtectedFolders {
			protected := filepath.Join(normVault, folder) + "/"
			if strings.HasPrefix(cleanFile, protected) {
				return Result{Allowed: false, Reason: systemBlockMsg(folder)}
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

	normVault := normalizePath(vaultDir)

	// Check for write operations targeting protected dirs.
	// Key improvement: verify the protected path appears in the write
	// *destination* (after the redirect operator or as a later argument),
	// not just anywhere in the command string.
	for _, folder := range ProtectedFolders {
		protected := filepath.Join(normVault, folder)

		if !strings.Contains(command, protected) {
			continue
		}

		// Check redirect operators: the protected path must appear
		// AFTER the operator to be a write destination.
		for _, op := range []string{">>", ">"} {
			if idx := strings.Index(command, op); idx >= 0 {
				afterOp := command[idx:]
				if strings.Contains(afterOp, protected) {
					return Result{Allowed: false, Reason: systemBlockMsg(folder)}
				}
			}
		}

		// Check file operation commands where protected path is the
		// destination (typically the last path argument).
		destPatterns := []string{
			"tee ", "cp ", "mv ", "cat >",
			"sed -i", "perl -pi", "install ", "rsync ", "dd ", "patch ",
		}
		for _, pattern := range destPatterns {
			if strings.Contains(command, pattern) && strings.Contains(command, protected) {
				return Result{Allowed: false, Reason: systemBlockMsg(folder)}
			}
		}
	}

	return Result{Allowed: true}
}

const projectVaultBlockMsg = "BLOCKED: Direct modification of project vault. " +
	"Use vlt vault=\"<path>\" commands instead. " +
	"vlt provides locking for concurrent agent safety."

// projectVaultPath is the relative path segment that identifies project vault files.
const projectVaultPath = "/.vault/knowledge/"

func checkProjectVault(projectRoot, filePath string) Result {
	if filePath == "" || projectRoot == "" {
		return Result{Allowed: true}
	}

	normRoot := normalizePath(projectRoot)
	normFile := normalizePath(filePath)

	vaultPrefix := normRoot + projectVaultPath
	if !strings.HasPrefix(normFile, vaultPrefix) {
		// Also check cleaned but non-resolved path (file may not exist)
		cleanFile := filepath.Clean(filePath)
		if !strings.HasPrefix(cleanFile, vaultPrefix) {
			return Result{Allowed: true}
		}
	}

	// Allow .settings.yaml -- managed by pvg settings (our own binary)
	if filepath.Base(filePath) == ".settings.yaml" {
		return Result{Allowed: true}
	}

	return Result{Allowed: false, Reason: projectVaultBlockMsg}
}

func checkBashProjectVault(projectRoot, command string) Result {
	if command == "" || projectRoot == "" {
		return Result{Allowed: true}
	}

	trimmed := strings.TrimSpace(command)
	if strings.HasPrefix(trimmed, "vlt ") || strings.HasPrefix(trimmed, "vlt\t") {
		return Result{Allowed: true}
	}

	normRoot := normalizePath(projectRoot)
	vaultSegment := normRoot + projectVaultPath

	if !strings.Contains(command, vaultSegment) {
		return Result{Allowed: true}
	}

	// Check redirect operators: protected path must be after the operator.
	for _, op := range []string{">>", ">"} {
		if idx := strings.Index(command, op); idx >= 0 {
			if strings.Contains(command[idx:], vaultSegment) {
				return Result{Allowed: false, Reason: projectVaultBlockMsg}
			}
		}
	}

	// Check write commands with protected path.
	writePatterns := []string{
		"tee ", "cp ", "mv ", "cat >", "mkdir ",
		"sed -i", "perl -pi", "install ", "rsync ", "dd ", "patch ",
	}
	for _, pattern := range writePatterns {
		if strings.Contains(command, pattern) {
			return Result{Allowed: false, Reason: projectVaultBlockMsg}
		}
	}

	return Result{Allowed: true}
}
