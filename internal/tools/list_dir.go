package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Edcko/techne-code/pkg/tool"
)

type ListDirTool struct{}

type listDirParams struct {
	Path      string `json:"path,omitempty"`
	All       bool   `json:"all,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
}

func (t *ListDirTool) Name() string { return "list_dir" }
func (t *ListDirTool) Description() string {
	return "List directory contents with file info (name, size, modified time). Shows hidden files with 'all', recurse with 'recursive'."
}
func (t *ListDirTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Directory path to list (default: current directory)"},
			"all": {"type": "boolean", "description": "Show hidden files (default: false)"},
			"recursive": {"type": "boolean", "description": "List subdirectories recursively (default: false)"}
		},
		"required": []
	}`)
}
func (t *ListDirTool) RequiresPermission() bool { return false }

func (t *ListDirTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	var params listDirParams
	if err := json.Unmarshal(input, &params); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error parsing parameters: %v", err), IsError: true}, nil
	}

	dir := params.Path
	if dir == "" {
		dir = "."
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error resolving path: %v", err), IsError: true}, nil
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error accessing directory: %v", err), IsError: true}, nil
	}
	if !info.IsDir() {
		return tool.ToolResult{Content: fmt.Sprintf("Path is not a directory: %s", absDir), IsError: true}, nil
	}

	var entries []dirEntry

	if params.Recursive {
		err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if path == absDir {
				return nil
			}
			rel, err := filepath.Rel(absDir, path)
			if err != nil {
				return nil
			}
			if !params.All && isHidden(rel) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			fi, err := d.Info()
			if err != nil {
				return nil
			}
			entries = append(entries, dirEntry{
				name:    rel,
				isDir:   d.IsDir(),
				size:    fi.Size(),
				modTime: fi.ModTime(),
			})
			return nil
		})
		if err != nil {
			return tool.ToolResult{Content: fmt.Sprintf("Error walking directory: %v", err), IsError: true}, nil
		}
	} else {
		des, err := os.ReadDir(absDir)
		if err != nil {
			return tool.ToolResult{Content: fmt.Sprintf("Error reading directory: %v", err), IsError: true}, nil
		}
		for _, d := range des {
			if !params.All && isHidden(d.Name()) {
				continue
			}
			fi, err := d.Info()
			if err != nil {
				continue
			}
			name := d.Name()
			entries = append(entries, dirEntry{
				name:    name,
				isDir:   d.IsDir(),
				size:    fi.Size(),
				modTime: fi.ModTime(),
			})
		}
	}

	if len(entries) == 0 {
		return tool.ToolResult{Content: "Empty directory."}, nil
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].isDir != entries[j].isDir {
			return entries[i].isDir
		}
		return entries[i].name < entries[j].name
	})

	var sb strings.Builder
	for _, e := range entries {
		name := e.name
		if e.isDir {
			name += "/"
		}
		fmt.Fprintf(&sb, "%-40s %8s  %s\n", name, formatSize(e.size), formatTime(e.modTime))
	}

	return tool.ToolResult{Content: sb.String()}, nil
}

type dirEntry struct {
	name    string
	isDir   bool
	size    int64
	modTime time.Time
}

func isHidden(name string) bool {
	base := filepath.Base(name)
	return len(base) > 0 && base[0] == '.'
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)
	switch {
	case size >= MB:
		return fmt.Sprintf("%.1fMB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.1fKB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%dB", size)
	}
}

func formatTime(t time.Time) string {
	now := time.Now()
	d := now.Sub(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		minutes := int(d.Minutes())
		return fmt.Sprintf("%dm ago", minutes)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		return fmt.Sprintf("%dh ago", hours)
	case d < 48*time.Hour:
		return "yesterday"
	case d < 30*24*time.Hour:
		days := int(d.Hours()) / 24
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 02 2006")
	}
}
