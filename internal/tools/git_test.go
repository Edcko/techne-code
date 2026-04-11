package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "techne-git-test-*")
	if err != nil {
		t.Fatal(err)
	}

	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup command %s %v failed: %v\n%s", name, args, err, out)
		}
	}

	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")

	err = os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello world"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	run("git", "add", "hello.txt")
	run("git", "commit", "-m", "initial commit")

	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestGitTool_Name(t *testing.T) {
	gt := NewGitTool()
	if gt.Name() != "git" {
		t.Errorf("expected name 'git', got %q", gt.Name())
	}
}

func TestGitTool_Description(t *testing.T) {
	gt := NewGitTool()
	if gt.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestGitTool_Parameters(t *testing.T) {
	gt := NewGitTool()
	params := gt.Parameters()
	if !json.Valid(params) {
		t.Error("parameters should be valid JSON")
	}
}

func TestGitTool_InvalidJSON(t *testing.T) {
	gt := NewGitTool()
	result, err := gt.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid JSON")
	}
}

func TestGitTool_DisallowedCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{"push", "push"},
		{"reset", "reset"},
		{"checkout", "checkout"},
		{"rebase", "rebase"},
		{"clean", "clean"},
		{"arbitrary", "arbitrary-command"},
		{"inject", "status; rm -rf /"},
		{"inject_pipe", "status | cat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gt := NewGitTool()
			input, _ := json.Marshal(map[string]string{"command": tt.command})
			result, err := gt.Execute(context.Background(), input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Errorf("expected error for disallowed command %q, got: %s", tt.command, result.Content)
			}
			if !strings.Contains(result.Content, "not allowed") {
				t.Errorf("expected 'not allowed' in error message, got: %s", result.Content)
			}
		})
	}
}

func TestGitTool_Status(t *testing.T) {
	dir := setupTestGitRepo(t)
	gt := NewGitTool()

	input, _ := json.Marshal(map[string]string{"command": "status"})

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "status", "--porcelain=v2")
	expected, _ := cmd.Output()

	result, err := gt.Execute(withDir(ctx, dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	_ = expected
}

func TestGitTool_Diff(t *testing.T) {
	dir := setupTestGitRepo(t)

	err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("modified content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	gt := NewGitTool()
	input, _ := json.Marshal(map[string]string{"command": "diff"})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "modified content") && !strings.Contains(result.Content, "hello.txt") {
		t.Errorf("expected diff output, got: %s", result.Content)
	}
}

func TestGitTool_DiffStaged(t *testing.T) {
	dir := setupTestGitRepo(t)

	err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("staged content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	runInDir(t, dir, "git", "add", "hello.txt")

	gt := NewGitTool()
	input, _ := json.Marshal(map[string]interface{}{
		"command": "diff",
		"staged":  true,
	})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestGitTool_DiffFile(t *testing.T) {
	dir := setupTestGitRepo(t)

	err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("changed"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	gt := NewGitTool()
	input, _ := json.Marshal(map[string]interface{}{
		"command": "diff",
		"file":    "hello.txt",
	})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestGitTool_Log(t *testing.T) {
	dir := setupTestGitRepo(t)
	gt := NewGitTool()

	input, _ := json.Marshal(map[string]interface{}{
		"command": "log",
		"count":   5,
	})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "initial commit") {
		t.Errorf("expected log to contain 'initial commit', got: %s", result.Content)
	}
}

func TestGitTool_LogDefault(t *testing.T) {
	dir := setupTestGitRepo(t)
	gt := NewGitTool()

	input, _ := json.Marshal(map[string]string{"command": "log"})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "initial commit") {
		t.Errorf("expected log to contain 'initial commit', got: %s", result.Content)
	}
}

func TestGitTool_LogOneline(t *testing.T) {
	dir := setupTestGitRepo(t)
	gt := NewGitTool()

	input, _ := json.Marshal(map[string]interface{}{
		"command": "log",
		"count":   5,
		"oneline": true,
	})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "initial commit") {
		t.Errorf("expected log to contain 'initial commit', got: %s", result.Content)
	}
}

func TestGitTool_Add(t *testing.T) {
	dir := setupTestGitRepo(t)

	err := os.WriteFile(filepath.Join(dir, "newfile.txt"), []byte("new content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	gt := NewGitTool()
	input, _ := json.Marshal(map[string]interface{}{
		"command": "add",
		"files":   []string{"newfile.txt"},
	})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestGitTool_AddNoFiles(t *testing.T) {
	gt := NewGitTool()
	input, _ := json.Marshal(map[string]interface{}{
		"command": "add",
		"files":   []string{},
	})

	result, err := gt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error when no files provided")
	}
	if !strings.Contains(result.Content, "files") {
		t.Errorf("expected 'files' in error, got: %s", result.Content)
	}
}

func TestGitTool_AddMissingFiles(t *testing.T) {
	gt := NewGitTool()
	input, _ := json.Marshal(map[string]string{"command": "add"})

	result, err := gt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error when files parameter missing")
	}
}

func TestGitTool_Commit(t *testing.T) {
	dir := setupTestGitRepo(t)

	err := os.WriteFile(filepath.Join(dir, "another.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	runInDir(t, dir, "git", "add", "another.txt")

	gt := NewGitTool()
	input, _ := json.Marshal(map[string]interface{}{
		"command": "commit",
		"message": "add another file",
	})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestGitTool_CommitNoMessage(t *testing.T) {
	gt := NewGitTool()
	input, _ := json.Marshal(map[string]string{"command": "commit"})

	result, err := gt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error when message missing")
	}
	if !strings.Contains(result.Content, "message") {
		t.Errorf("expected 'message' in error, got: %s", result.Content)
	}
}

func TestGitTool_CommitEmptyMessage(t *testing.T) {
	gt := NewGitTool()
	input, _ := json.Marshal(map[string]interface{}{
		"command": "commit",
		"message": "   ",
	})

	result, err := gt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for empty message")
	}
}

func TestGitTool_Branch(t *testing.T) {
	dir := setupTestGitRepo(t)
	gt := NewGitTool()

	input, _ := json.Marshal(map[string]string{"command": "branch"})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestGitTool_StashList(t *testing.T) {
	dir := setupTestGitRepo(t)
	gt := NewGitTool()

	input, _ := json.Marshal(map[string]interface{}{
		"command": "stash",
		"action":  "list",
	})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestGitTool_StashDefault(t *testing.T) {
	dir := setupTestGitRepo(t)
	gt := NewGitTool()

	input, _ := json.Marshal(map[string]string{"command": "stash"})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
}

func TestGitTool_StashInvalidAction(t *testing.T) {
	gt := NewGitTool()
	input, _ := json.Marshal(map[string]interface{}{
		"command": "stash",
		"action":  "invalid",
	})

	result, err := gt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid stash action")
	}
	if !strings.Contains(result.Content, "unknown stash action") {
		t.Errorf("expected 'unknown stash action' in error, got: %s", result.Content)
	}
}

func TestGitTool_NotGitRepo(t *testing.T) {
	dir, err := os.MkdirTemp("", "techne-git-norepo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	gt := NewGitTool()
	input, _ := json.Marshal(map[string]string{"command": "status"})

	result, err := gt.Execute(withDir(context.Background(), dir), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error when not in a git repo")
	}
}

func TestGitTool_RequiresPermissionForInput(t *testing.T) {
	gt := NewGitTool()

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantPerms bool
	}{
		{
			name:      "status is read-only",
			input:     map[string]interface{}{"command": "status"},
			wantPerms: false,
		},
		{
			name:      "diff is read-only",
			input:     map[string]interface{}{"command": "diff"},
			wantPerms: false,
		},
		{
			name:      "log is read-only",
			input:     map[string]interface{}{"command": "log"},
			wantPerms: false,
		},
		{
			name:      "branch is read-only",
			input:     map[string]interface{}{"command": "branch"},
			wantPerms: false,
		},
		{
			name:      "stash list is read-only",
			input:     map[string]interface{}{"command": "stash", "action": "list"},
			wantPerms: false,
		},
		{
			name:      "add requires permission",
			input:     map[string]interface{}{"command": "add", "files": []string{"file.go"}},
			wantPerms: true,
		},
		{
			name:      "commit requires permission",
			input:     map[string]interface{}{"command": "commit", "message": "test"},
			wantPerms: true,
		},
		{
			name:      "stash pop requires permission",
			input:     map[string]interface{}{"command": "stash", "action": "pop"},
			wantPerms: true,
		},
		{
			name:      "stash drop requires permission",
			input:     map[string]interface{}{"command": "stash", "action": "drop"},
			wantPerms: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, _ := json.Marshal(tt.input)
			got := gt.RequiresPermissionForInput(input)
			if got != tt.wantPerms {
				t.Errorf("RequiresPermissionForInput(%v) = %v, want %v", tt.input, got, tt.wantPerms)
			}
		})
	}
}

func TestGitTool_RequiresPermissionForInput_InvalidJSON(t *testing.T) {
	gt := NewGitTool()
	got := gt.RequiresPermissionForInput(json.RawMessage(`{invalid`))
	if !got {
		t.Error("expected true for invalid JSON (safe default)")
	}
}

func withDir(ctx context.Context, dir string) context.Context {
	return WithDir(ctx, dir)
}

func runInDir(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("command %s %v failed: %v\n%s", name, args, err, out)
	}
}
