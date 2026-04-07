package ollama

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Edcko/techne-code/pkg/provider"
)

func TestDetectToolCall(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "valid tool call with name and arguments",
			content:  `{"name": "read_file", "arguments": {"path": "main.go"}}`,
			expected: true,
		},
		{
			name:     "valid tool call with empty arguments",
			content:  `{"name": "list_files", "arguments": {}}`,
			expected: true,
		},
		{
			name:     "valid tool call without arguments field",
			content:  `{"name": "get_time"}`,
			expected: true,
		},
		{
			name:     "plain text not a tool call",
			content:  "Hello, how can I help you?",
			expected: false,
		},
		{
			name:     "json without name field",
			content:  `{"foo": "bar", "baz": 123}`,
			expected: false,
		},
		{
			name:     "empty string",
			content:  "",
			expected: false,
		},
		{
			name:     "whitespace only",
			content:  "   ",
			expected: false,
		},
		{
			name:     "json array not an object",
			content:  `[{"name": "test"}]`,
			expected: false,
		},
		{
			name:     "invalid json",
			content:  `{not valid json}`,
			expected: false,
		},
		{
			name:     "json with name but null value",
			content:  `{"name": null}`,
			expected: false,
		},
		{
			name:     "json with empty name",
			content:  `{"name": ""}`,
			expected: false,
		},
		{
			name:     "valid tool call with string arguments",
			content:  `{"name": "calculator", "arguments": "2 + 2"}`,
			expected: true,
		},
		{
			name:     "valid tool call with whitespace",
			content:  `  {"name": "test_tool", "arguments": {}}  `,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectToolCall(tt.content)
			if result != tt.expected {
				t.Errorf("detectToolCall(%q) = %v, expected %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestParseToolCall(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expected    *ollamaToolCall
		expectError bool
	}{
		{
			name:    "valid tool call with object arguments",
			content: `{"name": "read_file", "arguments": {"path": "main.go"}}`,
			expected: &ollamaToolCall{
				Name:      "read_file",
				Arguments: json.RawMessage(`{"path": "main.go"}`),
			},
			expectError: false,
		},
		{
			name:    "valid tool call with empty arguments",
			content: `{"name": "list_files", "arguments": {}}`,
			expected: &ollamaToolCall{
				Name:      "list_files",
				Arguments: json.RawMessage(`{}`),
			},
			expectError: false,
		},
		{
			name:    "valid tool call without arguments",
			content: `{"name": "get_time"}`,
			expected: &ollamaToolCall{
				Name:      "get_time",
				Arguments: nil,
			},
			expectError: false,
		},
		{
			name:    "valid tool call with string arguments",
			content: `{"name": "calculator", "arguments": "2 + 2"}`,
			expected: &ollamaToolCall{
				Name:      "calculator",
				Arguments: json.RawMessage(`"2 + 2"`),
			},
			expectError: false,
		},
		{
			name:        "invalid json",
			content:     `{not valid}`,
			expectError: true,
		},
		{
			name:        "missing name field",
			content:     `{"arguments": {"path": "test"}}`,
			expectError: true,
		},
		{
			name:        "empty name",
			content:     `{"name": "", "arguments": {}}`,
			expectError: true,
		},
		{
			name:        "null name",
			content:     `{"name": null}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseToolCall(tt.content)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Name != tt.expected.Name {
				t.Errorf("expected name %q, got %q", tt.expected.Name, result.Name)
			}

			if tt.expected.Arguments != nil {
				if string(result.Arguments) != string(tt.expected.Arguments) {
					t.Errorf("expected arguments %s, got %s", string(tt.expected.Arguments), string(result.Arguments))
				}
			}
		})
	}
}

func TestGenerateToolCallID(t *testing.T) {
	id1 := generateToolCallID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateToolCallID()

	if id1 == id2 {
		t.Errorf("expected different IDs, got same: %s", id1)
	}

	if !strings.HasPrefix(id1, "call_") {
		t.Errorf("expected ID to start with 'call_', got %s", id1)
	}

	if len(id1) <= 5 {
		t.Errorf("expected ID to have content after 'call_', got %s", id1)
	}
}

func TestConvertOllamaResponse(t *testing.T) {
	tests := []struct {
		name         string
		inputResp    *provider.ChatResponse
		content      string
		expectedType provider.ContentBlockType
		expectedName string
	}{
		{
			name: "convert tool call in content",
			inputResp: &provider.ChatResponse{
				Model:      "qwen2.5-coder",
				StopReason: "end_turn",
				Content: []provider.ContentBlock{
					{Type: provider.BlockText, Text: `{"name": "read_file", "arguments": {"path": "main.go"}}`},
				},
				Usage: provider.Usage{InputTokens: 10, OutputTokens: 5},
			},
			content:      `{"name": "read_file", "arguments": {"path": "main.go"}}`,
			expectedType: provider.BlockToolCall,
			expectedName: "read_file",
		},
		{
			name: "pass through plain text",
			inputResp: &provider.ChatResponse{
				Model:      "qwen2.5-coder",
				StopReason: "end_turn",
				Content: []provider.ContentBlock{
					{Type: provider.BlockText, Text: "Hello, how can I help?"},
				},
				Usage: provider.Usage{InputTokens: 10, OutputTokens: 5},
			},
			content:      "Hello, how can I help?",
			expectedType: provider.BlockText,
			expectedName: "",
		},
		{
			name: "pass through invalid json",
			inputResp: &provider.ChatResponse{
				Model:      "qwen2.5-coder",
				StopReason: "end_turn",
				Content: []provider.ContentBlock{
					{Type: provider.BlockText, Text: "{not valid json}"},
				},
				Usage: provider.Usage{InputTokens: 10, OutputTokens: 5},
			},
			content:      "{not valid json}",
			expectedType: provider.BlockText,
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertOllamaResponse(tt.inputResp, tt.content)

			if len(result.Content) == 0 {
				t.Fatalf("expected content blocks")
			}

			if result.Content[0].Type != tt.expectedType {
				t.Errorf("expected type %s, got %s", tt.expectedType, result.Content[0].Type)
			}

			if tt.expectedType == provider.BlockToolCall {
				if result.Content[0].ToolCall == nil {
					t.Fatalf("expected tool call block, got nil")
				}
				if result.Content[0].ToolCall.Name != tt.expectedName {
					t.Errorf("expected name %s, got %s", tt.expectedName, result.Content[0].ToolCall.Name)
				}
				if result.StopReason != "tool_use" {
					t.Errorf("expected stop_reason 'tool_use', got %s", result.StopReason)
				}
			}

			if tt.expectedType == provider.BlockText {
				if result.Content[0].Text != tt.content {
					t.Errorf("expected text %q, got %q", tt.content, result.Content[0].Text)
				}
			}
		})
	}
}

func TestConvertOllamaResponsePreservesUsage(t *testing.T) {
	inputResp := &provider.ChatResponse{
		Model:      "qwen2.5-coder",
		StopReason: "end_turn",
		Content: []provider.ContentBlock{
			{Type: provider.BlockText, Text: `{"name": "test", "arguments": {}}`},
		},
		Usage: provider.Usage{InputTokens: 100, OutputTokens: 50},
	}

	result := convertOllamaResponse(inputResp, `{"name": "test", "arguments": {}}`)

	if result.Usage.InputTokens != 100 {
		t.Errorf("expected InputTokens 100, got %d", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 50 {
		t.Errorf("expected OutputTokens 50, got %d", result.Usage.OutputTokens)
	}
}

func TestConvertOllamaResponseGeneratesUniqueIDs(t *testing.T) {
	inputResp := &provider.ChatResponse{
		Model:      "qwen2.5-coder",
		StopReason: "end_turn",
		Content: []provider.ContentBlock{
			{Type: provider.BlockText, Text: `{"name": "test", "arguments": {}}`},
		},
	}

	result1 := convertOllamaResponse(inputResp, `{"name": "test", "arguments": {}}`)
	result2 := convertOllamaResponse(inputResp, `{"name": "test", "arguments": {}}`)

	if len(result1.Content) == 0 || len(result2.Content) == 0 {
		t.Fatal("expected content blocks")
	}

	if result1.Content[0].ToolCall == nil || result2.Content[0].ToolCall == nil {
		t.Fatal("expected tool calls")
	}

	if result1.Content[0].ToolCall.ID == result2.Content[0].ToolCall.ID {
		t.Errorf("expected unique IDs, got same: %s", result1.Content[0].ToolCall.ID)
	}
}
