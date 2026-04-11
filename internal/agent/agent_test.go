package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/Edcko/techne-code/internal/event"
	"github.com/Edcko/techne-code/internal/llm"
	"github.com/Edcko/techne-code/internal/permission"
	pkgevent "github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/session"
	"github.com/Edcko/techne-code/pkg/tool"
)

type testRegistry struct {
	tools map[string]tool.Tool
}

func newTestRegistry() *testRegistry {
	return &testRegistry{tools: make(map[string]tool.Tool)}
}

func (r *testRegistry) Register(t tool.Tool) error {
	r.tools[t.Name()] = t
	return nil
}

func (r *testRegistry) Get(name string) (tool.Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *testRegistry) List() []tool.Tool {
	result := make([]tool.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

func (r *testRegistry) Schemas() []provider.ToolDef {
	result := make([]provider.ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, provider.ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return result
}

type MockProvider struct {
	LastRequest *provider.ChatRequest
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	m.LastRequest = &req
	return &provider.ChatResponse{
		Content: []provider.ContentBlock{
			{Type: provider.BlockText, Text: "test response"},
		},
		Usage: provider.Usage{
			InputTokens:  10,
			OutputTokens: 5,
		},
		Model:      "mock-model",
		StopReason: "end_turn",
	}, nil
}

func (m *MockProvider) Stream(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	m.LastRequest = &req
	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)
		ch <- provider.StreamChunk{
			Type: "text_delta",
			Text: "test response",
		}
		ch <- provider.StreamChunk{
			Type: "done",
		}
	}()
	return ch, nil
}

func (m *MockProvider) Models() []provider.ModelInfo {
	return []provider.ModelInfo{
		{
			ID:             "mock-model",
			MaxTokens:      4096,
			SupportsTools:  true,
			SupportsVision: false,
			ContextWindow:  8192,
		},
	}
}

func TestToolCallHash(t *testing.T) {
	hash1 := toolCallHash("read_file", json.RawMessage(`{"path":"/tmp/a.go"}`))
	hash2 := toolCallHash("read_file", json.RawMessage(`{"path":"/tmp/a.go"}`))
	hash3 := toolCallHash("read_file", json.RawMessage(`{"path":"/tmp/b.go"}`))

	if hash1 != hash2 {
		t.Error("same input should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("different input should produce different hash")
	}
}

func TestDetectLoop_NoLoop(t *testing.T) {
	hashes := []string{"a", "b", "c", "d", "e"}
	if detectLoop(hashes, 5, 10) {
		t.Error("should not detect loop with diverse hashes")
	}
}

func TestDetectLoop_Loop(t *testing.T) {
	// Same hash repeated 6 times
	hashes := []string{"a", "a", "a", "a", "a", "a"}
	if !detectLoop(hashes, 5, 10) {
		t.Error("should detect loop with 6 identical hashes (maxCount=5)")
	}
}

func TestDetectLoop_BelowThreshold(t *testing.T) {
	// Same hash 5 times (exactly at threshold, should NOT trigger)
	hashes := []string{"a", "a", "a", "a", "a"}
	if detectLoop(hashes, 5, 10) {
		t.Error("should not detect loop when count equals maxCount")
	}
}

func TestDetectLoop_ShortHistory(t *testing.T) {
	hashes := []string{"a", "a"}
	if detectLoop(hashes, 5, 10) {
		t.Error("should not detect loop with short history")
	}
}

func TestDetectLoop_WindowSize(t *testing.T) {
	// 10 unique hashes, then 6 identical — should detect in window
	hashes := []string{
		"b", "b", "b", "b", "b", "b", "b", "b", "b", "b",
		"a", "a", "a", "a", "a", "a",
	}
	if !detectLoop(hashes, 5, 10) {
		t.Error("should detect loop within window")
	}
}

func TestToJSON(t *testing.T) {
	result := toJSON(map[string]string{"key": "value"})
	if string(result) != `{"key":"value"}` {
		t.Errorf("unexpected JSON: %s", result)
	}
}

func TestToJSON_Nil(t *testing.T) {
	result := toJSON(nil)
	if string(result) != "null" {
		t.Errorf("unexpected JSON for nil: %s", result)
	}
}

type MockStore struct {
	messages map[string][]session.StoredMessage
	sessions map[string]*session.Session
	readFile map[string]map[string]bool
}

func NewMockStore() *MockStore {
	return &MockStore{
		messages: make(map[string][]session.StoredMessage),
		sessions: make(map[string]*session.Session),
		readFile: make(map[string]map[string]bool),
	}
}

func (m *MockStore) CreateSession(s *session.Session) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *MockStore) GetSession(id string) (*session.Session, error) {
	return m.sessions[id], nil
}

func (m *MockStore) ListSessions() ([]session.Session, error) {
	var result []session.Session
	for _, s := range m.sessions {
		result = append(result, *s)
	}
	return result, nil
}

func (m *MockStore) UpdateSessionTitle(id, title string) error {
	if s, ok := m.sessions[id]; ok {
		s.Title = title
	}
	return nil
}

func (m *MockStore) UpdateSessionSummary(id, summaryMessageID string) error {
	if s, ok := m.sessions[id]; ok {
		s.SummaryMessageID = &summaryMessageID
	}
	return nil
}

func (m *MockStore) DeleteSession(id string) error {
	delete(m.sessions, id)
	delete(m.messages, id)
	return nil
}

func (m *MockStore) SaveMessage(msg *session.StoredMessage) error {
	m.messages[msg.SessionID] = append(m.messages[msg.SessionID], *msg)
	return nil
}

func (m *MockStore) GetMessages(sessionID string) ([]session.StoredMessage, error) {
	return m.messages[sessionID], nil
}

func (m *MockStore) GetMessagesAfter(sessionID string, after time.Time) ([]session.StoredMessage, error) {
	var result []session.StoredMessage
	for _, msg := range m.messages[sessionID] {
		if msg.CreatedAt.After(after) {
			result = append(result, msg)
		}
	}
	return result, nil
}

func (m *MockStore) DeleteMessages(sessionID string) error {
	m.messages[sessionID] = nil
	return nil
}

func (m *MockStore) TrackReadFile(sessionID, path string) error {
	if m.readFile[sessionID] == nil {
		m.readFile[sessionID] = make(map[string]bool)
	}
	m.readFile[sessionID][path] = true
	return nil
}

func (m *MockStore) HasReadFile(sessionID, path string) (bool, error) {
	if m.readFile[sessionID] == nil {
		return false, nil
	}
	return m.readFile[sessionID][path], nil
}

type MockTool struct {
	name         string
	description  string
	parameters   json.RawMessage
	requiresPerm bool
}

func (m *MockTool) Name() string                { return m.name }
func (m *MockTool) Description() string         { return m.description }
func (m *MockTool) Parameters() json.RawMessage { return m.parameters }
func (m *MockTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	return tool.ToolResult{Content: "mock result"}, nil
}
func (m *MockTool) RequiresPermission() bool { return m.requiresPerm }

type MockProviderWithToolCalls struct {
	LastRequest *provider.ChatRequest
	responses   []provider.ChatResponse
	callIndex   int
	mu          sync.Mutex
}

func (m *MockProviderWithToolCalls) Name() string { return "mock_tc" }

func (m *MockProviderWithToolCalls) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastRequest = &req
	if m.callIndex < len(m.responses) {
		resp := m.responses[m.callIndex]
		m.callIndex++
		return &resp, nil
	}
	return &provider.ChatResponse{
		Content:    []provider.ContentBlock{{Type: provider.BlockText, Text: "done"}},
		StopReason: "end_turn",
	}, nil
}

func (m *MockProviderWithToolCalls) Stream(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	m.mu.Lock()
	m.LastRequest = &req
	m.mu.Unlock()

	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)
		ch <- provider.StreamChunk{Type: "text_delta", Text: "executing tool"}
		ch <- provider.StreamChunk{Type: "done"}
	}()
	return ch, nil
}

func (m *MockProviderWithToolCalls) Models() []provider.ModelInfo {
	return []provider.ModelInfo{{ID: "mock-model", MaxTokens: 4096, SupportsTools: true}}
}

type MockProviderWithPermToolCalls struct {
	LastRequest *provider.ChatRequest
	toolName    string
	toolInput   string
}

func (m *MockProviderWithPermToolCalls) Name() string { return "mock_perm_tc" }

func (m *MockProviderWithPermToolCalls) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	m.LastRequest = &req
	return &provider.ChatResponse{
		Content:    []provider.ContentBlock{{Type: provider.BlockText, Text: "done"}},
		StopReason: "end_turn",
	}, nil
}

func (m *MockProviderWithPermToolCalls) Stream(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	m.LastRequest = &req
	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)
		ch <- provider.StreamChunk{
			Type: "tool_call_delta",
			ToolCall: &provider.ToolCallDelta{
				ID:   "tc-1",
				Name: m.toolName,
			},
		}
		ch <- provider.StreamChunk{
			Type: "tool_call_delta",
			ToolCall: &provider.ToolCallDelta{
				ID:        "tc-1",
				InputJSON: m.toolInput,
				Done:      true,
			},
		}
		ch <- provider.StreamChunk{Type: "done"}
	}()
	return ch, nil
}

func (m *MockProviderWithPermToolCalls) Models() []provider.ModelInfo {
	return []provider.ModelInfo{{ID: "mock-model", MaxTokens: 4096, SupportsTools: true}}
}

type PermissionTool struct {
	name        string
	description string
	needsPerm   bool
}

func (p *PermissionTool) Name() string                { return p.name }
func (p *PermissionTool) Description() string         { return p.description }
func (p *PermissionTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (p *PermissionTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	return tool.ToolResult{Content: "tool executed: " + p.name}, nil
}
func (p *PermissionTool) RequiresPermission() bool { return p.needsPerm }

func TestAgentPermission_PermitsWhenGranted(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	var receivedRequest *pkgevent.PermissionRequestData
	bus.Subscribe(func(e pkgevent.Event) {
		if e.Type == pkgevent.EventPermissionReq {
			data, ok := e.Data.(pkgevent.PermissionRequestData)
			if ok {
				receivedRequest = &data
				data.Response <- pkgevent.PermissionResponseData{Allowed: true, Remember: true}
			}
		}
	})

	mockStore := NewMockStore()
	sessionID := "perm-test-1"
	mockStore.CreateSession(&session.Session{ID: sessionID, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	registry := newTestRegistry()
	dangerousTool := &PermissionTool{name: "bash", description: "Execute shell commands", needsPerm: true}
	safeTool := &PermissionTool{name: "read_file", description: "Read file contents", needsPerm: false}
	registry.Register(dangerousTool)
	registry.Register(safeTool)

	perm := permission.NewService(permission.ModeInteractive, nil)
	mockProvider := &MockProviderWithPermToolCalls{toolName: "bash", toolInput: `{"command":"ls"}`}
	client := llm.NewClient(mockProvider, bus)
	ag := New(client, mockStore, registry, perm, bus)

	config := Config{
		Model:         "mock-model",
		MaxTokens:     100,
		SystemPrompt:  "test",
		MaxIterations: 2,
		ToolsEnabled:  true,
	}

	ctx := context.Background()
	_ = ag.Run(ctx, sessionID, "test", config)

	if receivedRequest == nil {
		t.Fatal("expected permission request event to be published")
	}
	if receivedRequest.ToolName != "bash" {
		t.Errorf("expected tool name 'bash', got %q", receivedRequest.ToolName)
	}

	if !perm.IsAllowed(sessionID, "bash", "execute") {
		t.Error("expected bash to be granted after 'always allow'")
	}
}

func TestAgentPermission_DeniedSkipsTool(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	bus.Subscribe(func(e pkgevent.Event) {
		if e.Type == pkgevent.EventPermissionReq {
			data, ok := e.Data.(pkgevent.PermissionRequestData)
			if ok {
				data.Response <- pkgevent.PermissionResponseData{Allowed: false}
			}
		}
	})

	mockStore := NewMockStore()
	sessionID := "perm-deny-1"
	mockStore.CreateSession(&session.Session{ID: sessionID, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	registry := newTestRegistry()
	dangerousTool := &PermissionTool{name: "bash", description: "Execute shell commands", needsPerm: true}
	registry.Register(dangerousTool)

	perm := permission.NewService(permission.ModeInteractive, nil)
	mockProvider := &MockProviderWithPermToolCalls{toolName: "bash", toolInput: `{"command":"ls"}`}
	client := llm.NewClient(mockProvider, bus)
	ag := New(client, mockStore, registry, perm, bus)

	config := Config{
		Model:         "mock-model",
		MaxTokens:     100,
		SystemPrompt:  "test",
		MaxIterations: 2,
		ToolsEnabled:  true,
	}

	ctx := context.Background()
	_ = ag.Run(ctx, sessionID, "test", config)

	if perm.IsAllowed(sessionID, "bash", "execute") {
		t.Error("expected bash to NOT be granted after denial")
	}
}

func TestAgentPermission_AutoAllowSkipsDialog(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	var permissionRequested bool
	bus.Subscribe(func(e pkgevent.Event) {
		if e.Type == pkgevent.EventPermissionReq {
			permissionRequested = true
		}
	})

	mockStore := NewMockStore()
	sessionID := "auto-allow-1"
	mockStore.CreateSession(&session.Session{ID: sessionID, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	registry := newTestRegistry()
	dangerousTool := &PermissionTool{name: "bash", description: "Execute shell commands", needsPerm: true}
	registry.Register(dangerousTool)

	perm := permission.NewService(permission.ModeAutoAllow, nil)
	mockProvider := &MockProviderWithPermToolCalls{toolName: "bash", toolInput: `{"command":"ls"}`}
	client := llm.NewClient(mockProvider, bus)
	ag := New(client, mockStore, registry, perm, bus)

	config := Config{
		Model:         "mock-model",
		MaxTokens:     100,
		SystemPrompt:  "test",
		MaxIterations: 2,
		ToolsEnabled:  true,
	}

	ctx := context.Background()
	_ = ag.Run(ctx, sessionID, "test", config)

	if permissionRequested {
		t.Error("expected no permission request in auto_allow mode")
	}
}

func TestAgentPermission_SafeToolSkipsDialog(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	var permissionRequested bool
	bus.Subscribe(func(e pkgevent.Event) {
		if e.Type == pkgevent.EventPermissionReq {
			permissionRequested = true
		}
	})

	mockStore := NewMockStore()
	sessionID := "safe-tool-1"
	mockStore.CreateSession(&session.Session{ID: sessionID, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	registry := newTestRegistry()
	safeTool := &PermissionTool{name: "read_file", description: "Read file contents", needsPerm: false}
	registry.Register(safeTool)

	perm := permission.NewService(permission.ModeInteractive, nil)
	mockProvider := &MockProviderWithPermToolCalls{toolName: "read_file", toolInput: `{"path":"/tmp/test.go"}`}
	client := llm.NewClient(mockProvider, bus)
	ag := New(client, mockStore, registry, perm, bus)

	config := Config{
		Model:         "mock-model",
		MaxTokens:     100,
		SystemPrompt:  "test",
		MaxIterations: 2,
		ToolsEnabled:  true,
	}

	ctx := context.Background()
	_ = ag.Run(ctx, sessionID, "test", config)

	if permissionRequested {
		t.Error("expected no permission request for safe tool")
	}
}

func TestAgentToolInjection(t *testing.T) {
	testCases := []struct {
		name             string
		toolsEnabled     bool
		registerTools    bool
		expectTools      bool
		expectToolsCount int
	}{
		{
			name:             "ToolsEnabled_true_with_tools_in_registry",
			toolsEnabled:     true,
			registerTools:    true,
			expectTools:      true,
			expectToolsCount: 2,
		},
		{
			name:             "ToolsEnabled_false_no_tools_injected",
			toolsEnabled:     false,
			registerTools:    true,
			expectTools:      false,
			expectToolsCount: 0,
		},
		{
			name:             "ToolsEnabled_true_empty_registry",
			toolsEnabled:     true,
			registerTools:    false,
			expectTools:      false,
			expectToolsCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockProvider := &MockProvider{}
			mockStore := NewMockStore()
			bus := event.NewChannelEventBus()
			defer bus.Close()

			registry := newTestRegistry()
			if tc.registerTools {
				tool1 := &MockTool{
					name:        "read_file",
					description: "Read a file",
					parameters:  json.RawMessage(`{"type":"object"}`),
				}
				tool2 := &MockTool{
					name:        "write_file",
					description: "Write a file",
					parameters:  json.RawMessage(`{"type":"object"}`),
				}
				if err := registry.Register(tool1); err != nil {
					t.Fatalf("failed to register tool1: %v", err)
				}
				if err := registry.Register(tool2); err != nil {
					t.Fatalf("failed to register tool2: %v", err)
				}
			}

			client := llm.NewClient(mockProvider, bus)
			ag := New(client, mockStore, registry, nil, bus)

			sessionID := "test-session"
			err := mockStore.CreateSession(&session.Session{
				ID:        sessionID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})
			if err != nil {
				t.Fatalf("failed to create session: %v", err)
			}

			config := Config{
				Model:         "mock-model",
				MaxTokens:     100,
				SystemPrompt:  "test",
				MaxIterations: 1,
				ToolsEnabled:  tc.toolsEnabled,
			}

			ctx := context.Background()
			err = ag.Run(ctx, sessionID, "test prompt", config)
			if err != nil {
				t.Logf("agent run returned: %v (expected for mock)", err)
			}

			if mockProvider.LastRequest == nil {
				t.Fatal("LastRequest is nil - agent did not call provider")
			}

			toolsCount := len(mockProvider.LastRequest.Tools)
			if tc.expectTools {
				if toolsCount != tc.expectToolsCount {
					t.Errorf("expected %d tools, got %d", tc.expectToolsCount, toolsCount)
				}
				if toolsCount > 0 {
					foundNames := make(map[string]bool)
					for _, td := range mockProvider.LastRequest.Tools {
						foundNames[td.Name] = true
					}
					if !foundNames["read_file"] || !foundNames["write_file"] {
						t.Errorf("expected tools read_file and write_file, got: %+v", mockProvider.LastRequest.Tools)
					}
				}
			} else {
				if toolsCount != 0 {
					t.Errorf("expected no tools (0), got %d tools", toolsCount)
				}
			}
		})
	}
}
