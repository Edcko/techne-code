package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Edcko/techne-code/internal/llm/providers/openai"
	"github.com/Edcko/techne-code/pkg/provider"
)

var defaultBaseURL = "http://localhost:11434/v1"

var ollamaModels = []provider.ModelInfo{
	{ID: "llama3", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 8192},
	{ID: "llama3.1", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 128000},
	{ID: "llama3.2", MaxTokens: 4096, SupportsTools: true, SupportsVision: true, ContextWindow: 128000},
	{ID: "codellama", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 16384},
	{ID: "deepseek-coder", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 16384},
	{ID: "qwen2.5-coder", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 32768},
	{ID: "mistral", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 32768},
}

type ollamaToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type Adapter struct {
	*openai.Adapter
	baseURL string
	client  *http.Client
}

func New(baseURL string) *Adapter {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Adapter{
		Adapter: openai.NewAdapter("ollama", baseURL, ollamaModels),
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: 10 * time.Minute,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 5 * time.Minute,
			},
		},
	}
}

func (a *Adapter) Name() string {
	return "ollama"
}

func detectToolCall(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if !strings.HasPrefix(content, "{") || !strings.HasSuffix(content, "}") {
		return false
	}
	var tc ollamaToolCall
	if err := json.Unmarshal([]byte(content), &tc); err != nil {
		return false
	}
	return tc.Name != ""
}

func parseToolCall(content string) (*ollamaToolCall, error) {
	var tc ollamaToolCall
	if err := json.Unmarshal([]byte(content), &tc); err != nil {
		return nil, fmt.Errorf("parse tool call: %w", err)
	}
	if tc.Name == "" {
		return nil, fmt.Errorf("tool call missing name field")
	}
	return &tc, nil
}

func generateToolCallID() string {
	return fmt.Sprintf("call_%d", time.Now().UnixNano())
}

func convertOllamaResponse(resp *provider.ChatResponse, content string) *provider.ChatResponse {
	if !detectToolCall(content) {
		return resp
	}

	tc, err := parseToolCall(content)
	if err != nil {
		return resp
	}

	var args json.RawMessage
	if len(tc.Arguments) > 0 {
		args = tc.Arguments
	} else {
		args = json.RawMessage("{}")
	}

	return &provider.ChatResponse{
		Model:      resp.Model,
		StopReason: "tool_use",
		Content: []provider.ContentBlock{
			{
				Type: provider.BlockToolCall,
				ToolCall: &provider.ToolCallBlock{
					ID:    generateToolCallID(),
					Name:  tc.Name,
					Input: args,
				},
			},
		},
		Usage: resp.Usage,
	}
}

func (a *Adapter) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	resp, err := a.Adapter.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Content) == 1 && resp.Content[0].Type == provider.BlockText {
		text := resp.Content[0].Text
		if detectToolCall(text) {
			return convertOllamaResponse(resp, text), nil
		}
	}

	return resp, nil
}

func (a *Adapter) Stream(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	upstream, err := a.Adapter.Stream(ctx, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan provider.StreamChunk, 100)

	go func() {
		defer close(ch)
		var contentBuilder strings.Builder
		var hasContent bool

		for chunk := range upstream {
			switch chunk.Type {
			case "text_delta":
				contentBuilder.WriteString(chunk.Text)
				hasContent = true
			case "done":
				if hasContent {
					fullContent := contentBuilder.String()
					if detectToolCall(fullContent) {
						tc, err := parseToolCall(fullContent)
						if err == nil {
							var args json.RawMessage
							if len(tc.Arguments) > 0 {
								args = tc.Arguments
							} else {
								args = json.RawMessage("{}")
							}
							ch <- provider.StreamChunk{
								Type: "tool_call_delta",
								ToolCall: &provider.ToolCallDelta{
									ID:        generateToolCallID(),
									Name:      tc.Name,
									InputJSON: string(args),
									Done:      true,
								},
							}
							ch <- provider.StreamChunk{Type: "done"}
						} else {
							ch <- provider.StreamChunk{
								Type: "text_delta",
								Text: fullContent,
							}
							ch <- provider.StreamChunk{Type: "done"}
						}
					} else {
						ch <- provider.StreamChunk{
							Type: "text_delta",
							Text: fullContent,
						}
						ch <- provider.StreamChunk{Type: "done"}
					}
				} else {
					ch <- chunk
				}
			default:
				ch <- chunk
			}
		}
	}()

	return ch, nil
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Tools       []toolDef     `json:"tools,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`
}

type chatMessage struct {
	Role             string      `json:"role,omitempty"`
	Content          interface{} `json:"content,omitempty"`
	ReasoningContent string      `json:"reasoning_content,omitempty"`
	ToolCalls        []toolCall  `json:"tool_calls,omitempty"`
	ToolCallID       string      `json:"tool_call_id,omitempty"`
	Name             string      `json:"name,omitempty"`
}

type toolCall struct {
	Index    int          `json:"index,omitempty"`
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function functionCall `json:"function"`
}

type functionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type toolDef struct {
	Type     string       `json:"type"`
	Function functionTool `json:"function"`
}

type functionTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type chatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message,omitempty"`
	Delta        chatMessage `json:"delta,omitempty"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type streamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []streamChoice `json:"choices"`
}

type streamChoice struct {
	Index        int         `json:"index"`
	Delta        chatMessage `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}
