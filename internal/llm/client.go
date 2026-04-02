// Package llm provides the LLM client abstraction that wraps a provider
// with common functionality and event bus integration.
package llm

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/provider"
)

// Client wraps a provider with common functionality.
// It provides a higher-level interface for LLM interactions with
// automatic event bus integration for streaming responses.
type Client struct {
	provider provider.Provider
	bus      event.EventBus
}

// NewClient creates a new LLM client with the given provider and event bus.
func NewClient(p provider.Provider, bus event.EventBus) *Client {
	return &Client{
		provider: p,
		bus:      bus,
	}
}

// Chat sends a non-streaming request to the LLM provider.
func (c *Client) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	return c.provider.Chat(ctx, req)
}

// Stream sends a streaming request to the LLM and publishes events to the event bus.
// It returns the full accumulated response when the stream completes.
//
// Event types published:
//   - EventMessageDelta: for each text_delta and thinking_delta chunk
//   - EventError: if the provider returns an error
//   - EventDone: when the stream completes (always published, even on error)
func (c *Client) Stream(ctx context.Context, sessionID string, req provider.ChatRequest) (*provider.ChatResponse, error) {
	chunkChan, err := c.provider.Stream(ctx, req)
	if err != nil {
		// Publish error event and done event
		c.bus.Publish(event.NewEvent(event.EventError, sessionID, event.ErrorData{
			Message: err.Error(),
			Fatal:   true,
		}))
		c.bus.Publish(event.NewEvent(event.EventDone, sessionID, nil))
		return nil, err
	}

	// Accumulate the response
	response := &provider.ChatResponse{
		Content: []provider.ContentBlock{},
	}

	var mu sync.Mutex
	toolCallsAccum := make(map[string]*toolCallAccumulator)

	// Read chunks and process them
	for chunk := range chunkChan {
		switch chunk.Type {
		case "text_delta":
			// Publish message delta event
			c.bus.Publish(event.NewEvent(event.EventMessageDelta, sessionID, event.MessageDeltaData{
				Text: chunk.Text,
			}))
			// Accumulate text
			mu.Lock()
			response.Content = appendTextBlock(response.Content, chunk.Text)
			mu.Unlock()

		case "thinking_delta":
			// Publish message delta event for thinking
			c.bus.Publish(event.NewEvent(event.EventMessageDelta, sessionID, event.ThinkingDeltaData{
				Text: chunk.Thinking,
			}))
			// Accumulate thinking
			mu.Lock()
			response.Content = appendThinkingBlock(response.Content, chunk.Thinking)
			mu.Unlock()

		case "tool_call_delta":
			// Accumulate tool call
			if chunk.ToolCall != nil {
				mu.Lock()
				acc := toolCallsAccum[chunk.ToolCall.ID]
				if acc == nil {
					acc = &toolCallAccumulator{id: chunk.ToolCall.ID}
					toolCallsAccum[chunk.ToolCall.ID] = acc
				}
				// Only update name if provided (name is usually only in first delta)
				if chunk.ToolCall.Name != "" {
					acc.name = chunk.ToolCall.Name
				}
				acc.inputJSON += chunk.ToolCall.InputJSON
				if chunk.ToolCall.Done {
					acc.done = true
				}
				mu.Unlock()
			}

		case "usage":
			// Update usage stats
			if chunk.Usage != nil {
				mu.Lock()
				response.Usage = *chunk.Usage
				mu.Unlock()
			}

		case "error":
			// Publish error event
			if chunk.Error != nil {
				c.bus.Publish(event.NewEvent(event.EventError, sessionID, event.ErrorData{
					Message: chunk.Error.Error(),
					Fatal:   true,
				}))
			}

		case "done":
			// Stream completed
			// Collect final tool calls from accumulator
			mu.Lock()
			for _, acc := range toolCallsAccum {
				if acc.done {
					tc := provider.ToolCallBlock{
						ID:    acc.id,
						Name:  acc.name,
						Input: json.RawMessage(acc.inputJSON),
					}
					response.Content = append(response.Content, provider.ContentBlock{
						Type:     provider.BlockToolCall,
						ToolCall: &tc,
					})
				}
			}
			mu.Unlock()
		}
	}

	// Always publish done event
	c.bus.Publish(event.NewEvent(event.EventDone, sessionID, nil))

	return response, nil
}

// Provider returns the underlying provider.
func (c *Client) Provider() provider.Provider {
	return c.provider
}

// toolCallAccumulator accumulates tool call deltas into a complete tool call.
type toolCallAccumulator struct {
	id        string
	name      string
	inputJSON string
	done      bool
}

// appendTextBlock appends text to the last text block or creates a new one.
func appendTextBlock(blocks []provider.ContentBlock, text string) []provider.ContentBlock {
	if len(blocks) > 0 && blocks[len(blocks)-1].Type == provider.BlockText {
		blocks[len(blocks)-1].Text += text
		return blocks
	}
	return append(blocks, provider.ContentBlock{
		Type: provider.BlockText,
		Text: text,
	})
}

// appendThinkingBlock appends thinking to the last thinking block or creates a new one.
func appendThinkingBlock(blocks []provider.ContentBlock, thinking string) []provider.ContentBlock {
	if len(blocks) > 0 && blocks[len(blocks)-1].Type == provider.BlockThinking {
		if blocks[len(blocks)-1].Thinking != nil {
			blocks[len(blocks)-1].Thinking.Text += thinking
		}
		return blocks
	}
	return append(blocks, provider.ContentBlock{
		Type: provider.BlockThinking,
		Thinking: &provider.ThinkingBlock{
			Text: thinking,
		},
	})
}
