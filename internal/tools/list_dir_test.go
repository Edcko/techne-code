package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListDirTool_Name(t *testing.T) {
	tool := &ListDirTool{}
	if tool.Name() != "list_dir" {
		t.Errorf("expected name 'list_dir', got %q", tool.Name())
	}
}

func TestListDirTool_Description(t *testing.T) {
	tool := &ListDirTool{}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestListDirTool_Parameters(t *testing.T) {
	tool := &ListDirTool{}
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("parameters should be valid JSON")
	}
}

func TestListDirTool_RequiresPermission(t *testing.T) {
	tool := &ListDirTool{}
	if tool.RequiresPermission() {
		t.Error("list_dir tool should not require permission")
	}
}

func TestListDirTool_Execute_BasicListing(t *testing.T) {
	tool := &ListDirTool{}
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "alpha.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "beta.go"), []byte("package main"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)

	input, _ := json.Marshal(map[string]string{"path": tmpDir})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "alpha.txt") {
		t.Errorf("expected 'alpha.txt' in output, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "beta.go") {
		t.Errorf("expected 'beta.go' in output, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "subdir/") {
		t.Errorf("expected 'subdir/' with trailing slash in output, got: %s", result.Content)
	}
}

func TestListDirTool_Execute_HiddenFiles(t *testing.T) {
	tool := &ListDirTool{}
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("hi"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("secret"), 0644)

	input, _ := json.Marshal(map[string]interface{}{
		"path": tmpDir,
		"all":  false,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "visible.txt") {
		t.Errorf("expected 'visible.txt' in output, got: %s", result.Content)
	}
	if strings.Contains(result.Content, ".hidden") {
		t.Errorf("hidden file should not appear without 'all', got: %s", result.Content)
	}

	inputAll, _ := json.Marshal(map[string]interface{}{
		"path": tmpDir,
		"all":  true,
	})
	resultAll, err := tool.Execute(context.Background(), inputAll)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resultAll.IsError {
		t.Errorf("unexpected error: %s", resultAll.Content)
	}
	if !strings.Contains(resultAll.Content, ".hidden") {
		t.Errorf("expected '.hidden' with 'all' flag, got: %s", resultAll.Content)
	}
}

func TestListDirTool_Execute_Recursive(t *testing.T) {
	tool := &ListDirTool{}
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("root"), 0644)
	os.WriteFile(filepath.Join(subDir, "nested.go"), []byte("package main"), 0644)

	input, _ := json.Marshal(map[string]interface{}{
		"path":      tmpDir,
		"recursive": true,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "root.txt") {
		t.Errorf("expected 'root.txt' in output, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, filepath.Join("src", "nested.go")) {
		t.Errorf("expected nested file path in output, got: %s", result.Content)
	}
}

func TestListDirTool_Execute_RecursiveHiddenSkipped(t *testing.T) {
	tool := &ListDirTool{}
	tmpDir := t.TempDir()

	hiddenDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(hiddenDir, 0755)
	os.WriteFile(filepath.Join(hiddenDir, "config"), []byte("git config"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	input, _ := json.Marshal(map[string]interface{}{
		"path":      tmpDir,
		"recursive": true,
		"all":       false,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if strings.Contains(result.Content, ".git") {
		t.Errorf("hidden dirs should be skipped without 'all', got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "main.go") {
		t.Errorf("expected 'main.go' in output, got: %s", result.Content)
	}
}

func TestListDirTool_Execute_EmptyDir(t *testing.T) {
	tool := &ListDirTool{}
	tmpDir := t.TempDir()

	input, _ := json.Marshal(map[string]string{"path": tmpDir})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Empty directory") {
		t.Errorf("expected empty directory message, got: %s", result.Content)
	}
}

func TestListDirTool_Execute_NonExistentDir(t *testing.T) {
	tool := &ListDirTool{}

	input, _ := json.Marshal(map[string]string{"path": "/nonexistent/path/xyz"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for non-existent directory")
	}
}

func TestListDirTool_Execute_PathIsFile(t *testing.T) {
	tool := &ListDirTool{}
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	input, _ := json.Marshal(map[string]string{"path": testFile})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error when path is a file")
	}
}

func TestListDirTool_Execute_DefaultPath(t *testing.T) {
	tool := &ListDirTool{}

	input, _ := json.Marshal(map[string]string{})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestListDirTool_Execute_InvalidJSON(t *testing.T) {
	tool := &ListDirTool{}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid JSON")
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1572864, "1.5MB"},
		{5242880, "5.0MB"},
	}
	for _, tt := range tests {
		got := formatSize(tt.size)
		if got != tt.expected {
			t.Errorf("formatSize(%d) = %q, want %q", tt.size, got, tt.expected)
		}
	}
}

func TestFormatTime(t *testing.T) {
	now := strings.Contains(formatTime(time.Now()), "just now")
	if !now {
		t.Error("expected 'just now' for current time")
	}
}

func TestListDirTool_Execute_DirsSortedFirst(t *testing.T) {
	tool := &ListDirTool{}
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "zzz.txt"), []byte("last"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "aaa"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "bbb.txt"), []byte("mid"), 0644)

	input, _ := json.Marshal(map[string]string{"path": tmpDir})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}

	lines := strings.Split(strings.TrimSpace(result.Content), "\n")
	var dirIdx, fileIdx int
	for i, line := range lines {
		if strings.Contains(line, "aaa/") {
			dirIdx = i
		}
		if strings.Contains(line, "zzz.txt") {
			fileIdx = i
		}
	}
	if dirIdx > fileIdx {
		t.Errorf("directories should appear before files, got: %s", result.Content)
	}
}

func TestListDirTool_Execute_SizeFormatting(t *testing.T) {
	tool := &ListDirTool{}
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "small.txt"), []byte("hi"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "big.txt"), make([]byte, 2048), 0644)

	input, _ := json.Marshal(map[string]string{"path": tmpDir})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "B") && !strings.Contains(result.Content, "KB") {
		t.Errorf("expected size formatting in output, got: %s", result.Content)
	}
}
