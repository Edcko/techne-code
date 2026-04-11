package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Edcko/techne-code/internal/agent"
	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/session"
	"github.com/Edcko/techne-code/pkg/tool"
)

type SubAgentTool struct {
	config   agent.SubAgentConfig
	provider provider.Provider
	store    session.SessionStore
	registry tool.ToolRegistry
}

func NewSubAgentTool(
	config agent.SubAgentConfig,
	prov provider.Provider,
	store session.SessionStore,
	registry tool.ToolRegistry,
) *SubAgentTool {
	return &SubAgentTool{
		config:   config,
		provider: prov,
		store:    store,
		registry: registry,
	}
}

func (t *SubAgentTool) Name() string { return t.config.Name }

func (t *SubAgentTool) Description() string { return t.config.Description }

func (t *SubAgentTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"task": {"type": "string", "description": "Description of the task to delegate to this sub-agent"}
		},
		"required": ["task"]
	}`)
}

func (t *SubAgentTool) RequiresPermission() bool { return false }

func (t *SubAgentTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	var params struct {
		Task string `json:"task"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return tool.ToolResult{
			Content: fmt.Sprintf("Error parsing parameters: %v", err),
			IsError: true,
		}, nil
	}

	if params.Task == "" {
		return tool.ToolResult{
			Content: "Error: task is required",
			IsError: true,
		}, nil
	}

	allTools := t.registry.List()
	subAgent := agent.NewSubAgent(t.provider, t.store, t.config, allTools)

	output, err := subAgent.Run(ctx, params.Task)
	if err != nil {
		return tool.ToolResult{
			Content: fmt.Sprintf("Sub-agent error: %v", err),
			IsError: true,
		}, nil
	}

	return tool.ToolResult{Content: output}, nil
}

func NewResearcherConfig(model string) agent.SubAgentConfig {
	return agent.SubAgentConfig{
		Name:          "researcher",
		Description:   "Delegates a research task to a specialized sub-agent that can read files, search code, and fetch web content. Use this when you need to gather information, investigate the codebase, or research a topic.",
		SystemPrompt:  "You are a research specialist. Your job is to gather information, investigate code, and report findings clearly. Be thorough and systematic. Report your findings concisely.",
		AllowedTools:  []string{"read_file", "grep", "glob", "web_fetch"},
		MaxIterations: 15,
		Model:         model,
		MaxTokens:     4096,
	}
}

func NewCoderConfig(model string) agent.SubAgentConfig {
	return agent.SubAgentConfig{
		Name:          "coder",
		Description:   "Delegates a coding task to a specialized sub-agent that can read, write, and edit files, execute shell commands, and search code. Use this when you need to implement changes or write new code.",
		SystemPrompt:  "You are a coding specialist. Your job is to implement code changes accurately and efficiently. Follow existing code patterns and conventions. Write clean, working code.",
		AllowedTools:  []string{"read_file", "write_file", "edit_file", "bash", "grep", "glob"},
		MaxIterations: 20,
		Model:         model,
		MaxTokens:     4096,
	}
}

func NewReviewerConfig(model string) agent.SubAgentConfig {
	return agent.SubAgentConfig{
		Name:          "reviewer",
		Description:   "Delegates a code review task to a specialized read-only sub-agent that can read files and search code. Use this when you need code review, quality analysis, or feedback without making changes.",
		SystemPrompt:  "You are a code review specialist. Your job is to analyze code quality, identify issues, and provide constructive feedback. Focus on correctness, readability, and best practices.",
		AllowedTools:  []string{"read_file", "grep", "glob"},
		MaxIterations: 10,
		Model:         model,
		MaxTokens:     4096,
	}
}

func NewTesterConfig(model string) agent.SubAgentConfig {
	return agent.SubAgentConfig{
		Name:          "tester",
		Description:   "Delegates a testing task to a specialized sub-agent that can read and write files, execute shell commands, and search code. Use this when you need to write tests, run test suites, or debug test failures.",
		SystemPrompt:  "You are a testing specialist. Your job is to write tests and verify code correctness. Write comprehensive tests that cover edge cases.",
		AllowedTools:  []string{"read_file", "write_file", "bash", "grep", "glob"},
		MaxIterations: 20,
		Model:         model,
		MaxTokens:     4096,
	}
}
