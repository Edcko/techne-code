package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Edcko/techne-code/internal/agent"
)

func TestDelegateTool_Name(t *testing.T) {
	dt := NewDelegateTool(nil, nil, nil, nil)
	if dt.Name() != "delegate" {
		t.Errorf("expected name 'delegate', got %q", dt.Name())
	}
}

func TestDelegateTool_Description(t *testing.T) {
	dt := NewDelegateTool(nil, nil, nil, nil)
	if dt.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestDelegateTool_Parameters(t *testing.T) {
	dt := NewDelegateTool(nil, nil, nil, nil)
	params := dt.Parameters()
	if !json.Valid(params) {
		t.Error("parameters should be valid JSON")
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(params, &schema); err != nil {
		t.Fatalf("failed to unmarshal parameters: %v", err)
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties in schema")
	}
	if _, ok := props["tasks"]; !ok {
		t.Error("expected 'tasks' property in schema")
	}

	required, ok := schema["required"].([]interface{})
	if !ok {
		t.Fatal("expected required array in schema")
	}
	if len(required) != 1 || required[0] != "tasks" {
		t.Error("expected 'tasks' to be required")
	}
}

func TestDelegateTool_RequiresPermission(t *testing.T) {
	dt := NewDelegateTool(nil, nil, nil, nil)
	if dt.RequiresPermission() {
		t.Error("delegate tool should not require permission")
	}
}

func TestDelegateTool_Execute_InvalidJSON(t *testing.T) {
	dt := NewDelegateTool(nil, nil, nil, nil)
	result, err := dt.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for invalid JSON")
	}
}

func TestDelegateTool_Execute_EmptyTasks(t *testing.T) {
	dt := NewDelegateTool(nil, nil, nil, nil)
	input, _ := json.Marshal(map[string]interface{}{"tasks": []interface{}{}})
	result, err := dt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for empty tasks")
	}
}

func TestDelegateTool_Execute_ExceedsMaxTasks(t *testing.T) {
	dt := NewDelegateTool(nil, nil, nil, nil)
	tasks := make([]map[string]string, 4)
	for i := range tasks {
		tasks[i] = map[string]string{"agent": "researcher", "prompt": "do stuff"}
	}
	input, _ := json.Marshal(map[string]interface{}{"tasks": tasks})
	result, err := dt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result when exceeding max tasks")
	}
	if !strings.Contains(result.Content, "maximum 3") {
		t.Errorf("expected max limit message, got %q", result.Content)
	}
}

func TestDelegateTool_Execute_InvalidAgentName(t *testing.T) {
	dt := NewDelegateTool(nil, nil, nil, nil)
	input, _ := json.Marshal(map[string]interface{}{
		"tasks": []map[string]string{
			{"agent": "invalid_agent", "prompt": "do stuff"},
		},
	})
	result, err := dt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for invalid agent name")
	}
	if !strings.Contains(result.Content, "invalid agent") {
		t.Errorf("expected invalid agent message, got %q", result.Content)
	}
}

func TestDelegateTool_Execute_EmptyPrompt(t *testing.T) {
	dt := NewDelegateTool(nil, nil, nil, nil)
	input, _ := json.Marshal(map[string]interface{}{
		"tasks": []map[string]string{
			{"agent": "researcher", "prompt": ""},
		},
	})
	result, err := dt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for empty prompt")
	}
}

func TestDelegateTool_Execute_MissingConfig(t *testing.T) {
	dt := NewDelegateTool(nil, newMockStore(), nil, map[string]agent.SubAgentConfig{})
	input, _ := json.Marshal(map[string]interface{}{
		"tasks": []map[string]string{
			{"agent": "researcher", "prompt": "research something"},
		},
	})
	result, err := dt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "FAILED") {
		t.Errorf("expected failure in results for missing config, got %q", result.Content)
	}
}

func TestDelegateTool_Execute_NilProvider(t *testing.T) {
	configs := map[string]agent.SubAgentConfig{
		"researcher": NewResearcherConfig("test-model"),
	}
	registry := NewRegistry()
	dt := NewDelegateTool(nil, newMockStore(), registry, configs)

	input, _ := json.Marshal(map[string]interface{}{
		"tasks": []map[string]string{
			{"agent": "researcher", "prompt": "research something"},
		},
	})
	result, err := dt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Content, "FAILED") {
		t.Errorf("expected failure in results for nil provider, got %q", result.Content)
	}
}

func TestDelegateTool_Execute_ParallelExecution(t *testing.T) {
	var callCount int64

	configs := map[string]agent.SubAgentConfig{
		"researcher": {
			Name:          "researcher",
			SystemPrompt:  "test",
			AllowedTools:  []string{},
			MaxIterations: 1,
			Model:         "test",
		},
		"coder": {
			Name:          "coder",
			SystemPrompt:  "test",
			AllowedTools:  []string{},
			MaxIterations: 1,
			Model:         "test",
		},
	}

	registry := NewRegistry()

	originalRun := func(ctx context.Context, task DelegateTask, config agent.SubAgentConfig) (string, error) {
		atomic.AddInt64(&callCount, 1)
		return "result from " + config.Name, nil
	}

	_ = originalRun

	dt := &DelegateTool{
		provider: nil,
		store:    newMockStore(),
		registry: registry,
		configs:  configs,
	}

	input, _ := json.Marshal(map[string]interface{}{
		"tasks": []map[string]string{
			{"agent": "researcher", "prompt": "task 1"},
			{"agent": "coder", "prompt": "task 2"},
		},
	})

	result, err := dt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("expected success but got error: %q", result.Content)
	}

	if !strings.Contains(result.Content, "Delegated 2 task(s)") {
		t.Errorf("expected 2 tasks in output, got %q", result.Content)
	}

	if !strings.Contains(result.Content, "researcher") || !strings.Contains(result.Content, "coder") {
		t.Errorf("expected both agent names in output, got %q", result.Content)
	}
}

func TestDelegateTool_Execute_PartialFailure(t *testing.T) {
	configs := map[string]agent.SubAgentConfig{
		"researcher": {
			Name:          "researcher",
			SystemPrompt:  "test",
			AllowedTools:  []string{},
			MaxIterations: 1,
			Model:         "test",
		},
		"coder": {
			Name:          "coder",
			SystemPrompt:  "test",
			AllowedTools:  []string{},
			MaxIterations: 1,
			Model:         "test",
		},
	}

	registry := NewRegistry()
	dt := NewDelegateTool(nil, newMockStore(), registry, configs)

	input, _ := json.Marshal(map[string]interface{}{
		"tasks": []map[string]string{
			{"agent": "researcher", "prompt": "task 1"},
			{"agent": "coder", "prompt": "task 2"},
		},
	})

	result, err := dt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "FAILED") {
		t.Errorf("expected at least one failure in results (nil provider), got %q", result.Content)
	}
}

func TestDelegateTool_Execute_ThreeTasks(t *testing.T) {
	configs := map[string]agent.SubAgentConfig{
		"researcher": NewResearcherConfig("test"),
		"coder":      NewCoderConfig("test"),
		"reviewer":   NewReviewerConfig("test"),
	}
	registry := NewRegistry()
	dt := NewDelegateTool(nil, newMockStore(), registry, configs)

	input, _ := json.Marshal(map[string]interface{}{
		"tasks": []map[string]string{
			{"agent": "researcher", "prompt": "task 1"},
			{"agent": "coder", "prompt": "task 2"},
			{"agent": "reviewer", "prompt": "task 3"},
		},
	})

	result, err := dt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Content, "Delegated 3 task(s)") {
		t.Errorf("expected 3 tasks in output, got %q", result.Content)
	}
}

func TestDelegateTool_Execute_SingleTask(t *testing.T) {
	configs := map[string]agent.SubAgentConfig{
		"researcher": NewResearcherConfig("test"),
	}
	registry := NewRegistry()
	dt := NewDelegateTool(nil, newMockStore(), registry, configs)

	input, _ := json.Marshal(map[string]interface{}{
		"tasks": []map[string]string{
			{"agent": "researcher", "prompt": "single task"},
		},
	})

	result, err := dt.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Errorf("expected success but got error: %q", result.Content)
	}

	if !strings.Contains(result.Content, "Delegated 1 task(s)") {
		t.Errorf("expected 1 task in output, got %q", result.Content)
	}
}

func TestFormatDelegateResults_AllSuccess(t *testing.T) {
	results := []delegateResult{
		{Agent: "researcher", Output: "found 3 files", Err: nil},
		{Agent: "coder", Output: "implemented feature", Err: nil},
	}
	output := formatDelegateResults(results)

	if !strings.Contains(output, "COMPLETED") {
		t.Error("expected COMPLETED status in output")
	}
	if !strings.Contains(output, "found 3 files") || !strings.Contains(output, "implemented feature") {
		t.Error("expected both outputs in result")
	}
	if strings.Contains(output, "FAILED") {
		t.Error("should not contain FAILED when all succeed")
	}
}

func TestFormatDelegateResults_MixedResults(t *testing.T) {
	results := []delegateResult{
		{Agent: "researcher", Output: "found stuff", Err: nil},
		{Agent: "coder", Output: "", Err: fmt.Errorf("something broke")},
	}
	formatted := formatDelegateResults(results)

	if !strings.Contains(formatted, "COMPLETED") {
		t.Error("expected COMPLETED for successful task")
	}
	if !strings.Contains(formatted, "FAILED") {
		t.Error("expected FAILED for errored task")
	}
}
