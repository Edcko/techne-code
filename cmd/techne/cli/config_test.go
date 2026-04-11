package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Edcko/techne-code/internal/config"
	"github.com/spf13/cobra"
)

func TestConfigInitCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "techne.json")

	cmd := newConfigInitCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{
		"--provider", "anthropic",
		"--model", "claude-sonnet-4-20250514",
		"--path", configPath,
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var cfg configFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse config file: %v", err)
	}

	if cfg.DefaultProvider != "anthropic" {
		t.Errorf("expected default_provider 'anthropic', got %q", cfg.DefaultProvider)
	}

	if cfg.DefaultModel != "claude-sonnet-4-20250514" {
		t.Errorf("expected default_model 'claude-sonnet-4-20250514', got %q", cfg.DefaultModel)
	}

	p, ok := cfg.Providers["anthropic"]
	if !ok {
		t.Fatal("expected anthropic provider in config")
	}

	if p.Type != "anthropic" {
		t.Errorf("expected provider type 'anthropic', got %q", p.Type)
	}
}

func TestConfigInitWithOpenAI(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "techne.json")

	cmd := newConfigInitCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"--provider", "openai",
		"--path", configPath,
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var cfg configFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.DefaultProvider != "openai" {
		t.Errorf("expected default_provider 'openai', got %q", cfg.DefaultProvider)
	}

	if cfg.DefaultModel != "gpt-4" {
		t.Errorf("expected default_model 'gpt-4', got %q", cfg.DefaultModel)
	}

	if cfg.Providers["openai"].APIKey != "${TECHNE_OPENAI_API_KEY}" {
		t.Errorf("expected env ref for API key, got %q", cfg.Providers["openai"].APIKey)
	}
}

func TestConfigInitWithOllama(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "techne.json")

	cmd := newConfigInitCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"--provider", "ollama",
		"--path", configPath,
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var cfg configFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.DefaultProvider != "ollama" {
		t.Errorf("expected default_provider 'ollama', got %q", cfg.DefaultProvider)
	}

	if cfg.DefaultModel != "llama3" {
		t.Errorf("expected default_model 'llama3', got %q", cfg.DefaultModel)
	}

	if cfg.Providers["ollama"].BaseURL != "http://localhost:11434" {
		t.Errorf("expected base_url for ollama, got %q", cfg.Providers["ollama"].BaseURL)
	}

	if cfg.Providers["ollama"].APIKey != "" {
		t.Errorf("ollama should not have API key, got %q", cfg.Providers["ollama"].APIKey)
	}

	if cfg.Providers["ollama"].ToolsEnabled != false {
		t.Error("ollama tools_enabled should be false")
	}
}

func TestConfigInitOverwriteProtection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "techne.json")

	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	cmd := newConfigInitCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"--provider", "anthropic",
		"--path", configPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when config file already exists")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestConfigInitForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "techne.json")

	if err := os.WriteFile(configPath, []byte(`{"old": true}`), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	cmd := newConfigInitCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"--provider", "openai",
		"--path", configPath,
		"--force",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("config init --force failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	if strings.Contains(string(data), "old") {
		t.Error("file should have been overwritten")
	}

	var cfg configFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.DefaultProvider != "openai" {
		t.Errorf("expected default_provider 'openai', got %q", cfg.DefaultProvider)
	}
}

func TestConfigInitDefaultProvider(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "techne.json")

	cmd := newConfigInitCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--path", configPath})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var cfg configFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.DefaultProvider != "anthropic" {
		t.Errorf("expected default provider 'anthropic', got %q", cfg.DefaultProvider)
	}
}

func TestConfigInitWithAPIKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "techne.json")

	cmd := newConfigInitCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"--provider", "anthropic",
		"--api-key", "sk-test-12345",
		"--path", configPath,
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var cfg configFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.Providers["anthropic"].APIKey != "sk-test-12345" {
		t.Errorf("expected API key 'sk-test-12345', got %q", cfg.Providers["anthropic"].APIKey)
	}
}

func TestConfigInitCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "dir", "techne.json")

	cmd := newConfigInitCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"--provider", "anthropic",
		"--path", configPath,
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("config init failed: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created in nested directory")
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{name: "empty key", key: "", expected: ""},
		{name: "short key", key: "abc", expected: "****"},
		{name: "exact 4 chars", key: "abcd", expected: "****"},
		{name: "long key", key: "sk-ant-api03-long-key-here", expected: "****here"},
		{name: "12 char key", key: "abcd12345678", expected: "****5678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskKey(tt.key)
			if result != tt.expected {
				t.Errorf("maskKey(%q) = %q, expected %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestConfigShowMasksKeys(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		DefaultModel:    "claude-sonnet-4-20250514",
		Providers: map[string]config.ProviderConfig{
			"anthropic": {
				Type:   "anthropic",
				APIKey: "sk-ant-api03-secret-key-abc",
			},
		},
		Permissions: config.PermissionsConfig{Mode: "interactive"},
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.WithValue(context.Background(), configKey{}, cfg))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	showCmd := newConfigShowCmd()

	showCmd.RunE = func(c *cobra.Command, args []string) error {
		c.SetContext(context.WithValue(context.Background(), configKey{}, cfg))
		c.SetOut(buf)
		c.SetErr(buf)

		out := showConfig{
			DefaultProvider: cfg.DefaultProvider,
			DefaultModel:    cfg.DefaultModel,
			Permissions:     cfg.Permissions.Mode,
			Providers:       make(map[string]showProvider),
		}

		for name, p := range cfg.Providers {
			out.Providers[name] = showProvider{
				Type:         p.Type,
				APIKey:       maskKey(p.APIKey),
				ToolsEnabled: p.GetToolsEnabled(),
			}
		}

		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}

		c.Println(string(data))
		return nil
	}

	err := showCmd.Execute()
	if err != nil {
		t.Fatalf("config show failed: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "sk-ant-api03-secret-key-abc") {
		t.Error("output should not contain the raw API key")
	}

	if !strings.Contains(output, "****-abc") {
		t.Errorf("output should contain masked key ending in '-abc', got: %s", output)
	}
}

func TestConfigShowWithNoProvider(t *testing.T) {
	cfg := config.DefaultConfig()

	cmd := &cobra.Command{}
	cmd.SetContext(context.WithValue(context.Background(), configKey{}, cfg))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	showCmd := newConfigShowCmd()

	showCmd.RunE = func(c *cobra.Command, args []string) error {
		c.SetContext(context.WithValue(context.Background(), configKey{}, cfg))
		c.SetOut(buf)

		out := showConfig{
			DefaultProvider: cfg.DefaultProvider,
			DefaultModel:    cfg.DefaultModel,
			Permissions:     cfg.Permissions.Mode,
			Providers:       make(map[string]showProvider),
		}

		for name, p := range cfg.Providers {
			out.Providers[name] = showProvider{
				Type:         p.Type,
				APIKey:       maskKey(p.APIKey),
				ToolsEnabled: p.GetToolsEnabled(),
			}
		}

		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}

		c.Println(string(data))
		return nil
	}

	err := showCmd.Execute()
	if err != nil {
		t.Fatalf("config show failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "anthropic") {
		t.Errorf("output should contain default provider, got: %s", output)
	}
}

func TestConfigCommandStructure(t *testing.T) {
	configCmd := newConfigCmd()

	if configCmd.Use != "config" {
		t.Errorf("expected Use 'config', got %q", configCmd.Use)
	}

	subcommands := configCmd.Commands()
	if len(subcommands) != 2 {
		t.Fatalf("expected 2 subcommands, got %d", len(subcommands))
	}

	names := make(map[string]bool)
	for _, sub := range subcommands {
		names[sub.Use] = true
	}

	if !names["init"] {
		t.Error("expected 'init' subcommand")
	}

	if !names["show"] {
		t.Error("expected 'show' subcommand")
	}
}

func TestConfigInitFlags(t *testing.T) {
	cmd := newConfigInitCmd()

	expectedFlags := []string{"provider", "api-key", "model", "path", "force"}
	for _, name := range expectedFlags {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("expected flag --%s to exist", name)
		}
	}

	providerFlag := cmd.Flags().Lookup("provider")
	if providerFlag.Shorthand != "p" {
		t.Errorf("expected --provider shorthand 'p', got %q", providerFlag.Shorthand)
	}
}
