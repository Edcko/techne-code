package skills_test

import (
	"testing"

	"github.com/Edcko/techne-code/internal/skills"
	"github.com/Edcko/techne-code/pkg/skill"
)

func TestParseSkill(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "valid skill",
			input: `---
name: test-skill
description: A test skill
triggers:
  - type: file_pattern
    pattern: "*.go"
custom: value
---

# Instructions

This is a test skill.`,
			wantErr: false,
		},
		{
			name: "missing name",
			input: `---
description: No name
---

No name here.`,
			wantErr: true,
		},
		{
			name: "no frontmatter",
			input: `# Just markdown

No frontmatter here.`,
			wantErr: false,
		},
		{
			name: "empty frontmatter",
			input: `---
---

No content in frontmatter`,
			wantErr: true,
		},
		{
			name: "complex triggers",
			input: `---
name: multi-trigger
description: Multiple triggers
triggers:
  - type: file_pattern
    pattern: "*.ts"
  - type: file_pattern
    pattern: "*.tsx"
  - type: always
---

Has multiple triggers.`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := skills.ParseSkill([]byte(tt.input), skill.SourceBuiltin)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error for %s", tt.name)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for %s: %v", tt.name, err)
			}

			if s == nil {
				return
			}

			if s.Name() == "" {
				t.Fatalf("Expected non-empty name for %s", tt.name)
			}
		})
	}
}
