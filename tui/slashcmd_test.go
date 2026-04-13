package tui

import (
	"strings"
	"testing"

	"github.com/Edcko/techne-code/internal/config"
)

func TestParseSlashCommand_NotSlashCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"regular text", "hello world"},
		{"empty string", ""},
		{"text with slash in middle", "use /model here"},
		{"no leading slash", "model gemini"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdType, arg, isCmd := ParseSlashCommand(tt.input)
			if isCmd {
				t.Errorf("expected isCmd=false for %q, got true (cmdType=%s, arg=%s)", tt.input, cmdType, arg)
			}
		})
	}
}

func TestParseSlashCommand_SlashOnly(t *testing.T) {
	cmdType, _, isCmd := ParseSlashCommand("/")
	if !isCmd {
		t.Error("expected isCmd=true for '/'")
	}
	if cmdType != CommandUnknown {
		t.Errorf("expected CommandUnknown for '/', got %s", cmdType)
	}
}

func TestParseSlashCommand_ModelCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantArg   string
		wantEmpty bool
	}{
		{"with model name", "/model gemini-2.5-flash", "gemini-2.5-flash", false},
		{"with spaces", "/model  claude-sonnet-4-20250514 ", "claude-sonnet-4-20250514", false},
		{"no arg", "/model", "", true},
		{"only spaces", "/model   ", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdType, arg, isCmd := ParseSlashCommand(tt.input)
			if !isCmd {
				t.Error("expected isCmd=true")
			}
			if cmdType != CommandModel {
				t.Errorf("expected CommandModel, got %s", cmdType)
			}
			if tt.wantEmpty {
				if arg != "" {
					t.Errorf("expected empty arg, got %q", arg)
				}
			} else if arg != tt.wantArg {
				t.Errorf("expected arg %q, got %q", tt.wantArg, arg)
			}
		})
	}
}

func TestParseSlashCommand_ProviderCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantArg   string
		wantEmpty bool
	}{
		{"with provider name", "/provider openai", "openai", false},
		{"no arg", "/provider", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdType, arg, isCmd := ParseSlashCommand(tt.input)
			if !isCmd {
				t.Error("expected isCmd=true")
			}
			if cmdType != CommandProvider {
				t.Errorf("expected CommandProvider, got %s", cmdType)
			}
			if tt.wantEmpty {
				if arg != "" {
					t.Errorf("expected empty arg, got %q", arg)
				}
			} else if arg != tt.wantArg {
				t.Errorf("expected arg %q, got %q", tt.wantArg, arg)
			}
		})
	}
}

func TestParseSlashCommand_HelpCommand(t *testing.T) {
	cmdType, _, isCmd := ParseSlashCommand("/help")
	if !isCmd {
		t.Error("expected isCmd=true")
	}
	if cmdType != CommandHelp {
		t.Errorf("expected CommandHelp, got %s", cmdType)
	}
}

func TestParseSlashCommand_ClearCommand(t *testing.T) {
	cmdType, _, isCmd := ParseSlashCommand("/clear")
	if !isCmd {
		t.Error("expected isCmd=true")
	}
	if cmdType != CommandClear {
		t.Errorf("expected CommandClear, got %s", cmdType)
	}
}

func TestParseSlashCommand_UnknownCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
	}{
		{"unknown cmd", "/foobar", "foobar"},
		{"typo", "/modle", "modle"},
		{"numbers", "/123", "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdType, arg, isCmd := ParseSlashCommand(tt.input)
			if !isCmd {
				t.Error("expected isCmd=true")
			}
			if cmdType != CommandUnknown {
				t.Errorf("expected CommandUnknown, got %s", cmdType)
			}
			if arg != tt.wantName {
				t.Errorf("expected arg %q, got %q", tt.wantName, arg)
			}
		})
	}
}

func testConfig() *config.Config {
	return &config.Config{
		DefaultProvider: "anthropic",
		DefaultModel:    "claude-sonnet-4-20250514",
		Providers: map[string]config.ProviderConfig{
			"anthropic": {
				Type:   "anthropic",
				Models: []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514", "claude-haiku-4-20250414"},
			},
			"openai": {
				Type:   "openai",
				Models: []string{"gpt-4o", "gpt-4o-mini"},
			},
			"gemini": {
				Type:   "gemini",
				Models: []string{"gemini-2.5-flash", "gemini-2.5-pro"},
			},
		},
	}
}

func TestExecuteSlashCommand_RegularMessage(t *testing.T) {
	cfg := testConfig()
	result := ExecuteSlashCommand("hello world", cfg)
	if result.Handled {
		t.Error("expected regular message to not be handled")
	}
}

func TestExecuteSlashCommand_ModelSwitch(t *testing.T) {
	cfg := testConfig()

	result := ExecuteSlashCommand("/model claude-opus-4-20250514", cfg)
	if !result.Handled {
		t.Error("expected command to be handled")
	}
	if result.Command != CommandModel {
		t.Errorf("expected CommandModel, got %s", result.Command)
	}
	if result.IsError {
		t.Errorf("expected no error, got: %s", result.Message)
	}
	if result.NewModel != "claude-opus-4-20250514" {
		t.Errorf("expected NewModel=claude-opus-4-20250514, got %q", result.NewModel)
	}
	if !strings.Contains(result.Message, "claude-opus-4-20250514") {
		t.Errorf("expected message to contain model name, got: %s", result.Message)
	}
}

func TestExecuteSlashCommand_ModelSwitchInvalid(t *testing.T) {
	cfg := testConfig()

	result := ExecuteSlashCommand("/model nonexistent-model", cfg)
	if !result.Handled {
		t.Error("expected command to be handled")
	}
	if !result.IsError {
		t.Error("expected error for invalid model")
	}
	if result.NewModel != "" {
		t.Errorf("expected empty NewModel for invalid, got %q", result.NewModel)
	}
}

func TestExecuteSlashCommand_ModelSwitchNoArg(t *testing.T) {
	cfg := testConfig()

	result := ExecuteSlashCommand("/model", cfg)
	if !result.Handled {
		t.Error("expected command to be handled")
	}
	if !result.IsError {
		t.Error("expected error for missing arg")
	}
	if !strings.Contains(result.Message, "Usage") {
		t.Errorf("expected usage message, got: %s", result.Message)
	}
}

func TestExecuteSlashCommand_ProviderSwitch(t *testing.T) {
	cfg := testConfig()

	result := ExecuteSlashCommand("/provider openai", cfg)
	if !result.Handled {
		t.Error("expected command to be handled")
	}
	if result.IsError {
		t.Errorf("expected no error, got: %s", result.Message)
	}
	if result.NewProvider != "openai" {
		t.Errorf("expected NewProvider=openai, got %q", result.NewProvider)
	}
}

func TestExecuteSlashCommand_ProviderSwitchInvalid(t *testing.T) {
	cfg := testConfig()

	result := ExecuteSlashCommand("/provider nonexistent", cfg)
	if !result.Handled {
		t.Error("expected command to be handled")
	}
	if !result.IsError {
		t.Error("expected error for invalid provider")
	}
}

func TestExecuteSlashCommand_ProviderSwitchNoArg(t *testing.T) {
	cfg := testConfig()

	result := ExecuteSlashCommand("/provider", cfg)
	if !result.Handled {
		t.Error("expected command to be handled")
	}
	if !result.IsError {
		t.Error("expected error for missing arg")
	}
	if !strings.Contains(result.Message, "Usage") {
		t.Errorf("expected usage message, got: %s", result.Message)
	}
}

func TestExecuteSlashCommand_Help(t *testing.T) {
	cfg := testConfig()

	result := ExecuteSlashCommand("/help", cfg)
	if !result.Handled {
		t.Error("expected command to be handled")
	}
	if result.IsError {
		t.Error("expected no error for help")
	}
	if !strings.Contains(result.Message, "/model") {
		t.Errorf("expected help to list /model, got: %s", result.Message)
	}
	if !strings.Contains(result.Message, "/provider") {
		t.Errorf("expected help to list /provider, got: %s", result.Message)
	}
	if !strings.Contains(result.Message, "/clear") {
		t.Errorf("expected help to list /clear, got: %s", result.Message)
	}
	if !strings.Contains(result.Message, "/help") {
		t.Errorf("expected help to list /help, got: %s", result.Message)
	}
}

func TestExecuteSlashCommand_Clear(t *testing.T) {
	cfg := testConfig()

	result := ExecuteSlashCommand("/clear", cfg)
	if !result.Handled {
		t.Error("expected command to be handled")
	}
	if result.IsError {
		t.Error("expected no error for clear")
	}
	if !result.ClearChat {
		t.Error("expected ClearChat=true")
	}
}

func TestExecuteSlashCommand_UnknownCommand(t *testing.T) {
	cfg := testConfig()

	result := ExecuteSlashCommand("/foobar", cfg)
	if !result.Handled {
		t.Error("expected command to be handled")
	}
	if !result.IsError {
		t.Error("expected error for unknown command")
	}
	if result.Command != CommandUnknown {
		t.Errorf("expected CommandUnknown, got %s", result.Command)
	}
	if !strings.Contains(result.Message, "foobar") {
		t.Errorf("expected message to mention command name, got: %s", result.Message)
	}
	if !strings.Contains(result.Message, "/help") {
		t.Errorf("expected message to suggest /help, got: %s", result.Message)
	}
}

func TestIsValidModel_ValidInCurrentProvider(t *testing.T) {
	cfg := testConfig()
	if !isValidModel("claude-sonnet-4-20250514", cfg) {
		t.Error("expected claude-sonnet-4-20250514 to be valid")
	}
}

func TestIsValidModel_ValidInOtherProvider(t *testing.T) {
	cfg := testConfig()
	if !isValidModel("gpt-4o", cfg) {
		t.Error("expected gpt-4o to be valid (in openai provider)")
	}
}

func TestIsValidModel_Invalid(t *testing.T) {
	cfg := testConfig()
	if isValidModel("nonexistent", cfg) {
		t.Error("expected nonexistent model to be invalid")
	}
}

func TestIsValidProvider_Valid(t *testing.T) {
	cfg := testConfig()
	if !isValidProvider("anthropic", cfg) {
		t.Error("expected anthropic to be valid")
	}
	if !isValidProvider("openai", cfg) {
		t.Error("expected openai to be valid")
	}
}

func TestIsValidProvider_Invalid(t *testing.T) {
	cfg := testConfig()
	if isValidProvider("nonexistent", cfg) {
		t.Error("expected nonexistent provider to be invalid")
	}
}

func TestFormatModelList_CurrentProvider(t *testing.T) {
	cfg := testConfig()
	list := formatModelList(cfg)
	if !strings.Contains(list, "claude-sonnet-4-20250514") {
		t.Errorf("expected current provider models, got: %s", list)
	}
}

func TestFormatProviderList(t *testing.T) {
	cfg := testConfig()
	list := formatProviderList(cfg)
	if !strings.Contains(list, "anthropic") {
		t.Errorf("expected anthropic in list, got: %s", list)
	}
	if !strings.Contains(list, "openai") {
		t.Errorf("expected openai in list, got: %s", list)
	}
	if !strings.Contains(list, "gemini") {
		t.Errorf("expected gemini in list, got: %s", list)
	}
}

func TestHandleSubmit_SlashCommandDoesNotStream(t *testing.T) {
	m := initTestModel()

	m.inputBuf.SetText("/help")
	m.handleSubmit()

	if m.state != StateChatting {
		t.Errorf("expected StateChatting after slash command, got %d", m.state)
	}

	hasHelp := false
	for _, msg := range m.messages {
		if msg.Role == "system" && strings.Contains(msg.Content, "/model") {
			hasHelp = true
		}
	}
	if !hasHelp {
		t.Error("expected system message with help content")
	}
}

func TestHandleSubmit_ModelSwitchUpdatesConfig(t *testing.T) {
	m := initTestModel()
	m.cfg.Providers["anthropic"] = config.ProviderConfig{
		Type:   "anthropic",
		Models: []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514"},
	}

	m.inputBuf.SetText("/model claude-opus-4-20250514")
	m.handleSubmit()

	if m.cfg.DefaultModel != "claude-opus-4-20250514" {
		t.Errorf("expected model to be claude-opus-4-20250514, got %q", m.cfg.DefaultModel)
	}
	if m.state != StateChatting {
		t.Errorf("expected StateChatting after model switch, got %d", m.state)
	}
}

func TestHandleSubmit_ProviderSwitchUpdatesConfig(t *testing.T) {
	m := initTestModel()
	m.cfg.Providers["openai"] = config.ProviderConfig{
		Type:   "openai",
		Models: []string{"gpt-4o", "gpt-4o-mini"},
	}

	m.inputBuf.SetText("/provider openai")
	m.handleSubmit()

	if m.cfg.DefaultProvider != "openai" {
		t.Errorf("expected provider to be openai, got %q", m.cfg.DefaultProvider)
	}
	if m.cfg.DefaultModel != "gpt-4o" {
		t.Errorf("expected model to default to first in provider (gpt-4o), got %q", m.cfg.DefaultModel)
	}
}

func TestHandleSubmit_ClearCommand(t *testing.T) {
	m := initTestModel()

	m.messages = append(m.messages, ChatMessage{Role: "user", Content: "hello"})
	m.messages = append(m.messages, ChatMessage{Role: "assistant", Content: "hi"})

	m.inputBuf.SetText("/clear")
	m.handleSubmit()

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message (the clear confirmation), got %d", len(m.messages))
	}
	if m.messages[0].Role != "system" {
		t.Errorf("expected system message, got %q", m.messages[0].Role)
	}
}

func TestHandleSubmit_InvalidModelShowsError(t *testing.T) {
	m := initTestModel()

	m.inputBuf.SetText("/model nonexistent")
	m.handleSubmit()

	if m.state != StateChatting {
		t.Errorf("expected StateChatting, got %d", m.state)
	}

	found := false
	for _, msg := range m.messages {
		if msg.Role == "error" && strings.Contains(msg.Content, "nonexistent") {
			found = true
		}
	}
	if !found {
		t.Error("expected error message about unknown model")
	}
}

func TestHandleSubmit_UnknownSlashCommand(t *testing.T) {
	m := initTestModel()

	m.inputBuf.SetText("/foobar")
	m.handleSubmit()

	if m.state != StateChatting {
		t.Errorf("expected StateChatting, got %d", m.state)
	}

	found := false
	for _, msg := range m.messages {
		if msg.Role == "error" && strings.Contains(msg.Content, "foobar") {
			found = true
		}
	}
	if !found {
		t.Error("expected error message about unknown command")
	}
}

func TestHandleSubmit_RegularMessageStillSends(t *testing.T) {
	m := initTestModel()

	m.inputBuf.SetText("hello world")
	m.handleSubmit()

	if m.state != StateStreaming {
		t.Errorf("expected StateStreaming for regular message, got %d", m.state)
	}

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	if m.messages[0].Role != "user" {
		t.Errorf("expected user message, got %q", m.messages[0].Role)
	}
	if m.messages[0].Content != "hello world" {
		t.Errorf("expected content 'hello world', got %q", m.messages[0].Content)
	}
}

func TestHandleSubmit_StatusBarUpdatesOnModelSwitch(t *testing.T) {
	m := initTestModel()
	m.cfg.Providers["anthropic"] = config.ProviderConfig{
		Type:   "anthropic",
		Models: []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514"},
	}

	m.inputBuf.SetText("/model claude-opus-4-20250514")
	m.handleSubmit()

	if !strings.Contains(m.statusText, "claude-opus-4-20250514") {
		t.Errorf("expected statusText to contain new model, got %q", m.statusText)
	}
}
