package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileTool_Name(t *testing.T) {
	tool := &ReadFileTool{}
	if tool.Name() != "read_file" {
		t.Errorf("expected name 'read_file', got %q", tool.Name())
	}
}

func TestReadFileTool_Description(t *testing.T) {
	tool := &ReadFileTool{}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestReadFileTool_Parameters(t *testing.T) {
	tool := &ReadFileTool{}
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("parameters should be valid JSON")
	}
}

func TestReadFileTool_RequiresPermission(t *testing.T) {
	tool := &ReadFileTool{}
	if tool.RequiresPermission() {
		t.Error("read_file tool should not require permission")
	}
}

func TestReadFileTool_Execute_SimpleRead(t *testing.T) {
	tool := &ReadFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world\nline 2\nline 3"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{"path": testFile})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "hello world") {
		t.Errorf("expected content to contain 'hello world', got: %s", result.Content)
	}
}

func TestReadFileTool_Execute_WithLineNumbers(t *testing.T) {
	tool := &ReadFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{"path": testFile})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "1: line 1") {
		t.Errorf("expected line numbers, got: %s", result.Content)
	}
}

func TestReadFileTool_Execute_WithOffset(t *testing.T) {
	tool := &ReadFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]interface{}{
		"path":   testFile,
		"offset": 2,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "line 2") {
		t.Errorf("expected content to contain 'line 2', got: %s", result.Content)
	}
	if strings.Contains(result.Content, "1: line 1") {
		t.Errorf("should not contain line 1 when offset is 2, got: %s", result.Content)
	}
}

func TestReadFileTool_Execute_WithLimit(t *testing.T) {
	tool := &ReadFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]interface{}{
		"path":  testFile,
		"limit": 2,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if strings.Contains(result.Content, "line 3") {
		t.Errorf("should not contain line 3 with limit 2, got: %s", result.Content)
	}
}

func TestReadFileTool_Execute_WithOffsetAndLimit(t *testing.T) {
	tool := &ReadFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]interface{}{
		"path":   testFile,
		"offset": 2,
		"limit":  2,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "line 2") {
		t.Errorf("expected 'line 2', got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "line 3") {
		t.Errorf("expected 'line 3', got: %s", result.Content)
	}
	if strings.Contains(result.Content, "line 4") {
		t.Errorf("should not contain 'line 4', got: %s", result.Content)
	}
}

func TestReadFileTool_Execute_FileNotFound(t *testing.T) {
	tool := &ReadFileTool{}

	input, _ := json.Marshal(map[string]string{"path": "/nonexistent/file.txt"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for non-existent file")
	}
}

func TestReadFileTool_Execute_MissingPath(t *testing.T) {
	tool := &ReadFileTool{}

	input, _ := json.Marshal(map[string]string{})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing path")
	}
}

func TestReadFileTool_Execute_InvalidJSON(t *testing.T) {
	tool := &ReadFileTool{}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid JSON")
	}
}

func TestReadFileTool_Execute_EmptyFile(t *testing.T) {
	tool := &ReadFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{"path": testFile})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestReadFileTool_Execute_OffsetOutOfRange(t *testing.T) {
	tool := &ReadFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]interface{}{
		"path":   testFile,
		"offset": 100,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}
