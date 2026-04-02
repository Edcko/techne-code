package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Edcko/techne-code/pkg/tool"
)

// EditFileTool performs find/replace within a file.
type EditFileTool struct{}

type editParams struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

func (t *EditFileTool) Name() string { return "edit_file" }
func (t *EditFileTool) Description() string {
	return "Find and replace text in a file. The old_string must match exactly (including whitespace). Returns the number of replacements made."
}
func (t *EditFileTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Path to the file to edit"},
			"old_string": {"type": "string", "description": "Exact text to find and replace"},
			"new_string": {"type": "string", "description": "Text to replace with"}
		},
		"required": ["path", "old_string", "new_string"]
	}`)
}
func (t *EditFileTool) RequiresPermission() bool { return true }

func (t *EditFileTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	var params editParams
	if err := json.Unmarshal(input, &params); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error parsing parameters: %v", err), IsError: true}, nil
	}

	data, err := os.ReadFile(params.Path)
	if err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error reading file: %v", err), IsError: true}, nil
	}

	content := string(data)
	count := strings.Count(content, params.OldString)
	if count == 0 {
		return tool.ToolResult{Content: fmt.Sprintf("Error: old_string not found in %s", params.Path), IsError: true}, nil
	}

	newContent := strings.ReplaceAll(content, params.OldString, params.NewString)
	if err := os.WriteFile(params.Path, []byte(newContent), 0644); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error writing file: %v", err), IsError: true}, nil
	}

	return tool.ToolResult{Content: fmt.Sprintf("Replaced %d occurrence(s) in %s", count, params.Path)}, nil
}
