package builtin

import (
	"context"
	"strings"

	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
)

type PythonSkill struct{}

func NewPythonSkill() *PythonSkill {
	return &PythonSkill{}
}

func (s *PythonSkill) Name() string { return "python" }

func (s *PythonSkill) Description() string {
	return "Python development best practices for typing, testing, and modern frameworks"
}

func (s *PythonSkill) Instructions() string {
	return `Python Guidelines:

1. Type System
- Use type hints on all function signatures (parameters and return types)
- Prefer modern union syntax X | Y over Union[X, Y] (Python 3.10+)
- Use typing.Protocol for structural subtyping over ABC for nominal
- Prefer dataclasses or Pydantic models over raw dicts
- Use Literal, TypedDict, and Final for precision where needed

2. Project Structure
- Use pyproject.toml as the single source of config (PEP 621)
- Prefer uv or poetry for dependency management
- Keep environments isolated: never install globally
- Use src/ layout for publishable packages
- Separate concerns: models, services, repositories

3. Error Handling
- Use specific exception types, never bare except
- Chain exceptions with raise X from Y for context
- Use try/except around the minimal code that can fail
- Prefer returning Result patterns or Optional over raising for control flow
- Log exceptions with structured context, not just str(e)

4. Testing with Pytest
- Write tests in test_*.py files alongside or in tests/ directory
- Use fixtures for setup, parametrize for data-driven tests
- Mock at boundaries (external APIs, databases), not internals
- Aim for fast unit tests, separate slow integration tests with marks
- Use conftest.py for shared fixtures, keep them minimal

5. Framework Patterns
- FastAPI: use dependency injection, Pydantic models for request/response
- Django: keep views thin, business logic in services or models
- Use async/await for I/O-bound work, not for CPU-bound
- Prefer environment variables for config with pydantic-settings`
}

func (s *PythonSkill) Triggers() []skill.Trigger {
	return []skill.Trigger{
		{Type: skill.TriggerFilePattern, Pattern: "*.py"},
		{Type: skill.TriggerFilePattern, Pattern: "**/*.py"},
	}
}

func (s *PythonSkill) Tools() []tool.Tool {
	return nil
}

func (s *PythonSkill) IsActive(ctx context.Context, context skill.SkillContext) bool {
	for _, t := range s.Triggers() {
		switch t.Type {
		case skill.TriggerFilePattern:
			if context.CurrentFile != "" && strings.HasSuffix(context.CurrentFile, ".py") {
				return true
			}
		}
	}
	return false
}
