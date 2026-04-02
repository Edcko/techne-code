package openai

import (
	"encoding/json"
	"strings"

	"github.com/Edcko/techne-code/pkg/provider"
)

func convertMessages(msgs []provider.Message) []chatMessage {
	result := make([]chatMessage, 0, len(msgs))

	for _, msg := range msgs {
		switch msg.Role {
		case provider.RoleSystem:
			result = append(result, chatMessage{
				Role:    "system",
				Content: extractTextContent(msg.Content),
			})

		case provider.RoleUser:
			result = append(result, chatMessage{
				Role:    "user",
				Content: extractTextContent(msg.Content),
			})

		case provider.RoleAssistant:
			cm := chatMessage{
				Role:    "assistant",
				Content: extractTextContent(msg.Content),
			}
			toolCalls := extractToolCalls(msg.Content)
			if len(toolCalls) > 0 {
				cm.ToolCalls = toolCalls
			}
			result = append(result, cm)

		case provider.RoleTool:
			for _, block := range msg.Content {
				if block.Type == provider.BlockToolResult && block.ToolResult != nil {
					result = append(result, chatMessage{
						Role:       "tool",
						ToolCallID: block.ToolResult.ToolCallID,
						Content:    block.ToolResult.Content,
					})
				}
			}
		}
	}

	return result
}

func extractTextContent(blocks []provider.ContentBlock) string {
	var parts []string
	for _, block := range blocks {
		if block.Type == provider.BlockText {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func extractToolCalls(blocks []provider.ContentBlock) []toolCall {
	var result []toolCall
	for _, block := range blocks {
		if block.Type == provider.BlockToolCall && block.ToolCall != nil {
			tc := toolCall{
				ID:   block.ToolCall.ID,
				Type: "function",
				Function: functionCall{
					Name:      block.ToolCall.Name,
					Arguments: string(block.ToolCall.Input),
				},
			}
			result = append(result, tc)
		}
	}
	return result
}

func convertTools(tools []provider.ToolDef) []toolDef {
	result := make([]toolDef, 0, len(tools))
	for _, t := range tools {
		var params interface{}
		if len(t.Parameters) > 0 {
			var schema map[string]interface{}
			if err := json.Unmarshal(t.Parameters, &schema); err == nil {
				params = schema
			}
		}

		result = append(result, toolDef{
			Type: "function",
			Function: functionTool{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}
	return result
}

func convertResponse(resp *chatResponse) *provider.ChatResponse {
	result := &provider.ChatResponse{
		Model:      resp.Model,
		StopReason: "end_turn",
		Usage: provider.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.FinishReason != "" {
			result.StopReason = convertFinishReason(choice.FinishReason)
		}

		if choice.Message.Content != "" {
			result.Content = append(result.Content, provider.ContentBlock{
				Type: provider.BlockText,
				Text: choice.Message.Content.(string),
			})
		}

		for _, tc := range choice.Message.ToolCalls {
			result.Content = append(result.Content, provider.ContentBlock{
				Type: provider.BlockToolCall,
				ToolCall: &provider.ToolCallBlock{
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: json.RawMessage(tc.Function.Arguments),
				},
			})
		}
	}

	return result
}

func convertFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return reason
	}
}

func convertError(err error, statusCode int) *provider.ProviderError {
	pe := &provider.ProviderError{
		Message:    err.Error(),
		StatusCode: statusCode,
	}
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "rate_limit") || statusCode == 429:
		pe.Type = "rate_limit"
		pe.Retry = true
	case strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "invalid") && strings.Contains(errStr, "key") || statusCode == 401:
		pe.Type = "auth"
	case strings.Contains(errStr, "context") || strings.Contains(errStr, "token") || statusCode == 400:
		pe.Type = "context_too_long"
	case strings.Contains(errStr, "timeout"):
		pe.Type = "timeout"
		pe.Retry = true
	default:
		pe.Type = "provider"
	}

	return pe
}
