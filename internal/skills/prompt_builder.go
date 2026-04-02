package skills

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Edcko/techne-code/pkg/skill"
)

type PromptBuilder struct {
	registry *Registry
}

func NewPromptBuilder(registry *Registry) *PromptBuilder {
	return &PromptBuilder{registry: registry}
}

func (b *PromptBuilder) BuildSystemPrompt(ctx context.Context, context skill.SkillContext) string {
	active := b.registry.ActiveSkills(ctx, context)
	if len(active) == 0 {
		return ""
	}

	var sections []string
	for _, s := range active {
		instructions := strings.TrimSpace(s.Instructions())
		if instructions == "" {
			continue
		}
		sections = append(sections, fmt.Sprintf("## Skill: %s\n\n%s", s.Name(), instructions))
	}

	if len(sections) == 0 {
		return ""
	}

	return "\n\n" + strings.Join(sections, "\n\n") + "\n"
}

func (b *PromptBuilder) BuildSkillList() []string {
	skills := b.registry.List()
	result := make([]string, len(skills))
	for i, s := range skills {
		result[i] = fmt.Sprintf("- %s: %s", s.Name, s.Description)
	}
	sort.Strings(result)
	return result
}

func (b *PromptBuilder) BuildActiveSkillList(ctx context.Context, context skill.SkillContext) []string {
	active := b.registry.ActiveSkills(ctx, context)
	result := make([]string, len(active))
	for i, s := range active {
		result[i] = s.Name()
	}
	sort.Strings(result)
	return result
}
