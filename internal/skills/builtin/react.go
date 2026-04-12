package builtin

import (
	"context"
	"strings"

	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
)

type ReactSkill struct{}

func NewReactSkill() *ReactSkill {
	return &ReactSkill{}
}

func (s *ReactSkill) Name() string { return "react" }

func (s *ReactSkill) Description() string {
	return "React development patterns for component design, hooks, and state management"
}

func (s *ReactSkill) Instructions() string {
	return `React Guidelines:

1. Component Design
- Prefer function components with hooks over class components
- One component per file, named exports by default
- Keep components small and focused on a single responsibility
- Use composition over props drilling or deep nesting
- Separate container/presentational: logic vs UI components

2. Hooks Rules
- Only call hooks at the top level, never inside loops or conditions
- Only call hooks from React function components or custom hooks
- Name custom hooks with use prefix (useAuth, useDebounce)
- Keep useEffect focused: one effect per concern
- Clean up effects: return cleanup function for subscriptions and timers

3. State Management
- Use useState for local UI state
- Lift state up to the nearest common ancestor when shared
- Use useReducer for complex state with multiple sub-values
- Consider Zustand or Jotai for global state, avoid Redux for new projects
- Never store derived state — compute it during render

4. Performance
- Let React Compiler handle memoization (React 19+)
- Avoid premature useMemo/useCallback unless profiling proves the need
- Use lazy loading for route-level components with React.lazy
- Keep render output deterministic for the same props and state
- Profile before optimizing — measure, then act

5. Testing
- Use Vitest + React Testing Library for component tests
- Test behavior (what the user sees), not implementation details
- Prefer userEvent over fireEvent for realistic interactions
- Mock at the network layer (MSW) over mocking components
- Use screen.getByRole and accessible queries, not test IDs`
}

func (s *ReactSkill) Triggers() []skill.Trigger {
	return []skill.Trigger{
		{Type: skill.TriggerFilePattern, Pattern: "*.tsx"},
		{Type: skill.TriggerFilePattern, Pattern: "*.jsx"},
		{Type: skill.TriggerFilePattern, Pattern: "**/*.tsx"},
		{Type: skill.TriggerFilePattern, Pattern: "**/*.jsx"},
	}
}

func (s *ReactSkill) Tools() []tool.Tool {
	return nil
}

func (s *ReactSkill) IsActive(ctx context.Context, context skill.SkillContext) bool {
	for _, t := range s.Triggers() {
		switch t.Type {
		case skill.TriggerFilePattern:
			if context.CurrentFile != "" {
				if strings.HasSuffix(context.CurrentFile, ".tsx") || strings.HasSuffix(context.CurrentFile, ".jsx") {
					return true
				}
			}
		}
	}
	return false
}
