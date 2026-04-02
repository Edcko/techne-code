package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlobTool_Name(t *testing.T) {
	tool := &GlobTool{}
	if tool.Name() != "glob" {
		t.Errorf("expected name 'glob', got %q", tool.Name())
	}
}

func TestGlobTool_Description(t *testing.T) {
	tool := &GlobTool{}
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestGlobTool_Parameters(t *testing.T) {
	tool := &GlobTool{}
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("parameters should be valid JSON")
	}
}

func TestGlobTool_RequiresPermission(t *testing.T) {
	tool := &GlobTool{}
	if tool.RequiresPermission() {
		t.Error("glob tool should not require permission")
	}
}

func TestGlobTool_Execute_SimplePattern(t *testing.T) {
	tool := &GlobTool{}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"pattern": "*.txt",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "test.txt") {
		t.Errorf("expected 'test.txt' in results, got: %s", result.Content)
	}
}

func TestGlobTool_Execute_NoMatches(t *testing.T) {
	tool := &GlobTool{}

	tmpDir := t.TempDir()

	input, _ := json.Marshal(map[string]string{
		"pattern": "*.nonexistent",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "No files matched") {
		t.Errorf("expected 'No files matched' message, got: %s", result.Content)
	}
}

func TestGlobTool_Execute_MultipleMatches(t *testing.T) {
	tool := &GlobTool{}

	tmpDir := t.TempDir()
	for i := 0; i < 3; i++ {
		testFile := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	input, _ := json.Marshal(map[string]string{
		"pattern": "*.txt",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}

	lines := strings.Count(result.Content, "\n")
	if lines != 2 {
		t.Errorf("expected 3 matches (2 newlines), got %d lines: %s", lines, result.Content)
	}
}

func TestGlobTool_Execute_DoubleStarPattern(t *testing.T) {
	tool := &GlobTool{}

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	testFile := filepath.Join(subDir, "nested.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"pattern": "**/*.txt",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "nested.txt") {
		t.Errorf("expected 'nested.txt' in results, got: %s", result.Content)
	}
}

func TestGlobTool_Execute_SpecificExtension(t *testing.T) {
	tool := &GlobTool{}

	tmpDir := t.TempDir()
	txtFile := filepath.Join(tmpDir, "file.txt")
	goFile := filepath.Join(tmpDir, "file.go")

	if err := os.WriteFile(txtFile, []byte("text"), 0644); err != nil {
		t.Fatalf("failed to create txt file: %v", err)
	}
	if err := os.WriteFile(goFile, []byte("code"), 0644); err != nil {
		t.Fatalf("failed to create go file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"pattern": "*.go",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "file.go") {
		t.Errorf("expected 'file.go' in results, got: %s", result.Content)
	}
	if strings.Contains(result.Content, "file.txt") {
		t.Errorf("should not contain 'file.txt', got: %s", result.Content)
	}
}

func TestGlobTool_Execute_DefaultPath(t *testing.T) {
	tool := &GlobTool{}

	input, _ := json.Marshal(map[string]string{
		"pattern": "*.go",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestGlobTool_Execute_MissingPattern(t *testing.T) {
	tool := &GlobTool{}

	input, _ := json.Marshal(map[string]string{})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

func TestGlobTool_Execute_InvalidJSON(t *testing.T) {
	tool := &GlobTool{}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid JSON")
	}
}

func TestGlobTool_Execute_InvalidPattern(t *testing.T) {
	tool := &GlobTool{}

	input, _ := json.Marshal(map[string]string{
		"pattern": "[invalid",
		"path":    ".",
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid glob pattern")
	}
}

func TestGlobTool_Execute_SubdirectoryPattern(t *testing.T) {
	tool := &GlobTool{}

	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	testFile := filepath.Join(srcDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"pattern": "src/*.go",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "main.go") {
		t.Errorf("expected 'main.go' in results, got: %s", result.Content)
	}
}

func TestGlobTool_Execute_HiddenFiles(t *testing.T) {
	tool := &GlobTool{}

	tmpDir := t.TempDir()
	hiddenFile := filepath.Join(tmpDir, ".hidden")
	if err := os.WriteFile(hiddenFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("failed to create hidden file: %v", err)
	}

	input, _ := json.Marshal(map[string]string{
		"pattern": ".*",
		"path":    tmpDir,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}
