package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/RamXX/paivot-graph/pvg-cli/internal/dispatcher"
)

// userPromptInput matches the JSON Claude Code sends to UserPromptSubmit hooks.
type userPromptInput struct {
	Prompt string `json:"prompt"`
}

// triggerPhrases are case-insensitive phrases that activate dispatcher mode.
var triggerPhrases = []string{
	"use paivot",
	"paivot this",
	"run paivot",
	"engage paivot",
	"with paivot",
}

// UserPromptSubmit detects Paivot trigger phrases in user prompts and
// auto-enables dispatcher mode. Outputs JSON with additionalContext when
// dispatcher mode is activated.
func UserPromptSubmit() error {
	var input userPromptInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return nil // fail-open
	}

	if !containsTriggerPhrase(input.Prompt) {
		return nil // silent exit
	}

	cwd, _ := os.Getwd()
	if cwd == "" {
		return nil
	}

	// Enable dispatcher mode
	if err := dispatcher.On(cwd); err != nil {
		// Log but don't block
		fmt.Fprintf(os.Stderr, "pvg: failed to enable dispatcher mode: %v\n", err)
		return nil
	}

	// Output hook response with context reinforcement
	resp := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName": "UserPromptSubmit",
			"additionalContext": "DISPATCHER MODE ACTIVE. You are a coordinator only. " +
				"Do NOT write D&F files, source code, or stories directly. Spawn the appropriate agent instead. " +
				"BLT QUESTIONING PROTOCOL: When a BLT agent (BA, Designer, Architect) returns output, " +
				"check for a QUESTIONS_FOR_USER block BEFORE checking for a document. " +
				"The agent's first output in any D&F engagement MUST be questions, not a document. " +
				"If the agent produced a document on its first turn without any questioning round, " +
				"this is a protocol violation -- re-spawn the agent with an explicit reminder to ask questions first.",
		},
	}
	return json.NewEncoder(os.Stdout).Encode(resp)
}

// negationPrefixes are words that negate the trigger when they appear
// immediately before the trigger phrase.
var negationPrefixes = []string{
	"don't ", "dont ", "do not ", "not ", "no ", "without ",
	"never ", "stop ", "disable ", "skip ",
}

// containsTriggerPhrase checks if the prompt contains any Paivot trigger phrase,
// excluding negated forms like "don't use paivot" or "not paivot".
func containsTriggerPhrase(prompt string) bool {
	lower := strings.ToLower(prompt)
	for _, phrase := range triggerPhrases {
		idx := strings.Index(lower, phrase)
		if idx < 0 {
			continue
		}

		// Check if the phrase is preceded by a negation word.
		prefix := lower[:idx]
		negated := false
		for _, neg := range negationPrefixes {
			if strings.HasSuffix(prefix, neg) {
				negated = true
				break
			}
		}
		if !negated {
			return true
		}
	}
	return false
}
