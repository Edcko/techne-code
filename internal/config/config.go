// Package config provides configuration management for Techne Code.
// It supports loading configuration from multiple sources with a clear priority order:
// defaults → global file → project file → environment variables.
package config

// Config is the root configuration for Techne Code.
type Config struct {
	// DefaultProvider is the name of the default AI provider to use.
	// Can reference a provider from the Providers map or be a built-in provider name.
	DefaultProvider string `koanf:"default_provider"`

	// DefaultModel is the default model to use for the provider.
	DefaultModel string `koanf:"default_model"`

	// Providers contains per-provider configuration.
	// Keys are provider names (e.g., "anthropic", "openai", "ollama").
	Providers map[string]ProviderConfig `koanf:"providers"`

	// Permissions contains the permission system configuration.
	Permissions PermissionsConfig `koanf:"permissions"`

	// Skills contains the skill system configuration.
	Skills SkillsConfig `koanf:"skills"`

	// Options contains general configuration options.
	Options OptionsConfig `koanf:"options"`
}

// ProviderConfig contains per-provider configuration.
type ProviderConfig struct {
	// Type is the provider type: "anthropic", "openai", or "ollama".
	Type string `koanf:"type"`

	// APIKey is the API key for the provider.
	// Can reference environment variables using ${VAR} or $VAR syntax.
	APIKey string `koanf:"api_key"`

	// BaseURL is the base URL for the provider's API.
	// Useful for custom endpoints or Ollama.
	BaseURL string `koanf:"base_url"`

	// Models is a list of available models for this provider.
	Models []string `koanf:"models"`
}

// PermissionsConfig contains the permission system configuration.
type PermissionsConfig struct {
	// Mode determines how permissions are handled:
	// - "interactive": ask user for each permission (default)
	// - "auto_allow": automatically allow all tool executions
	// - "auto_deny": automatically deny all tool executions
	Mode string `koanf:"mode"`

	// AllowedTools is a list of tools that are automatically approved.
	// These tools bypass the permission check.
	AllowedTools []string `koanf:"allowed_tools"`
}

// SkillsConfig contains the skill system configuration.
type SkillsConfig struct {
	// Enabled is a list of skill names to explicitly enable.
	// If empty, all skills are enabled by default.
	Enabled []string `koanf:"enabled"`

	// Disabled is a list of skill names to explicitly disable.
	// These skills will not be activated even if their triggers match.
	Disabled []string `koanf:"disabled"`

	// UserSkillsPath is the path to user-defined skills directory.
	// Default: ~/.config/techne/skills/
	UserSkillsPath string `koanf:"user_skills_path"`

	// ProjectSkillsPath is the path to project-specific skills directory.
	// Default: .techne/skills/
	ProjectSkillsPath string `koanf:"project_skills_path"`
}

// OptionsConfig contains general configuration options.
type OptionsConfig struct {
	// ContextPaths is a list of file paths to include in the system prompt.
	// These files provide context about the project to the AI.
	// Example: ["AGENTS.md", ".cursorrules", "CLAUDE.md"]
	ContextPaths []string `koanf:"context_paths"`

	// MaxBashTimeout is the maximum allowed timeout for bash commands in milliseconds.
	// Default: 120000 (2 minutes)
	MaxBashTimeout int `koanf:"max_bash_timeout"`

	// MaxOutputChars is the maximum number of characters to display in output.
	// Output exceeding this will be truncated.
	// Default: 20000
	MaxOutputChars int `koanf:"max_output_chars"`

	// DataDirectory is the directory where Techne stores its data.
	// Default: ".techne/"
	DataDirectory string `koanf:"data_directory"`
}
