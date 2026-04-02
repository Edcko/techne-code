package skills

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Edcko/techne-code/pkg/skill"
)

type entry struct {
	skill  skill.Skill
	source skill.SkillSource
	config skill.SkillConfig
}

type Registry struct {
	mu      sync.RWMutex
	skills  map[string]entry
	enabled map[string]bool
}

func NewRegistry() *Registry {
	return &Registry{
		skills:  make(map[string]entry),
		enabled: make(map[string]bool),
	}
}

func (r *Registry) Register(s skill.Skill, source skill.SkillSource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := s.Name()
	if _, exists := r.skills[name]; exists {
		return fmt.Errorf("skill %q already registered", name)
	}

	r.skills[name] = entry{
		skill:  s,
		source: source,
		config: skill.SkillConfig{Enabled: true, Priority: 0},
	}
	r.enabled[name] = true
	return nil
}

func (r *Registry) Get(name string) (skill.Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.skills[name]
	if !ok {
		return nil, false
	}
	return e.skill, true
}

func (r *Registry) List() []skill.SkillInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]skill.SkillInfo, 0, len(r.skills))
	for name, e := range r.skills {
		result = append(result, skill.SkillInfo{
			Name:        name,
			Description: e.skill.Description(),
			Triggers:    e.skill.Triggers(),
			Config:      e.config,
			Source:      e.source,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func (r *Registry) ActiveSkills(ctx context.Context, context skill.SkillContext) []skill.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []skill.Skill
	for name, e := range r.skills {
		if !r.enabled[name] {
			continue
		}
		if e.skill.IsActive(ctx, context) {
			result = append(result, e.skill)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		p1 := r.skills[result[i].Name()].config.Priority
		p2 := r.skills[result[j].Name()].config.Priority
		if p1 != p2 {
			return p1 > p2
		}
		return result[i].Name() < result[j].Name()
	})
	return result
}

func (r *Registry) BuildSystemPrompt(ctx context.Context, context skill.SkillContext) string {
	active := r.ActiveSkills(ctx, context)
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

func (r *Registry) Enable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.skills[name]; !ok {
		return fmt.Errorf("skill %q not found", name)
	}
	r.enabled[name] = true
	return nil
}

func (r *Registry) Disable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.skills[name]; !ok {
		return fmt.Errorf("skill %q not found", name)
	}
	r.enabled[name] = false
	return nil
}

func (r *Registry) LoadFromPath(path string, source skill.SkillSource) error {
	skills, err := LoadSkillsFromDir(path, source)
	if err != nil {
		return fmt.Errorf("load skills from %s: %w", path, err)
	}

	for _, s := range skills {
		if err := r.Register(s, source); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) IsEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.enabled[name]
}
