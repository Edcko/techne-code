package builtin

import (
	"context"

	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
)

type SecuritySkill struct{}

func NewSecuritySkill() *SecuritySkill {
	return &SecuritySkill{}
}

func (s *SecuritySkill) Name() string { return "security" }

func (s *SecuritySkill) Description() string {
	return "Security best practices - always active, provides security-conscious coding guidelines"
}

func (s *SecuritySkill) Instructions() string {
	return `Security Guidelines - ALWAYS follow these rules:

1. Secrets and Credentials
- NEVER commit secrets, API keys, or credentials to version control
- NEVER log sensitive information (passwords, tokens, personal data)
- Use environment variables or secure secret management for credentials
- Mask or redact sensitive values in logs and error messages
- Assume all user input is potentially malicious

2. Input Validation
- Validate all user input at system boundaries
- Use allowlists over blocklists for validation
- Sanitize input before using in commands, queries, or file paths
- Escape output appropriately for the target context
- Never trust client-side validation alone

3. File Operations
- Validate and sanitize file paths to prevent path traversal
- Check file permissions before operations
- Use secure file handling practices (close files, handle errors)
- Never execute files from untrusted sources

4. Command Execution
- Avoid shell command injection by using proper argument passing
- Never concatenate user input directly into shell commands
- Use the full path to executables when possible
- Limit command execution timeout to prevent hangs

5. Data Protection
- Use encryption for sensitive data at rest and in transit
- Implement proper access controls and authorization checks
- Follow the principle of least privilege
- Keep dependencies updated to avoid known vulnerabilities`
}

func (s *SecuritySkill) Triggers() []skill.Trigger {
	return []skill.Trigger{
		{Type: skill.TriggerAlways, Pattern: ""},
	}
}

func (s *SecuritySkill) Tools() []tool.Tool {
	return nil
}

func (s *SecuritySkill) IsActive(ctx context.Context, context skill.SkillContext) bool {
	return true
}
