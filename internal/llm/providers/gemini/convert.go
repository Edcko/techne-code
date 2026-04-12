package gemini

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Edcko/techne-code/pkg/provider"
)

func convertMessages(msgs []provider.Message) []content {
	result := make([]content, 0, len(msgs))
	for _, msg := range msgs {
		switch msg.Role {
		case provider.RoleUser:
			result = append(result, content{
				Role:  "user",
				Parts: extractParts(msg.Content),
			})
		case provider.RoleAssistant:
			result = append(result, content{
				Role:  "model",
				Parts: extractAssistantParts(msg.Content),
			})
		case provider.RoleTool:
			for _, block := range msg.Content {
				if block.Type == provider.BlockToolResult && block.ToolResult != nil {
					result = append(result, content{
						Role: "function",
						Parts: []part{{
							FunctionResponse: &functionRespPart{
								Name:     block.ToolResult.Name,
								Response: map[string]string{"content": block.ToolResult.Content},
							},
						}},
					})
				}
			}
		}
	}
	return result
}

func extractParts(blocks []provider.ContentBlock) []part {
	var parts []part
	for _, block := range blocks {
		if block.Type == provider.BlockText {
			parts = append(parts, part{Text: block.Text})
		}
	}
	if len(parts) == 0 {
		parts = append(parts, part{Text: ""})
	}
	return parts
}

func extractAssistantParts(blocks []provider.ContentBlock) []part {
	var parts []part
	for _, block := range blocks {
		switch block.Type {
		case provider.BlockText:
			if block.Text != "" {
				parts = append(parts, part{Text: block.Text})
			}
		case provider.BlockToolCall:
			if block.ToolCall != nil {
				args := block.ToolCall.Input
				if len(args) == 0 {
					args = json.RawMessage("{}")
				}
				parts = append(parts, part{
					FunctionCall: &functionCallPart{
						Name: block.ToolCall.Name,
						Args: args,
					},
				})
			}
		}
	}
	if len(parts) == 0 {
		parts = append(parts, part{Text: ""})
	}
	return parts
}

func convertTools(tools []provider.ToolDef) []toolConfig {
	if len(tools) == 0 {
		return nil
	}

	decls := make([]functionDecl, 0, len(tools))
	for _, t := range tools {
		var params interface{}
		if len(t.Parameters) > 0 {
			var schema map[string]interface{}
			if err := json.Unmarshal(t.Parameters, &schema); err == nil {
				params = schema
			}
		}
		decls = append(decls, functionDecl{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  params,
		})
	}
	return []toolConfig{{FunctionDeclarations: decls}}
}

func convertResponse(resp *generateResponse) *provider.ChatResponse {
	result := &provider.ChatResponse{
		StopReason: "end_turn",
	}
	if resp.ModelVersion != "" {
		result.Model = resp.ModelVersion
	}
	if resp.UsageMetadata != nil {
		result.Usage = provider.Usage{
			InputTokens:  resp.UsageMetadata.PromptTokenCount,
			OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
		}
	}

	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		cand := resp.Candidates[0]
		if cand.FinishReason != "" {
			result.StopReason = convertFinishReason(cand.FinishReason)
		}
		hasToolCall := false
		for _, p := range cand.Content.Parts {
			if p.Text != "" {
				result.Content = append(result.Content, provider.ContentBlock{
					Type: provider.BlockText,
					Text: p.Text,
				})
			}
			if p.FunctionCall != nil {
				hasToolCall = true
				args := p.FunctionCall.Args
				if len(args) == 0 {
					args = json.RawMessage("{}")
				}
				result.Content = append(result.Content, provider.ContentBlock{
					Type: provider.BlockToolCall,
					ToolCall: &provider.ToolCallBlock{
						ID:    generateCallID(),
						Name:  p.FunctionCall.Name,
						Input: args,
					},
				})
			}
		}
		if hasToolCall && result.StopReason == "end_turn" {
			result.StopReason = "tool_use"
		}
	}

	return result
}

func convertFinishReason(reason string) string {
	switch strings.ToUpper(reason) {
	case "STOP":
		return "end_turn"
	case "MAX_TOKENS":
		return "max_tokens"
	case "SAFETY", "RECITATION":
		return "end_turn"
	default:
		return strings.ToLower(reason)
	}
}

func convertError(err error, statusCode int) *provider.ProviderError {
	pe := &provider.ProviderError{
		Message:    err.Error(),
		StatusCode: statusCode,
	}
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "resource_exhausted") || statusCode == 429:
		pe.Type = "rate_limit"
		pe.Retry = true
	case strings.Contains(errStr, "api_key") || strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "permission") || statusCode == 401 || statusCode == 403:
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

func generateCallID() string {
	return fmt.Sprintf("call_%d", time.Now().UnixNano())
}
