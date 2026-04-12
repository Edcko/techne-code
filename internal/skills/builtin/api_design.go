package builtin

import (
	"context"
	"strings"

	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
)

type APIDesignSkill struct{}

func NewAPIDesignSkill() *APIDesignSkill {
	return &APIDesignSkill{}
}

func (s *APIDesignSkill) Name() string { return "api_design" }

func (s *APIDesignSkill) Description() string {
	return "API design best practices for REST, OpenAPI, and service interfaces"
}

func (s *APIDesignSkill) Instructions() string {
	return `API Design Guidelines:

1. RESTful Principles
- Use nouns for resources, not verbs: /users not /getUsers
- Use HTTP methods semantically: GET (read), POST (create), PUT (replace), PATCH (update), DELETE
- Return appropriate status codes: 201 for creation, 204 for deletion, 404 for not found
- Use plural nouns for collections: /users/{id}/orders
- Keep URLs predictable and consistent across the entire API

2. Request and Response Design
- Accept JSON request bodies with Content-Type: application/json
- Return consistent envelope: { data, error, meta } pattern
- Use snake_case for JSON field names (or camelCase, but pick one and be consistent)
- Version the API in the URL path: /v1/users, not in headers
- Support pagination with cursor or offset, include total count in meta

3. Error Handling
- Use a consistent error format: { code, message, details }
- Return machine-readable error codes, not just human messages
- Use Problem Details (RFC 7807) for standard error responses
- Distinguish client errors (4xx) from server errors (5xx)
- Never expose stack traces or internal details in production errors

4. OpenAPI and Documentation
- Write OpenAPI specs first for public APIs, keep them in sync
- Use operationId for every endpoint — it becomes the function name in generated clients
- Document all status codes, not just the happy path
- Include examples in schema definitions
- Validate request/response against the spec in tests

5. Versioning and Evolution
- Additive changes only: new optional fields, new endpoints
- Never remove or rename existing fields without a version bump
- Use deprecation headers and documentation before removal
- Support at most 2 active versions simultaneously
- Communicate breaking changes with migration guides and timelines`
}

func (s *APIDesignSkill) Triggers() []skill.Trigger {
	return []skill.Trigger{
		{Type: skill.TriggerFilePattern, Pattern: "openapi.yaml"},
		{Type: skill.TriggerFilePattern, Pattern: "openapi.yml"},
		{Type: skill.TriggerFilePattern, Pattern: "swagger.json"},
		{Type: skill.TriggerFilePattern, Pattern: "swagger.yaml"},
		{Type: skill.TriggerFilePattern, Pattern: "*.proto"},
		{Type: skill.TriggerFilePattern, Pattern: "**/*.proto"},
	}
}

func (s *APIDesignSkill) Tools() []tool.Tool {
	return nil
}

func (s *APIDesignSkill) IsActive(ctx context.Context, context skill.SkillContext) bool {
	for _, t := range s.Triggers() {
		switch t.Type {
		case skill.TriggerFilePattern:
			if context.CurrentFile != "" {
				file := context.CurrentFile
				if strings.Contains(file, "openapi") ||
					strings.Contains(file, "swagger") ||
					strings.HasSuffix(file, ".proto") {
					return true
				}
			}
		}
	}
	return false
}
