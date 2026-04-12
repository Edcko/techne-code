package builtin

import (
	"context"
	"strings"

	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
)

type DatabaseSkill struct{}

func NewDatabaseSkill() *DatabaseSkill {
	return &DatabaseSkill{}
}

func (s *DatabaseSkill) Name() string { return "database" }

func (s *DatabaseSkill) Description() string {
	return "Database best practices for SQL optimization, migrations, and schema design"
}

func (s *DatabaseSkill) Instructions() string {
	return `Database Guidelines:

1. Schema Design
- Normalize to 3NF by default, denormalize deliberately for read performance
- Use appropriate data types: don't store numbers as strings
- Add NOT NULL constraints by default, make nullable explicit and documented
- Use foreign keys with ON DELETE behavior matching domain rules
- Add CHECK constraints for domain validation at the database level

2. Indexing Strategy
- Index columns used in WHERE, JOIN, and ORDER BY clauses
- Use composite indexes when queries filter on multiple columns
- Put the most selective column first in composite indexes
- Avoid over-indexing: every index slows writes
- Use EXPLAIN/EXPLAIN ANALYZE to verify index usage, not guess

3. Query Optimization
- Select only needed columns, avoid SELECT *
- Use JOINs over subqueries when the optimizer handles them better
- Paginate with cursor-based approach for large result sets
- Use prepared statements to prevent SQL injection and enable plan caching
- Avoid N+1: fetch related data with JOINs or batch queries

4. Migration Patterns
- Migrations must be forward-only and reversible (up/down)
- Never modify a migration after it has been applied
- Keep migrations small and focused: one concern per migration
- Add columns as nullable first, backfill, then add NOT NULL constraint
- Use transactions in migrations for DDL where the database supports it

5. ORM Best Practices
- Understand what SQL your ORM generates
- Use eager loading to avoid N+1, lazy loading only when justified
- Keep business logic out of models, use service or repository layer
- Map ORM entities to domain types at the boundary
- Always use parameterized queries, never interpolate user input`
}

func (s *DatabaseSkill) Triggers() []skill.Trigger {
	return []skill.Trigger{
		{Type: skill.TriggerFilePattern, Pattern: "*.sql"},
		{Type: skill.TriggerFilePattern, Pattern: "**/*.sql"},
		{Type: skill.TriggerFilePattern, Pattern: "schema.prisma"},
		{Type: skill.TriggerFilePattern, Pattern: "**/schema.prisma"},
	}
}

func (s *DatabaseSkill) Tools() []tool.Tool {
	return nil
}

func (s *DatabaseSkill) IsActive(ctx context.Context, context skill.SkillContext) bool {
	for _, t := range s.Triggers() {
		switch t.Type {
		case skill.TriggerFilePattern:
			if context.CurrentFile != "" {
				file := context.CurrentFile
				if strings.HasSuffix(file, ".sql") || strings.HasSuffix(file, "schema.prisma") {
					return true
				}
			}
		}
	}
	return false
}
