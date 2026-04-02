package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Edcko/techne-code/pkg/provider"
)

type Adapter struct {
	apiKey  string
	baseURL string
	client  *http.Client
	models  []provider.ModelInfo
}

func NewAdapter(apiKey, baseURL string, models []provider.ModelInfo) *Adapter {
	if baseURL == "" {
		baseURL = "https://api.z.ai/api/coding/paas/v4"
	}

	if len(models) == 0 {
		models = []provider.ModelInfo{
			{ID: "glm-5", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 128000},
			{ID: "glm-4.6", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 128000},
			{ID: "glm-4.7", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 128000},
		}
	}

	return &Adapter{
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
		models: models,
	}
}

func (a *Adapter) Name() string {
	return "openai"
}

func (a *Adapter) Models() []provider.ModelInfo {
	return a.models
}

func (a *Adapter) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	body := chatRequest{
		Model:       req.Config.Model,
		Messages:    convertMessages(req.Messages),
		MaxTokens:   req.Config.MaxTokens,
		Temperature: req.Config.Temperature,
		TopP:        req.Config.TopP,
		Stream:      false,
	}

	if len(req.Tools) > 0 {
		body.Tools = convertTools(req.Tools)
	}

	if req.System != "" {
		systemMsg := chatMessage{
			Role:    "system",
			Content: req.System,
		}
		body.Messages = append([]chatMessage{systemMsg}, body.Messages...)
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, convertError(err, 0)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, convertError(fmt.Errorf("API error: %s", string(bodyBytes)), resp.StatusCode)
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return convertResponse(&chatResp), nil
}

func (a *Adapter) Stream(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	body := chatRequest{
		Model:       req.Config.Model,
		Messages:    convertMessages(req.Messages),
		MaxTokens:   req.Config.MaxTokens,
		Temperature: req.Config.Temperature,
		TopP:        req.Config.TopP,
		Stream:      true,
	}

	if len(req.Tools) > 0 {
		body.Tools = convertTools(req.Tools)
	}

	if req.System != "" {
		systemMsg := chatMessage{
			Role:    "system",
			Content: req.System,
		}
		body.Messages = append([]chatMessage{systemMsg}, body.Messages...)
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, convertError(err, 0)
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, convertError(fmt.Errorf("API error: %s", string(bodyBytes)), resp.StatusCode)
	}

	ch := make(chan provider.StreamChunk, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()
		a.processStream(resp.Body, ch)
	}()

	return ch, nil
}

func (a *Adapter) processStream(reader io.Reader, ch chan<- provider.StreamChunk) {
	type toolCallAccum struct {
		id        string
		name      string
		arguments strings.Builder
	}
	toolCalls := make(map[int]*toolCallAccum)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			ch <- provider.StreamChunk{Type: "done"}
			return
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		delta := choice.Delta

		if delta.ReasoningContent != "" {
			ch <- provider.StreamChunk{
				Type:     "thinking_delta",
				Thinking: delta.ReasoningContent,
			}
		}

		if delta.Content != nil {
			if content, ok := delta.Content.(string); ok && content != "" {
				ch <- provider.StreamChunk{
					Type: "text_delta",
					Text: content,
				}
			}
		}

		for _, tc := range delta.ToolCalls {
			accum := toolCalls[tc.Index]
			if accum == nil {
				accum = &toolCallAccum{}
				toolCalls[tc.Index] = accum
			}

			if tc.ID != "" {
				accum.id = tc.ID
			}
			if tc.Function.Name != "" {
				accum.name = tc.Function.Name
			}
			accum.arguments.WriteString(tc.Function.Arguments)
		}

		if choice.FinishReason != nil && *choice.FinishReason == "tool_calls" {
			for _, accum := range toolCalls {
				ch <- provider.StreamChunk{
					Type: "tool_call_delta",
					ToolCall: &provider.ToolCallDelta{
						ID:        accum.id,
						Name:      accum.name,
						InputJSON: accum.arguments.String(),
						Done:      true,
					},
				}
			}
		}

		if choice.FinishReason != nil && *choice.FinishReason == "stop" {
			ch <- provider.StreamChunk{Type: "done"}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- provider.StreamChunk{
			Type:  "error",
			Error: convertError(err, 0),
		}
	}
}
