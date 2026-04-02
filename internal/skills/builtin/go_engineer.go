package builtin

import (
	"context"

	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
)

type GoEngineerSkill struct{}

func NewGoEngineerSkill() *GoEngineerSkill {
	return &GoEngineerSkill{}
}

func (s *GoEngineerSkill) Name() string { return "go_engineer" }

func (s *GoEngineerSkill) Description() string {
	return "Go engineering best practices for idiomatic code, error handling, and testing patterns"
}

func (s *GoEngineerSkill) Instructions() string {
	return `When writing Go code, follow these principles:

1. Idiomatic Go
- Use meaningful names: short for local variables, longer for exported names
- Prefer composition over inheritance
- Return errors as values, not exceptions
- Use defer for cleanup operations
- Avoid init() functions when possible

2. Error Handling
- Always check errors immediately after the call
- Wrap errors with context using fmt.Errorf("operation failed: %w", err)
- Never ignore errors with _ unless explicitly documented
- Use sentinel errors for expected conditions
- Create custom error types for domain-specific errors

3. Testing Patterns
- Write table-driven tests for multiple scenarios
- Use t.Run() for subtests to organize test cases
- Prefer httptest for HTTP handler testing
- Use build tags for integration tests
- Mock interfaces, not concrete types

4. Concurrency
- Use errgroup for concurrent operations that need error handling
- Prefer channels for communication, mutexes for state protection
- Always close channels from the sender side
- Use context for cancellation propagation
- Avoid goroutine leaks with proper cleanup

5. Code Organization
- Group imports: standard library, external packages, internal packages
- Keep interfaces small and focused
- Use internal/ for implementation details
- Document all exported types and functions
- Use gofmt and goimports consistently`
}

func (s *GoEngineerSkill) Triggers() []skill.Trigger {
	return []skill.Trigger{
		{Type: skill.TriggerFilePattern, Pattern: "*.go"},
		{Type: skill.TriggerFilePattern, Pattern: "**/*.go"},
	}
}

func (s *GoEngineerSkill) Tools() []tool.Tool {
	return nil
}

func (s *GoEngineerSkill) IsActive(ctx context.Context, context skill.SkillContext) bool {
	for _, t := range s.Triggers() {
		switch t.Type {
		case skill.TriggerFilePattern:
			if context.CurrentFile != "" && matchPattern(t.Pattern, context.CurrentFile) {
				return true
			}
		}
	}
	return false
}

func matchPattern(pattern, path string) bool {
	if pattern == "*.go" {
		return len(path) > 3 && path[len(path)-3:] == ".go"
	}
	return false
}
