package builtin

import (
	"context"
	"strings"

	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
)

type TypeScriptSkill struct{}

func NewTypeScriptSkill() *TypeScriptSkill {
	return &TypeScriptSkill{}
}

func (s *TypeScriptSkill) Name() string { return "typescript" }

func (s *TypeScriptSkill) Description() string {
	return "TypeScript best practices for strict typing, interfaces, and generics"
}

func (s *TypeScriptSkill) Instructions() string {
	return `TypeScript Guidelines:

1. Type System
- Prefer strict mode with strict: true in tsconfig.json
- Use explicit return types for functions
- Avoid any - use unknown when type is truly unknown
- Prefer type aliases for unions, interfaces for object shapes
- Use const assertions for literal types

2. Interfaces and Types
- Use interfaces for objects that can be extended
- Use type aliases for unions, intersections, and tuples
- Prefer readonly for immutable data
- Use optional chaining (?.) and nullish coalescing (??)
- Define strict prop types in React components

3. Generics
- Use generics for reusable components and functions
- Constrain generic types with extends when possible
- Use inference over explicit typing when obvious
- Avoid generic proliferation - keep it simple

4. Best Practices
- Use enum for sets of related constants
- Prefer async/await over raw Promises
- Use Record<K, V> for record types
- Leverage utility types: Partial, Required, Pick, Omit
- Use template literal types for string patterns

5. Code Organization
- One export per file for components
- Use barrel exports (index.ts) sparingly
- Keep types close to their usage
- Document complex types with JSDoc comments`
}

func (s *TypeScriptSkill) Triggers() []skill.Trigger {
	return []skill.Trigger{
		{Type: skill.TriggerFilePattern, Pattern: "*.ts"},
		{Type: skill.TriggerFilePattern, Pattern: "*.tsx"},
		{Type: skill.TriggerFilePattern, Pattern: "**/*.ts"},
		{Type: skill.TriggerFilePattern, Pattern: "**/*.tsx"},
	}
}

func (s *TypeScriptSkill) Tools() []tool.Tool {
	return nil
}

func (s *TypeScriptSkill) IsActive(ctx context.Context, context skill.SkillContext) bool {
	for _, t := range s.Triggers() {
		switch t.Type {
		case skill.TriggerFilePattern:
			if context.CurrentFile != "" {
				if strings.HasSuffix(context.CurrentFile, ".ts") || strings.HasSuffix(context.CurrentFile, ".tsx") {
					return true
				}
			}
		}
	}
	return false
}
