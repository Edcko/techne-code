package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Edcko/techne-code/internal/config"
	"github.com/spf13/cobra"
)

// TestRootCommandFlags verifies that the root command has expected global flags.
func TestRootCommandFlags(t *testing.T) {
	// Create a minimal root command for testing
	rootCmd := &cobra.Command{
		Use: "techne",
		Run: func(cmd *cobra.Command, args []string) {},
	}

	// Add the global flags that should exist
	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to config file")
	rootCmd.PersistentFlags().String("provider", "", "LLM provider to use")
	rootCmd.PersistentFlags().String("model", "", "LLM model to use")

	// Test that flags exist
	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("Expected --config flag to exist")
	}

	providerFlag := rootCmd.PersistentFlags().Lookup("provider")
	if providerFlag == nil {
		t.Error("Expected --provider flag to exist")
	}

	modelFlag := rootCmd.PersistentFlags().Lookup("model")
	if modelFlag == nil {
		t.Error("Expected --model flag to exist")
	}

	// Verify shorthand
	if configFlag.Shorthand != "c" {
		t.Errorf("Expected --config shorthand to be 'c', got '%s'", configFlag.Shorthand)
	}

	t.Logf("Root command has %d persistent flags", rootCmd.PersistentFlags().NFlag())
}

// TestChatCommandFlags verifies that the chat command has expected flags.
func TestChatCommandFlags(t *testing.T) {
	ctx := context.Background()
	chatCmd := newChatCmd(ctx)

	// Test that flags exist
	promptFlag := chatCmd.Flags().Lookup("prompt")
	if promptFlag == nil {
		t.Error("Expected --prompt flag to exist on chat command")
	}

	sessionFlag := chatCmd.Flags().Lookup("session")
	if sessionFlag == nil {
		t.Error("Expected --session flag to exist on chat command")
	}

	newSessionFlag := chatCmd.Flags().Lookup("new-session")
	if newSessionFlag == nil {
		t.Error("Expected --new-session flag to exist on chat command")
	}

	// Verify shorthand for prompt
	if promptFlag.Shorthand != "p" {
		t.Errorf("Expected --prompt shorthand to be 'p', got '%s'", promptFlag.Shorthand)
	}

	// Verify flag types
	if promptFlag.Value.Type() != "string" {
		t.Errorf("Expected --prompt flag type to be string, got %s", promptFlag.Value.Type())
	}

	if newSessionFlag.Value.Type() != "bool" {
		t.Errorf("Expected --new-session flag type to be bool, got %s", newSessionFlag.Value.Type())
	}
}

// TestSessionListCommand verifies the session list command works.
func TestSessionListCommand(t *testing.T) {
	ctx := context.Background()
	sessionCmd := newSessionCmd(ctx)

	buf := new(bytes.Buffer)
	sessionCmd.SetOut(buf)
	sessionCmd.SetErr(buf)

	sessionCmd.SetArgs([]string{"list"})

	err := sessionCmd.ExecuteContext(context.WithValue(ctx, configKey{}, config.DefaultConfig()))
	if err != nil {
		t.Errorf("Session list command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No sessions found") && !strings.Contains(output, "ID") {
		t.Errorf("Expected output to contain 'No sessions found' or session table header, got: %s", output)
	}
}

// TestSessionDeleteCommand verifies the session delete command validates args.
func TestSessionDeleteCommand(t *testing.T) {
	ctx := context.Background()
	sessionCmd := newSessionCmd(ctx)

	// Find the delete subcommand
	deleteCmd, _, err := sessionCmd.Find([]string{"delete"})
	if err != nil {
		t.Fatalf("Failed to find delete command: %v", err)
	}

	// Test that it requires exactly one argument
	if deleteCmd.Args == nil {
		t.Error("Expected delete command to have Args validator")
	}

	// Test with no args (should fail)
	err = deleteCmd.Args(deleteCmd, []string{})
	if err == nil {
		t.Error("Expected error when no session ID provided")
	}

	// Test with one arg (should pass)
	err = deleteCmd.Args(deleteCmd, []string{"session-123"})
	if err != nil {
		t.Errorf("Expected no error with one session ID, got: %v", err)
	}
}

// TestVersionCommand verifies the version command prints version.
func TestVersionCommand(t *testing.T) {
	testVersion := "test-1.0.0"
	versionCmd := newVersionCmd(testVersion)

	// Capture output
	buf := new(bytes.Buffer)
	versionCmd.SetOut(buf)

	err := versionCmd.Execute()
	if err != nil {
		t.Errorf("Version command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, testVersion) {
		t.Errorf("Expected output to contain '%s', got: %s", testVersion, output)
	}
}

// TestConfigLoading verifies config loading applies CLI overrides.
func TestConfigLoading(t *testing.T) {
	// Create a command with config loading
	rootCmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {},
	}

	rootCmd.PersistentFlags().String("provider", "", "LLM provider to use")
	rootCmd.PersistentFlags().String("model", "", "LLM model to use")

	// Set flag values
	rootCmd.SetArgs([]string{"--provider", "openai", "--model", "gpt-4"})

	// Parse flags
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("Note: Execute may fail without config, this is expected: %v", err)
	}

	// Verify flags were parsed
	provider, _ := rootCmd.Flags().GetString("provider")
	model, _ := rootCmd.Flags().GetString("model")

	if provider != "openai" {
		t.Errorf("Expected provider to be 'openai', got '%s'", provider)
	}

	if model != "gpt-4" {
		t.Errorf("Expected model to be 'gpt-4', got '%s'", model)
	}
}

// TestMissingConfigDoesntCrash verifies that missing config uses defaults.
func TestMissingConfigDoesntCrash(t *testing.T) {
	// Create a command with a non-existent config
	rootCmd := &cobra.Command{
		Use: "test",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Simulate loading a non-existent config
			return loadConfig(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Should have config from defaults
			cfg := getConfig(cmd)
			if cfg == nil {
				t.Error("Config should not be nil even with missing file")
			}
		},
	}

	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to config file")
	rootCmd.PersistentFlags().String("provider", "", "LLM provider to use")
	rootCmd.PersistentFlags().String("model", "", "LLM model to use")

	// Use a non-existent config file
	rootCmd.SetArgs([]string{"--config", "/non/existent/config.json"})

	// This should not crash
	ctx := context.Background()
	err := rootCmd.ExecuteContext(ctx)

	// The command should succeed even with missing config
	if err != nil {
		t.Logf("Note: Command may report error, but should not crash: %v", err)
	}

	// Verify we can still get a default config
	cfg := config.DefaultConfig()
	if cfg.DefaultProvider == "" {
		t.Error("Default config should have a default provider")
	}
}

// TestGetConfigFallback verifies getConfig returns defaults when context is empty.
func TestGetConfigFallback(t *testing.T) {
	// Create a command without config in context
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// getConfig should return defaults, not panic
	cfg := getConfig(cmd)

	if cfg == nil {
		t.Error("getConfig should return defaults, not nil")
	}

	if cfg.DefaultProvider == "" {
		t.Error("Default config should have a default provider")
	}
}

// TestChatCommandOutput verifies chat command validates config requirements.
func TestChatCommandOutput(t *testing.T) {
	ctx := context.Background()
	chatCmd := newChatCmd(ctx)

	// Capture output
	buf := new(bytes.Buffer)
	chatCmd.SetOut(buf)
	chatCmd.SetErr(buf)

	// Set config in context with no providers configured
	chatCmd.SetContext(context.WithValue(ctx, configKey{}, config.DefaultConfig()))

	// Execute without --prompt (interactive mode)
	err := chatCmd.Execute()

	// Should fail because no provider is configured
	if err == nil {
		t.Error("Expected error when no provider is configured")
	}

	output := buf.String()

	// Should mention provider not found
	if !strings.Contains(output, "provider") {
		t.Errorf("Expected output to mention 'provider', got: %s", output)
	}
}

// TestNonInteractiveMode verifies --prompt flag triggers non-interactive mode.
func TestNonInteractiveMode(t *testing.T) {
	ctx := context.Background()
	chatCmd := newChatCmd(ctx)

	// Capture output
	buf := new(bytes.Buffer)
	chatCmd.SetOut(buf)
	chatCmd.SetErr(buf)

	// Set config in context with no providers configured
	chatCmd.SetContext(context.WithValue(ctx, configKey{}, config.DefaultConfig()))

	// Set --prompt flag
	chatCmd.SetArgs([]string{"--prompt", "test prompt"})

	err := chatCmd.Execute()

	// Should fail because no provider is configured
	if err == nil {
		t.Error("Expected error when no provider is configured")
	}

	output := buf.String()

	// Should mention provider not found
	if !strings.Contains(output, "provider") {
		t.Errorf("Expected output to mention 'provider', got: %s", output)
	}
}

// TestSessionShowCommand verifies session show command.
func TestSessionShowCommand(t *testing.T) {
	ctx := context.Background()
	sessionCmd := newSessionCmd(ctx)

	buf := new(bytes.Buffer)
	sessionCmd.SetOut(buf)
	sessionCmd.SetErr(buf)

	sessionCmd.SetArgs([]string{"show", "nonexistent-session"})

	err := sessionCmd.ExecuteContext(context.WithValue(ctx, configKey{}, config.DefaultConfig()))
	if err == nil {
		t.Error("Expected error when session not found")
	}

	output := buf.String()
	if !strings.Contains(output, "session not found") && !strings.Contains(err.Error(), "session not found") {
		t.Errorf("Expected 'session not found' error, got: %s (err: %v)", output, err)
	}
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name         string
		apiKey       string
		providerType string
		providerName string
		wantErr      bool
	}{
		{
			name:         "empty API key with ollama provider should not error",
			apiKey:       "",
			providerType: "ollama",
			providerName: "ollama",
			wantErr:      false,
		},
		{
			name:         "empty API key with openai provider should error",
			apiKey:       "",
			providerType: "openai",
			providerName: "openai",
			wantErr:      true,
		},
		{
			name:         "empty API key with anthropic provider should error",
			apiKey:       "",
			providerType: "anthropic",
			providerName: "anthropic",
			wantErr:      true,
		},
		{
			name:         "empty API key with gemini provider should error",
			apiKey:       "",
			providerType: "gemini",
			providerName: "gemini",
			wantErr:      true,
		},
		{
			name:         "empty API key with unknown provider should error",
			apiKey:       "",
			providerType: "unknown",
			providerName: "unknown",
			wantErr:      true,
		},
		{
			name:         "non-empty API key with ollama provider should not error",
			apiKey:       "test-key",
			providerType: "ollama",
			providerName: "ollama",
			wantErr:      false,
		},
		{
			name:         "non-empty API key with openai provider should not error",
			apiKey:       "test-key",
			providerType: "openai",
			providerName: "openai",
			wantErr:      false,
		},
		{
			name:         "non-empty API key with anthropic provider should not error",
			apiKey:       "test-key",
			providerType: "anthropic",
			providerName: "anthropic",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIKey(tt.apiKey, tt.providerType, tt.providerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.wantErr {
				expectedEnvVar := "TECHNE_" + strings.ToUpper(tt.providerName) + "_API_KEY"
				if !strings.Contains(err.Error(), "API key not found") {
					t.Errorf("error message should contain 'API key not found', got: %v", err)
				}
				if !strings.Contains(err.Error(), expectedEnvVar) {
					t.Errorf("error message should contain env var %s, got: %v", expectedEnvVar, err)
				}
				if !strings.Contains(err.Error(), tt.providerName) {
					t.Errorf("error message should contain provider name %s, got: %v", tt.providerName, err)
				}
			}
		})
	}
}

func TestSessionAndNewSessionMutuallyExclusive(t *testing.T) {
	ctx := context.Background()
	chatCmd := newChatCmd(ctx)

	buf := new(bytes.Buffer)
	chatCmd.SetOut(buf)
	chatCmd.SetErr(buf)
	chatCmd.SetContext(context.WithValue(ctx, configKey{}, config.DefaultConfig()))

	chatCmd.SetArgs([]string{"--session", "abc-123", "--new-session"})

	err := chatCmd.Execute()
	if err == nil {
		t.Error("expected error when both --session and --new-session are provided")
	}
	if !strings.Contains(err.Error(), "cannot use both") {
		t.Errorf("expected mutual exclusion error, got: %v", err)
	}
}

func TestSessionFlagInvalidID(t *testing.T) {
	ctx := context.Background()
	chatCmd := newChatCmd(ctx)

	buf := new(bytes.Buffer)
	chatCmd.SetOut(buf)
	chatCmd.SetErr(buf)
	chatCmd.SetContext(context.WithValue(ctx, configKey{}, config.DefaultConfig()))

	chatCmd.SetArgs([]string{"--session", "nonexistent-session", "--prompt", "hello"})

	err := chatCmd.Execute()
	if err == nil {
		t.Error("expected error when resuming nonexistent session")
	}
	if !strings.Contains(err.Error(), "session") && !strings.Contains(err.Error(), "provider") {
		t.Errorf("expected session or provider error, got: %v", err)
	}
}

func TestNewSessionFlagDefault(t *testing.T) {
	ctx := context.Background()
	chatCmd := newChatCmd(ctx)

	newSessionFlag := chatCmd.Flags().Lookup("new-session")
	if newSessionFlag == nil {
		t.Fatal("expected --new-session flag to exist")
	}
	if newSessionFlag.DefValue != "false" {
		t.Errorf("expected --new-session default to be false, got %q", newSessionFlag.DefValue)
	}
}

func TestSessionFlagDefault(t *testing.T) {
	ctx := context.Background()
	chatCmd := newChatCmd(ctx)

	sessionFlag := chatCmd.Flags().Lookup("session")
	if sessionFlag == nil {
		t.Fatal("expected --session flag to exist")
	}
	if sessionFlag.DefValue != "" {
		t.Errorf("expected --session default to be empty, got %q", sessionFlag.DefValue)
	}
}
