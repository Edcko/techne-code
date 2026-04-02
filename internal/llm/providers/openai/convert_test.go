package openai

import (
	"encoding/json"
	"testing"

	"github.com/Edcko/techne-code/pkg/provider"
)

func TestConvertMessages(t *testing.T) {
	tests := []struct {
		name     string
		input    []provider.Message
		expected []chatMessage
	}{
		{
			name: "simple user message",
			input: []provider.Message{
				{
					Role: provider.RoleUser,
					Content: []provider.ContentBlock{
						{Type: provider.BlockText, Text: "Hello"},
					},
				},
			},
			expected: []chatMessage{
				{Role: "user", Content: "Hello"},
			},
		},
		{
			name: "system message",
			input: []provider.Message{
				{
					Role: provider.RoleSystem,
					Content: []provider.ContentBlock{
						{Type: provider.BlockText, Text: "You are helpful"},
					},
				},
			},
			expected: []chatMessage{
				{Role: "system", Content: "You are helpful"},
			},
		},
		{
			name: "assistant with tool calls",
			input: []provider.Message{
				{
					Role: provider.RoleAssistant,
					Content: []provider.ContentBlock{
						{Type: provider.BlockText, Text: "Let me help"},
						{
							Type: provider.BlockToolCall,
							ToolCall: &provider.ToolCallBlock{
								ID:    "call_123",
								Name:  "read_file",
								Input: json.RawMessage(`{"path": "main.go"}`),
							},
						},
					},
				},
			},
			expected: []chatMessage{
				{
					Role:    "assistant",
					Content: "Let me help",
					ToolCalls: []toolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: functionCall{
								Name:      "read_file",
								Arguments: `{"path": "main.go"}`,
							},
						},
					},
				},
			},
		},
		{
			name: "tool result message",
			input: []provider.Message{
				{
					Role: provider.RoleTool,
					Content: []provider.ContentBlock{
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
				},
			},
			expected: []chatMessage{
				{
					Role:       "tool",
					ToolCallID: "call_123",
					Content:    "file contents",
				},
			},
		},
		{
			name: "multiple messages",
			input: []provider.Message{
				{
					Role: provider.RoleUser,
					Content: []provider.ContentBlock{
						{Type: provider.BlockText, Text: "Hello"},
					},
				},
				{
					Role: provider.RoleAssistant,
					Content: []provider.ContentBlock{
						{Type: provider.BlockText, Text: "Hi there"},
					},
				},
			},
			expected: []chatMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertMessages(tt.input)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d messages, got %d", len(tt.expected), len(result))
			}

			for i, msg := range result {
				exp := tt.expected[i]
				if msg.Role != exp.Role {
					t.Errorf("message %d: expected role %s, got %s", i, exp.Role, msg.Role)
				}

				if msg.Role == "tool" {
					if msg.ToolCallID != exp.ToolCallID {
						t.Errorf("message %d: expected tool_call_id %s, got %s", i, exp.ToolCallID, msg.ToolCallID)
					}
					if msg.Content != exp.Content {
						t.Errorf("message %d: expected content %s, got %s", i, exp.Content, msg.Content)
					}
				} else {
					if msg.Content != exp.Content {
						t.Errorf("message %d: expected content %s, got %s", i, exp.Content, msg.Content)
					}
					if len(msg.ToolCalls) != len(exp.ToolCalls) {
						t.Errorf("message %d: expected %d tool calls, got %d", i, len(exp.ToolCalls), len(msg.ToolCalls))
					}
					for j, tc := range msg.ToolCalls {
						expTC := exp.ToolCalls[j]
						if tc.ID != expTC.ID {
							t.Errorf("message %d tool call %d: expected ID %s, got %s", i, j, expTC.ID, tc.ID)
						}
						if tc.Function.Name != expTC.Function.Name {
							t.Errorf("message %d tool call %d: expected name %s, got %s", i, j, expTC.Function.Name, tc.Function.Name)
						}
						if tc.Function.Arguments != expTC.Function.Arguments {
							t.Errorf("message %d tool call %d: expected arguments %s, got %s", i, j, expTC.Function.Arguments, tc.Function.Arguments)
						}
					}
				}
			}
		})
	}
}

func TestConvertTools(t *testing.T) {
	tests := []struct {
		name     string
		input    []provider.ToolDef
		expected []toolDef
	}{
		{
			name: "single tool",
			input: []provider.ToolDef{
				{
					Name:        "read_file",
					Description: "Read a file",
					Parameters:  json.RawMessage(`{"type": "object", "properties": {"path": {"type": "string"}}}`),
				},
			},
			expected: []toolDef{
				{
					Type: "function",
					Function: functionTool{
						Name:        "read_file",
						Description: "Read a file",
						Parameters: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"path": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			},
		},
		{
			name:     "empty tools",
			input:    []provider.ToolDef{},
			expected: []toolDef{},
		},
		{
			name: "tool with empty parameters",
			input: []provider.ToolDef{
				{
					Name:        "simple_tool",
					Description: "A simple tool",
					Parameters:  json.RawMessage{},
				},
			},
			expected: []toolDef{
				{
					Type: "function",
					Function: functionTool{
						Name:        "simple_tool",
						Description: "A simple tool",
						Parameters:  nil,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertTools(tt.input)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d tools, got %d", len(tt.expected), len(result))
			}

			for i, tool := range result {
				exp := tt.expected[i]
				if tool.Type != exp.Type {
					t.Errorf("tool %d: expected type %s, got %s", i, exp.Type, tool.Type)
				}
				if tool.Function.Name != exp.Function.Name {
					t.Errorf("tool %d: expected name %s, got %s", i, exp.Function.Name, tool.Function.Name)
				}
				if tool.Function.Description != exp.Function.Description {
					t.Errorf("tool %d: expected description %s, got %s", i, exp.Function.Description, tool.Function.Description)
				}
			}
		})
	}
}

func TestConvertResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    *chatResponse
		expected *provider.ChatResponse
	}{
		{
			name: "simple text response",
			input: &chatResponse{
				Model: "glm-5",
				Choices: []choice{
					{
						Message: chatMessage{
							Role:    "assistant",
							Content: "Hello!",
						},
						FinishReason: "stop",
					},
				},
				Usage: usage{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
			},
			expected: &provider.ChatResponse{
				Model:      "glm-5",
				StopReason: "end_turn",
				Content: []provider.ContentBlock{
					{Type: provider.BlockText, Text: "Hello!"},
				},
				Usage: provider.Usage{
					InputTokens:  10,
					OutputTokens: 5,
				},
			},
		},
		{
			name: "response with tool calls",
			input: &chatResponse{
				Model: "glm-5",
				Choices: []choice{
					{
						Message: chatMessage{
							Role:    "assistant",
							Content: "",
							ToolCalls: []toolCall{
								{
									ID:   "call_123",
									Type: "function",
									Function: functionCall{
										Name:      "read_file",
										Arguments: `{"path": "main.go"}`,
									},
								},
							},
						},
						FinishReason: "tool_calls",
					},
				},
				Usage: usage{
					PromptTokens:     15,
					CompletionTokens: 10,
					TotalTokens:      25,
				},
			},
			expected: &provider.ChatResponse{
				Model:      "glm-5",
				StopReason: "tool_use",
				Content: []provider.ContentBlock{
					{
						Type: provider.BlockToolCall,
						ToolCall: &provider.ToolCallBlock{
							ID:    "call_123",
							Name:  "read_file",
							Input: json.RawMessage(`{"path": "main.go"}`),
						},
					},
				},
				Usage: provider.Usage{
					InputTokens:  15,
					OutputTokens: 10,
				},
			},
		},
		{
			name: "response with max_tokens finish",
			input: &chatResponse{
				Model: "glm-5",
				Choices: []choice{
					{
						Message: chatMessage{
							Role:    "assistant",
							Content: "Truncated",
						},
						FinishReason: "length",
					},
				},
				Usage: usage{
					PromptTokens:     10,
					CompletionTokens: 100,
					TotalTokens:      110,
				},
			},
			expected: &provider.ChatResponse{
				Model:      "glm-5",
				StopReason: "max_tokens",
				Content: []provider.ContentBlock{
					{Type: provider.BlockText, Text: "Truncated"},
				},
				Usage: provider.Usage{
					InputTokens:  10,
					OutputTokens: 100,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertResponse(tt.input)

			if result.Model != tt.expected.Model {
				t.Errorf("expected model %s, got %s", tt.expected.Model, result.Model)
			}

			if result.StopReason != tt.expected.StopReason {
				t.Errorf("expected stop_reason %s, got %s", tt.expected.StopReason, result.StopReason)
			}

			if result.Usage.InputTokens != tt.expected.Usage.InputTokens {
				t.Errorf("expected input_tokens %d, got %d", tt.expected.Usage.InputTokens, result.Usage.InputTokens)
			}

			if result.Usage.OutputTokens != tt.expected.Usage.OutputTokens {
				t.Errorf("expected output_tokens %d, got %d", tt.expected.Usage.OutputTokens, result.Usage.OutputTokens)
			}

			if len(result.Content) != len(tt.expected.Content) {
				t.Fatalf("expected %d content blocks, got %d", len(tt.expected.Content), len(result.Content))
			}

			for i, block := range result.Content {
				exp := tt.expected.Content[i]
				if block.Type != exp.Type {
					t.Errorf("content block %d: expected type %s, got %s", i, exp.Type, block.Type)
				}
				if block.Type == provider.BlockText && block.Text != exp.Text {
					t.Errorf("content block %d: expected text %s, got %s", i, exp.Text, block.Text)
				}
				if block.Type == provider.BlockToolCall {
					if block.ToolCall.ID != exp.ToolCall.ID {
						t.Errorf("content block %d: expected tool call ID %s, got %s", i, exp.ToolCall.ID, block.ToolCall.ID)
					}
					if block.ToolCall.Name != exp.ToolCall.Name {
						t.Errorf("content block %d: expected tool call name %s, got %s", i, exp.ToolCall.Name, block.ToolCall.Name)
					}
				}
			}
		})
	}
}

func TestConvertFinishReason(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"stop", "end_turn"},
		{"length", "max_tokens"},
		{"tool_calls", "tool_use"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertFinishReason(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name     string
		input    []provider.ContentBlock
		expected string
	}{
		{
			name: "single text block",
			input: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Hello"},
			},
			expected: "Hello",
		},
		{
			name: "multiple text blocks",
			input: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Hello"},
				{Type: provider.BlockText, Text: "World"},
			},
			expected: "Hello\nWorld",
		},
		{
			name: "mixed blocks",
			input: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Text"},
				{Type: provider.BlockToolCall, ToolCall: &provider.ToolCallBlock{ID: "123"}},
				{Type: provider.BlockText, Text: "More"},
			},
			expected: "Text\nMore",
		},
		{
			name:     "empty blocks",
			input:    []provider.ContentBlock{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextContent(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    []provider.ContentBlock
		expected []toolCall
	}{
		{
			name: "single tool call",
			input: []provider.ContentBlock{
				{
					Type: provider.BlockToolCall,
					ToolCall: &provider.ToolCallBlock{
						ID:    "call_123",
						Name:  "read_file",
						Input: json.RawMessage(`{"path": "main.go"}`),
					},
				},
			},
			expected: []toolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: functionCall{
						Name:      "read_file",
						Arguments: `{"path": "main.go"}`,
					},
				},
			},
		},
		{
			name: "multiple tool calls",
			input: []provider.ContentBlock{
				{
					Type: provider.BlockToolCall,
					ToolCall: &provider.ToolCallBlock{
						ID:    "call_1",
						Name:  "read_file",
						Input: json.RawMessage(`{"path": "a.go"}`),
					},
				},
				{
					Type: provider.BlockToolCall,
					ToolCall: &provider.ToolCallBlock{
						ID:    "call_2",
						Name:  "write_file",
						Input: json.RawMessage(`{"path": "b.go"}`),
					},
				},
			},
			expected: []toolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: functionCall{
						Name:      "read_file",
						Arguments: `{"path": "a.go"}`,
					},
				},
				{
					ID:   "call_2",
					Type: "function",
					Function: functionCall{
						Name:      "write_file",
						Arguments: `{"path": "b.go"}`,
					},
				},
			},
		},
		{
			name:     "no tool calls",
			input:    []provider.ContentBlock{},
			expected: []toolCall{},
		},
		{
			name: "mixed blocks",
			input: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Some text"},
				{
					Type: provider.BlockToolCall,
					ToolCall: &provider.ToolCallBlock{
						ID:    "call_123",
						Name:  "test",
						Input: json.RawMessage(`{}`),
					},
				},
			},
			expected: []toolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: functionCall{
						Name:      "test",
						Arguments: `{}`,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractToolCalls(tt.input)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d tool calls, got %d", len(tt.expected), len(result))
			}

			for i, tc := range result {
				exp := tt.expected[i]
				if tc.ID != exp.ID {
					t.Errorf("tool call %d: expected ID %s, got %s", i, exp.ID, tc.ID)
				}
				if tc.Type != exp.Type {
					t.Errorf("tool call %d: expected type %s, got %s", i, exp.Type, tc.Type)
				}
				if tc.Function.Name != exp.Function.Name {
					t.Errorf("tool call %d: expected name %s, got %s", i, exp.Function.Name, tc.Function.Name)
				}
				if tc.Function.Arguments != exp.Function.Arguments {
					t.Errorf("tool call %d: expected arguments %s, got %s", i, exp.Function.Arguments, tc.Function.Arguments)
				}
			}
		})
	}
}
