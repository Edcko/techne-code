package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Edcko/techne-code/pkg/provider"
)

func TestAdapterName(t *testing.T) {
	a := NewAdapter("key", "", nil)
	if a.Name() != "gemini" {
		t.Errorf("expected name 'gemini', got %s", a.Name())
	}
}

func TestAdapterModels(t *testing.T) {
	a := NewAdapter("key", "", nil)
	models := a.Models()
	if len(models) != 3 {
		t.Fatalf("expected 3 models, got %d", len(models))
	}

	expectedIDs := []string{"gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.0-flash"}
	for i, expected := range expectedIDs {
		if models[i].ID != expected {
			t.Errorf("model %d: expected ID %s, got %s", i, expected, models[i].ID)
		}
	}

	for _, m := range models {
		if !m.SupportsTools {
			t.Errorf("model %s should support tools", m.ID)
		}
		if !m.SupportsVision {
			t.Errorf("model %s should support vision", m.ID)
		}
	}
}

func TestAdapterModelsCustom(t *testing.T) {
	custom := []provider.ModelInfo{
		{ID: "custom-model", MaxTokens: 1000, SupportsTools: false, ContextWindow: 5000},
	}
	a := NewAdapter("key", "", custom)
	models := a.Models()
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].ID != "custom-model" {
		t.Errorf("expected custom-model, got %s", models[0].ID)
	}
}

func TestChatSimpleText(t *testing.T) {
	respBody := generateResponse{
		Candidates: []candidate{
			{
				Content: &content{
					Role:  "model",
					Parts: []part{{Text: "Hello from Gemini!"}},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: &usageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 5,
			TotalTokenCount:      15,
		},
		ModelVersion: "gemini-2.0-flash",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Errorf("expected generateContent endpoint, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("expected API key in query params")
		}

		var req generateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if len(req.Contents) == 0 {
			t.Error("expected contents in request")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respBody)
	}))
	defer server.Close()

	a := NewAdapter("test-key", server.URL, nil)
	resp, err := a.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hello"}}},
		},
		Config: provider.ProviderConfig{Model: "gemini-2.0-flash"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Model != "gemini-2.0-flash" {
		t.Errorf("expected model gemini-2.0-flash, got %s", resp.Model)
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("expected stop_reason end_turn, got %s", resp.StopReason)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(resp.Content))
	}
	if resp.Content[0].Type != provider.BlockText {
		t.Errorf("expected text block, got %s", resp.Content[0].Type)
	}
	if resp.Content[0].Text != "Hello from Gemini!" {
		t.Errorf("expected text 'Hello from Gemini!', got %s", resp.Content[0].Text)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected input tokens 10, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("expected output tokens 5, got %d", resp.Usage.OutputTokens)
	}
}

func TestChatWithToolCall(t *testing.T) {
	respBody := generateResponse{
		Candidates: []candidate{
			{
				Content: &content{
					Role: "model",
					Parts: []part{
						{
							FunctionCall: &functionCallPart{
								Name: "read_file",
								Args: json.RawMessage(`{"path": "main.go"}`),
							},
						},
					},
				},
				FinishReason: "STOP",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respBody)
	}))
	defer server.Close()

	a := NewAdapter("test-key", server.URL, nil)
	resp, err := a.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Read main.go"}}},
		},
		Tools: []provider.ToolDef{
			{Name: "read_file", Description: "Read a file", Parameters: json.RawMessage(`{"type": "object"}`)},
		},
		Config: provider.ProviderConfig{Model: "gemini-2.5-flash"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
		t.Fatal("expected tool call data")
	}
	if resp.Content[0].ToolCall.Name != "read_file" {
		t.Errorf("expected tool name read_file, got %s", resp.Content[0].ToolCall.Name)
	}
	var expectedArgs map[string]string
	json.Unmarshal(resp.Content[0].ToolCall.Input, &expectedArgs)
	if expectedArgs["path"] != "main.go" {
		t.Errorf("expected path 'main.go', got %s", expectedArgs["path"])
	}
	if resp.Content[0].ToolCall.ID == "" {
		t.Error("expected non-empty tool call ID")
	}
}

func TestChatWithSystemPrompt(t *testing.T) {
	var receivedReq generateRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		respBody := generateResponse{
			Candidates: []candidate{{
				Content:      &content{Role: "model", Parts: []part{{Text: "ok"}}},
				FinishReason: "STOP",
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respBody)
	}))
	defer server.Close()

	a := NewAdapter("test-key", server.URL, nil)
	_, err := a.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hi"}}},
		},
		System: "You are a coding assistant",
		Config: provider.ProviderConfig{Model: "gemini-2.0-flash"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.SystemInstruction == nil {
		t.Fatal("expected systemInstruction to be set")
	}
	if len(receivedReq.SystemInstruction.Parts) == 0 || receivedReq.SystemInstruction.Parts[0].Text != "You are a coding assistant" {
		t.Error("system instruction text mismatch")
	}
}

func TestChatError401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"error":{"code":401,"message":"API key not valid"}}`)
	}))
	defer server.Close()

	a := NewAdapter("bad-key", server.URL, nil)
	_, err := a.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hi"}}},
		},
		Config: provider.ProviderConfig{Model: "gemini-2.0-flash"},
	})

	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	pe, ok := err.(*provider.ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if pe.Type != "auth" {
		t.Errorf("expected error type auth, got %s", pe.Type)
	}
	if pe.StatusCode != 401 {
		t.Errorf("expected status code 401, got %d", pe.StatusCode)
	}
}

func TestChatError429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		fmt.Fprint(w, `{"error":{"code":429,"message":"Resource exhausted"}}`)
	}))
	defer server.Close()

	a := NewAdapter("test-key", server.URL, nil)
	_, err := a.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hi"}}},
		},
		Config: provider.ProviderConfig{Model: "gemini-2.0-flash"},
	})

	if err == nil {
		t.Fatal("expected error for 429 response")
	}

	pe, ok := err.(*provider.ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if pe.Type != "rate_limit" {
		t.Errorf("expected error type rate_limit, got %s", pe.Type)
	}
	if !pe.Retry {
		t.Error("expected retry to be true for rate limit")
	}
}

func TestChatError400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		fmt.Fprint(w, `{"error":{"code":400,"message":"Request too long"}}`)
	}))
	defer server.Close()

	a := NewAdapter("test-key", server.URL, nil)
	_, err := a.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hi"}}},
		},
		Config: provider.ProviderConfig{Model: "gemini-2.0-flash"},
	})

	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	pe, ok := err.(*provider.ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if pe.Type != "context_too_long" {
		t.Errorf("expected error type context_too_long, got %s", pe.Type)
	}
}

func TestChatMaxTokens(t *testing.T) {
	respBody := generateResponse{
		Candidates: []candidate{
			{
				Content: &content{
					Role:  "model",
					Parts: []part{{Text: "Truncated..."}},
				},
				FinishReason: "MAX_TOKENS",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respBody)
	}))
	defer server.Close()

	a := NewAdapter("test-key", server.URL, nil)
	resp, err := a.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hi"}}},
		},
		Config: provider.ProviderConfig{Model: "gemini-2.0-flash"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StopReason != "max_tokens" {
		t.Errorf("expected stop_reason max_tokens, got %s", resp.StopReason)
	}
}

func TestStreamText(t *testing.T) {
	chunk1 := generateResponse{
		Candidates: []candidate{{
			Content: &content{Role: "model", Parts: []part{{Text: "Hello"}}},
		}},
	}
	chunk2 := generateResponse{
		Candidates: []candidate{{
			Content: &content{Role: "model", Parts: []part{{Text: " world"}}},
		}},
	}
	chunk3 := generateResponse{
		Candidates: []candidate{{
			Content:      &content{Role: "model", Parts: []part{{Text: "!"}}},
			FinishReason: "STOP",
		}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":streamGenerateContent") {
			t.Errorf("expected streamGenerateContent endpoint, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("alt") != "sse" {
			t.Errorf("expected alt=sse query param")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		data1, _ := json.Marshal(chunk1)
		fmt.Fprintf(w, "data: %s\n\n", data1)
		flusher.Flush()

		data2, _ := json.Marshal(chunk2)
		fmt.Fprintf(w, "data: %s\n\n", data2)
		flusher.Flush()

		data3, _ := json.Marshal(chunk3)
		fmt.Fprintf(w, "data: %s\n\n", data3)
		flusher.Flush()
	}))
	defer server.Close()

	a := NewAdapter("test-key", server.URL, nil)
	ch, err := a.Stream(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hello"}}},
		},
		Config: provider.ProviderConfig{Model: "gemini-2.0-flash"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var texts []string
	var gotDone bool
	for chunk := range ch {
		switch chunk.Type {
		case "text_delta":
			texts = append(texts, chunk.Text)
		case "done":
			gotDone = true
		}
	}

	if !gotDone {
		t.Error("expected done chunk")
	}
	if len(texts) != 3 {
		t.Fatalf("expected 3 text chunks, got %d", len(texts))
	}
	expected := []string{"Hello", " world", "!"}
	for i, exp := range expected {
		if texts[i] != exp {
			t.Errorf("chunk %d: expected %q, got %q", i, exp, texts[i])
		}
	}
}

func TestStreamWithToolCall(t *testing.T) {
	chunk1 := generateResponse{
		Candidates: []candidate{{
			Content: &content{Role: "model", Parts: []part{{Text: "Let me read that."}}},
		}},
	}
	chunk2 := generateResponse{
		Candidates: []candidate{{
			Content: &content{
				Role: "model",
				Parts: []part{{
					FunctionCall: &functionCallPart{
						Name: "read_file",
						Args: json.RawMessage(`{"path": "main.go"}`),
					},
				}},
			},
			FinishReason: "STOP",
		}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		data1, _ := json.Marshal(chunk1)
		fmt.Fprintf(w, "data: %s\n\n", data1)
		flusher.Flush()

		data2, _ := json.Marshal(chunk2)
		fmt.Fprintf(w, "data: %s\n\n", data2)
		flusher.Flush()
	}))
	defer server.Close()

	a := NewAdapter("test-key", server.URL, nil)
	ch, err := a.Stream(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Read main.go"}}},
		},
		Tools: []provider.ToolDef{
			{Name: "read_file", Description: "Read a file", Parameters: json.RawMessage(`{"type": "object"}`)},
		},
		Config: provider.ProviderConfig{Model: "gemini-2.0-flash"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var texts []string
	var toolCalls []*provider.ToolCallDelta
	var gotDone bool
	for chunk := range ch {
		switch chunk.Type {
		case "text_delta":
			texts = append(texts, chunk.Text)
		case "tool_call_delta":
			toolCalls = append(toolCalls, chunk.ToolCall)
		case "done":
			gotDone = true
		}
	}

	if !gotDone {
		t.Error("expected done chunk")
	}
	if len(texts) != 1 || texts[0] != "Let me read that." {
		t.Errorf("unexpected text chunks: %v", texts)
	}
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
	}
	if toolCalls[0].Name != "read_file" {
		t.Errorf("expected tool name read_file, got %s", toolCalls[0].Name)
	}
	var args map[string]string
	json.Unmarshal([]byte(toolCalls[0].InputJSON), &args)
	if args["path"] != "main.go" {
		t.Errorf("expected path 'main.go', got %s", args["path"])
	}
	if !toolCalls[0].Done {
		t.Error("expected tool call to be done")
	}
	if toolCalls[0].ID == "" {
		t.Error("expected non-empty tool call ID")
	}
}

func TestStreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"error":{"code":500,"message":"Internal error"}}`)
	}))
	defer server.Close()

	a := NewAdapter("test-key", server.URL, nil)
	_, err := a.Stream(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hi"}}},
		},
		Config: provider.ProviderConfig{Model: "gemini-2.0-flash"},
	})

	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	pe, ok := err.(*provider.ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if pe.Type != "provider" {
		t.Errorf("expected error type provider, got %s", pe.Type)
	}
}

func TestConvertMessagesUser(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hello"}}},
	}
	result := convertMessages(msgs)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Role != "user" {
		t.Errorf("expected role user, got %s", result[0].Role)
	}
	if len(result[0].Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(result[0].Parts))
	}
	if result[0].Parts[0].Text != "Hello" {
		t.Errorf("expected text Hello, got %s", result[0].Parts[0].Text)
	}
}

func TestConvertMessagesAssistant(t *testing.T) {
	msgs := []provider.Message{
		{
			Role: provider.RoleAssistant,
			Content: []provider.ContentBlock{
				{Type: provider.BlockText, Text: "Let me help"},
				{
					Type: provider.BlockToolCall,
					ToolCall: &provider.ToolCallBlock{
						ID:    "call_1",
						Name:  "read_file",
						Input: json.RawMessage(`{"path": "test.go"}`),
					},
				},
			},
		},
	}
	result := convertMessages(msgs)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Role != "model" {
		t.Errorf("expected role model, got %s", result[0].Role)
	}
	if len(result[0].Parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(result[0].Parts))
	}
	if result[0].Parts[0].Text != "Let me help" {
		t.Errorf("expected text 'Let me help', got %s", result[0].Parts[0].Text)
	}
	if result[0].Parts[1].FunctionCall == nil {
		t.Fatal("expected function call part")
	}
	if result[0].Parts[1].FunctionCall.Name != "read_file" {
		t.Errorf("expected function name read_file, got %s", result[0].Parts[1].FunctionCall.Name)
	}
}

func TestConvertMessagesToolResult(t *testing.T) {
	msgs := []provider.Message{
		{
			Role: provider.RoleTool,
			Content: []provider.ContentBlock{
				{
					Type: provider.BlockToolResult,
					ToolResult: &provider.ToolResultBlock{
						ToolCallID: "call_1",
						Name:       "read_file",
						Content:    "file contents here",
					},
				},
			},
		},
	}
	result := convertMessages(msgs)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Role != "function" {
		t.Errorf("expected role function, got %s", result[0].Role)
	}
	if len(result[0].Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(result[0].Parts))
	}
	if result[0].Parts[0].FunctionResponse == nil {
		t.Fatal("expected function response part")
	}
	if result[0].Parts[0].FunctionResponse.Name != "read_file" {
		t.Errorf("expected name read_file, got %s", result[0].Parts[0].FunctionResponse.Name)
	}
}

func TestConvertMessagesSkipsSystem(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "System prompt"}}},
		{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hello"}}},
	}
	result := convertMessages(msgs)

	if len(result) != 1 {
		t.Fatalf("expected 1 message (system skipped), got %d", len(result))
	}
	if result[0].Role != "user" {
		t.Errorf("expected role user, got %s", result[0].Role)
	}
}

func TestConvertToolsConversion(t *testing.T) {
	tools := []provider.ToolDef{
		{
			Name:        "read_file",
			Description: "Read a file",
			Parameters:  json.RawMessage(`{"type": "object", "properties": {"path": {"type": "string"}}}`),
		},
	}
	result := convertTools(tools)

	if len(result) != 1 {
		t.Fatalf("expected 1 tool config, got %d", len(result))
	}
	if len(result[0].FunctionDeclarations) != 1 {
		t.Fatalf("expected 1 function declaration, got %d", len(result[0].FunctionDeclarations))
	}
	decl := result[0].FunctionDeclarations[0]
	if decl.Name != "read_file" {
		t.Errorf("expected name read_file, got %s", decl.Name)
	}
	if decl.Description != "Read a file" {
		t.Errorf("expected description 'Read a file', got %s", decl.Description)
	}
}

func TestConvertToolsEmpty(t *testing.T) {
	result := convertTools(nil)
	if result != nil {
		t.Errorf("expected nil for empty tools, got %v", result)
	}

	result = convertTools([]provider.ToolDef{})
	if result != nil {
		t.Errorf("expected nil for empty tools, got %v", result)
	}
}

func TestConvertResponseSimple(t *testing.T) {
	resp := &generateResponse{
		ModelVersion: "gemini-2.0-flash",
		Candidates: []candidate{{
			Content:      &content{Role: "model", Parts: []part{{Text: "Hi!"}}},
			FinishReason: "STOP",
		}},
		UsageMetadata: &usageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 3,
		},
	}

	result := convertResponse(resp)

	if result.Model != "gemini-2.0-flash" {
		t.Errorf("expected model gemini-2.0-flash, got %s", result.Model)
	}
	if result.StopReason != "end_turn" {
		t.Errorf("expected stop_reason end_turn, got %s", result.StopReason)
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}
	if result.Content[0].Text != "Hi!" {
		t.Errorf("expected text 'Hi!', got %s", result.Content[0].Text)
	}
	if result.Usage.InputTokens != 10 {
		t.Errorf("expected input tokens 10, got %d", result.Usage.InputTokens)
	}
}

func TestConvertResponseToolCall(t *testing.T) {
	resp := &generateResponse{
		Candidates: []candidate{{
			Content: &content{
				Role: "model",
				Parts: []part{
					{Text: "Let me check."},
					{
						FunctionCall: &functionCallPart{
							Name: "bash",
							Args: json.RawMessage(`{"command": "ls"}`),
						},
					},
				},
			},
			FinishReason: "STOP",
		}},
	}

	result := convertResponse(resp)

	if result.StopReason != "tool_use" {
		t.Errorf("expected stop_reason tool_use, got %s", result.StopReason)
	}
	if len(result.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(result.Content))
	}
	if result.Content[0].Type != provider.BlockText {
		t.Errorf("expected first block to be text")
	}
	if result.Content[1].Type != provider.BlockToolCall {
		t.Errorf("expected second block to be tool_call")
	}
	if result.Content[1].ToolCall.Name != "bash" {
		t.Errorf("expected tool name bash, got %s", result.Content[1].ToolCall.Name)
	}
}

func TestConvertResponseNoArgs(t *testing.T) {
	resp := &generateResponse{
		Candidates: []candidate{{
			Content: &content{
				Role: "model",
				Parts: []part{{
					FunctionCall: &functionCallPart{
						Name: "list_files",
					},
				}},
			},
			FinishReason: "STOP",
		}},
	}

	result := convertResponse(resp)

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}
	if string(result.Content[0].ToolCall.Input) != "{}" {
		t.Errorf("expected empty object for nil args, got %s", string(result.Content[0].ToolCall.Input))
	}
}

func TestConvertResponseEmpty(t *testing.T) {
	resp := &generateResponse{}

	result := convertResponse(resp)

	if result.StopReason != "end_turn" {
		t.Errorf("expected default stop_reason end_turn, got %s", result.StopReason)
	}
	if len(result.Content) != 0 {
		t.Errorf("expected no content blocks, got %d", len(result.Content))
	}
}

func TestConvertFinishReason(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"STOP", "end_turn"},
		{"stop", "end_turn"},
		{"Stop", "end_turn"},
		{"MAX_TOKENS", "max_tokens"},
		{"SAFETY", "end_turn"},
		{"RECITATION", "end_turn"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertFinishReason(tt.input)
			if result != tt.expected {
				t.Errorf("convertFinishReason(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		statusCode    int
		expectedType  string
		expectedRetry bool
	}{
		{"rate limit 429", fmt.Errorf("too many requests"), 429, "rate_limit", true},
		{"resource exhausted", fmt.Errorf("resource_exhausted"), 0, "rate_limit", true},
		{"unauthorized 401", fmt.Errorf("unauthorized"), 401, "auth", false},
		{"forbidden 403", fmt.Errorf("forbidden"), 403, "auth", false},
		{"api key invalid", fmt.Errorf("api_key not valid"), 0, "auth", false},
		{"bad request 400", fmt.Errorf("bad request"), 400, "context_too_long", false},
		{"timeout", fmt.Errorf("connection timeout"), 0, "timeout", true},
		{"generic error", fmt.Errorf("something went wrong"), 500, "provider", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pe := convertError(tt.err, tt.statusCode)
			if pe.Type != tt.expectedType {
				t.Errorf("expected type %s, got %s", tt.expectedType, pe.Type)
			}
			if pe.Retry != tt.expectedRetry {
				t.Errorf("expected retry %v, got %v", tt.expectedRetry, pe.Retry)
			}
			if pe.StatusCode != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, pe.StatusCode)
			}
		})
	}
}

func TestGenerateCallID(t *testing.T) {
	id1 := generateCallID()
	time.Sleep(1 * time.Millisecond)
	id2 := generateCallID()

	if id1 == id2 {
		t.Errorf("expected different IDs, got same: %s", id1)
	}
	if !strings.HasPrefix(id1, "call_") {
		t.Errorf("expected ID to start with 'call_', got %s", id1)
	}
}

func TestChatSendsToolsInRequest(t *testing.T) {
	var receivedReq generateRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		respBody := generateResponse{
			Candidates: []candidate{{
				Content:      &content{Role: "model", Parts: []part{{Text: "ok"}}},
				FinishReason: "STOP",
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respBody)
	}))
	defer server.Close()

	a := NewAdapter("test-key", server.URL, nil)
	_, err := a.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hi"}}},
		},
		Tools: []provider.ToolDef{
			{Name: "bash", Description: "Run a command", Parameters: json.RawMessage(`{"type": "object"}`)},
			{Name: "read_file", Description: "Read a file", Parameters: json.RawMessage(`{"type": "object"}`)},
		},
		Config: provider.ProviderConfig{Model: "gemini-2.0-flash"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedReq.Tools) != 1 {
		t.Fatalf("expected 1 tool config, got %d", len(receivedReq.Tools))
	}
	if len(receivedReq.Tools[0].FunctionDeclarations) != 2 {
		t.Fatalf("expected 2 function declarations, got %d", len(receivedReq.Tools[0].FunctionDeclarations))
	}
	if receivedReq.Tools[0].FunctionDeclarations[0].Name != "bash" {
		t.Errorf("expected first tool bash, got %s", receivedReq.Tools[0].FunctionDeclarations[0].Name)
	}
	if receivedReq.Tools[0].FunctionDeclarations[1].Name != "read_file" {
		t.Errorf("expected second tool read_file, got %s", receivedReq.Tools[0].FunctionDeclarations[1].Name)
	}
}

func TestChatSendsGenerationConfig(t *testing.T) {
	var receivedReq generateRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)
		respBody := generateResponse{
			Candidates: []candidate{{
				Content:      &content{Role: "model", Parts: []part{{Text: "ok"}}},
				FinishReason: "STOP",
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respBody)
	}))
	defer server.Close()

	a := NewAdapter("test-key", server.URL, nil)
	_, err := a.Chat(context.Background(), provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hi"}}},
		},
		Config: provider.ProviderConfig{
			Model:       "gemini-2.0-flash",
			MaxTokens:   2048,
			Temperature: 0.7,
			TopP:        0.9,
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedReq.GenerationConfig == nil {
		t.Fatal("expected generation config to be set")
	}
	if receivedReq.GenerationConfig.MaxOutputTokens != 2048 {
		t.Errorf("expected max output tokens 2048, got %d", receivedReq.GenerationConfig.MaxOutputTokens)
	}
	if receivedReq.GenerationConfig.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", receivedReq.GenerationConfig.Temperature)
	}
	if receivedReq.GenerationConfig.TopP != 0.9 {
		t.Errorf("expected top_p 0.9, got %f", receivedReq.GenerationConfig.TopP)
	}
}

func TestDefaultBaseURL(t *testing.T) {
	a := NewAdapter("key", "", nil)
	if a.baseURL != "https://generativelanguage.googleapis.com/v1beta" {
		t.Errorf("expected default base URL, got %s", a.baseURL)
	}
}

func TestCustomBaseURL(t *testing.T) {
	a := NewAdapter("key", "https://custom.api.com/v1", nil)
	if a.baseURL != "https://custom.api.com/v1" {
		t.Errorf("expected custom base URL, got %s", a.baseURL)
	}
}

func TestBaseURLTrailingSlash(t *testing.T) {
	a := NewAdapter("key", "https://custom.api.com/v1/", nil)
	if a.baseURL != "https://custom.api.com/v1" {
		t.Errorf("expected trailing slash trimmed, got %s", a.baseURL)
	}
}
