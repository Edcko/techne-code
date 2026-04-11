package agent

import (
	"github.com/Edcko/techne-code/pkg/provider"
)

const charsPerToken = 4

func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + charsPerToken - 1) / charsPerToken
}

func EstimateMessageTokens(msg provider.Message) int {
	total := 0
	for _, block := range msg.Content {
		switch block.Type {
		case provider.BlockText:
			total += EstimateTokens(block.Text)
		case provider.BlockThinking:
			if block.Thinking != nil {
				total += EstimateTokens(block.Thinking.Text)
			}
		case provider.BlockToolCall:
			if block.ToolCall != nil {
				total += EstimateTokens(block.ToolCall.Name)
				total += EstimateTokens(string(block.ToolCall.Input))
			}
		case provider.BlockToolResult:
			if block.ToolResult != nil {
				total += EstimateTokens(block.ToolResult.Content)
			}
		case provider.BlockImage:
			if block.Image != nil {
				total += len(block.Image.Data) / charsPerToken
			}
		}
	}
	total += 4
	return total
}

func EstimateMessagesTokens(messages []provider.Message) int {
	total := 0
	for _, msg := range messages {
		total += EstimateMessageTokens(msg)
	}
	return total
}

func EstimateSystemPromptTokens(prompt string) int {
	if prompt == "" {
		return 0
	}
	return EstimateTokens(prompt) + 4
}

func GetContextWindow(models []provider.ModelInfo, modelID string) int {
	for _, m := range models {
		if m.ID == modelID {
			return m.ContextWindow
		}
	}
	return 128000
}

func IsApproachingLimit(usedTokens, contextWindow int, threshold float64) bool {
	if contextWindow <= 0 {
		return false
	}
	ratio := float64(usedTokens) / float64(contextWindow)
	return ratio >= threshold
}
