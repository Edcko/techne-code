// Package tool defines the tool system for Techne Code.
// Tools are the building blocks that allow the agent to interact with
// the filesystem, execute commands, and perform other actions.
package tool

import (
	"context"
	"encoding/json"

	"github.com/Edcko/techne-code/pkg/provider"
)

// ToolResult represents the output of a tool execution.
type ToolResult struct {
	// Content contains the output from the tool.
	Content string `json:"content"`
	// IsError indicates whether the tool execution failed.
	IsError bool `json:"is_error"`
}

// Tool defines the interface for agent tools.
// Tools are executable actions that the agent can invoke to interact
// with the environment (filesystem, shell, network, etc.).
type Tool interface {
	// Name returns the unique identifier for this tool.
	Name() string
	// Description returns a human-readable explanation of what the tool does.
	// This is shown to the LLM to help it decide when to use the tool.
	Description() string
	// Parameters returns a JSON Schema describing the tool's input parameters.
	Parameters() json.RawMessage
	// Execute runs the tool with the given input and returns the result.
	Execute(ctx context.Context, input json.RawMessage) (ToolResult, error)
	// RequiresPermission indicates whether this tool needs user approval before execution.
	// Dangerous operations (file writes, shell commands) should require permission.
	RequiresPermission() bool
}

// ToolRegistry defines the interface for managing tool registration and lookup.
// The registry maintains all available tools and provides methods to access them.
type ToolRegistry interface {
	// Register adds a tool to the registry.
	// Returns an error if a tool with the same name already exists.
	Register(t Tool) error
	// Get retrieves a tool by name.
	// Returns the tool and true if found, nil and false otherwise.
	Get(name string) (Tool, bool)
	// List returns all registered tools.
	List() []Tool
	// Schemas returns tool definitions in the format required for LLM function calling.
	Schemas() []provider.ToolDef
}
