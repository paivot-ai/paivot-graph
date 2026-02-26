package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSettings_NoFile(t *testing.T) {
	s := loadSettings("/nonexistent/.settings.yaml")
	if len(s) != 0 {
		t.Errorf("expected empty map for missing file, got %d entries", len(s))
	}
}

func TestLoadSettings_ParsesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".settings.yaml")
	content := "session_start_max_notes: 5\nauto_capture: false\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := loadSettings(path)
	if s["session_start_max_notes"] != "5" {
		t.Errorf("expected 5, got %q", s["session_start_max_notes"])
	}
	if s["auto_capture"] != "false" {
		t.Errorf("expected false, got %q", s["auto_capture"])
	}
}

func TestWriteSettings_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", ".settings.yaml")

	settings := map[string]string{
		"session_start_max_notes": "20",
		"staleness_days":          "60",
	}

	if err := writeSettings(path, settings); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "session_start_max_notes: 20") {
		t.Error("expected session_start_max_notes: 20 in output")
	}
	if !strings.Contains(content, "staleness_days: 60") {
		t.Error("expected staleness_days: 60 in output")
	}
}

func TestWriteAndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".settings.yaml")

	original := map[string]string{
		"auto_capture":            "true",
		"session_start_max_notes": "15",
		"staleness_days":          "45",
	}

	if err := writeSettings(path, original); err != nil {
		t.Fatal(err)
	}

	loaded := loadSettings(path)
	for k, v := range original {
		if loaded[k] != v {
			t.Errorf("key %q: expected %q, got %q", k, v, loaded[k])
		}
	}
}

func TestLoadSettings_SkipsComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".settings.yaml")
	content := "# This is a comment\nsession_start_max_notes: 5\n# Another comment\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := loadSettings(path)
	if len(s) != 1 {
		t.Errorf("expected 1 entry (comments should be skipped), got %d", len(s))
	}
	if s["session_start_max_notes"] != "5" {
		t.Errorf("expected 5, got %q", s["session_start_max_notes"])
	}
}
