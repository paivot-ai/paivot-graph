package guard

import (
	"testing"
)

const testVaultDir = "/Users/test/Library/Mobile Documents/iCloud~md~obsidian/Documents/Claude"

func TestCheckFilePath_AllowsNonVaultPaths(t *testing.T) {
	input := HookInput{
		ToolName:  "Edit",
		ToolInput: ToolInput{FilePath: "/tmp/safe.md"},
	}
	result := Check(testVaultDir, input)
	if !result.Allowed {
		t.Errorf("expected allowed, got blocked: %s", result.Reason)
	}
}

func TestCheckFilePath_AllowsInbox(t *testing.T) {
	input := HookInput{
		ToolName:  "Write",
		ToolInput: ToolInput{FilePath: testVaultDir + "/_inbox/Proposal.md"},
	}
	result := Check(testVaultDir, input)
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
			result := Check(testVaultDir, input)
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
	result := Check(testVaultDir, input)
	if !result.Allowed {
		t.Errorf("expected vlt command allowed, got blocked: %s", result.Reason)
	}
}

func TestCheckBash_AllowsSafeCommands(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: "ls /tmp"},
	}
	result := Check(testVaultDir, input)
	if !result.Allowed {
		t.Errorf("expected safe command allowed, got blocked: %s", result.Reason)
	}
}

func TestCheckBash_BlocksRedirectToProtectedDir(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `cat > "` + testVaultDir + `/methodology/Developer Agent.md"`},
	}
	result := Check(testVaultDir, input)
	if result.Allowed {
		t.Errorf("expected blocked for redirect to methodology/, got allowed")
	}
}

func TestCheckBash_BlocksCpToProtectedDir(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `cp /tmp/new.md "` + testVaultDir + `/conventions/Session Operating Mode.md"`},
	}
	result := Check(testVaultDir, input)
	if result.Allowed {
		t.Errorf("expected blocked for cp to conventions/, got allowed")
	}
}

func TestCheckBash_AllowsReadFromProtectedDir(t *testing.T) {
	input := HookInput{
		ToolName:  "Bash",
		ToolInput: ToolInput{Command: `cat "` + testVaultDir + `/methodology/Developer Agent.md"`},
	}
	result := Check(testVaultDir, input)
	if !result.Allowed {
		t.Errorf("expected read from vault allowed, got blocked: %s", result.Reason)
	}
}

func TestCheckFilePath_EmptyPath(t *testing.T) {
	input := HookInput{
		ToolName:  "Edit",
		ToolInput: ToolInput{FilePath: ""},
	}
	result := Check(testVaultDir, input)
	if !result.Allowed {
		t.Errorf("expected allowed for empty path, got blocked")
	}
}

func TestCheckUnknownTool_Allowed(t *testing.T) {
	input := HookInput{
		ToolName:  "Grep",
		ToolInput: ToolInput{},
	}
	result := Check(testVaultDir, input)
	if !result.Allowed {
		t.Errorf("expected unknown tool allowed, got blocked")
	}
}
