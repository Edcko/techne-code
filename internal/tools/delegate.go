package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Edcko/techne-code/internal/agent"
	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/session"
	"github.com/Edcko/techne-code/pkg/tool"
)

const maxParallelTasks = 3

var validAgentNames = map[string]bool{
	"researcher": true,
	"coder":      true,
	"reviewer":   true,
	"tester":     true,
}

type DelegateTask struct {
	Agent  string `json:"agent"`
	Prompt string `json:"prompt"`
}

type delegateResult struct {
	Agent  string
	Output string
	Err    error
}

type DelegateTool struct {
	provider provider.Provider
	store    session.SessionStore
	registry tool.ToolRegistry
	configs  map[string]agent.SubAgentConfig
}

func NewDelegateTool(
	prov provider.Provider,
	store session.SessionStore,
	registry tool.ToolRegistry,
	configs map[string]agent.SubAgentConfig,
) *DelegateTool {
	return &DelegateTool{
		provider: prov,
		store:    store,
		registry: registry,
		configs:  configs,
	}
}

func (t *DelegateTool) Name() string { return "delegate" }

func (t *DelegateTool) Description() string {
	return "Run multiple sub-agents in parallel and collect results. Each task specifies an agent type (researcher, coder, reviewer, tester) and a prompt. Up to 3 tasks can run concurrently. Use this when independent tasks can benefit from parallel execution."
}

func (t *DelegateTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"tasks": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"agent": {"type": "string", "enum": ["researcher", "coder", "reviewer", "tester"], "description": "The sub-agent type to use"},
						"prompt": {"type": "string", "description": "The task description to send to the sub-agent"}
					},
					"required": ["agent", "prompt"]
				},
				"description": "List of tasks to delegate to sub-agents in parallel. Maximum 3 tasks.",
				"maxItems": 3
			}
		},
		"required": ["tasks"]
	}`)
}

func (t *DelegateTool) RequiresPermission() bool { return false }

func (t *DelegateTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	var params struct {
		Tasks []DelegateTask `json:"tasks"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return tool.ToolResult{
			Content: fmt.Sprintf("Error parsing parameters: %v", err),
			IsError: true,
		}, nil
	}

	if len(params.Tasks) == 0 {
		return tool.ToolResult{
			Content: "Error: at least one task is required",
			IsError: true,
		}, nil
	}

	if len(params.Tasks) > maxParallelTasks {
		return tool.ToolResult{
			Content: fmt.Sprintf("Error: maximum %d parallel tasks allowed, got %d", maxParallelTasks, len(params.Tasks)),
			IsError: true,
		}, nil
	}

	for i, task := range params.Tasks {
		if !validAgentNames[task.Agent] {
			return tool.ToolResult{
				Content: fmt.Sprintf("Error: task %d has invalid agent %q. Valid agents: researcher, coder, reviewer, tester", i, task.Agent),
				IsError: true,
			}, nil
		}
		if task.Prompt == "" {
			return tool.ToolResult{
				Content: fmt.Sprintf("Error: task %d has empty prompt", i),
				IsError: true,
			}, nil
		}
	}

	results := make([]delegateResult, len(params.Tasks))
	var wg sync.WaitGroup

	for i, task := range params.Tasks {
		wg.Add(1)
		go func(idx int, dt DelegateTask) {
			defer wg.Done()
			output, err := t.runSubAgent(ctx, dt)
			results[idx] = delegateResult{
				Agent:  dt.Agent,
				Output: output,
				Err:    err,
			}
		}(i, task)
	}

	wg.Wait()

	return tool.ToolResult{Content: formatDelegateResults(results)}, nil
}

func (t *DelegateTool) runSubAgent(ctx context.Context, task DelegateTask) (string, error) {
	config, ok := t.configs[task.Agent]
	if !ok {
		return "", fmt.Errorf("no config found for agent %q", task.Agent)
	}

	allTools := t.registry.List()
	subAgent := agent.NewSubAgent(t.provider, t.store, config, allTools)

	output, err := subAgent.Run(ctx, task.Prompt)
	if err != nil {
		return "", fmt.Errorf("sub-agent %q failed: %w", task.Agent, err)
	}

	return output, nil
}

func formatDelegateResults(results []delegateResult) string {
	output := fmt.Sprintf("Delegated %d task(s):\n\n", len(results))
	for i, r := range results {
		output += fmt.Sprintf("## Task %d [%s]\n", i+1, r.Agent)
		if r.Err != nil {
			output += fmt.Sprintf("Status: FAILED\nError: %s\n\n", r.Err)
		} else {
			output += fmt.Sprintf("Status: COMPLETED\n%s\n\n", r.Output)
		}
	}
	return output
}
