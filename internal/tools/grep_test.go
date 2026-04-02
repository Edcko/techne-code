package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrepTool_Name(t *testing.T) {
	tool := &GrepTool{}
	if tool.Name() != "grep" {
		t.Errorf("expected name 'grep', got %q", tool.Name())
	}
}

func TestGrepTool_Description(t *testing.T) {
	tool := &GrepTool{}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestGrepTool_Parameters(t *testing.T) {
	tool := &GrepTool{}
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("parameters should be valid JSON")
	}
}

func TestGrepTool_RequiresPermission(t *testing.T) {
	tool := &GrepTool{}
	if tool.RequiresPermission() {
		t.Error("grep tool should not require permission")
	}
}

func TestGrepTool_Execute_SimplePattern(t *testing.T) {
	tool := &GrepTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world\nfoo bar\nhello again"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"pattern": "hello",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "hello") {
		t.Errorf("expected 'hello' in results, got: %s", result.Content)
	}
}

func TestGrepTool_Execute_NoMatches(t *testing.T) {
	tool := &GrepTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world\nfoo bar"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"pattern": "nonexistent",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "No matches") {
		t.Errorf("expected 'No matches' message, got: %s", result.Content)
	}
}

func TestGrepTool_Execute_WithInclude(t *testing.T) {
	tool := &GrepTool{}

	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "test.go")
	txtFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(goFile, []byte("package main\nfunc test() {}"), 0644); err != nil {
		t.Fatalf("failed to create go file: %v", err)
	}
	if err := os.WriteFile(txtFile, []byte("package not this"), 0644); err != nil {
		t.Fatalf("failed to create txt file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"pattern": "package",
		"include": "*.go",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "test.go") {
		t.Errorf("expected 'test.go' in results, got: %s", result.Content)
	}
}

func TestGrepTool_Execute_RegexPattern(t *testing.T) {
	tool := &GrepTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "func test123()\nfunc other()\nnot a func"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"pattern": "func.*\\(\\)",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "func") {
		t.Errorf("expected 'func' in results, got: %s", result.Content)
	}
}

func TestGrepTool_Execute_MissingPattern(t *testing.T) {
	tool := &GrepTool{}

	input, _ := json.Marshal(map[string]string{})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

func TestGrepTool_Execute_InvalidJSON(t *testing.T) {
	tool := &GrepTool{}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid JSON")
	}
}

func TestGrepTool_Execute_DefaultPath(t *testing.T) {
	tool := &GrepTool{}

	input, _ := json.Marshal(map[string]string{
		"pattern": "func",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestGrepTool_Execute_InvalidRegex(t *testing.T) {
	tool := &GrepTool{}

	input, _ := json.Marshal(map[string]string{
		"pattern": "[invalid(regex",
		"path":    ".",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

func TestGrepTool_Execute_MultipleFiles(t *testing.T) {
	tool := &GrepTool{}

	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	if err := os.WriteFile(file1, []byte("match this line"), 0644); err != nil {
		t.Fatalf("failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("also match this"), 0644); err != nil {
		t.Fatalf("failed to create file2: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"pattern": "match",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "match") {
		t.Errorf("expected 'match' in results, got: %s", result.Content)
	}
}

func TestGrepTool_Execute_NonexistentPath(t *testing.T) {
	tool := &GrepTool{}

	input, _ := json.Marshal(map[string]string{
		"pattern": "test",
		"path":    "/nonexistent/directory",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}
