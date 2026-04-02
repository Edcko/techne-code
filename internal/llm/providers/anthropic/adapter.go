// Package anthropic implements the LLM provider interface for Anthropic Claude models.
package anthropic

import (
	"context"
	"fmt"
	"strings"

	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
)

// AnthropicAdapter implements provider.Provider for Anthropic Claude models.
type AnthropicAdapter struct {
	client *anthropic.Client
}

// New creates a new Anthropic adapter with the given API key.
func New(apiKey string) *AnthropicAdapter {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &AnthropicAdapter{client: &client}
}

// Name returns the provider identifier.
func (a *AnthropicAdapter) Name() string {
	return "anthropic"
}

// Models returns the list of supported Anthropic models.
func (a *AnthropicAdapter) Models() []provider.ModelInfo {
	return []provider.ModelInfo{
		{ID: "claude-sonnet-4-20250514", MaxTokens: 64000, SupportsTools: true, SupportsVision: true, ContextWindow: 200000},
		{ID: "claude-opus-4-20250514", MaxTokens: 32000, SupportsTools: true, SupportsVision: true, ContextWindow: 200000},
		{ID: "claude-haiku-4-20250414", MaxTokens: 64000, SupportsTools: true, SupportsVision: true, ContextWindow: 200000},
	}
}

// Chat sends a non-streaming request to the Anthropic Messages API.
func (a *AnthropicAdapter) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	msgs, err := convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("convert messages: %w", err)
	}

	params := anthropic.MessageNewParams{
		Model:     req.Config.Model,
		MaxTokens: int64(req.Config.MaxTokens),
		Messages:  msgs,
	}

	tools := convertTools(req.Tools)
	if len(tools) > 0 {
		params.Tools = tools
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}

	msg, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return nil, convertError(err)
	}

	return convertResponse(msg), nil
}

// Stream sends a streaming request to the Anthropic Messages API.
// Returns a channel of StreamChunk that the caller can read from.
func (a *AnthropicAdapter) Stream(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	msgs, err := convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("convert messages: %w", err)
	}

	params := anthropic.MessageNewParams{
		Model:     req.Config.Model,
		MaxTokens: int64(req.Config.MaxTokens),
		Messages:  msgs,
	}

	tools := convertTools(req.Tools)
	if len(tools) > 0 {
		params.Tools = tools
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}

	stream := a.client.Messages.NewStreaming(ctx, params)
	ch := make(chan provider.StreamChunk, 100)

	go func() {
		defer close(ch)
		processStream(stream, ch)
	}()

	return ch, nil
}

// processStream reads events from the Anthropic SSE stream and publishes chunks.
func processStream(stream *ssestream.Stream[anthropic.MessageStreamEventUnion], ch chan<- provider.StreamChunk) {
	type toolCallAccum struct {
		id    string
		name  string
		input strings.Builder
	}
	toolCalls := make(map[int64]*toolCallAccum)

	for stream.Next() {
		event := stream.Current()

		switch v := event.AsAny().(type) {
		case anthropic.ContentBlockStartEvent:
			switch b := v.ContentBlock.AsAny().(type) {
			case anthropic.ToolUseBlock:
				toolCalls[v.Index] = &toolCallAccum{
					id:   b.ID,
					name: b.Name,
				}
			}

		case anthropic.ContentBlockDeltaEvent:
			switch d := v.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				ch <- provider.StreamChunk{Type: "text_delta", Text: d.Text}
			case anthropic.InputJSONDelta:
				if tc, ok := toolCalls[v.Index]; ok {
					tc.input.WriteString(d.PartialJSON)
				}
			}

		case anthropic.ContentBlockStopEvent:
			if tc, ok := toolCalls[v.Index]; ok {
				ch <- provider.StreamChunk{
					Type: "tool_call_delta",
					ToolCall: &provider.ToolCallDelta{
						ID:        tc.id,
						Name:      tc.name,
						InputJSON: tc.input.String(),
						Done:      true,
					},
				}
			}

		case anthropic.MessageStartEvent:
			// Message started — model info available if needed

		case anthropic.MessageDeltaEvent:
			ch <- provider.StreamChunk{Type: "done"}
		}
	}

	if err := stream.Err(); err != nil {
		ch <- provider.StreamChunk{Type: "error", Error: convertError(err)}
	}
}
