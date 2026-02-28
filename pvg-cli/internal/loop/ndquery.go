package loop

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// WorkCounts holds the counts of issues in each state.
type WorkCounts struct {
	Ready      int
	Delivered  int
	InProgress int
	Blocked    int
}

// ndIssue matches the PascalCase JSON output of nd.
type ndIssue struct {
	ID     string   `json:"ID"`
	Status string   `json:"Status"`
	Labels []string `json:"Labels"`
	Type   string   `json:"Type"`
}

// QueryWorkCounts returns work counts based on mode ("all" or "epic").
func QueryWorkCounts(mode, targetEpic string) (WorkCounts, error) {
	if mode == "epic" && targetEpic != "" {
		return queryEpicCounts(targetEpic)
	}
	return queryAllCounts()
}

// queryAllCounts uses nd subcommands to gather counts across the whole backlog.
func queryAllCounts() (WorkCounts, error) {
	var wc WorkCounts

	// Ready issues
	readyIssues, err := runND("ready", "--json")
	if err == nil {
		wc.Ready = len(readyIssues)
	}

	// In-progress issues (includes delivered -- we separate below)
	ipIssues, err := runND("list", "--status", "in_progress", "--json")
	if err == nil {
		for _, issue := range ipIssues {
			if hasLabel(issue.Labels, "delivered") {
				wc.Delivered++
			} else {
				wc.InProgress++
			}
		}
	}

	// Blocked issues
	blockedIssues, err := runND("blocked", "--json")
	if err == nil {
		wc.Blocked = len(blockedIssues)
	}

	return wc, nil
}

// queryEpicCounts uses nd children to count work within a specific epic.
func queryEpicCounts(epicID string) (WorkCounts, error) {
	var wc WorkCounts

	issues, err := runND("children", epicID, "--json")
	if err != nil {
		return wc, fmt.Errorf("query epic children: %w", err)
	}

	for _, issue := range issues {
		switch strings.ToLower(issue.Status) {
		case "ready":
			wc.Ready++
		case "in_progress":
			if hasLabel(issue.Labels, "delivered") {
				wc.Delivered++
			} else {
				wc.InProgress++
			}
		case "blocked":
			wc.Blocked++
		// closed/done issues are not counted
		}
	}

	return wc, nil
}

// ValidateEpic checks that an epic ID exists and is a valid epic.
func ValidateEpic(epicID string) error {
	issues, err := runND("show", epicID, "--json")
	if err != nil {
		return fmt.Errorf("epic %s not found: %w", epicID, err)
	}
	if len(issues) == 0 {
		return fmt.Errorf("epic %s not found", epicID)
	}
	issue := issues[0]
	if !strings.EqualFold(issue.Type, "epic") {
		return fmt.Errorf("%s is not an epic (type: %s)", epicID, issue.Type)
	}
	return nil
}

// runND executes an nd command and parses JSON output.
// Returns empty slice (not error) when nd outputs nothing.
func runND(args ...string) ([]ndIssue, error) {
	cmd := exec.Command("nd", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nd %s: %w", strings.Join(args, " "), err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" || trimmed == "[]" || trimmed == "null" {
		return nil, nil
	}

	// nd may return a single object or an array
	var issues []ndIssue
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &issues); err != nil {
			return nil, fmt.Errorf("parse nd output: %w", err)
		}
	} else {
		var single ndIssue
		if err := json.Unmarshal([]byte(trimmed), &single); err != nil {
			return nil, fmt.Errorf("parse nd output: %w", err)
		}
		issues = []ndIssue{single}
	}

	return issues, nil
}

// hasLabel checks if a label exists in a slice (case-insensitive).
func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, target) {
			return true
		}
	}
	return false
}
