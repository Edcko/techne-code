package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/Edcko/techne-code/pkg/provider"
)

func TestConvertUserContent(t *testing.T) {
	blocks := []provider.ContentBlock{
		{Type: provider.BlockText, Text: "hello"},
		{Type: provider.BlockText, Text: "world"},
	}
	result := convertUserContent(blocks)
	if len(result) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(result))
	}
}

func TestConvertAssistantContent_Text(t *testing.T) {
	blocks := []provider.ContentBlock{
		{Type: provider.BlockText, Text: "response"},
	}
	result := convertAssistantContent(blocks)
	if len(result) != 1 {
		t.Fatalf("expected 1 block, got %d", len(result))
	}
}

func TestConvertAssistantContent_ToolCall(t *testing.T) {
	input := json.RawMessage(`{"path": "/tmp/test.go"}`)
	blocks := []provider.ContentBlock{
		{
			Type: provider.BlockToolCall,
			ToolCall: &provider.ToolCallBlock{
				ID:    "call_123",
				Name:  "read_file",
				Input: input,
			},
		},
	}
	result := convertAssistantContent(blocks)
	if len(result) != 1 {
		t.Fatalf("expected 1 block, got %d", len(result))
	}
}

func TestConvertToolResultContent(t *testing.T) {
	tests := []struct {
		name     string
		blocks   []provider.ContentBlock
		expected int
	}{
		{
			name: "success result",
			blocks: []provider.ContentBlock{
				{
					Type: provider.BlockToolResult,
					ToolResult: &provider.ToolResultBlock{
						ToolCallID: "call_123",
						Name:       "read_file",
						Content:    "file contents",
						IsError:    false,
					},
				},
			},
			expected: 1,
		},
		{
			name: "error result",
			blocks: []provider.ContentBlock{
				{
					Type: provider.BlockToolResult,
					ToolResult: &provider.ToolResultBlock{
						ToolCallID: "call_456",
						Name:       "bash",
						Content:    "command failed",
						IsError:    true,
					},
				},
			},
			expected: 1,
		},
		{
			name:     "empty blocks",
			blocks:   []provider.ContentBlock{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToolResultContent(tt.blocks)
			if len(result) != tt.expected {
				t.Errorf("expected %d blocks, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestConvertTools(t *testing.T) {
	params := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "file path"}
		},
		"required": ["path"]
	}`)

	tools := []provider.ToolDef{
		{
			Name:        "read_file",
			Description: "Read a file",
			Parameters:  params,
		},
	}

	result := convertTools(tools)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}
}

func TestConvertTools_Empty(t *testing.T) {
	result := convertTools(nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 tools, got %d", len(result))
	}
}

func TestConvertError(t *testing.T) {
	tests := []struct {
		name      string
		errMsg    string
		wantType  string
		wantRetry bool
	}{
		{"rate limit", "rate_limit exceeded 429", "rate_limit", true},
		{"auth error", "authentication error 401 invalid_api_key", "auth", false},
		{"context too long", "too many tokens in context", "context_too_long", false},
		{"timeout", "request timeout after 30s", "timeout", true},
		{"generic error", "something went wrong", "provider", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := convertError(javaError{msg: tt.errMsg})
			if pe.Type != tt.wantType {
				t.Errorf("expected type %q, got %q", tt.wantType, pe.Type)
			}
			if pe.Retry != tt.wantRetry {
				t.Errorf("expected retry %v, got %v", tt.wantRetry, pe.Retry)
			}
		})
	}
}

// javaError is a simple error for testing
type javaError struct {
	msg string
}

func (e javaError) Error() string { return e.msg }

func TestNew(t *testing.T) {
	adapter := New("test-key")
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if adapter.Name() != "anthropic" {
		t.Errorf("expected name 'anthropic', got %q", adapter.Name())
	}
}

func TestModels(t *testing.T) {
	adapter := New("test-key")
	models := adapter.Models()
	if len(models) != 3 {
		t.Fatalf("expected 3 models, got %d", len(models))
	}
	found := false
	for _, m := range models {
		if m.ID == "claude-sonnet-4-20250514" {
			found = true
			if !m.SupportsTools {
				t.Error("expected SupportsTools=true")
			}
		}
	}
	if !found {
		t.Error("expected to find claude-sonnet-4-20250514")
	}
}
