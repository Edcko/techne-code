package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditFileTool_Name(t *testing.T) {
	tool := &EditFileTool{}
	if tool.Name() != "edit_file" {
		t.Errorf("expected name 'edit_file', got %q", tool.Name())
	}
}

func TestEditFileTool_Description(t *testing.T) {
	tool := &EditFileTool{}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestEditFileTool_Parameters(t *testing.T) {
	tool := &EditFileTool{}
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("parameters should be valid JSON")
	}
}

func TestEditFileTool_RequiresPermission(t *testing.T) {
	tool := &EditFileTool{}
	if !tool.RequiresPermission() {
		t.Error("edit_file tool should require permission")
	}
}

func TestEditFileTool_Execute_SingleReplacement(t *testing.T) {
	tool := &EditFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	original := "hello world"
	if err := os.WriteFile(testFile, []byte(original), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"path":       testFile,
		"old_string": "world",
		"new_string": "universe",
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
	if string(data) != "hello universe" {
		t.Errorf("expected 'hello universe', got %q", string(data))
	}
}

func TestEditFileTool_Execute_MultipleReplacements(t *testing.T) {
	tool := &EditFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	original := "foo bar foo bar foo"
	if err := os.WriteFile(testFile, []byte(original), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"path":       testFile,
		"old_string": "foo",
		"new_string": "baz",
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
	expected := "baz bar baz bar baz"
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
	if !strings.Contains(result.Content, "3 occurrence") {
		t.Errorf("expected 3 occurrences in result, got: %s", result.Content)
	}
}

func TestEditFileTool_Execute_OldStringNotFound(t *testing.T) {
	tool := &EditFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	original := "hello world"
	if err := os.WriteFile(testFile, []byte(original), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"path":       testFile,
		"old_string": "nonexistent",
		"new_string": "replacement",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error when old_string not found")
	}
}

func TestEditFileTool_Execute_FileNotFound(t *testing.T) {
	tool := &EditFileTool{}

	input, _ := json.Marshal(map[string]string{
		"path":       "/nonexistent/file.txt",
		"old_string": "old",
		"new_string": "new",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for non-existent file")
	}
}

func TestEditFileTool_Execute_MissingPath(t *testing.T) {
	tool := &EditFileTool{}

	input, _ := json.Marshal(map[string]string{
		"old_string": "old",
		"new_string": "new",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing path")
	}
}

func TestEditFileTool_Execute_MissingOldString(t *testing.T) {
	tool := &EditFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"path":       testFile,
		"new_string": "new",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

func TestEditFileTool_Execute_MissingNewString(t *testing.T) {
	tool := &EditFileTool{}

	input, _ := json.Marshal(map[string]string{
		"path":       "/tmp/test.txt",
		"old_string": "old",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing new_string")
	}
}

func TestEditFileTool_Execute_InvalidJSON(t *testing.T) {
	tool := &EditFileTool{}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid JSON")
	}
}

func TestEditFileTool_Execute_ExactMatchWithWhitespace(t *testing.T) {
	tool := &EditFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	original := "hello   world"
	if err := os.WriteFile(testFile, []byte(original), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"path":       testFile,
		"old_string": "hello   world",
		"new_string": "replaced",
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
	if string(data) != "replaced" {
		t.Errorf("expected 'replaced', got %q", string(data))
	}
}

func TestEditFileTool_Execute_EmptyReplacement(t *testing.T) {
	tool := &EditFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	original := "hello world"
	if err := os.WriteFile(testFile, []byte(original), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"path":       testFile,
		"old_string": " world",
		"new_string": "",
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
	if string(data) != "hello" {
		t.Errorf("expected 'hello', got %q", string(data))
	}
}

func TestEditFileTool_Execute_MultilineOldString(t *testing.T) {
	tool := &EditFileTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	original := "line 1\nline 2\nline 3"
	if err := os.WriteFile(testFile, []byte(original), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"path":       testFile,
		"old_string": "line 1\nline 2",
		"new_string": "replaced",
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
	expected := "replaced\nline 3"
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}
