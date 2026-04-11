package tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Edcko/techne-code/internal/agent"
	"github.com/Edcko/techne-code/pkg/session"
)

type mockStore struct {
	sessions map[string]*session.Session
	messages map[string][]session.StoredMessage
}

func newMockStore() *mockStore {
	return &mockStore{
		sessions: make(map[string]*session.Session),
		messages: make(map[string][]session.StoredMessage),
	}
}

func (m *mockStore) CreateSession(s *session.Session) error {
	m.sessions[s.ID] = s
	return nil
}
func (m *mockStore) GetSession(id string) (*session.Session, error) {
	return m.sessions[id], nil
}
func (m *mockStore) ListSessions() ([]session.Session, error) {
	var result []session.Session
	for _, s := range m.sessions {
		result = append(result, *s)
	}
	return result, nil
}
func (m *mockStore) UpdateSessionTitle(id, title string) error { return nil }
func (m *mockStore) UpdateSessionSummary(id, summaryMessageID string) error {
	return nil
}
func (m *mockStore) DeleteSession(id string) error { return nil }
func (m *mockStore) SaveMessage(msg *session.StoredMessage) error {
	m.messages[msg.SessionID] = append(m.messages[msg.SessionID], *msg)
	return nil
}
func (m *mockStore) GetMessages(sessionID string) ([]session.StoredMessage, error) {
	return m.messages[sessionID], nil
}
func (m *mockStore) GetMessagesAfter(sessionID string, after time.Time) ([]session.StoredMessage, error) {
	return nil, nil
}
func (m *mockStore) DeleteMessages(sessionID string) error { return nil }
func (m *mockStore) TrackReadFile(sessionID, path string) error {
	return nil
}
func (m *mockStore) HasReadFile(sessionID, path string) (bool, error) {
	return false, nil
}

func TestSubAgentTool_Name(t *testing.T) {
	config := agent.SubAgentConfig{
		Name:         "researcher",
		Description:  "test desc",
		SystemPrompt: "test prompt",
	}
	sat := NewSubAgentTool(config, nil, nil, nil)

	if sat.Name() != "researcher" {
		t.Errorf("expected name 'researcher', got %q", sat.Name())
	}
}

func TestSubAgentTool_Description(t *testing.T) {
	config := agent.SubAgentConfig{
		Name:         "researcher",
		Description:  "Researches things",
		SystemPrompt: "test prompt",
	}
	sat := NewSubAgentTool(config, nil, nil, nil)

	if sat.Description() != "Researches things" {
		t.Errorf("expected description 'Researches things', got %q", sat.Description())
	}
}

func TestSubAgentTool_Parameters(t *testing.T) {
	config := agent.SubAgentConfig{
		Name:         "test",
		Description:  "test",
		SystemPrompt: "test",
	}
	sat := NewSubAgentTool(config, nil, nil, nil)

	params := sat.Parameters()
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
	if _, ok := props["task"]; !ok {
		t.Error("expected 'task' property in schema")
	}

	required, ok := schema["required"].([]interface{})
	if !ok {
		t.Fatal("expected required array in schema")
	}
	if len(required) != 1 || required[0] != "task" {
		t.Error("expected 'task' to be required")
	}
}

func TestSubAgentTool_RequiresPermission(t *testing.T) {
	config := agent.SubAgentConfig{
		Name:         "test",
		Description:  "test",
		SystemPrompt: "test",
	}
	sat := NewSubAgentTool(config, nil, nil, nil)

	if sat.RequiresPermission() {
		t.Error("sub-agent tool should not require permission")
	}
}

func TestSubAgentTool_Execute_InvalidJSON(t *testing.T) {
	config := agent.SubAgentConfig{
		Name:         "test",
		Description:  "test",
		SystemPrompt: "test",
	}
	sat := NewSubAgentTool(config, nil, nil, nil)

	result, err := sat.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for invalid JSON")
	}
}

func TestSubAgentTool_Execute_EmptyTask(t *testing.T) {
	config := agent.SubAgentConfig{
		Name:         "test",
		Description:  "test",
		SystemPrompt: "test",
	}
	sat := NewSubAgentTool(config, nil, nil, nil)

	input, _ := json.Marshal(map[string]string{"task": ""})
	result, err := sat.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for empty task")
	}
}

func TestSubAgentTool_Execute_MissingTaskField(t *testing.T) {
	config := agent.SubAgentConfig{
		Name:         "test",
		Description:  "test",
		SystemPrompt: "test",
	}
	sat := NewSubAgentTool(config, nil, nil, nil)

	input, _ := json.Marshal(map[string]string{"other": "value"})
	result, err := sat.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing task")
	}
}

func TestNewResearcherConfig(t *testing.T) {
	cfg := NewResearcherConfig("claude-3")

	if cfg.Name != "researcher" {
		t.Errorf("expected name 'researcher', got %q", cfg.Name)
	}
	if cfg.Model != "claude-3" {
		t.Errorf("expected model 'claude-3', got %q", cfg.Model)
	}
	if cfg.MaxIterations != 15 {
		t.Errorf("expected 15 max iterations, got %d", cfg.MaxIterations)
	}
	allowedSet := map[string]bool{}
	for _, name := range cfg.AllowedTools {
		allowedSet[name] = true
	}
	if !allowedSet["read_file"] || !allowedSet["grep"] || !allowedSet["glob"] || !allowedSet["web_fetch"] {
		t.Errorf("unexpected allowed tools: %v", cfg.AllowedTools)
	}
	if allowedSet["write_file"] || allowedSet["bash"] {
		t.Error("researcher should not have write_file or bash")
	}
}

func TestNewCoderConfig(t *testing.T) {
	cfg := NewCoderConfig("gpt-4")

	if cfg.Name != "coder" {
		t.Errorf("expected name 'coder', got %q", cfg.Name)
	}
	if cfg.MaxIterations != 20 {
		t.Errorf("expected 20 max iterations, got %d", cfg.MaxIterations)
	}
	allowedSet := map[string]bool{}
	for _, name := range cfg.AllowedTools {
		allowedSet[name] = true
	}
	if !allowedSet["read_file"] || !allowedSet["write_file"] || !allowedSet["edit_file"] || !allowedSet["bash"] {
		t.Errorf("unexpected allowed tools: %v", cfg.AllowedTools)
	}
}

func TestNewReviewerConfig(t *testing.T) {
	cfg := NewReviewerConfig("claude-3")

	if cfg.Name != "reviewer" {
		t.Errorf("expected name 'reviewer', got %q", cfg.Name)
	}
	if cfg.MaxIterations != 10 {
		t.Errorf("expected 10 max iterations, got %d", cfg.MaxIterations)
	}
	for _, name := range cfg.AllowedTools {
		if name == "write_file" || name == "bash" || name == "edit_file" {
			t.Errorf("reviewer should not have %q", name)
		}
	}
}

func TestNewTesterConfig(t *testing.T) {
	cfg := NewTesterConfig("gpt-4")

	if cfg.Name != "tester" {
		t.Errorf("expected name 'tester', got %q", cfg.Name)
	}
	if cfg.MaxIterations != 20 {
		t.Errorf("expected 20 max iterations, got %d", cfg.MaxIterations)
	}
	allowedSet := map[string]bool{}
	for _, name := range cfg.AllowedTools {
		allowedSet[name] = true
	}
	if !allowedSet["read_file"] || !allowedSet["write_file"] || !allowedSet["bash"] {
		t.Errorf("unexpected allowed tools: %v", cfg.AllowedTools)
	}
}

func TestSubAgentTool_Execute_NilProvider(t *testing.T) {
	config := agent.SubAgentConfig{
		Name:          "test",
		Description:   "test",
		SystemPrompt:  "test",
		AllowedTools:  []string{},
		MaxIterations: 1,
		Model:         "test",
	}

	registry := NewRegistry()
	sat := NewSubAgentTool(config, nil, newMockStore(), registry)

	input, _ := json.Marshal(map[string]string{"task": "do something"})
	result, err := sat.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result with nil provider")
	}
}
