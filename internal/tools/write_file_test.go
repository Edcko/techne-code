package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileTool_Name(t *testing.T) {
	tool := &WriteFileTool{}
	if tool.Name() != "write_file" {
		t.Errorf("expected name 'write_file', got %q", tool.Name())
	}
}

func TestWriteFileTool_Description(t *testing.T) {
	tool := &WriteFileTool{}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestWriteFileTool_Parameters(t *testing.T) {
	tool := &WriteFileTool{}
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("parameters should be valid JSON")
	}
}

func TestWriteFileTool_RequiresPermission(t *testing.T) {
	tool := &WriteFileTool{}
	if !tool.RequiresPermission() {
		t.Error("write_file tool should require permission")
	}
}

func TestWriteFileTool_Execute_CreateFile(t *testing.T) {
	tool := &WriteFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	input, _ := json.Marshal(map[string]string{
		"path":    testFile,
		"content": "hello world",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(data))
	}
}

func TestWriteFileTool_Execute_OverwriteFile(t *testing.T) {
	tool := &WriteFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("original content"), 0644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"path":    testFile,
		"content": "new content",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "new content" {
		t.Errorf("expected 'new content', got %q", string(data))
	}
}

func TestWriteFileTool_Execute_CreateParentDirs(t *testing.T) {
	tool := &WriteFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "deep", "test.txt")

	input, _ := json.Marshal(map[string]string{
		"path":    testFile,
		"content": "nested content",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "nested content" {
		t.Errorf("expected 'nested content', got %q", string(data))
	}
}

func TestWriteFileTool_Execute_EmptyContent(t *testing.T) {
	tool := &WriteFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	input, _ := json.Marshal(map[string]string{
		"path":    testFile,
		"content": "",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "" {
		t.Errorf("expected empty file, got %q", string(data))
	}
}

func TestWriteFileTool_Execute_MultilineContent(t *testing.T) {
	tool := &WriteFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multiline.txt")
	content := "line 1\nline 2\nline 3\n"

	input, _ := json.Marshal(map[string]string{
		"path":    testFile,
		"content": content,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected %q, got %q", content, string(data))
	}
}

func TestWriteFileTool_Execute_MissingPath(t *testing.T) {
	tool := &WriteFileTool{}

	input, _ := json.Marshal(map[string]string{"content": "test"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing path")
	}
}

func TestWriteFileTool_Execute_MissingContent(t *testing.T) {
	tool := &WriteFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	input, _ := json.Marshal(map[string]string{"path": testFile})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

func TestWriteFileTool_Execute_InvalidJSON(t *testing.T) {
	tool := &WriteFileTool{}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid JSON")
	}
}

func TestWriteFileTool_Execute_ReportsBytesWritten(t *testing.T) {
	tool := &WriteFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world"

	input, _ := json.Marshal(map[string]string{
		"path":    testFile,
		"content": content,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "11 bytes") {
		t.Errorf("expected bytes count in result, got: %s", result.Content)
	}
}
