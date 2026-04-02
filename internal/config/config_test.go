package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test default values
	if cfg.DefaultProvider != "anthropic" {
		t.Errorf("expected default_provider to be 'anthropic', got %q", cfg.DefaultProvider)
	}

	if cfg.DefaultModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected default_model to be 'claude-sonnet-4-20250514', got %q", cfg.DefaultModel)
	}

	if cfg.Permissions.Mode != "interactive" {
		t.Errorf("expected permissions.mode to be 'interactive', got %q", cfg.Permissions.Mode)
	}

	if cfg.Options.MaxBashTimeout != 120000 {
		t.Errorf("expected options.max_bash_timeout to be 120000, got %d", cfg.Options.MaxBashTimeout)
	}

	if cfg.Options.MaxOutputChars != 20000 {
		t.Errorf("expected options.max_output_chars to be 20000, got %d", cfg.Options.MaxOutputChars)
	}

	if cfg.Options.DataDirectory != ".techne/" {
		t.Errorf("expected options.data_directory to be '.techne/', got %q", cfg.Options.DataDirectory)
	}

	expectedContextPaths := []string{"AGENTS.md", ".cursorrules", "CLAUDE.md"}
	if len(cfg.Options.ContextPaths) != len(expectedContextPaths) {
		t.Errorf("expected %d context_paths, got %d", len(expectedContextPaths), len(cfg.Options.ContextPaths))
	}
}

func TestLoadWithNoFiles(t *testing.T) {
	// Create a temp directory with no config files
	tmpDir := t.TempDir()

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should return defaults
	if cfg.DefaultProvider != "anthropic" {
		t.Errorf("expected default_provider to be 'anthropic', got %q", cfg.DefaultProvider)
	}

	if cfg.DefaultModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected default_model to be 'claude-sonnet-4-20250514', got %q", cfg.DefaultModel)
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("TECHNE_DEFAULT_PROVIDER", "openai")
	os.Setenv("TECHNE_DEFAULT_MODEL", "gpt-4")
	defer func() {
		os.Unsetenv("TECHNE_DEFAULT_PROVIDER")
		os.Unsetenv("TECHNE_DEFAULT_MODEL")
	}()

	tmpDir := t.TempDir()

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Environment variables should override defaults
	if cfg.DefaultProvider != "openai" {
		t.Errorf("expected default_provider to be 'openai', got %q", cfg.DefaultProvider)
	}

	if cfg.DefaultModel != "gpt-4" {
		t.Errorf("expected default_model to be 'gpt-4', got %q", cfg.DefaultModel)
	}
}

func TestLoadWithProjectConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .techne directory and config file
	techneDir := filepath.Join(tmpDir, ".techne")
	if err := os.Mkdir(techneDir, 0755); err != nil {
		t.Fatalf("failed to create .techne directory: %v", err)
	}

	configContent := `{
		"default_provider": "ollama",
		"default_model": "llama2"
	}`
	configPath := filepath.Join(techneDir, "techne.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Project config should override defaults
	if cfg.DefaultProvider != "ollama" {
		t.Errorf("expected default_provider to be 'ollama', got %q", cfg.DefaultProvider)
	}

	if cfg.DefaultModel != "llama2" {
		t.Errorf("expected default_model to be 'llama2', got %q", cfg.DefaultModel)
	}
}

func TestValidateCatchesInvalidProviderType(t *testing.T) {
	cfg := &Config{
		DefaultProvider: "anthropic",
		DefaultModel:    "claude-sonnet-4-20250514",
		Providers: map[string]ProviderConfig{
			"myprovider": {
				Type: "invalid_type",
			},
		},
		Permissions: PermissionsConfig{
			Mode: "interactive",
		},
		Options: OptionsConfig{
			MaxBashTimeout: 120000,
			MaxOutputChars: 20000,
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for invalid provider type, got nil")
	}
}

func TestValidateCatchesMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "missing default_provider",
			config: &Config{
				DefaultProvider: "",
				DefaultModel:    "claude-sonnet-4-20250514",
				Permissions:     PermissionsConfig{Mode: "interactive"},
				Options: OptionsConfig{
					MaxBashTimeout: 120000,
					MaxOutputChars: 20000,
				},
			},
			wantErr: true,
		},
		{
			name: "missing default_model",
			config: &Config{
				DefaultProvider: "anthropic",
				DefaultModel:    "",
				Permissions:     PermissionsConfig{Mode: "interactive"},
				Options: OptionsConfig{
					MaxBashTimeout: 120000,
					MaxOutputChars: 20000,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid max_bash_timeout",
			config: &Config{
				DefaultProvider: "anthropic",
				DefaultModel:    "claude-sonnet-4-20250514",
				Permissions:     PermissionsConfig{Mode: "interactive"},
				Options: OptionsConfig{
					MaxBashTimeout: 0,
					MaxOutputChars: 20000,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid max_output_chars",
			config: &Config{
				DefaultProvider: "anthropic",
				DefaultModel:    "claude-sonnet-4-20250514",
				Permissions:     PermissionsConfig{Mode: "interactive"},
				Options: OptionsConfig{
					MaxBashTimeout: 120000,
					MaxOutputChars: -1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid permissions mode",
			config: &Config{
				DefaultProvider: "anthropic",
				DefaultModel:    "claude-sonnet-4-20250514",
				Permissions:     PermissionsConfig{Mode: "invalid"},
				Options: OptionsConfig{
					MaxBashTimeout: 120000,
					MaxOutputChars: 20000,
				},
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &Config{
				DefaultProvider: "anthropic",
				DefaultModel:    "claude-sonnet-4-20250514",
				Permissions:     PermissionsConfig{Mode: "interactive"},
				Options: OptionsConfig{
					MaxBashTimeout: 120000,
					MaxOutputChars: 20000,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvVarExpansionBraceSyntax(t *testing.T) {
	os.Setenv("TEST_API_KEY", "my-secret-key-123")
	defer os.Unsetenv("TEST_API_KEY")

	cfg := &Config{
		Providers: map[string]ProviderConfig{
			"anthropic": {
				APIKey: "${TEST_API_KEY}",
			},
		},
	}

	expandAPIKeys(cfg)

	if cfg.Providers["anthropic"].APIKey != "my-secret-key-123" {
		t.Errorf("expected API key to be 'my-secret-key-123', got %q", cfg.Providers["anthropic"].APIKey)
	}
}

func TestEnvVarExpansionDollarSyntax(t *testing.T) {
	os.Setenv("TEST_API_KEY2", "my-secret-key-456")
	defer os.Unsetenv("TEST_API_KEY2")

	cfg := &Config{
		Providers: map[string]ProviderConfig{
			"openai": {
				APIKey: "$TEST_API_KEY2",
			},
		},
	}

	expandAPIKeys(cfg)

	if cfg.Providers["openai"].APIKey != "my-secret-key-456" {
		t.Errorf("expected API key to be 'my-secret-key-456', got %q", cfg.Providers["openai"].APIKey)
	}
}

func TestEnvVarExpansionNoExpansion(t *testing.T) {
	cfg := &Config{
		Providers: map[string]ProviderConfig{
			"anthropic": {
				APIKey: "plain-api-key",
			},
		},
	}

	expandAPIKeys(cfg)

	if cfg.Providers["anthropic"].APIKey != "plain-api-key" {
		t.Errorf("expected API key to remain 'plain-api-key', got %q", cfg.Providers["anthropic"].APIKey)
	}
}

func TestEnvVarExpansionEmptyVar(t *testing.T) {
	cfg := &Config{
		Providers: map[string]ProviderConfig{
			"anthropic": {
				APIKey: "",
			},
		},
	}

	expandAPIKeys(cfg)

	if cfg.Providers["anthropic"].APIKey != "" {
		t.Errorf("expected API key to be empty, got %q", cfg.Providers["anthropic"].APIKey)
	}
}

func TestLoadFromFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.json")
	configContent := `{
		"default_provider": "openai",
		"default_model": "gpt-4-turbo",
		"providers": {
			"openai": {
				"type": "openai",
				"api_key": "test-key"
			}
		}
	}`
	if err := os.WriteFile(tmpFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.DefaultProvider != "openai" {
		t.Errorf("expected default_provider to be 'openai', got %q", cfg.DefaultProvider)
	}

	if cfg.DefaultModel != "gpt-4-turbo" {
		t.Errorf("expected default_model to be 'gpt-4-turbo', got %q", cfg.DefaultModel)
	}

	if cfg.Providers["openai"].Type != "openai" {
		t.Errorf("expected provider type to be 'openai', got %q", cfg.Providers["openai"].Type)
	}
}

func TestConfigPriority(t *testing.T) {
	// Create temp directory with project config
	tmpDir := t.TempDir()
	techneDir := filepath.Join(tmpDir, ".techne")
	if err := os.Mkdir(techneDir, 0755); err != nil {
		t.Fatalf("failed to create .techne directory: %v", err)
	}

	// Project config sets provider to "ollama"
	projectConfig := `{"default_provider": "ollama", "default_model": "llama2"}`
	if err := os.WriteFile(filepath.Join(techneDir, "techne.json"), []byte(projectConfig), 0644); err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	// Environment variable sets provider to "openai" (should win)
	os.Setenv("TECHNE_DEFAULT_PROVIDER", "openai")
	os.Setenv("TECHNE_DEFAULT_MODEL", "gpt-4")
	defer func() {
		os.Unsetenv("TECHNE_DEFAULT_PROVIDER")
		os.Unsetenv("TECHNE_DEFAULT_MODEL")
	}()

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Env var should override project config
	if cfg.DefaultProvider != "openai" {
		t.Errorf("expected default_provider to be 'openai' (from env), got %q", cfg.DefaultProvider)
	}

	if cfg.DefaultModel != "gpt-4" {
		t.Errorf("expected default_model to be 'gpt-4' (from env), got %q", cfg.DefaultModel)
	}
}

func TestValidateDefaultProviderReference(t *testing.T) {
	// When providers are defined and default_provider doesn't exist in providers
	// but is a valid built-in type, it should be valid
	cfg := &Config{
		DefaultProvider: "anthropic", // Built-in type, not in Providers map
		DefaultModel:    "claude-sonnet-4-20250514",
		Providers: map[string]ProviderConfig{
			"ollama": {
				Type: "ollama",
			},
		},
		Permissions: PermissionsConfig{Mode: "interactive"},
		Options: OptionsConfig{
			MaxBashTimeout: 120000,
			MaxOutputChars: 20000,
		},
	}

	err := Validate(cfg)
	if err != nil {
		t.Errorf("expected validation to pass for built-in provider type, got error: %v", err)
	}
}
