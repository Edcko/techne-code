package builtin

import (
	"context"
	"strings"

	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
)

type DockerSkill struct{}

func NewDockerSkill() *DockerSkill {
	return &DockerSkill{}
}

func (s *DockerSkill) Name() string { return "docker" }

func (s *DockerSkill) Description() string {
	return "Docker and containerization best practices for builds, compose, and security"
}

func (s *DockerSkill) Instructions() string {
	return `Docker Guidelines:

1. Dockerfile Best Practices
- Use multi-stage builds to keep final images small
- Pin base image versions: node:20-alpine, not node:latest
- Order layers from least to most frequently changed
- Copy dependency files first, then install, then copy source
- Use .dockerignore to exclude node_modules, .git, and build artifacts

2. Multi-Stage Patterns
- Build stage: full SDK for compilation and asset building
- Runtime stage: minimal base (alpine, distroless, scratch)
- Never include build tools, debuggers, or source in production images
- Use COPY --from=build to extract only needed artifacts
- One process per container: use separate containers for app, worker, scheduler

3. Docker Compose
- Use compose for local development environments
- Define healthchecks for service dependencies
- Use named volumes for persistent data, bind mounts for code
- Keep docker-compose.yml for dev, docker-compose.prod.yml for overrides
- Use depends_on with condition: service_healthy for startup order

4. Security
- Run containers as non-root user (USER directive)
- Read-only root filesystem where possible
- Never embed secrets in images — use Docker secrets or env at runtime
- Scan images for vulnerabilities (trivy, grype) in CI
- Minimize installed packages to reduce attack surface

5. Optimization
- Use BuildKit for parallel builds and cache mounts
- Cache pip/npm installs with --mount=type=cache
- Use COPY --link for better cache invalidation
- Tag images with both version and SHA, not just latest
- Prune unused images and volumes regularly`
}

func (s *DockerSkill) Triggers() []skill.Trigger {
	return []skill.Trigger{
		{Type: skill.TriggerCommand, Pattern: "docker"},
		{Type: skill.TriggerFilePattern, Pattern: "Dockerfile"},
		{Type: skill.TriggerFilePattern, Pattern: "Dockerfile.*"},
		{Type: skill.TriggerFilePattern, Pattern: "docker-compose.yml"},
		{Type: skill.TriggerFilePattern, Pattern: "docker-compose.yaml"},
		{Type: skill.TriggerFilePattern, Pattern: "*.dockerfile"},
	}
}

func (s *DockerSkill) Tools() []tool.Tool {
	return nil
}

func (s *DockerSkill) IsActive(ctx context.Context, context skill.SkillContext) bool {
	for _, t := range s.Triggers() {
		switch t.Type {
		case skill.TriggerFilePattern:
			if context.CurrentFile != "" {
				name := context.CurrentFile
				if strings.Contains(name, "Dockerfile") ||
					strings.Contains(name, "docker-compose") ||
					strings.HasSuffix(name, ".dockerfile") {
					return true
				}
			}
		case skill.TriggerCommand:
			if context.ActiveCommand == "docker" {
				return true
			}
		}
	}
	return false
}
