package skill

import (
	"context"

	"github.com/Edcko/techne-code/pkg/tool"
)

type TriggerType string

const (
	TriggerFilePattern TriggerType = "file_pattern"
	TriggerCommand     TriggerType = "command"
	TriggerAlways      TriggerType = "always"
)

type Trigger struct {
	Type    TriggerType `json:"type"`
	Pattern string      `json:"pattern"`
}

type Skill interface {
	Name() string
	Description() string
	Instructions() string
	Triggers() []Trigger
	Tools() []tool.Tool
	IsActive(ctx context.Context, context SkillContext) bool
}

type SkillContext struct {
	CurrentFile   string
	UserMessage   string
	ActiveCommand string
}

type SkillConfig struct {
	Enabled  bool              `json:"enabled"`
	Params   map[string]string `json:"params"`
	Priority int               `json:"priority"`
}

type SkillInfo struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Triggers    []Trigger   `json:"triggers"`
	Config      SkillConfig `json:"config"`
	Source      SkillSource `json:"source"`
}

type SkillSource string

const (
	SourceBuiltin SkillSource = "builtin"
	SourceUser    SkillSource = "user"
	SourceProject SkillSource = "project"
)

type SkillRegistry interface {
	Register(s Skill, source SkillSource) error
	Get(name string) (Skill, bool)
	List() []SkillInfo
	ActiveSkills(ctx context.Context, context SkillContext) []Skill
	BuildSystemPrompt(ctx context.Context, context SkillContext) string
	Enable(name string) error
	Disable(name string) error
	LoadFromPath(path string, source SkillSource) error
}
