package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Edcko/techne-code/pkg/tool"
)

// WriteFileTool creates or overwrites a file with the given content.
type WriteFileTool struct{}

type writeParams struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string {
	return "Create or overwrite a file with the given content. Creates parent directories if needed."
}
func (t *WriteFileTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Path to the file to write"},
			"content": {"type": "string", "description": "Content to write to the file"}
		},
		"required": ["path", "content"]
	}`)
}
func (t *WriteFileTool) RequiresPermission() bool { return true }

func (t *WriteFileTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	var params writeParams
	if err := json.Unmarshal(input, &params); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error parsing parameters: %v", err), IsError: true}, nil
	}

	// Create parent directories if needed
	dir := filepath.Dir(params.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error creating directory: %v", err), IsError: true}, nil
	}

	if err := os.WriteFile(params.Path, []byte(params.Content), 0644); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error writing file: %v", err), IsError: true}, nil
	}

	return tool.ToolResult{Content: fmt.Sprintf("Successfully wrote %d bytes to %s", len(params.Content), params.Path)}, nil
}
