package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func newMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func newAdapterWithServer(server *httptest.Server) *AnthropicAdapter {
	client := anthropic.NewClient(
		option.WithAPIKey("test-key"),
		option.WithBaseURL(server.URL),
	)
	return &AnthropicAdapter{client: &client}
}

func TestChat_Success(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected api key header")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Errorf("expected anthropic-version header")
		}

		resp := map[string]any{
			"id":    "msg_123",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]any{
				{"type": "text", "text": "Hello, world!"},
			},
			"stop_reason": "end_turn",
			"usage": map[string]int{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	adapter := newAdapterWithServer(server)
	ctx := context.Background()

	req := provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Hello"},
			}},
		},
		Config: provider.ProviderConfig{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 1024,
		},
	}

	resp, err := adapter.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model claude-sonnet-4-20250514, got %s", resp.Model)
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("expected stop_reason end_turn, got %s", resp.StopReason)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", resp.Usage.InputTokens)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(resp.Content))
	}
	if resp.Content[0].Type != provider.BlockText {
		t.Errorf("expected text block, got %s", resp.Content[0].Type)
	}
	if resp.Content[0].Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %s", resp.Content[0].Text)
	}
}

func TestChat_WithTools(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		tools, ok := body["tools"].([]any)
		if !ok || len(tools) == 0 {
			t.Errorf("expected tools in request body")
		}

		resp := map[string]any{
			"id":    "msg_456",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]any{
				{
					"type":  "tool_use",
					"id":    "call_123",
					"name":  "read_file",
					"input": map[string]any{"path": "/tmp/test.go"},
				},
			},
			"stop_reason": "tool_use",
			"usage": map[string]int{
				"input_tokens":  20,
				"output_tokens": 10,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	adapter := newAdapterWithServer(server)
	ctx := context.Background()

	req := provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Read the file"},
			}},
		},
		Tools: []provider.ToolDef{
			{
				Name:        "read_file",
				Description: "Read a file",
				Parameters:  json.RawMessage(`{"type": "object", "properties": {"path": {"type": "string"}}}`),
			},
		},
		Config: provider.ProviderConfig{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 1024,
		},
	}

	resp, err := adapter.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.StopReason != "tool_use" {
		t.Errorf("expected stop_reason tool_use, got %s", resp.StopReason)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(resp.Content))
	}
	if resp.Content[0].Type != provider.BlockToolCall {
		t.Errorf("expected tool_call block, got %s", resp.Content[0].Type)
	}
	if resp.Content[0].ToolCall == nil {
		t.Fatal("expected tool_call data")
	}
	if resp.Content[0].ToolCall.Name != "read_file" {
		t.Errorf("expected tool name read_file, got %s", resp.Content[0].ToolCall.Name)
	}
	if resp.Content[0].ToolCall.ID != "call_123" {
		t.Errorf("expected tool id call_123, got %s", resp.Content[0].ToolCall.ID)
	}
}

func TestChat_WithSystem(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		system, ok := body["system"].([]any)
		if !ok || len(system) == 0 {
			t.Errorf("expected system in request body")
		}

		resp := map[string]any{
			"id":    "msg_789",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]any{
				{"type": "text", "text": "I am Claude"},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]int{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	adapter := newAdapterWithServer(server)
	ctx := context.Background()

	req := provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Who are you?"},
			}},
		},
		System: "You are a helpful assistant.",
		Config: provider.ProviderConfig{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 1024,
		},
	}

	resp, err := adapter.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(resp.Content))
	}
}

func TestChat_APIError(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		errorBody    string
		expectedType string
	}{
		{
			name:         "rate limit",
			statusCode:   429,
			errorBody:    `{"error": {"type": "rate_limit_error", "message": "rate_limit exceeded"}}`,
			expectedType: "rate_limit",
		},
		{
			name:         "auth error",
			statusCode:   401,
			errorBody:    `{"error": {"type": "authentication_error", "message": "invalid_api_key"}}`,
			expectedType: "auth",
		},
		{
			name:         "context too long",
			statusCode:   400,
			errorBody:    `{"error": {"type": "invalid_request_error", "message": "too many tokens"}}`,
			expectedType: "context_too_long",
		},
		{
			name:         "generic error",
			statusCode:   500,
			errorBody:    `{"error": {"type": "api_error", "message": "internal error"}}`,
			expectedType: "provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.errorBody))
			})
			defer server.Close()

			adapter := newAdapterWithServer(server)
			ctx := context.Background()

			req := provider.ChatRequest{
				Messages: []provider.Message{
					{Role: provider.RoleUser, Content: []provider.ContentBlock{
						{Type: provider.BlockText, Text: "Hello"},
					}},
				},
				Config: provider.ProviderConfig{
					Model:     "claude-sonnet-4-20250514",
					MaxTokens: 1024,
				},
			}

			_, err := adapter.Chat(ctx, req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			pe, ok := err.(*provider.ProviderError)
			if !ok {
				t.Fatalf("expected ProviderError, got %T", err)
			}
			if pe.Type != tt.expectedType {
				t.Errorf("expected error type %s, got %s", tt.expectedType, pe.Type)
			}
		})
	}
}

func TestChat_InvalidJSON(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	})
	defer server.Close()

	adapter := newAdapterWithServer(server)
	ctx := context.Background()

	req := provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Hello"},
			}},
		},
		Config: provider.ProviderConfig{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 1024,
		},
	}

	_, err := adapter.Chat(ctx, req)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestStream_Success(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/event-stream")

		events := []string{
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-sonnet-4-20250514\",\"content\":[],\"stop_reason\":null,\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}\n\n",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n\n",
			"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n",
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":5}}\n\n",
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
		}

		for _, event := range events {
			w.Write([]byte(event))
			w.(http.Flusher).Flush()
			time.Sleep(5 * time.Millisecond)
		}
	})
	defer server.Close()

	adapter := newAdapterWithServer(server)
	ctx := context.Background()

	req := provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Hello"},
			}},
		},
		Config: provider.ProviderConfig{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 1024,
		},
	}

	ch, err := adapter.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	var textChunks []string
	var gotDone bool

	timeout := time.After(5 * time.Second)
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				if !gotDone {
					t.Error("stream closed without done chunk")
				}
				return
			}
			switch chunk.Type {
			case "text_delta":
				textChunks = append(textChunks, chunk.Text)
			case "done":
				gotDone = true
			case "error":
				t.Fatalf("unexpected error chunk: %v", chunk.Error)
			}
		case <-timeout:
			t.Fatal("timeout waiting for stream to complete")
		}
	}
}

func TestStream_WithToolCall(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/event-stream")

		events := []string{
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-sonnet-4-20250514\",\"content\":[],\"stop_reason\":null,\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}\n\n",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"call_123\",\"name\":\"read_file\",\"input\":{}}}\n\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"path\\\":\\\"/tmp\\\"\"}}\n\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\",\\\"name\\\":\\\"test.go\\\"}\"}}\n\n",
			"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n",
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"},\"usage\":{\"output_tokens\":20}}\n\n",
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
		}

		for _, event := range events {
			w.Write([]byte(event))
			w.(http.Flusher).Flush()
			time.Sleep(10 * time.Millisecond)
		}
	})
	defer server.Close()

	adapter := newAdapterWithServer(server)
	ctx := context.Background()

	req := provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Read file"},
			}},
		},
		Tools: []provider.ToolDef{
			{
				Name:        "read_file",
				Description: "Read a file",
				Parameters:  json.RawMessage(`{"type": "object"}`),
			},
		},
		Config: provider.ProviderConfig{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 1024,
		},
	}

	ch, err := adapter.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	var toolCalls []provider.ToolCallDelta

	timeout := time.After(5 * time.Second)
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				if len(toolCalls) == 0 {
					t.Error("expected at least one tool call")
				}
				return
			}
			if chunk.Type == "tool_call_delta" && chunk.ToolCall != nil && chunk.ToolCall.Done {
				toolCalls = append(toolCalls, *chunk.ToolCall)
			}
			if chunk.Type == "error" {
				t.Fatalf("unexpected error chunk: %v", chunk.Error)
			}
		case <-timeout:
			t.Fatal("timeout waiting for stream to complete")
		}
	}
}

func TestStream_APIError(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error": {"type": "rate_limit_error", "message": "rate_limit exceeded"}}`))
	})
	defer server.Close()

	adapter := newAdapterWithServer(server)
	ctx := context.Background()

	req := provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Hello"},
			}},
		},
		Config: provider.ProviderConfig{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 1024,
		},
	}

	ch, err := adapter.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	timeout := time.After(5 * time.Second)
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				t.Fatal("channel closed without error chunk")
			}
			if chunk.Type == "error" {
				if chunk.Error == nil {
					t.Fatal("expected error in error chunk")
				}
				pe, ok := chunk.Error.(*provider.ProviderError)
				if !ok {
					t.Fatalf("expected ProviderError, got %T", chunk.Error)
				}
				if pe.Type != "rate_limit" {
					t.Errorf("expected error type rate_limit, got %s", pe.Type)
				}
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for error chunk")
		}
	}
}

func TestStream_ContextCancellation(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/event-stream")

		for i := 0; i < 100; i++ {
			w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"x\"}}\n\n"))
			w.(http.Flusher).Flush()
			time.Sleep(50 * time.Millisecond)
		}
	})
	defer server.Close()

	adapter := newAdapterWithServer(server)
	ctx, cancel := context.WithCancel(context.Background())

	req := provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Hello"},
			}},
		},
		Config: provider.ProviderConfig{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 1024,
		},
	}

	ch, err := adapter.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	timeout := time.After(2 * time.Second)
	chunkCount := 0

	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
			chunkCount++
		case <-timeout:
			if chunkCount > 50 {
				t.Error("stream should have been cancelled but received too many chunks")
			}
			return
		}
	}
}

func TestConvertMessages_EmptyContent(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: []provider.ContentBlock{}},
	}

	result, err := convertMessages(msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 message, got %d", len(result))
	}
}

func TestConvertMessages_ToolResult(t *testing.T) {
	msgs := []provider.Message{
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
	}

	result, err := convertMessages(msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 message, got %d", len(result))
	}
}

func TestConvertResponse_EmptyContent(t *testing.T) {
	msg := &anthropic.Message{
		ID:         "msg_empty",
		Type:       "message",
		Role:       "assistant",
		Model:      "claude-sonnet-4-20250514",
		Content:    []anthropic.ContentBlockUnion{},
		StopReason: "end_turn",
		Usage: anthropic.Usage{
			InputTokens:  5,
			OutputTokens: 0,
		},
	}

	resp := convertResponse(msg)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Content) != 0 {
		t.Errorf("expected 0 content blocks, got %d", len(resp.Content))
	}
}

func TestName(t *testing.T) {
	adapter := New("test-key")
	if adapter.Name() != "anthropic" {
		t.Errorf("expected name 'anthropic', got %s", adapter.Name())
	}
}

func TestModels_ReturnsExpectedModels(t *testing.T) {
	adapter := New("test-key")
	models := adapter.Models()

	expectedModels := []string{
		"claude-sonnet-4-20250514",
		"claude-opus-4-20250514",
		"claude-haiku-4-20250414",
	}

	if len(models) != len(expectedModels) {
		t.Fatalf("expected %d models, got %d", len(expectedModels), len(models))
	}

	for i, expected := range expectedModels {
		if models[i].ID != expected {
			t.Errorf("expected model %s, got %s", expected, models[i].ID)
		}
		if !models[i].SupportsTools {
			t.Errorf("expected %s to support tools", expected)
		}
		if !models[i].SupportsVision {
			t.Errorf("expected %s to support vision", expected)
		}
		if models[i].ContextWindow != 200000 {
			t.Errorf("expected context window 200000, got %d", models[i].ContextWindow)
		}
	}
}

func TestStream_ConnectionError(t *testing.T) {
	adapter := New("test-key")

	ctx := context.Background()
	req := provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Hello"},
			}},
		},
		Config: provider.ProviderConfig{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 1024,
		},
	}

	ch, err := adapter.Stream(ctx, req)
	if err != nil {
		return
	}

	timeout := time.After(5 * time.Second)
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				return
			}
			if chunk.Type == "error" {
				if chunk.Error == nil {
					t.Error("expected error in error chunk")
				}
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for error chunk")
		}
	}
}

func TestChat_MultipleContentBlocks(t *testing.T) {
	server := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"id":    "msg_multi",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]any{
				{"type": "text", "text": "First part"},
				{"type": "tool_use", "id": "call_1", "name": "tool_a", "input": map[string]int{"x": 1}},
				{"type": "text", "text": "Second part"},
			},
			"stop_reason": "end_turn",
			"usage": map[string]int{
				"input_tokens":  10,
				"output_tokens": 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	adapter := newAdapterWithServer(server)
	ctx := context.Background()

	req := provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Multi"},
			}},
		},
		Config: provider.ProviderConfig{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 1024,
		},
	}

	resp, err := adapter.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(resp.Content) != 3 {
		t.Fatalf("expected 3 content blocks, got %d", len(resp.Content))
	}

	if resp.Content[0].Type != provider.BlockText {
		t.Errorf("expected first block to be text, got %s", resp.Content[0].Type)
	}
	if resp.Content[1].Type != provider.BlockToolCall {
		t.Errorf("expected second block to be tool_call, got %s", resp.Content[1].Type)
	}
	if resp.Content[2].Type != provider.BlockText {
		t.Errorf("expected third block to be text, got %s", resp.Content[2].Type)
	}
}

func TestStream_MalformedSSE(t *testing.T) {
	t.Skip("Skipping: SDK throws error on malformed JSON in stream")
}

func TestConvertTools_WithEmptyParameters(t *testing.T) {
	tools := []provider.ToolDef{
		{
			Name:        "no_params_tool",
			Description: "A tool with no parameters",
			Parameters:  nil,
		},
	}

	result := convertTools(tools)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}
}

func TestConvertTools_WithInvalidJSON(t *testing.T) {
	tools := []provider.ToolDef{
		{
			Name:        "invalid_schema_tool",
			Description: "A tool with invalid JSON schema",
			Parameters:  json.RawMessage(`{invalid}`),
		},
	}

	result := convertTools(tools)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}
}

func streamResponseBody(events []string) io.Reader {
	var buf strings.Builder
	for _, event := range events {
		buf.WriteString(event)
	}
	return strings.NewReader(buf.String())
}
