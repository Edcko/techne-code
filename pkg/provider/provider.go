// Package provider defines the LLM provider types and interfaces for Techne Code.
// These types represent the contract between the agent and various LLM backends
// (Anthropic Claude, OpenAI, Google Gemini, etc.).
package provider

import (
	"context"
	"encoding/json"
)

// Role represents the role of a message in a conversation.
type Role string

// Message roles following the standard LLM conversation format.
const (
	// RoleSystem represents system-level instructions that guide the assistant's behavior.
	RoleSystem Role = "system"
	// RoleUser represents messages from the human user.
	RoleUser Role = "user"
	// RoleAssistant represents messages from the AI assistant.
	RoleAssistant Role = "assistant"
	// RoleTool represents results from tool executions.
	RoleTool Role = "tool"
)

// ContentBlockType represents the type of content in a message block.
type ContentBlockType string

// Content block types for different kinds of message content.
const (
	// BlockText represents plain text content.
	BlockText ContentBlockType = "text"
	// BlockToolCall represents a request to execute a tool.
	BlockToolCall ContentBlockType = "tool_call"
	// BlockToolResult represents the result of a tool execution.
	BlockToolResult ContentBlockType = "tool_result"
	// BlockThinking represents internal reasoning (for models that support it).
	BlockThinking ContentBlockType = "thinking"
	// BlockImage represents image content in base64 format.
	BlockImage ContentBlockType = "image"
)

// ContentBlock represents a piece of message content.
// A message can contain multiple content blocks of different types.
type ContentBlock struct {
	// Type identifies the kind of content block.
	Type ContentBlockType `json:"type"`
	// Text contains the text content for text and thinking blocks.
	Text string `json:"text,omitempty"`
	// ToolCall contains tool call data for tool_call blocks.
	ToolCall *ToolCallBlock `json:"tool_call,omitempty"`
	// ToolResult contains tool result data for tool_result blocks.
	ToolResult *ToolResultBlock `json:"tool_result,omitempty"`
	// Thinking contains thinking/reasoning data for thinking blocks.
	Thinking *ThinkingBlock `json:"thinking,omitempty"`
	// Image contains image data for image blocks.
	Image *ImageBlock `json:"image,omitempty"`
}

// ToolCallBlock represents a tool call request from the LLM.
type ToolCallBlock struct {
	// ID is the unique identifier for this tool call.
	ID string `json:"id"`
	// Name is the name of the tool to execute.
	Name string `json:"name"`
	// Input contains the tool parameters as raw JSON.
	Input json.RawMessage `json:"input"`
}

// ToolResultBlock represents the result of a tool execution.
type ToolResultBlock struct {
	// ToolCallID references the tool call this result is for.
	ToolCallID string `json:"tool_call_id"`
	// Name is the name of the tool that was executed.
	Name string `json:"name"`
	// Content contains the output from the tool execution.
	Content string `json:"content"`
	// IsError indicates whether the tool execution failed.
	IsError bool `json:"is_error"`
}

// ThinkingBlock represents internal reasoning from extended thinking models.
type ThinkingBlock struct {
	// Text contains the thinking/reasoning content.
	Text string `json:"text"`
	// Signature is an optional signature for verification (used by some providers).
	Signature string `json:"signature,omitempty"`
}

// ImageBlock represents an image in a message.
type ImageBlock struct {
	// MediaType is the MIME type of the image (e.g., "image/png", "image/jpeg").
	MediaType string `json:"media_type"`
	// Data contains the base64-encoded image data.
	Data string `json:"data"`
}

// Message represents a single message in an LLM conversation.
type Message struct {
	// Role indicates who sent the message.
	Role Role `json:"role"`
	// Content contains the message content as a list of content blocks.
	Content []ContentBlock `json:"content"`
}

// Usage represents token usage statistics from an LLM provider.
type Usage struct {
	// InputTokens is the number of tokens in the request.
	InputTokens int `json:"input_tokens"`
	// OutputTokens is the number of tokens in the response.
	OutputTokens int `json:"output_tokens"`
	// CacheReadTokens is the number of tokens read from cache (if supported).
	CacheReadTokens int `json:"cache_read_tokens,omitempty"`
	// CacheWriteTokens is the number of tokens written to cache (if supported).
	CacheWriteTokens int `json:"cache_write_tokens,omitempty"`
}

// ProviderConfig contains model-specific configuration for LLM requests.
type ProviderConfig struct {
	// Model is the identifier of the model to use.
	Model string `json:"model"`
	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int `json:"max_tokens"`
	// Temperature controls randomness in the output (0.0 to 1.0).
	Temperature float64 `json:"temperature"`
	// TopP controls diversity via nucleus sampling (0.0 to 1.0).
	TopP float64 `json:"top_p,omitempty"`
}

// ChatRequest represents a request to an LLM provider.
type ChatRequest struct {
	// Messages is the conversation history.
	Messages []Message `json:"messages"`
	// Tools is the list of tools available to the LLM.
	Tools []ToolDef `json:"tools,omitempty"`
	// System contains system-level instructions.
	System string `json:"system,omitempty"`
	// Config contains model-specific configuration.
	Config ProviderConfig `json:"config"`
}

// ChatResponse represents a response from an LLM provider.
type ChatResponse struct {
	// Content contains the generated content blocks.
	Content []ContentBlock `json:"content"`
	// Usage contains token usage statistics.
	Usage Usage `json:"usage"`
	// Model is the model that generated this response.
	Model string `json:"model"`
	// StopReason indicates why generation stopped (e.g., "end_turn", "max_tokens", "tool_use").
	StopReason string `json:"stop_reason"`
}

// StreamChunk represents a chunk from a streaming LLM response.
type StreamChunk struct {
	// Type identifies the kind of chunk (e.g., "text_delta", "tool_call_delta", "thinking_delta", "usage", "done", "error").
	Type string `json:"type"`
	// Text contains text content for text_delta chunks.
	Text string `json:"text,omitempty"`
	// ToolCall contains tool call data for tool_call_delta chunks.
	ToolCall *ToolCallDelta `json:"tool_call,omitempty"`
	// Thinking contains thinking content for thinking_delta chunks.
	Thinking string `json:"thinking,omitempty"`
	// Usage contains token usage for usage chunks.
	Usage *Usage `json:"usage,omitempty"`
	// Error contains error information for error chunks.
	Error error `json:"error,omitempty"`
}

// ToolCallDelta represents a partial tool call in a streaming response.
type ToolCallDelta struct {
	// ID is the unique identifier for this tool call (may be empty in early deltas).
	ID string `json:"id,omitempty"`
	// Name is the name of the tool (may be empty in early deltas).
	Name string `json:"name,omitempty"`
	// InputJSON contains partial or complete tool input as JSON string.
	InputJSON string `json:"input_json,omitempty"`
	// Done indicates whether this tool call is complete.
	Done bool `json:"done"`
}

// ToolDef represents a tool definition for LLM function calling.
type ToolDef struct {
	// Name is the unique identifier for the tool.
	Name string `json:"name"`
	// Description explains what the tool does (shown to the LLM).
	Description string `json:"description"`
	// Parameters is a JSON Schema describing the tool's input parameters.
	Parameters json.RawMessage `json:"parameters"`
}

// ModelInfo contains information about an LLM model.
type ModelInfo struct {
	// ID is the model identifier used in API calls.
	ID string `json:"id"`
	// MaxTokens is the maximum output tokens the model can generate.
	MaxTokens int `json:"max_tokens"`
	// SupportsTools indicates whether the model supports function calling.
	SupportsTools bool `json:"supports_tools"`
	// SupportsVision indicates whether the model supports image inputs.
	SupportsVision bool `json:"supports_vision"`
	// ContextWindow is the maximum context length in tokens.
	ContextWindow int `json:"context_window"`
}

// ProviderError represents a typed error from an LLM provider.
type ProviderError struct {
	// Type categorizes the error (e.g., "rate_limit", "auth", "context_too_long", "timeout", "provider").
	Type string
	// Message contains the human-readable error description.
	Message string
	// Retry indicates whether the request can be retried.
	Retry bool
	// StatusCode is the HTTP status code (if applicable).
	StatusCode int
}

// Error implements the error interface for ProviderError.
func (e *ProviderError) Error() string {
	return e.Message
}

// Provider defines the interface for LLM providers.
// Implementations handle communication with specific LLM backends.
type Provider interface {
	// Name returns the provider's identifier (e.g., "anthropic", "openai", "gemini").
	Name() string
	// Chat sends a non-streaming request to the LLM and returns the response.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	// Stream sends a streaming request to the LLM and returns a channel of chunks.
	Stream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
	// Models returns information about available models.
	Models() []ModelInfo
}
