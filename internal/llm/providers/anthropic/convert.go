package anthropic

import (
	"encoding/json"
	"strings"

	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/anthropics/anthropic-sdk-go"
)

// convertMessages converts provider messages to Anthropic MessageParam slice.
func convertMessages(msgs []provider.Message) ([]anthropic.MessageParam, error) {
	var result []anthropic.MessageParam
	for _, msg := range msgs {
		switch msg.Role {
		case provider.RoleUser:
			blocks := convertUserContent(msg.Content)
			result = append(result, anthropic.NewUserMessage(blocks...))
		case provider.RoleAssistant:
			blocks := convertAssistantContent(msg.Content)
			result = append(result, anthropic.NewAssistantMessage(blocks...))
		case provider.RoleTool:
			blocks := convertToolResultContent(msg.Content)
			result = append(result, anthropic.NewUserMessage(blocks...))
		}
	}
	return result, nil
}

func convertUserContent(blocks []provider.ContentBlock) []anthropic.ContentBlockParamUnion {
	var result []anthropic.ContentBlockParamUnion
	for _, b := range blocks {
		if b.Type == provider.BlockText {
			result = append(result, anthropic.NewTextBlock(b.Text))
		}
	}
	return result
}

func convertAssistantContent(blocks []provider.ContentBlock) []anthropic.ContentBlockParamUnion {
	var result []anthropic.ContentBlockParamUnion
	for _, b := range blocks {
		switch b.Type {
		case provider.BlockText:
			result = append(result, anthropic.NewTextBlock(b.Text))
		default:
			// Handle other block types (tool_call, etc.) if needed
			if b.ToolCall != nil {
				result = append(result, anthropic.NewToolUseBlock(
					b.ToolCall.ID,
					json.RawMessage(b.ToolCall.Input),
					b.ToolCall.Name,
				))
			}
		}
	}
	return result
}

func convertToolResultContent(blocks []provider.ContentBlock) []anthropic.ContentBlockParamUnion {
	var result []anthropic.ContentBlockParamUnion
	for _, b := range blocks {
		if b.Type == provider.BlockToolResult && b.ToolResult != nil {
			result = append(result, anthropic.NewToolResultBlock(
				b.ToolResult.ToolCallID,
				b.ToolResult.Content,
				b.ToolResult.IsError,
			))
		}
	}
	return result
}

// convertTools converts provider tool definitions to Anthropic format.
func convertTools(tools []provider.ToolDef) []anthropic.ToolUnionParam {
	result := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		var schema map[string]any
		if len(t.Parameters) > 0 {
			_ = json.Unmarshal(t.Parameters, &schema)
		}

		tp := anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: schema,
			},
		}
		result = append(result, anthropic.ToolUnionParam{OfTool: &tp})
	}
	return result
}

// convertResponse converts an Anthropic Message to a provider ChatResponse.
func convertResponse(msg *anthropic.Message) *provider.ChatResponse {
	resp := &provider.ChatResponse{
		Model:      string(msg.Model),
		StopReason: string(msg.StopReason),
		Usage: provider.Usage{
			InputTokens:  int(msg.Usage.InputTokens),
			OutputTokens: int(msg.Usage.OutputTokens),
		},
	}

	for _, block := range msg.Content {
		switch v := block.AsAny().(type) {
		case anthropic.TextBlock:
			resp.Content = append(resp.Content, provider.ContentBlock{
				Type: provider.BlockText,
				Text: v.Text,
			})
		case anthropic.ToolUseBlock:
			resp.Content = append(resp.Content, provider.ContentBlock{
				Type: provider.BlockToolCall,
				ToolCall: &provider.ToolCallBlock{
					ID:    v.ID,
					Name:  v.Name,
					Input: json.RawMessage(v.JSON.Input.Raw()),
				},
			})
		}
	}

	return resp
}

// convertError wraps an Anthropic API error into a ProviderError.
func convertError(err error) *provider.ProviderError {
	pe := &provider.ProviderError{
		Message: err.Error(),
	}
	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "rate_limit") || strings.Contains(errStr, "429"):
		pe.Type = "rate_limit"
		pe.Retry = true
	case strings.Contains(errStr, "authentication") || strings.Contains(errStr, "401") || strings.Contains(errStr, "invalid_api_key"):
		pe.Type = "auth"
	case strings.Contains(errStr, "too many tokens") || strings.Contains(errStr, "max_tokens") || strings.Contains(errStr, "context"):
		pe.Type = "context_too_long"
	case strings.Contains(errStr, "timeout"):
		pe.Type = "timeout"
		pe.Retry = true
	default:
		pe.Type = "provider"
	}
	return pe
}
