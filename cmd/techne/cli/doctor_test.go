package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Edcko/techne-code/internal/config"
)

func TestCheckConfigFileValid(t *testing.T) {
	tmpDir := t.TempDir()
	techneDir := filepath.Join(tmpDir, ".techne")
	os.MkdirAll(techneDir, 0755)

	configContent := `{"default_provider": "anthropic"}`
	configPath := filepath.Join(techneDir, "techne.json")
	os.WriteFile(configPath, []byte(configContent), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	result := checkConfigFile()

	if !result.Pass {
		t.Errorf("expected check to pass with valid config, got: %s", result.Message)
	}
	if !strings.Contains(result.Message, "valid config") {
		t.Errorf("expected message to mention valid config, got: %s", result.Message)
	}
}

func TestCheckConfigFileInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	techneDir := filepath.Join(tmpDir, ".techne")
	os.MkdirAll(techneDir, 0755)

	configContent := `{invalid json}`
	configPath := filepath.Join(techneDir, "techne.json")
	os.WriteFile(configPath, []byte(configContent), 0644)

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	result := checkConfigFile()

	if result.Pass {
		t.Error("expected check to fail with invalid JSON")
	}
	if !strings.Contains(result.Message, "invalid JSON") {
		t.Errorf("expected message to mention invalid JSON, got: %s", result.Message)
	}
}

func TestCheckConfigFileMissing(t *testing.T) {
	tmpDir := t.TempDir()

	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	result := checkConfigFile()

	if result.Pass {
		t.Error("expected check to fail with no config file")
	}
	if !strings.Contains(result.Message, "No config file found") {
		t.Errorf("expected message to mention no config file, got: %s", result.Message)
	}
}

func TestCheckDefaultProvider(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		wantPass   bool
		wantSubstr string
	}{
		{
			name:       "provider set",
			provider:   "anthropic",
			wantPass:   true,
			wantSubstr: "anthropic",
		},
		{
			name:       "provider empty",
			provider:   "",
			wantPass:   false,
			wantSubstr: "No default provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{DefaultProvider: tt.provider}
			result := checkDefaultProvider(cfg)

			if result.Pass != tt.wantPass {
				t.Errorf("expected Pass=%v, got Pass=%v", tt.wantPass, result.Pass)
			}
			if !strings.Contains(result.Message, tt.wantSubstr) {
				t.Errorf("expected message to contain %q, got: %s", tt.wantSubstr, result.Message)
			}
		})
	}
}

func TestCheckDefaultModel(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		wantPass   bool
		wantSubstr string
	}{
		{
			name:       "model set",
			model:      "claude-sonnet-4-20250514",
			wantPass:   true,
			wantSubstr: "claude-sonnet-4-20250514",
		},
		{
			name:       "model empty",
			model:      "",
			wantPass:   false,
			wantSubstr: "No default model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{DefaultModel: tt.model}
			result := checkDefaultModel(cfg)

			if result.Pass != tt.wantPass {
				t.Errorf("expected Pass=%v, got Pass=%v", tt.wantPass, result.Pass)
			}
			if !strings.Contains(result.Message, tt.wantSubstr) {
				t.Errorf("expected message to contain %q, got: %s", tt.wantSubstr, result.Message)
			}
		})
	}
}

func TestCheckProviderAPIKeys(t *testing.T) {
	tests := []struct {
		name       string
		providers  map[string]config.ProviderConfig
		envKey     string
		envValue   string
		wantLen    int
		wantPass   bool
		wantSubstr string
	}{
		{
			name:       "no providers configured",
			providers:  map[string]config.ProviderConfig{},
			wantLen:    0,
			wantPass:   true,
			wantSubstr: "",
		},
		{
			name: "ollama provider skipped",
			providers: map[string]config.ProviderConfig{
				"ollama": {Type: "ollama"},
			},
			wantLen:    0,
			wantPass:   true,
			wantSubstr: "",
		},
		{
			name: "openai with API key in config",
			providers: map[string]config.ProviderConfig{
				"openai": {Type: "openai", APIKey: "sk-test-key-12345678"},
			},
			wantLen:    1,
			wantPass:   true,
			wantSubstr: "API key present",
		},
		{
			name: "anthropic missing API key",
			providers: map[string]config.ProviderConfig{
				"anthropic": {Type: "anthropic", APIKey: ""},
			},
			wantLen:    1,
			wantPass:   false,
			wantSubstr: "No API key found",
		},
		{
			name: "openai with env var API key",
			providers: map[string]config.ProviderConfig{
				"openai": {Type: "openai", APIKey: ""},
			},
			envKey:     "TECHNE_OPENAI_API_KEY",
			envValue:   "sk-env-key-12345678",
			wantLen:    1,
			wantPass:   true,
			wantSubstr: "API key present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envKey != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			cfg := &config.Config{Providers: tt.providers}
			results := checkProviderAPIKeys(cfg)

			if len(results) != tt.wantLen {
				t.Fatalf("expected %d results, got %d", tt.wantLen, len(results))
			}

			if len(results) > 0 {
				r := results[0]
				if r.Pass != tt.wantPass {
					t.Errorf("expected Pass=%v, got Pass=%v: %s", tt.wantPass, r.Pass, r.Message)
				}
				if tt.wantSubstr != "" && !strings.Contains(r.Message, tt.wantSubstr) {
					t.Errorf("expected message to contain %q, got: %s", tt.wantSubstr, r.Message)
				}
			}
		})
	}
}

func TestCheckProviderAPIKeysMasking(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"openai": {Type: "openai", APIKey: "sk-1234567890abcdef"},
		},
	}

	results := checkProviderAPIKeys(cfg)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !results[0].Pass {
		t.Error("expected check to pass")
	}
	if strings.Contains(results[0].Message, "sk-1234567890abcdef") {
		t.Error("API key should be masked in output")
	}
}

func TestCheckOllamaProvidersReachable(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"local": {Type: "ollama"},
		},
	}

	mockChecker := func(baseURL string) CheckResult {
		return CheckResult{
			Pass:    true,
			Message: fmt.Sprintf("Ollama reachable at %s", baseURL),
		}
	}

	results := checkOllamaProviders(cfg, mockChecker)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !results[0].Pass {
		t.Errorf("expected check to pass, got: %s", results[0].Message)
	}
	if !strings.Contains(results[0].Name, "Ollama connectivity") {
		t.Errorf("expected name to contain 'Ollama connectivity', got: %s", results[0].Name)
	}
}

func TestCheckOllamaProvidersUnreachable(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"local": {Type: "ollama"},
		},
	}

	mockChecker := func(baseURL string) CheckResult {
		return CheckResult{
			Pass:    false,
			Message: "Cannot reach Ollama",
		}
	}

	results := checkOllamaProviders(cfg, mockChecker)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Pass {
		t.Error("expected check to fail")
	}
}

func TestCheckOllamaProvidersCustomBaseURL(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"remote": {Type: "ollama", BaseURL: "http://myserver:11434/v1"},
		},
	}

	var capturedURL string
	mockChecker := func(baseURL string) CheckResult {
		capturedURL = baseURL
		return CheckResult{Pass: true, Message: "ok"}
	}

	checkOllamaProviders(cfg, mockChecker)

	if capturedURL != "http://myserver:11434" {
		t.Errorf("expected base URL to be http://myserver:11434, got: %s", capturedURL)
	}
}

func TestCheckOllamaProvidersSkipsNonOllama(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"anthropic": {Type: "anthropic"},
			"openai":    {Type: "openai"},
		},
	}

	mockChecker := func(baseURL string) CheckResult {
		return CheckResult{Pass: true}
	}

	results := checkOllamaProviders(cfg, mockChecker)
	if len(results) != 0 {
		t.Errorf("expected 0 results for non-ollama providers, got %d", len(results))
	}
}

func TestCheckDataDirectory(t *testing.T) {
	cfg := &config.Config{
		Options: config.OptionsConfig{
			DataDirectory: filepath.Join(t.TempDir(), "techne_data"),
		},
	}

	result := checkDataDirectory(cfg)

	if !result.Pass {
		t.Errorf("expected check to pass for writable directory, got: %s", result.Message)
	}
	if !strings.Contains(result.Message, "writable") {
		t.Errorf("expected message to mention writable, got: %s", result.Message)
	}
}

func TestCheckDataDirectoryDefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	cfg := &config.Config{
		Options: config.OptionsConfig{DataDirectory: ""},
	}

	result := checkDataDirectory(cfg)

	if !result.Pass {
		t.Errorf("expected check to pass with default data dir, got: %s", result.Message)
	}
	if !strings.Contains(result.Name, "Data directory") {
		t.Errorf("expected name to be 'Data directory', got: %s", result.Name)
	}
}

func TestCheckGoVersion(t *testing.T) {
	result := checkGoVersion()

	if !result.Pass {
		t.Error("Go version check should always pass")
	}
	if !strings.Contains(result.Message, "go") {
		t.Errorf("expected message to contain Go version info, got: %s", result.Message)
	}
}

func TestRunDoctorChecks(t *testing.T) {
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		DefaultModel:    "claude-sonnet-4-20250514",
		Providers: map[string]config.ProviderConfig{
			"anthropic": {Type: "anthropic", APIKey: "sk-test-key-12345678"},
		},
		Options: config.OptionsConfig{
			DataDirectory: filepath.Join(t.TempDir(), "techne_data"),
		},
	}

	mockOllama := func(baseURL string) CheckResult {
		return CheckResult{Pass: true, Message: "ok"}
	}

	results := runDoctorChecks(cfg, mockOllama)

	if len(results) < 5 {
		t.Errorf("expected at least 5 check results, got %d", len(results))
	}

	foundNames := make(map[string]bool)
	for _, r := range results {
		foundNames[r.Name] = true
	}

	expectedChecks := []string{
		"Config file",
		"Default provider",
		"Default model",
		"Go version",
		"Data directory",
	}

	for _, expected := range expectedChecks {
		if !foundNames[expected] {
			t.Errorf("expected check %q not found in results", expected)
		}
	}
}

func TestDoctorCommandRegistration(t *testing.T) {
	cmd := newDoctorCmd()

	if cmd.Use != "doctor" {
		t.Errorf("expected Use to be 'doctor', got: %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if cmd.RunE == nil {
		t.Error("expected RunE to be set")
	}
}

func TestDoctorCommandOutput(t *testing.T) {
	cmd := newDoctorCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	tmpDir := t.TempDir()
	cfg := &config.Config{
		DefaultProvider: "anthropic",
		DefaultModel:    "claude-sonnet-4-20250514",
		Providers:       map[string]config.ProviderConfig{},
		Options: config.OptionsConfig{
			DataDirectory: filepath.Join(tmpDir, "techne_data"),
		},
	}

	cmd.SetContext(context.WithValue(context.Background(), configKey{}, cfg))

	err := cmd.Execute()
	if err != nil {
		t.Logf("doctor command returned error (expected with no config file): %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Default provider") {
		t.Errorf("expected output to contain 'Default provider', got: %s", output)
	}
	if !strings.Contains(output, "Go version") {
		t.Errorf("expected output to contain 'Go version', got: %s", output)
	}
}

func TestDoctorCommandExitCode(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		DefaultProvider: "",
		DefaultModel:    "",
		Providers:       map[string]config.ProviderConfig{},
		Options: config.OptionsConfig{
			DataDirectory: filepath.Join(tmpDir, "techne_data"),
		},
	}

	cmd := newDoctorCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetContext(context.WithValue(context.Background(), configKey{}, cfg))

	err := cmd.Execute()

	if err == nil {
		t.Error("expected error when checks fail")
	}
	if !strings.Contains(err.Error(), "doctor checks failed") {
		t.Errorf("expected 'doctor checks failed' error, got: %v", err)
	}
}

func TestMaskKeyForDoctor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "short", input: "abc", want: "****"},
		{name: "exact4", input: "abcd", want: "****"},
		{name: "long", input: "sk-1234567890abcdef", want: "****cdef"},
		{name: "5chars", input: "abcde", want: "****bcde"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskKey(tt.input)
			if got != tt.want {
				t.Errorf("maskKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPrintResultPass(t *testing.T) {
	cmd := newDoctorCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := CheckResult{
		Name:    "Test check",
		Pass:    true,
		Message: "All good",
	}

	printResult(cmd, result)

	output := buf.String()
	if !strings.Contains(output, "✅") {
		t.Errorf("expected checkmark icon in output, got: %s", output)
	}
	if !strings.Contains(output, "Test check") {
		t.Errorf("expected check name in output, got: %s", output)
	}
}

func TestPrintResultFail(t *testing.T) {
	cmd := newDoctorCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	result := CheckResult{
		Name:    "Test check",
		Pass:    false,
		Message: "Something broke",
	}

	printResult(cmd, result)

	output := buf.String()
	if !strings.Contains(output, "❌") {
		t.Errorf("expected X icon in output, got: %s", output)
	}
	if !strings.Contains(output, "Something broke") {
		t.Errorf("expected message in output, got: %s", output)
	}
}
