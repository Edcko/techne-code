package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/tool"
)

func TestNewSubAgent_SetsDefaults(t *testing.T) {
	config := SubAgentConfig{
		Name:         "test",
		SystemPrompt: "test prompt",
		AllowedTools: []string{"read_file"},
		Model:        "test-model",
	}

	mockStore := NewMockStore()
	mockProvider := &MockProvider{}
	allTools := []tool.Tool{&MockTool{name: "read_file"}, &MockTool{name: "write_file"}}

	sa := NewSubAgent(mockProvider, mockStore, config, allTools)

	if sa.config.MaxIterations != 20 {
		t.Errorf("expected MaxIterations 20, got %d", sa.config.MaxIterations)
	}
	if sa.config.MaxTokens != 4096 {
		t.Errorf("expected MaxTokens 4096, got %d", sa.config.MaxTokens)
	}
}

func TestNewSubAgent_PreservesExplicitValues(t *testing.T) {
	config := SubAgentConfig{
		Name:          "test",
		SystemPrompt:  "test prompt",
		AllowedTools:  []string{"read_file"},
		MaxIterations: 10,
		MaxTokens:     2048,
		Model:         "test-model",
	}

	mockStore := NewMockStore()
	mockProvider := &MockProvider{}
	allTools := []tool.Tool{&MockTool{name: "read_file"}}

	sa := NewSubAgent(mockProvider, mockStore, config, allTools)

	if sa.config.MaxIterations != 10 {
		t.Errorf("expected MaxIterations 10, got %d", sa.config.MaxIterations)
	}
	if sa.config.MaxTokens != 2048 {
		t.Errorf("expected MaxTokens 2048, got %d", sa.config.MaxTokens)
	}
}

func TestNewSubAgent_FiltersTools(t *testing.T) {
	config := SubAgentConfig{
		Name:         "test",
		SystemPrompt: "test prompt",
		AllowedTools: []string{"read_file", "grep"},
		Model:        "test-model",
	}

	mockStore := NewMockStore()
	mockProvider := &MockProvider{}
	allTools := []tool.Tool{
		&MockTool{name: "read_file"},
		&MockTool{name: "write_file"},
		&MockTool{name: "grep"},
		&MockTool{name: "bash"},
	}

	sa := NewSubAgent(mockProvider, mockStore, config, allTools)

	if sa.ToolCount() != 2 {
		t.Fatalf("expected 2 tools, got %d", sa.ToolCount())
	}
	if !sa.HasTool("read_file") {
		t.Error("expected read_file to be available")
	}
	if !sa.HasTool("grep") {
		t.Error("expected grep to be available")
	}
	if sa.HasTool("write_file") {
		t.Error("expected write_file to NOT be available")
	}
	if sa.HasTool("bash") {
		t.Error("expected bash to NOT be available")
	}
}

func TestNewSubAgent_EmptyAllowedTools(t *testing.T) {
	config := SubAgentConfig{
		Name:         "test",
		SystemPrompt: "test prompt",
		AllowedTools: []string{},
		Model:        "test-model",
	}

	mockStore := NewMockStore()
	mockProvider := &MockProvider{}
	allTools := []tool.Tool{
		&MockTool{name: "read_file"},
		&MockTool{name: "write_file"},
	}

	sa := NewSubAgent(mockProvider, mockStore, config, allTools)

	if sa.ToolCount() != 0 {
		t.Errorf("expected 0 tools, got %d", sa.ToolCount())
	}
}

func TestNewSubAgent_NoMatchingTools(t *testing.T) {
	config := SubAgentConfig{
		Name:         "test",
		SystemPrompt: "test prompt",
		AllowedTools: []string{"nonexistent_tool"},
		Model:        "test-model",
	}

	mockStore := NewMockStore()
	mockProvider := &MockProvider{}
	allTools := []tool.Tool{&MockTool{name: "read_file"}}

	sa := NewSubAgent(mockProvider, mockStore, config, allTools)

	if sa.ToolCount() != 0 {
		t.Errorf("expected 0 tools when no match, got %d", sa.ToolCount())
	}
}

func TestSubAgent_Run_CollectsOutput(t *testing.T) {
	config := SubAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a test agent.",
		AllowedTools:  []string{"read_file"},
		MaxIterations: 5,
		Model:         "mock-model",
		MaxTokens:     1024,
	}

	mockStore := NewMockStore()
	mockProvider := &MockProvider{}
	allTools := []tool.Tool{&MockTool{name: "read_file"}}

	sa := NewSubAgent(mockProvider, mockStore, config, allTools)

	output, err := sa.Run(context.Background(), "do something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != "test response" {
		t.Errorf("expected 'test response', got %q", output)
	}
}

func TestSubAgent_Run_CreatesChildSession(t *testing.T) {
	config := SubAgentConfig{
		Name:          "test-agent",
		SystemPrompt:  "You are a test agent.",
		AllowedTools:  []string{},
		MaxIterations: 5,
		Model:         "mock-model",
		MaxTokens:     1024,
	}

	mockStore := NewMockStore()
	mockProvider := &MockProvider{}

	sa := NewSubAgent(mockProvider, mockStore, config, nil)

	_, err := sa.Run(context.Background(), "test task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sessions, _ := mockStore.ListSessions()
	found := false
	for _, s := range sessions {
		if s.Title == "[sub-agent:test-agent] test task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected child session with correct title to be created")
	}
}

func TestSilentEventBus(t *testing.T) {
	bus := &SilentEventBus{}

	bus.Publish(event.Event{})

	unsub := bus.Subscribe(func(e event.Event) {})
	unsub()

	bus.Close()
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello"},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"ab", 5, "ab"},
	}

	for _, tt := range tests {
		got := truncateString(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestScopedRegistry(t *testing.T) {
	tools := map[string]tool.Tool{
		"read_file": &MockTool{name: "read_file"},
		"grep":      &MockTool{name: "grep"},
	}

	reg := newScopedRegistry(tools)

	if len(reg.List()) != 2 {
		t.Errorf("expected 2 tools, got %d", len(reg.List()))
	}

	t1, ok := reg.Get("read_file")
	if !ok {
		t.Error("expected read_file to be found")
	}
	if t1.Name() != "read_file" {
		t.Errorf("expected read_file, got %q", t1.Name())
	}

	_, ok = reg.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent tool to not be found")
	}

	schemas := reg.Schemas()
	if len(schemas) != 2 {
		t.Errorf("expected 2 schemas, got %d", len(schemas))
	}

	newTool := &MockTool{name: "bash"}
	if err := reg.Register(newTool); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reg.List()) != 3 {
		t.Errorf("expected 3 tools after register, got %d", len(reg.List()))
	}
}

func TestScopedRegistry_Schemas(t *testing.T) {
	tools := map[string]tool.Tool{
		"test": &MockTool{
			name:        "test",
			description: "test tool",
			parameters:  json.RawMessage(`{"type":"object"}`),
		},
	}

	reg := newScopedRegistry(tools)
	schemas := reg.Schemas()

	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}

	if schemas[0].Name != "test" {
		t.Errorf("expected schema name 'test', got %q", schemas[0].Name)
	}
	if schemas[0].Description != "test tool" {
		t.Errorf("expected schema description 'test tool', got %q", schemas[0].Description)
	}
}
