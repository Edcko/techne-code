// Package event defines the event system for Techne Code.
// Events are used to communicate state changes, tool executions, and other
// occurrences throughout the agent's lifecycle.
package event

import (
	"encoding/json"
	"time"
)

// EventType represents the type of an event in the system.
type EventType string

// Event types for different kinds of system occurrences.
const (
	// EventMessageDelta is emitted for each chunk of text from the LLM.
	EventMessageDelta EventType = "message_delta"
	// EventToolStart is emitted when a tool begins execution.
	EventToolStart EventType = "tool_start"
	// EventToolResult is emitted when a tool completes execution.
	EventToolResult EventType = "tool_result"
	// EventError is emitted when an error occurs.
	EventError EventType = "error"
	// EventDone is emitted when the agent completes its response.
	EventDone EventType = "done"
	// EventSummarize is emitted when context summarization is needed.
	EventSummarize EventType = "summarize"
	// EventSessionUpdate is emitted when session metadata changes.
	EventSessionUpdate EventType = "session_update"
	// EventPermissionReq is emitted when user permission is required for an action.
	EventPermissionReq EventType = "permission_request"
	// EventPermissionRes is emitted when the user responds to a permission request.
	EventPermissionRes EventType = "permission_response"
	// EventTokenUsage is emitted when token usage stats are updated.
	EventTokenUsage EventType = "token_usage"
)

// Event represents an occurrence in the system.
type Event struct {
	// Type identifies the kind of event.
	Type EventType `json:"type"`
	// SessionID identifies which session this event belongs to.
	SessionID string `json:"session_id"`
	// Data contains the event-specific payload.
	Data interface{} `json:"data"`
	// Timestamp records when the event occurred.
	Timestamp time.Time `json:"timestamp"`
}

// MessageDeltaData contains data for EventMessageDelta events.
type MessageDeltaData struct {
	// Text is the chunk of text from the LLM.
	Text string `json:"text"`
}

// ThinkingDeltaData contains data for thinking/reasoning chunks.
type ThinkingDeltaData struct {
	// Text is the chunk of thinking content from the LLM.
	Text string `json:"text"`
}

// ToolStartData contains data for EventToolStart events.
type ToolStartData struct {
	// ToolName is the name of the tool being executed.
	ToolName string `json:"tool_name"`
	// Input contains the tool parameters as raw JSON.
	Input json.RawMessage `json:"input"`
}

// ToolResultData contains data for EventToolResult events.
type ToolResultData struct {
	// ToolName is the name of the tool that was executed.
	ToolName string `json:"tool_name"`
	// Content contains the output from the tool execution.
	Content string `json:"content"`
	// IsError indicates whether the tool execution failed.
	IsError bool `json:"is_error"`
}

// ErrorData contains data for EventError events.
type ErrorData struct {
	// Message describes the error.
	Message string `json:"message"`
	// Fatal indicates whether the error is unrecoverable.
	Fatal bool `json:"fatal"`
}

// SessionUpdateData contains data for EventSessionUpdate events.
type SessionUpdateData struct {
	// SessionID is the ID of the updated session.
	SessionID string `json:"session_id"`
	// Title is the new title of the session.
	Title string `json:"title"`
}

// PermissionRequestData contains data for EventPermissionReq events.
type PermissionRequestData struct {
	// ToolName is the name of the tool requiring permission.
	ToolName string
	// Action describes the action being requested.
	Action string
	// Description provides details about the action for user review.
	Description string
	// Params contains the parameters for the action as raw JSON.
	Params json.RawMessage
	// Response is the channel where the user's response is sent back.
	Response chan<- PermissionResponseData
}

// PermissionResponseData contains the user's response to a permission request.
type PermissionResponseData struct {
	// Allowed indicates whether the user granted permission.
	Allowed bool
	// Remember indicates whether to remember this permission for the session.
	Remember bool
}

// TokenUsageData contains cumulative token usage statistics for a session.
type TokenUsageData struct {
	// InputTokens is the cumulative input tokens used.
	InputTokens int `json:"input_tokens"`
	// OutputTokens is the cumulative output tokens used.
	OutputTokens int `json:"output_tokens"`
	// TotalTokens is the sum of input and output tokens.
	TotalTokens int `json:"total_tokens"`
	// CachedTokens is the cumulative tokens read from cache.
	CachedTokens int `json:"cached_tokens,omitempty"`
	// EstimatedContextUsage is the estimated current context window usage in tokens.
	EstimatedContextUsage int `json:"estimated_context_usage,omitempty"`
	// ContextWindow is the model's maximum context window size.
	ContextWindow int `json:"context_window,omitempty"`
}

// EventHandler is a callback function for handling events.
type EventHandler func(Event)

// EventBus defines the interface for a publish/subscribe event system.
type EventBus interface {
	// Publish sends an event to all subscribers.
	Publish(event Event)
	// Subscribe registers a handler and returns an unsubscribe function.
	Subscribe(handler EventHandler) func()
	// Close shuts down the event bus and releases resources.
	Close()
}

// NewEvent creates a new Event with the current timestamp.
func NewEvent(typ EventType, sessionID string, data interface{}) Event {
	return Event{
		Type:      typ,
		SessionID: sessionID,
		Data:      data,
		Timestamp: time.Now(),
	}
}
