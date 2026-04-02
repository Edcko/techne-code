package config

// DefaultConfig returns the built-in default configuration.
// These defaults are used when no configuration files or environment
// variables are present.
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider: "anthropic",
		DefaultModel:    "claude-sonnet-4-20250514",
		Providers:       make(map[string]ProviderConfig),
		Permissions: PermissionsConfig{
			Mode:         "interactive",
			AllowedTools: []string{},
		},
		Options: OptionsConfig{
			ContextPaths:   []string{"AGENTS.md", ".cursorrules", "CLAUDE.md"},
			MaxBashTimeout: 120000, // 2 minutes
			MaxOutputChars: 20000,
			DataDirectory:  ".techne/",
		},
	}
}
