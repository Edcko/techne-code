package agent

import (
	"context"
	"fmt"

	"github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/provider"
)

const (
	summarizePrompt = `Summarize the following conversation concisely. Preserve:
1. Key decisions and their reasoning
2. Important technical details (file paths, function names, error messages)
3. The current state of work (what's done, what's pending)
4. Any constraints or requirements mentioned

Be thorough but concise. This summary replaces the original messages, so include anything the AI would need to continue the conversation effectively.`
)

type SummarizeData struct {
	OriginalMessages int `json:"original_messages"`
	SummaryTokens    int `json:"summary_tokens"`
}

type chatCaller interface {
	Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error)
	Provider() provider.Provider
}

type Summarizer struct {
	client chatCaller
	bus    event.EventBus
}

func NewSummarizer(client chatCaller, bus event.EventBus) *Summarizer {
	return &Summarizer{
		client: client,
		bus:    bus,
	}
}

func (s *Summarizer) Summarize(ctx context.Context, sessionID string, model string, messages []provider.Message) (string, error) {
	var conversationText string
	for _, msg := range messages {
		conversationText += formatMessageForSummary(msg)
	}

	if conversationText == "" {
		return "", nil
	}

	req := provider.ChatRequest{
		System: summarizePrompt,
		Config: provider.ProviderConfig{
			Model:     model,
			MaxTokens: 1024,
		},
		Messages: []provider.Message{
			{
				Role: provider.RoleUser,
				Content: []provider.ContentBlock{
					{Type: provider.BlockText, Text: conversationText},
				},
			},
		},
	}

	resp, err := s.client.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("summarization request failed: %w", err)
	}

	var summary string
	for _, block := range resp.Content {
		if block.Type == provider.BlockText {
			summary += block.Text
		}
	}

	if summary == "" {
		return "", fmt.Errorf("summarization returned empty response")
	}

	s.bus.Publish(event.NewEvent(event.EventSummarize, sessionID, SummarizeData{
		OriginalMessages: len(messages),
		SummaryTokens:    EstimateTokens(summary),
	}))

	return summary, nil
}

func formatMessageForSummary(msg provider.Message) string {
	var text string
	for _, block := range msg.Content {
		switch block.Type {
		case provider.BlockText:
			text += block.Text + "\n"
		case provider.BlockToolCall:
			if block.ToolCall != nil {
				text += fmt.Sprintf("[Tool Call: %s(%s)]\n", block.ToolCall.Name, string(block.ToolCall.Input))
			}
		case provider.BlockToolResult:
			if block.ToolResult != nil {
				text += fmt.Sprintf("[Tool Result (%s): %s]\n", block.ToolResult.Name, truncateForSummary(block.ToolResult.Content, 500))
			}
		case provider.BlockThinking:
			if block.Thinking != nil {
				text += fmt.Sprintf("[Thinking: %s]\n", truncateForSummary(block.Thinking.Text, 200))
			}
		}
	}

	if text == "" {
		return ""
	}

	return fmt.Sprintf("\n--- %s ---\n%s", string(msg.Role), text)
}

func truncateForSummary(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "... [truncated]"
}
