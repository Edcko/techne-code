package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Edcko/techne-code/pkg/tool"
)

// ReadFileTool reads file contents with optional line range.
type ReadFileTool struct{}

type readFileParams struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func (t *ReadFileTool) Name() string { return "read_file" }
func (t *ReadFileTool) Description() string {
	return "Read the contents of a file. Returns the file content with line numbers. Use offset and limit to read specific line ranges."
}
func (t *ReadFileTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Path to the file to read"},
			"offset": {"type": "integer", "description": "Starting line number (1-based, optional)"},
			"limit": {"type": "integer", "description": "Maximum number of lines to read (optional)"}
		},
		"required": ["path"]
	}`)
}
func (t *ReadFileTool) RequiresPermission() bool { return false }

func (t *ReadFileTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	var params readFileParams
	if err := json.Unmarshal(input, &params); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error parsing parameters: %v", err), IsError: true}, nil
	}

	data, err := os.ReadFile(params.Path)
	if err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error reading file: %v", err), IsError: true}, nil
	}

	lines := strings.Split(string(data), "\n")

	// Apply offset (1-based to 0-based)
	offset := params.Offset
	if offset < 1 {
		offset = 1
	}
	limit := params.Limit
	if limit <= 0 {
		limit = len(lines)
	}

	end := offset + limit - 1
	if end > len(lines) {
		end = len(lines)
	}

	var sb strings.Builder
	for i := offset - 1; i < end && i < len(lines); i++ {
		sb.WriteString(fmt.Sprintf("%s: %s\n", strconv.Itoa(i+1), lines[i]))
	}

	return tool.ToolResult{Content: sb.String()}, nil
}
