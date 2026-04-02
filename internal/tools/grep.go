package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Edcko/techne-code/pkg/tool"
)

// GrepTool searches file contents using regex patterns.
// Uses ripgrep (rg) if available, falls back to Go regex.
type GrepTool struct{}

type grepParams struct {
	Pattern string `json:"pattern"`
	Include string `json:"include,omitempty"`
	Path    string `json:"path,omitempty"`
}

func (t *GrepTool) Name() string { return "grep" }
func (t *GrepTool) Description() string {
	return "Search file contents using a regex pattern. Returns matching lines with file paths and line numbers. Uses ripgrep if available."
}
func (t *GrepTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {"type": "string", "description": "Regex pattern to search for"},
			"include": {"type": "string", "description": "File glob to include (e.g., '*.go', '*.ts')"},
			"path": {"type": "string", "description": "Directory to search in (defaults to current directory)"}
		},
		"required": ["pattern"]
	}`)
}
func (t *GrepTool) RequiresPermission() bool { return false }

func (t *GrepTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	var params grepParams
	if err := json.Unmarshal(input, &params); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error parsing parameters: %v", err), IsError: true}, nil
	}

	searchPath := params.Path
	if searchPath == "" {
		searchPath = "."
	}

	// Try ripgrep first
	if rgPath, err := exec.LookPath("rg"); err == nil {
		return t.grepWithRG(ctx, rgPath, params, searchPath)
	}

	// Fallback to Go regex
	return t.grepWithGoRegex(ctx, params, searchPath)
}

func (t *GrepTool) grepWithRG(ctx context.Context, rgPath string, params grepParams, searchPath string) (tool.ToolResult, error) {
	args := []string{"--no-heading", "--with-filename", "--line-number", "--color=never", "--max-count", "50"}
	if params.Include != "" {
		args = append(args, "--glob", params.Include)
	}
	args = append(args, params.Pattern, searchPath)

	cmd := exec.CommandContext(ctx, rgPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// rg returns exit code 1 when no matches found
		if strings.Contains(err.Error(), "exit status 1") {
			return tool.ToolResult{Content: "No matches found."}, nil
		}
		return tool.ToolResult{Content: string(output), IsError: true}, nil
	}

	result := string(output)
	if len(result) > 20000 {
		half := 10000
		result = result[:half] + "\n\n... [truncated] ...\n\n" + result[len(result)-half:]
	}

	return tool.ToolResult{Content: result}, nil
}

func (t *GrepTool) grepWithGoRegex(ctx context.Context, params grepParams, searchPath string) (tool.ToolResult, error) {
	re, err := regexp.Compile(params.Pattern)
	if err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Invalid regex pattern: %v", err), IsError: true}, nil
	}

	// Use glob tool to find files, then search each
	globber := &GlobTool{}
	globInput, _ := json.Marshal(globParams{
		Pattern: params.Include,
		Path:    searchPath,
	})
	if params.Include == "" {
		globInput, _ = json.Marshal(globParams{
			Pattern: "**/*",
			Path:    searchPath,
		})
	}

	globResult, _ := globber.Execute(ctx, globInput)
	if globResult.IsError {
		return globResult, nil
	}

	files := strings.Split(globResult.Content, "\n")
	var matches []string

	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" || file == "No files matched the pattern." {
			continue
		}

		data, err := readFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				matches = append(matches, fmt.Sprintf("%s:%d: %s", file, i+1, line))
				if len(matches) >= 50 {
					break
				}
			}
		}
		if len(matches) >= 50 {
			break
		}
	}

	if len(matches) == 0 {
		return tool.ToolResult{Content: "No matches found."}, nil
	}

	return tool.ToolResult{Content: strings.Join(matches, "\n")}, nil
}

// readFile is a helper to read a file's content.
func readFile(path string) ([]byte, error) {
	return exec.Command("cat", path).Output()
}
