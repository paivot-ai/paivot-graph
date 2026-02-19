package main

import (
	"testing"
)

func TestExtractFrontmatter(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantYAML  string
		wantStart int
		wantFound bool
	}{
		{
			name:      "standard frontmatter",
			text:      "---\ntype: decision\nstatus: active\n---\n\n# Note\n",
			wantYAML:  "type: decision\nstatus: active",
			wantStart: 4,
			wantFound: true,
		},
		{
			name:      "no frontmatter",
			text:      "# Just a heading\n\nSome content.\n",
			wantYAML:  "",
			wantStart: 0,
			wantFound: false,
		},
		{
			name:      "unclosed frontmatter",
			text:      "---\ntype: broken\n# No closing delimiter\n",
			wantYAML:  "",
			wantStart: 0,
			wantFound: false,
		},
		{
			name:      "empty frontmatter",
			text:      "---\n---\n\n# Empty props\n",
			wantYAML:  "",
			wantStart: 2,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml, start, found := extractFrontmatter(tt.text)
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
			if yaml != tt.wantYAML {
				t.Errorf("yaml = %q, want %q", yaml, tt.wantYAML)
			}
			if start != tt.wantStart {
				t.Errorf("bodyStart = %d, want %d", start, tt.wantStart)
			}
		})
	}
}

func TestFrontmatterGetList(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		key  string
		want []string
	}{
		{
			name: "inline list",
			yaml: "type: decision\ntags: [project, important]\nstatus: active",
			key:  "tags",
			want: []string{"project", "important"},
		},
		{
			name: "block list",
			yaml: "type: decision\ntags:\n  - project\n  - important\nstatus: active",
			key:  "tags",
			want: []string{"project", "important"},
		},
		{
			name: "inline list with quotes",
			yaml: "aliases: [\"My Note\", 'Alt Name']",
			key:  "aliases",
			want: []string{"My Note", "Alt Name"},
		},
		{
			name: "single value returned as list",
			yaml: "tags: solo-tag\nstatus: active",
			key:  "tags",
			want: []string{"solo-tag"},
		},
		{
			name: "key not found",
			yaml: "type: decision\nstatus: active",
			key:  "tags",
			want: nil,
		},
		{
			name: "block list with blank lines",
			yaml: "aliases:\n  - Name One\n\n  - Name Two\nstatus: active",
			key:  "aliases",
			want: []string{"Name One", "Name Two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := frontmatterGetList(tt.yaml, tt.key)

			if tt.want == nil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %d items %v, want %d items %v", len(got), got, len(tt.want), tt.want)
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("item[%d] = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}

func TestFrontmatterGetValue(t *testing.T) {
	yaml := "type: decision\nstatus: active\ncreated: 2024-01-15"

	tests := []struct {
		key       string
		wantVal   string
		wantFound bool
	}{
		{"type", "decision", true},
		{"status", "active", true},
		{"missing", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			val, found := frontmatterGetValue(yaml, tt.key)
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
			if val != tt.wantVal {
				t.Errorf("val = %q, want %q", val, tt.wantVal)
			}
		})
	}
}

func TestFrontmatterRemoveKey(t *testing.T) {
	tests := []struct {
		name string
		text string
		key  string
		want string
	}{
		{
			name: "remove simple key",
			text: "---\ntype: decision\nstatus: active\ncreated: 2024-01-15\n---\n\n# Note\n",
			key:  "status",
			want: "---\ntype: decision\ncreated: 2024-01-15\n---\n\n# Note\n",
		},
		{
			name: "remove block list",
			text: "---\ntype: note\ntags:\n  - a\n  - b\nstatus: active\n---\n\n# Note\n",
			key:  "tags",
			want: "---\ntype: note\nstatus: active\n---\n\n# Note\n",
		},
		{
			name: "key not found returns original",
			text: "---\ntype: note\n---\n\n# Note\n",
			key:  "missing",
			want: "---\ntype: note\n---\n\n# Note\n",
		},
		{
			name: "no frontmatter returns original",
			text: "# Note\n\nContent.\n",
			key:  "type",
			want: "# Note\n\nContent.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := frontmatterRemoveKey(tt.text, tt.key)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFrontmatterReadAll(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "with frontmatter",
			text: "---\ntype: decision\nstatus: active\n---\n\n# Note\n",
			want: "---\ntype: decision\nstatus: active\n---",
		},
		{
			name: "no frontmatter",
			text: "# Note\n\nContent.\n",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := frontmatterReadAll(tt.text)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
