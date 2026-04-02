package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Edcko/techne-code/pkg/tool"
)

// GlobTool finds files matching a pattern.
type GlobTool struct{}

type globParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

func (t *GlobTool) Name() string { return "glob" }
func (t *GlobTool) Description() string {
	return "Find files matching a glob pattern. Returns matching file paths sorted by modification time."
}
func (t *GlobTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {"type": "string", "description": "Glob pattern (e.g., '**/*.go', 'src/**/*.ts')"},
			"path": {"type": "string", "description": "Base directory to search in (defaults to current directory)"}
		},
		"required": ["pattern"]
	}`)
}
func (t *GlobTool) RequiresPermission() bool { return false }

func (t *GlobTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	var params globParams
	if err := json.Unmarshal(input, &params); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error parsing parameters: %v", err), IsError: true}, nil
	}

	basePath := params.Path
	if basePath == "" {
		basePath = "."
	}

	matches, err := filepath.Glob(filepath.Join(basePath, params.Pattern))
	if err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error matching pattern: %v", err), IsError: true}, nil
	}

	// Also try walking for ** patterns
	if strings.Contains(params.Pattern, "**") {
		pattern := params.Pattern
		filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(basePath, path)
			matched, _ := filepath.Match(pattern, rel)
			if matched && !info.IsDir() {
				matches = append(matches, path)
			}
			return nil
		})
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := make([]string, 0, len(matches))
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			unique = append(unique, m)
		}
	}
	matches = unique

	sort.Strings(matches)

	if len(matches) == 0 {
		return tool.ToolResult{Content: "No files matched the pattern."}, nil
	}

	return tool.ToolResult{Content: strings.Join(matches, "\n")}, nil
}
