package loop

import (
	"encoding/json"
	"testing"
)

func TestHasLabel_Found(t *testing.T) {
	labels := []string{"bug", "delivered", "urgent"}
	if !hasLabel(labels, "delivered") {
		t.Error("expected to find 'delivered'")
	}
}

func TestHasLabel_CaseInsensitive(t *testing.T) {
	labels := []string{"Bug", "Delivered", "Urgent"}
	if !hasLabel(labels, "delivered") {
		t.Error("expected case-insensitive match for 'delivered'")
	}
}

func TestHasLabel_NotFound(t *testing.T) {
	labels := []string{"bug", "urgent"}
	if hasLabel(labels, "delivered") {
		t.Error("expected not to find 'delivered'")
	}
}

func TestHasLabel_EmptyLabels(t *testing.T) {
	if hasLabel(nil, "delivered") {
		t.Error("expected false for nil labels")
	}
	if hasLabel([]string{}, "delivered") {
		t.Error("expected false for empty labels")
	}
}

func TestNDIssue_JSONParsing_Array(t *testing.T) {
	input := `[
		{"ID": "PROJ-a1b", "Status": "in_progress", "Labels": ["delivered", "bug"], "Type": "story"},
		{"ID": "PROJ-c3d", "Status": "ready", "Labels": [], "Type": "story"}
	]`

	var issues []ndIssue
	if err := json.Unmarshal([]byte(input), &issues); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].ID != "PROJ-a1b" {
		t.Errorf("expected PROJ-a1b, got %s", issues[0].ID)
	}
	if issues[0].Status != "in_progress" {
		t.Errorf("expected in_progress, got %s", issues[0].Status)
	}
	if !hasLabel(issues[0].Labels, "delivered") {
		t.Error("expected delivered label on first issue")
	}
	if issues[1].Type != "story" {
		t.Errorf("expected story type, got %s", issues[1].Type)
	}
}

func TestNDIssue_JSONParsing_Single(t *testing.T) {
	input := `{"ID": "PROJ-x1y", "Status": "ready", "Labels": ["epic"], "Type": "epic"}`

	var issue ndIssue
	if err := json.Unmarshal([]byte(input), &issue); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if issue.ID != "PROJ-x1y" {
		t.Errorf("expected PROJ-x1y, got %s", issue.ID)
	}
	if issue.Type != "epic" {
		t.Errorf("expected epic, got %s", issue.Type)
	}
}

func TestNDIssue_JSONParsing_EmptyLabels(t *testing.T) {
	input := `{"ID": "TEST-001", "Status": "ready", "Labels": null, "Type": "story"}`

	var issue ndIssue
	if err := json.Unmarshal([]byte(input), &issue); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if issue.Labels != nil {
		t.Errorf("expected nil labels, got %v", issue.Labels)
	}
	if hasLabel(issue.Labels, "delivered") {
		t.Error("hasLabel should return false for nil labels")
	}
}
