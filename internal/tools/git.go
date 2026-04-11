package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Edcko/techne-code/pkg/tool"
)

type GitTool struct{}

type gitParams struct {
	Command string   `json:"command"`
	Staged  bool     `json:"staged,omitempty"`
	File    string   `json:"file,omitempty"`
	Count   int      `json:"count,omitempty"`
	Oneline bool     `json:"oneline,omitempty"`
	Files   []string `json:"files,omitempty"`
	Message string   `json:"message,omitempty"`
	Action  string   `json:"action,omitempty"`
}

var allowedGitCommands = map[string]bool{
	"status": true,
	"diff":   true,
	"log":    true,
	"add":    true,
	"commit": true,
	"branch": true,
	"stash":  true,
}

var writeCommands = map[string]bool{
	"add":    true,
	"commit": true,
}

var writeStashActions = map[string]bool{
	"pop":  true,
	"drop": true,
}

type dirKey struct{}

func WithDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, dirKey{}, dir)
}

func NewGitTool() *GitTool {
	return &GitTool{}
}

func (t *GitTool) Name() string { return "git" }

func (t *GitTool) Description() string {
	return "Execute git commands with structured output. Supported commands: status, diff, log, add, commit, branch, stash."
}

func (t *GitTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "Git subcommand to execute",
				"enum": ["status", "diff", "log", "add", "commit", "branch", "stash"]
			},
			"staged": {"type": "boolean", "description": "Show staged changes (diff command only)"},
			"file": {"type": "string", "description": "Specific file to diff"},
			"count": {"type": "integer", "description": "Number of log entries to show (default 10)"},
			"oneline": {"type": "boolean", "description": "Use oneline format for log"},
			"files": {"type": "array", "items": {"type": "string"}, "description": "Files to add"},
			"message": {"type": "string", "description": "Commit message"},
			"action": {"type": "string", "description": "Stash action: list, pop, drop", "enum": ["list", "pop", "drop"]}
		},
		"required": ["command"]
	}`)
}

func (t *GitTool) RequiresPermission() bool {
	return false
}

func (t *GitTool) RequiresPermissionForInput(input json.RawMessage) bool {
	var params gitParams
	if err := json.Unmarshal(input, &params); err != nil {
		return true
	}

	if writeCommands[params.Command] {
		return true
	}

	if params.Command == "stash" && writeStashActions[params.Action] {
		return true
	}

	return false
}

func (t *GitTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	var params gitParams
	if err := json.Unmarshal(input, &params); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error parsing parameters: %v", err), IsError: true}, nil
	}

	if !allowedGitCommands[params.Command] {
		return tool.ToolResult{
			Content: fmt.Sprintf("Error: git subcommand %q is not allowed. Allowed commands: status, diff, log, add, commit, branch, stash", params.Command),
			IsError: true,
		}, nil
	}

	args, err := t.buildArgs(params)
	if err != nil {
		return tool.ToolResult{Content: err.Error(), IsError: true}, nil
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	if dir, ok := ctx.Value(dirKey{}).(string); ok && dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	execErr := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if execErr != nil {
		output += fmt.Sprintf("\n[error] %v", execErr)
		return tool.ToolResult{Content: output, IsError: true}, nil
	}

	if output == "" {
		output = "(no output)"
	}

	return tool.ToolResult{Content: output}, nil
}

func (t *GitTool) buildArgs(params gitParams) ([]string, error) {
	switch params.Command {
	case "status":
		return []string{"status", "--porcelain=v2"}, nil

	case "diff":
		args := []string{"diff"}
		if params.Staged {
			args = append(args, "--cached")
		}
		if params.File != "" {
			args = append(args, "--", params.File)
		}
		return args, nil

	case "log":
		count := params.Count
		if count <= 0 {
			count = 10
		}
		args := []string{"log", fmt.Sprintf("-%d", count)}
		if params.Oneline {
			args = append(args, "--oneline")
		}
		return args, nil

	case "add":
		if len(params.Files) == 0 {
			return nil, fmt.Errorf("Error: 'files' parameter is required for add command")
		}
		args := []string{"add"}
		args = append(args, params.Files...)
		return args, nil

	case "commit":
		if strings.TrimSpace(params.Message) == "" {
			return nil, fmt.Errorf("Error: 'message' parameter is required for commit command")
		}
		return []string{"commit", "-m", params.Message}, nil

	case "branch":
		return []string{"branch", "-a"}, nil

	case "stash":
		action := params.Action
		if action == "" {
			action = "list"
		}
		switch action {
		case "list":
			return []string{"stash", "list"}, nil
		case "pop":
			return []string{"stash", "pop"}, nil
		case "drop":
			return []string{"stash", "drop"}, nil
		default:
			return nil, fmt.Errorf("Error: unknown stash action %q. Allowed: list, pop, drop", action)
		}

	default:
		return nil, fmt.Errorf("Error: unknown git command %q", params.Command)
	}
}
