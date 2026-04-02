package builtin

import (
	"github.com/Edcko/techne-code/pkg/skill"
)

func AllSkills() []skill.Skill {
	return []skill.Skill{
		NewGoEngineerSkill(),
		NewSecuritySkill(),
		NewTypeScriptSkill(),
	}
}

func RegisterAll(registry skill.SkillRegistry) error {
	for _, s := range AllSkills() {
		if err := registry.Register(s, skill.SourceBuiltin); err != nil {
			return err
		}
	}
	return nil
}
