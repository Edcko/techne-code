package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBashTool_Name(t *testing.T) {
	tool := NewBashTool()
	if tool.Name() != "bash" {
		t.Errorf("expected name 'bash', got %q", tool.Name())
	}
}

func TestBashTool_Description(t *testing.T) {
	tool := NewBashTool()
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestBashTool_Parameters(t *testing.T) {
	tool := NewBashTool()
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("parameters should be valid JSON")
	}
}

func TestBashTool_RequiresPermission(t *testing.T) {
	tool := NewBashTool()
	if !tool.RequiresPermission() {
		t.Error("bash tool should require permission")
	}
}

func TestBashTool_Execute_SimpleCommand(t *testing.T) {
	tool := NewBashTool()
	input, _ := json.Marshal(map[string]string{"command": "echo hello"})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error in result: %s", result.Content)
	}
	if !strings.Contains(result.Content, "hello") {
		t.Errorf("expected output to contain 'hello', got: %s", result.Content)
	}
}

func TestBashTool_Execute_CommandWithStderr(t *testing.T) {
	tool := NewBashTool()
	input, _ := json.Marshal(map[string]string{"command": "echo stderr >&2"})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "stderr") {
		t.Errorf("expected output to contain 'stderr', got: %s", result.Content)
	}
}

func TestBashTool_Execute_MissingCommand(t *testing.T) {
	tool := NewBashTool()
	input, _ := json.Marshal(map[string]string{})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

func TestBashTool_Execute_InvalidJSON(t *testing.T) {
	tool := NewBashTool()

	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid JSON")
	}
}

func TestBashTool_Execute_BannedCommand_RmRf(t *testing.T) {
	tool := NewBashTool()
	input, _ := json.Marshal(map[string]string{"command": "rm -rf /"})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for banned command")
	}
	if !strings.Contains(result.Content, "banned") {
		t.Errorf("expected 'banned' in error message, got: %s", result.Content)
	}
}

func TestBashTool_Execute_BannedCommand_ForkBomb(t *testing.T) {
	tool := NewBashTool()
	input, _ := json.Marshal(map[string]string{"command": ":(){ :|:& };:"})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for banned command")
	}
}

func TestBashTool_Execute_Timeout(t *testing.T) {
	tool := NewBashTool()
	input, _ := json.Marshal(map[string]interface{}{
		"command":    "sleep 10",
		"timeout_ms": 100,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "timeout") {
		t.Errorf("expected timeout in output, got: %s", result.Content)
	}
}

func TestBashTool_Execute_NonZeroExitCode(t *testing.T) {
	tool := NewBashTool()
	input, _ := json.Marshal(map[string]string{"command": "exit 1"})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "exit code") && !strings.Contains(result.Content, "non-zero") {
		t.Errorf("expected exit code message in output, got: %s", result.Content)
	}
}

func TestBashTool_Execute_CommandNotFound(t *testing.T) {
	tool := NewBashTool()
	input, _ := json.Marshal(map[string]string{"command": "nonexistentcommand12345"})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError && !strings.Contains(result.Content, "command") {
		t.Errorf("expected command-related error, got: %s", result.Content)
	}
}

func TestBashTool_Execute_CustomTimeout(t *testing.T) {
	tool := NewBashTool()
	input, _ := json.Marshal(map[string]interface{}{
		"command":    "echo fast",
		"timeout_ms": 5000,
	})

	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestBashTool_IsSafeCommand(t *testing.T) {
	tool := NewBashTool()

	safeCommands := []string{
		"git status",
		"git log",
		"ls -la",
		"pwd",
		"echo hello",
		"go version",
	}

	for _, cmd := range safeCommands {
		if !tool.IsSafeCommand(cmd) {
			t.Errorf("expected %q to be safe", cmd)
		}
	}

	unsafeCommands := []string{
		"rm file.txt",
		"sudo apt install",
		"curl http://example.com",
	}

	for _, cmd := range unsafeCommands {
		if tool.IsSafeCommand(cmd) {
			t.Errorf("expected %q to be unsafe", cmd)
		}
	}
}
