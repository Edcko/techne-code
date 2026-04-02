package skills

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
	"gopkg.in/yaml.v3"
)

func LoadSkillsFromDir(dir string, source skill.SkillSource) ([]skill.Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var skills []skill.Skill
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		path := filepath.Join(dir, name)
		s, err := ParseSkillFile(path, source)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if s != nil {
			skills = append(skills, s)
		}
	}

	return skills, nil
}

func ParseSkillFile(path string, source skill.SkillSource) (skill.Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseSkill(data, source)
}

func ParseSkill(data []byte, source skill.SkillSource) (skill.Skill, error) {
	frontmatter, content, err := ExtractFrontmatter(data)
	if err != nil {
		return nil, err
	}

	if frontmatter == nil {
		return nil, nil
	}

	metadata, err := parseFrontmatter(frontmatter)
	if err != nil {
		return nil, err
	}

	name, ok := metadata["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("skill missing required field: name")
	}

	description, _ := metadata["description"].(string)
	instructions := strings.TrimSpace(string(content))

	var triggers []skill.Trigger
	if rawTriggers, ok := metadata["triggers"].([]interface{}); ok {
		for _, t := range rawTriggers {
			if triggerMap, ok := t.(map[interface{}]interface{}); ok {
				trigger := skill.Trigger{}
				if typ, ok := triggerMap["type"].(string); ok {
					trigger.Type = skill.TriggerType(typ)
				}
				if pattern, ok := triggerMap["pattern"].(string); ok {
					trigger.Pattern = pattern
				}
				triggers = append(triggers, trigger)
			}
		}
	}

	return &parsedSkill{
		name:         name,
		description:  description,
		instructions: instructions,
		triggers:     triggers,
		source:       source,
	}, nil
}

func ExtractFrontmatter(data []byte) ([]byte, []byte, error) {
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, data, nil
	}

	end := bytes.Index(data[4:], []byte("\n---\n"))
	if end == -1 {
		return nil, nil, fmt.Errorf("unclosed frontmatter")
	}

	frontmatter := data[4 : 4+end]
	content := data[4+end+5:]
	return frontmatter, content, nil
}

func parseFrontmatter(data []byte) (map[string]interface{}, error) {
	if len(data) == 0 {
		return make(map[string]interface{}), nil
	}
	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

type parsedSkill struct {
	name         string
	description  string
	instructions string
	triggers     []skill.Trigger
	source       skill.SkillSource
}

func (s *parsedSkill) Name() string              { return s.name }
func (s *parsedSkill) Description() string       { return s.description }
func (s *parsedSkill) Instructions() string      { return s.instructions }
func (s *parsedSkill) Triggers() []skill.Trigger { return s.triggers }
func (s *parsedSkill) Tools() []tool.Tool        { return nil }

func (s *parsedSkill) IsActive(ctx context.Context, context skill.SkillContext) bool {
	for _, t := range s.triggers {
		switch t.Type {
		case skill.TriggerAlways:
			return true
		case skill.TriggerFilePattern:
			if context.CurrentFile != "" && MatchPattern(t.Pattern, context.CurrentFile) {
				return true
			}
		case skill.TriggerCommand:
			if context.ActiveCommand != "" && t.Pattern == context.ActiveCommand {
				return true
			}
		}
	}
	return false
}

func MatchPattern(pattern, path string) bool {
	matched, _ := filepath.Match(pattern, path)
	if matched {
		return true
	}

	if strings.Contains(pattern, "**") {
		regex := globToRegex(pattern)
		matched, _ = filepath.Match(regex, path)
		return matched
	}

	return false
}

func globToRegex(pattern string) string {
	result := strings.ReplaceAll(pattern, ".", "\\.")
	result = strings.ReplaceAll(result, "**", ".*")
	result = strings.ReplaceAll(result, "*", "[^/]*")
	result = strings.ReplaceAll(result, "?", ".")
	return result
}
