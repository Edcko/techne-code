package skills_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Edcko/techne-code/internal/skills"
	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
)

func TestRegistry_Register(t *testing.T) {
	r := skills.NewRegistry()

	ms := &mockSkill{nameVal: "test1", descVal: "Test skill 1"}

	err := r.Register(ms, skill.SourceBuiltin)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	s, ok := r.Get("test1")
	if !ok {
		t.Fatal("Get failed to find registered skill")
	}

	if s.Name() != "test1" {
		t.Fatalf("Expected name test1, got %s", s.Name())
	}

	err = r.Register(ms, skill.SourceBuiltin)
	if err == nil {
		t.Fatalf("Expected error on duplicate register")
	}
}

func TestRegistry_EnableDisable(t *testing.T) {
	r := skills.NewRegistry()

	ms := &mockSkill{nameVal: "test1", descVal: "Test skill 1"}
	r.Register(ms, skill.SourceBuiltin)

	if !r.IsEnabled("test1") {
		t.Fatal("Skill should be enabled by default")
	}

	if err := r.Disable("test1"); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	if r.IsEnabled("test1") {
		t.Fatal("Skill should be disabled")
	}

	if err := r.Enable("test1"); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	if !r.IsEnabled("test1") {
		t.Fatal("Skill should be enabled")
	}

	if err := r.Disable("nonexistent"); err == nil {
		t.Fatal("Disable should fail for nonexistent skill")
	}
}

func TestRegistry_List(t *testing.T) {
	r := skills.NewRegistry()

	s1 := &mockSkill{nameVal: "alpha", descVal: "First skill"}
	s2 := &mockSkill{nameVal: "beta", descVal: "Second skill"}

	r.Register(s1, skill.SourceBuiltin)
	r.Register(s2, skill.SourceUser)

	list := r.List()
	if len(list) != 2 {
		t.Fatalf("Expected 2 skills, got %d", len(list))
	}

	names := make(map[string]bool)
	for _, info := range list {
		names[info.Name] = true
		if info.Name == "alpha" && info.Source != skill.SourceBuiltin {
			t.Fatal("alpha should have source builtin")
		}
		if info.Name == "beta" && info.Source != skill.SourceUser {
			t.Fatal("beta should have source user")
		}
	}

	if !names["alpha"] || !names["beta"] {
		t.Fatal("Missing expected skills in list")
	}
}

func TestRegistry_ActiveSkills(t *testing.T) {
	r := skills.NewRegistry()

	s1 := &mockSkill{
		nameVal:     "always_on",
		descVal:     "Always active",
		triggerType: skill.TriggerAlways,
	}
	s2 := &mockSkill{
		nameVal:     "file_trigger",
		descVal:     "File triggered",
		triggerType: skill.TriggerFilePattern,
		pattern:     "*.go",
	}
	s3 := &mockSkill{
		nameVal:     "never_active",
		descVal:     "Never active",
		triggerType: skill.TriggerCommand,
		pattern:     "nonexistent",
	}

	r.Register(s1, skill.SourceBuiltin)
	r.Register(s2, skill.SourceBuiltin)
	r.Register(s3, skill.SourceBuiltin)

	ctx := context.Background()

	active := r.ActiveSkills(ctx, skill.SkillContext{})
	if len(active) != 1 {
		t.Fatalf("Expected 1 active skill, got %d", len(active))
	}

	if active[0].Name() != "always_on" {
		t.Fatal("Expected always_on to be active")
	}

	active = r.ActiveSkills(ctx, skill.SkillContext{CurrentFile: "main.go"})
	if len(active) != 2 {
		t.Fatalf("Expected 2 active skills with .go file, got %d", len(active))
	}

	names := make(map[string]bool)
	for _, s := range active {
		names[s.Name()] = true
	}

	if !names["always_on"] || !names["file_trigger"] {
		t.Fatal("Expected both always_on and file_trigger to be active")
	}
}

func TestRegistry_BuildSystemPrompt(t *testing.T) {
	r := skills.NewRegistry()

	s1 := &mockSkill{
		nameVal:      "skill1",
		descVal:      "First skill",
		instructions: "Do this and that.",
	}
	s2 := &mockSkill{
		nameVal:      "skill2",
		descVal:      "Second skill",
		instructions: "Do something else.",
	}

	r.Register(s1, skill.SourceBuiltin)
	r.Register(s2, skill.SourceBuiltin)

	ctx := context.Background()
	prompt := r.BuildSystemPrompt(ctx, skill.SkillContext{})

	if prompt == "" {
		t.Fatal("Expected non-empty prompt")
	}

	if !contains(prompt, "## Skill: skill1") {
		t.Fatal("Expected skill1 section in prompt")
	}

	if !contains(prompt, "## Skill: skill2") {
		t.Fatal("Expected skill2 section in prompt")
	}

	if !contains(prompt, "Do this and that") {
		t.Fatal("Expected skill1 instructions in prompt")
	}
}

type mockSkill struct {
	nameVal      string
	descVal      string
	instructions string
	triggerType  skill.TriggerType
	pattern      string
}

func (s *mockSkill) Name() string         { return s.nameVal }
func (s *mockSkill) Description() string  { return s.descVal }
func (s *mockSkill) Instructions() string { return s.instructions }
func (s *mockSkill) Triggers() []skill.Trigger {
	if s.triggerType == "" {
		return nil
	}
	return []skill.Trigger{{Type: s.triggerType, Pattern: s.pattern}}
}
func (s *mockSkill) Tools() []tool.Tool { return nil }
func (s *mockSkill) IsActive(_ context.Context, ctx skill.SkillContext) bool {
	if s.triggerType == "" || s.triggerType == skill.TriggerAlways {
		return true
	}
	if s.triggerType == skill.TriggerFilePattern {
		return strings.HasSuffix(ctx.CurrentFile, ".go")
	}
	return false
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
