package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	koanf "github.com/knadh/koanf/v2"
)

const (
	// EnvPrefix is the prefix for environment variables.
	EnvPrefix = "TECHNE_"

	// EnvDelimiter is the delimiter used to separate nested keys in env vars.
	// Example: TECHNE_PERMISSIONS_MODE maps to permissions.mode
	EnvDelimiter = "_"
)

// Load loads configuration from all sources with the following priority (lowest to highest):
// 1. Built-in defaults
// 2. Global config file (~/.config/techne/techne.json)
// 3. Project config file (.techne/techne.json)
// 4. Environment variables (TECHNE_*)
//
// The projectDir parameter is used to locate the project config file.
// If projectDir is empty, the project config file is not loaded.
func Load(projectDir string) (*Config, error) {
	k := koanf.New(".")

	// 1. Load defaults
	defaults := DefaultConfig()
	if err := loadDefaults(k, defaults); err != nil {
		return nil, fmt.Errorf("failed to load defaults: %w", err)
	}

	// 2. Load global config file
	globalPath := globalConfigPath()
	if globalPath != "" {
		if _, err := os.Stat(globalPath); err == nil {
			if err := loadFile(k, globalPath); err != nil {
				return nil, fmt.Errorf("failed to load global config: %w", err)
			}
		}
	}

	// 3. Load project config file
	if projectDir != "" {
		projectPath := filepath.Join(projectDir, ".techne", "techne.json")
		if _, err := os.Stat(projectPath); err == nil {
			if err := loadFile(k, projectPath); err != nil {
				return nil, fmt.Errorf("failed to load project config: %w", err)
			}
		}
	}

	// 4. Load environment variables
	if err := loadEnv(k); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// Unmarshal into config struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Initialize empty maps if nil
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}
	if cfg.Options.ContextPaths == nil {
		cfg.Options.ContextPaths = []string{}
	}
	if cfg.Permissions.AllowedTools == nil {
		cfg.Permissions.AllowedTools = []string{}
	}

	// Expand environment variables in API keys
	expandAPIKeys(&cfg)

	return &cfg, nil
}

// LoadFromFile loads configuration from a specific file path.
// This is useful for testing or loading config from a custom location.
// Note: This does not apply defaults or load environment variables.
func LoadFromFile(path string) (*Config, error) {
	k := koanf.New(".")

	if err := loadFile(k, path); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Initialize empty maps if nil
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}

	expandAPIKeys(&cfg)

	return &cfg, nil
}

// globalConfigPath returns the path to the global config file.
func globalConfigPath() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "techne", "techne.json")
	}
	return ""
}

// loadDefaults loads the default configuration into koanf using confmap provider.
func loadDefaults(k *koanf.Koanf, defaults *Config) error {
	// Convert defaults to a flat map with dot notation
	defaultsMap := map[string]interface{}{
		"default_provider":          defaults.DefaultProvider,
		"default_model":             defaults.DefaultModel,
		"permissions.mode":          defaults.Permissions.Mode,
		"permissions.allowed_tools": defaults.Permissions.AllowedTools,
		"options.context_paths":     defaults.Options.ContextPaths,
		"options.max_bash_timeout":  defaults.Options.MaxBashTimeout,
		"options.max_output_chars":  defaults.Options.MaxOutputChars,
		"options.data_directory":    defaults.Options.DataDirectory,
	}

	return k.Load(confmap.Provider(defaultsMap, "."), nil)
}

// loadFile loads configuration from a JSON file.
func loadFile(k *koanf.Koanf, path string) error {
	return k.Load(file.Provider(path), json.Parser())
}

// loadEnv loads configuration from environment variables.
// Variables are expected in the format TECHNE_KEY (e.g., TECHNE_DEFAULT_PROVIDER).
// For nested keys, use double underscore: TECHNE_PERMISSIONS__MODE -> permissions.mode
func loadEnv(k *koanf.Koanf) error {
	// Use __ as the delimiter for nested keys
	// TECHNE_DEFAULT_PROVIDER -> default_provider
	// TECHNE_PERMISSIONS__MODE -> permissions.mode
	return k.Load(env.Provider(EnvPrefix, "__", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, EnvPrefix))
	}), nil)
}

// expandAPIKeys expands environment variable references in API keys.
// Supports both ${VAR} and $VAR syntax.
func expandAPIKeys(cfg *Config) {
	for name, provider := range cfg.Providers {
		provider.APIKey = expandEnvVar(provider.APIKey)
		cfg.Providers[name] = provider
	}
}

// expandEnvVar expands environment variable references in a string.
// Supports both ${VAR} and $VAR syntax.
func expandEnvVar(s string) string {
	if s == "" {
		return s
	}

	// Handle ${VAR} syntax
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		envVar := s[2 : len(s)-1]
		return os.Getenv(envVar)
	}

	// Handle $VAR syntax
	if strings.HasPrefix(s, "$") && !strings.HasPrefix(s, "${") {
		envVar := s[1:]
		// Handle cases like $VAR with more text after
		if idx := strings.Index(envVar, " "); idx > 0 {
			envVar = envVar[:idx]
		}
		return os.Getenv(envVar)
	}

	return s
}
