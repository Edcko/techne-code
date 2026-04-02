package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Edcko/techne-code/pkg/tool"
)

// BashTool executes shell commands.
type BashTool struct {
	MaxTimeoutMs int
	MaxOutput    int
}

func NewBashTool() *BashTool {
	return &BashTool{
		MaxTimeoutMs: 120000, // 2 minutes
		MaxOutput:    20000,  // 20K chars
	}
}

type bashParams struct {
	Command   string `json:"command"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
}

// Banned commands that should never be executed.
var bannedCommands = []string{
	"rm -rf /", "mkfs.", "dd if=", "> /dev/sd",
	":(){ :|:& };:", "fork bomb",
}

// Safe commands that can be auto-approved.
var safeCommands = []string{
	"git status", "git log", "git diff", "git branch", "git remote",
	"ls", "cat ", "find ", "wc ", "pwd", "echo ", "head ", "tail ",
	"which ", "type ", "env", "printenv", "go version", "go env",
	"go list", "go mod", "node --version", "npm --version",
}

func (t *BashTool) Name() string { return "bash" }
func (t *BashTool) Description() string {
	return "Execute a shell command and return stdout/stderr. Commands run in the project directory. Output is truncated if too long."
}
func (t *BashTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {"type": "string", "description": "Shell command to execute"},
			"timeout_ms": {"type": "integer", "description": "Timeout in milliseconds (default 120000)"}
		},
		"required": ["command"]
	}`)
}
func (t *BashTool) RequiresPermission() bool { return true }

func (t *BashTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	var params bashParams
	if err := json.Unmarshal(input, &params); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error parsing parameters: %v", err), IsError: true}, nil
	}

	// Check banned commands
	cmdLower := strings.ToLower(params.Command)
	for _, banned := range bannedCommands {
		if strings.Contains(cmdLower, banned) {
			return tool.ToolResult{
				Content: fmt.Sprintf("Error: command contains banned pattern: %q", banned),
				IsError: true,
			}, nil
		}
	}

	// Set timeout
	timeoutMs := params.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = t.MaxTimeoutMs
	}
	timeout := time.Duration(timeoutMs) * time.Millisecond

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", params.Command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n[stderr]\n" + stderr.String()
	}

	// Truncate if too long
	if len(output) > t.MaxOutput {
		half := t.MaxOutput / 2
		output = output[:half] + "\n\n... [truncated] ...\n\n" + output[len(output)-half:]
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			output += fmt.Sprintf("\n[timeout] Command timed out after %dms", timeoutMs)
		} else {
			output += fmt.Sprintf("\n[exit code non-zero] %v", err)
		}
	}

	return tool.ToolResult{Content: output}, nil
}

// IsSafeCommand checks if a command is in the safe list.
func (t *BashTool) IsSafeCommand(cmd string) bool {
	cmdLower := strings.ToLower(cmd)
	for _, safe := range safeCommands {
		if strings.HasPrefix(cmdLower, safe) {
			return true
		}
	}
	return false
}
