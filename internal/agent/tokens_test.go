package agent

import (
	"testing"

	"github.com/Edcko/techne-code/pkg/provider"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty", "", 0},
		{"single_char", "a", 1},
		{"four_chars", "abcd", 1},
		{"five_chars", "abcde", 2},
		{"eight_chars", "abcdefgh", 2},
		{"sixteen_chars", "abcdefghijklmnop", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokens(tt.input)
			if result != tt.expected {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	msg := provider.Message{
		Role: provider.RoleUser,
		Content: []provider.ContentBlock{
			{Type: provider.BlockText, Text: "Hello world"},
			{Type: provider.BlockText, Text: "Second block"},
		},
	}

	tokens := EstimateMessageTokens(msg)
	if tokens <= 0 {
		t.Error("EstimateMessageTokens should return positive value")
	}

	textTokens := EstimateTokens("Hello world") + EstimateTokens("Second block") + 4
	if tokens != textTokens {
		t.Errorf("EstimateMessageTokens = %d, want %d", tokens, textTokens)
	}
}

func TestEstimateMessageTokens_ToolCall(t *testing.T) {
	msg := provider.Message{
		Role: provider.RoleAssistant,
		Content: []provider.ContentBlock{
			{
				Type: provider.BlockToolCall,
				ToolCall: &provider.ToolCallBlock{
					ID:    "tc_1",
					Name:  "read_file",
					Input: []byte(`{"path":"/tmp/test.go"}`),
				},
			},
		},
	}

	tokens := EstimateMessageTokens(msg)
	if tokens <= 0 {
		t.Error("should count tool call tokens")
	}
}

func TestEstimateMessageTokens_ToolResult(t *testing.T) {
	msg := provider.Message{
		Role: provider.RoleTool,
		Content: []provider.ContentBlock{
			{
				Type: provider.BlockToolResult,
				ToolResult: &provider.ToolResultBlock{
					ToolCallID: "tc_1",
					Name:       "read_file",
					Content:    "file contents here",
				},
			},
		},
	}

	tokens := EstimateMessageTokens(msg)
	if tokens <= 0 {
		t.Error("should count tool result tokens")
	}
}

func TestEstimateMessagesTokens(t *testing.T) {
	messages := []provider.Message{
		{
			Role:    provider.RoleUser,
			Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hello"}},
		},
		{
			Role:    provider.RoleAssistant,
			Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hi there"}},
		},
	}

	total := EstimateMessagesTokens(messages)
	sum := EstimateMessageTokens(messages[0]) + EstimateMessageTokens(messages[1])
	if total != sum {
		t.Errorf("EstimateMessagesTokens = %d, want sum %d", total, sum)
	}
}

func TestEstimateSystemPromptTokens(t *testing.T) {
	if EstimateSystemPromptTokens("") != 0 {
		t.Error("empty prompt should be 0 tokens")
	}

	tokens := EstimateSystemPromptTokens("You are a helpful assistant")
	if tokens <= 0 {
		t.Error("should return positive tokens for non-empty prompt")
	}
}

func TestGetContextWindow(t *testing.T) {
	models := []provider.ModelInfo{
		{ID: "claude-3", ContextWindow: 200000},
		{ID: "gpt-4", ContextWindow: 128000},
	}

	if GetContextWindow(models, "claude-3") != 200000 {
		t.Error("should find claude-3 context window")
	}

	if GetContextWindow(models, "gpt-4") != 128000 {
		t.Error("should find gpt-4 context window")
	}

	if GetContextWindow(models, "unknown") != 128000 {
		t.Error("should return default for unknown model")
	}

	if GetContextWindow(nil, "anything") != 128000 {
		t.Error("should return default for nil models")
	}
}

func TestIsApproachingLimit(t *testing.T) {
	tests := []struct {
		name          string
		usedTokens    int
		contextWindow int
		threshold     float64
		expected      bool
	}{
		{"under_limit", 50000, 200000, 0.9, false},
		{"at_threshold", 180000, 200000, 0.9, true},
		{"over_limit", 190000, 200000, 0.9, true},
		{"zero_window", 100, 0, 0.9, false},
		{"exact_90", 90, 100, 0.9, true},
		{"just_under", 89, 100, 0.9, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsApproachingLimit(tt.usedTokens, tt.contextWindow, tt.threshold)
			if result != tt.expected {
				t.Errorf("IsApproachingLimit(%d, %d, %.1f) = %v, want %v",
					tt.usedTokens, tt.contextWindow, tt.threshold, result, tt.expected)
			}
		})
	}
}
