package guard

import (
	"testing"
)

const testVaultDir = "/Users/test/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude"
const testProjectRoot = "/Users/test/workspace/my-project"

func TestCheckFilePath_AllowsNonVaultPaths(t *testing.T) {
	input := HookInput{
		ToolName:  "Edit",
		ToolInput: ToolInput{FilePath: "/tmp/safe.md"},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if !result.Allowed {
		t.Errorf("expected allowed, got blocked: %s", result.Reason)
	}
}

func TestCheckFilePath_AllowsInbox(t *testing.T) {
	input := HookInput{
		ToolName:  "Write",
		ToolInput: ToolInput{FilePath: testVaultDir + "/_inbox/Proposal.md"},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if !result.Allowed {
		t.Errorf("expected allowed for _inbox/, got blocked: %s", result.Reason)
	}
}

func TestCheckFilePath_BlocksProtectedDirs(t *testing.T) {
	for _, folder := range ProtectedFolders {
		t.Run(folder, func(t *testing.T) {
			input := HookInput{
				ToolName:  "Edit",
				ToolInput: ToolInput{FilePath: testVaultDir + "/" + folder + "/Some Note.md"},
			}
			result := Check(testVaultDir, testProjectRoot, input)
			if result.Allowed {
				t.Errorf("expected blocked for %s/, got allowed", folder)
			}
		})
	}
}

func TestCheckBash_AllowsVltCommands(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `vlt vault="Claude" append file="Developer Agent" content="test"`},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if !result.Allowed {
		t.Errorf("expected vlt command allowed, got blocked: %s", result.Reason)
	}
}

func TestCheckBash_AllowsSafeCommands(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: "ls /tmp"},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if !result.Allowed {
		t.Errorf("expected safe command allowed, got blocked: %s", result.Reason)
	}
}

func TestCheckBash_BlocksRedirectToProtectedDir(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `cat > "` + testVaultDir + `/methodology/Developer Agent.md"`},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if result.Allowed {
		t.Errorf("expected blocked for redirect to methodology/, got allowed")
	}
}

func TestCheckBash_BlocksCpToProtectedDir(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `cp /tmp/new.md "` + testVaultDir + `/conventions/Session Operating Mode.md"`},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if result.Allowed {
		t.Errorf("expected blocked for cp to conventions/, got allowed")
	}
}

func TestCheckBash_AllowsReadFromProtectedDir(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `cat "` + testVaultDir + `/methodology/Developer Agent.md"`},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if !result.Allowed {
		t.Errorf("expected read from vault allowed, got blocked: %s", result.Reason)
	}
}

func TestCheckBash_AllowsGrepWithProtectedDirInOutput(t *testing.T) {
	// A grep that reads from a protected dir should NOT be blocked.
	// The old guard would false-positive on this because ">" appears in the command
	// before the protected path (as part of the grep output).
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `grep "pattern" /tmp/file.txt > /tmp/output.txt`},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if !result.Allowed {
		t.Errorf("expected grep with redirect to non-vault path allowed, got blocked: %s", result.Reason)
	}
}

func TestCheckBash_BlocksSedInPlace(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `sed -i 's/old/new/' "` + testVaultDir + `/conventions/Mode.md"`},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if result.Allowed {
		t.Errorf("expected blocked for sed -i on conventions/, got allowed")
	}
}

func TestCheckBash_BlocksAppendRedirect(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `echo "new content" >> "` + testVaultDir + `/methodology/Agent.md"`},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if result.Allowed {
		t.Errorf("expected blocked for >> to methodology/, got allowed")
	}
}

func TestCheckFilePath_EmptyPath(t *testing.T) {
	input := HookInput{
		ToolName:  "Edit",
		ToolInput: ToolInput{FilePath: ""},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if !result.Allowed {
		t.Errorf("expected allowed for empty path, got blocked")
	}
}

func TestCheckUnknownTool_Allowed(t *testing.T) {
	input := HookInput{
		ToolName:  "Grep",
		ToolInput: ToolInput{},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if !result.Allowed {
		t.Errorf("expected unknown tool allowed, got blocked")
	}
}

func TestCheckFilePath_BlocksTrailingDotDot(t *testing.T) {
	// Path traversal: go up from _inbox and back into protected dir.
	input := HookInput{
		ToolName:  "Edit",
		ToolInput: ToolInput{FilePath: testVaultDir + "/_inbox/../methodology/Hack.md"},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if result.Allowed {
		t.Errorf("expected blocked for path traversal via .., got allowed")
	}
}

// --- Project vault tests ---

func TestCheckFilePath_BlocksProjectVault(t *testing.T) {
	input := HookInput{
		ToolName:  "Edit",
		ToolInput: ToolInput{FilePath: testProjectRoot + "/.vault/knowledge/decisions/test.md"},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if result.Allowed {
		t.Errorf("expected blocked for .vault/knowledge/ edit, got allowed")
	}
	if result.Reason != projectVaultBlockMsg {
		t.Errorf("unexpected reason: %s", result.Reason)
	}
}

func TestCheckFilePath_AllowsProjectVaultSettings(t *testing.T) {
	input := HookInput{
		ToolName:  "Edit",
		ToolInput: ToolInput{FilePath: testProjectRoot + "/.vault/knowledge/.settings.yaml"},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if !result.Allowed {
		t.Errorf("expected allowed for .settings.yaml, got blocked: %s", result.Reason)
	}
}

func TestCheckBash_BlocksProjectVaultWrite(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `cat > ` + testProjectRoot + `/.vault/knowledge/patterns/test.md << 'EOF'
content
EOF`},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if result.Allowed {
		t.Errorf("expected blocked for cat > .vault/knowledge/, got allowed")
	}
	if result.Reason != projectVaultBlockMsg {
		t.Errorf("unexpected reason: %s", result.Reason)
	}
}

func TestCheckBash_AllowsProjectVaultVlt(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `vlt vault=".vault/knowledge" create name="test" path="patterns/test.md" content="..." silent`},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if !result.Allowed {
		t.Errorf("expected vlt command for project vault allowed, got blocked: %s", result.Reason)
	}
}

func TestCheckBash_BlocksProjectVaultSedInPlace(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `sed -i 's/old/new/' ` + testProjectRoot + `/.vault/knowledge/patterns/test.md`},
	}
	result := Check(testVaultDir, testProjectRoot, input)
	if result.Allowed {
		t.Errorf("expected blocked for sed -i on .vault/knowledge/, got allowed")
	}
}
