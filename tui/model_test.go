package tui

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Edcko/techne-code/internal/config"
	eventbus "github.com/Edcko/techne-code/internal/event"
	"github.com/Edcko/techne-code/internal/llm"
	"github.com/Edcko/techne-code/internal/permission"
	"github.com/Edcko/techne-code/internal/tools"
	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/session"
)

type mockProviderForTUI struct{}

func (m *mockProviderForTUI) Name() string { return "mock" }
func (m *mockProviderForTUI) Chat(_ context.Context, _ provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		Content:    []provider.ContentBlock{{Type: provider.BlockText, Text: "hi"}},
		Usage:      provider.Usage{InputTokens: 1, OutputTokens: 1},
		Model:      "mock",
		StopReason: "end_turn",
	}, nil
}
func (m *mockProviderForTUI) Stream(_ context.Context, _ provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	return nil, nil
}
func (m *mockProviderForTUI) Models() []provider.ModelInfo {
	return nil
}

type mockStoreForTUI struct {
	sessions map[string]*session.Session
	messages map[string][]session.StoredMessage
}

func newMockStoreForTUI() *mockStoreForTUI {
	return &mockStoreForTUI{
		sessions: make(map[string]*session.Session),
		messages: make(map[string][]session.StoredMessage),
	}
}

func (s *mockStoreForTUI) CreateSession(sess *session.Session) error {
	if sess.ID == "" {
		sess.ID = "generated-id"
	}
	sess.CreatedAt = time.Now()
	sess.UpdatedAt = time.Now()
	s.sessions[sess.ID] = sess
	return nil
}

func (s *mockStoreForTUI) GetSession(id string) (*session.Session, error) {
	return s.sessions[id], nil
}

func (s *mockStoreForTUI) ListSessions() ([]session.Session, error) {
	var result []session.Session
	for _, sess := range s.sessions {
		result = append(result, *sess)
	}
	return result, nil
}

func (s *mockStoreForTUI) UpdateSessionTitle(id, title string) error {
	if sess, ok := s.sessions[id]; ok {
		sess.Title = title
	}
	return nil
}

func (s *mockStoreForTUI) UpdateSessionSummary(id, summaryMessageID string) error {
	if sess, ok := s.sessions[id]; ok {
		sess.SummaryMessageID = &summaryMessageID
	}
	return nil
}

func (s *mockStoreForTUI) DeleteSession(id string) error {
	delete(s.sessions, id)
	delete(s.messages, id)
	return nil
}

func (s *mockStoreForTUI) SaveMessage(m *session.StoredMessage) error {
	s.messages[m.SessionID] = append(s.messages[m.SessionID], *m)
	return nil
}

func (s *mockStoreForTUI) GetMessages(sessionID string) ([]session.StoredMessage, error) {
	return s.messages[sessionID], nil
}

func (s *mockStoreForTUI) GetMessagesAfter(sessionID string, after time.Time) ([]session.StoredMessage, error) {
	var result []session.StoredMessage
	for _, msg := range s.messages[sessionID] {
		if msg.CreatedAt.After(after) {
			result = append(result, msg)
		}
	}
	return result, nil
}

func (s *mockStoreForTUI) DeleteMessages(sessionID string) error {
	s.messages[sessionID] = nil
	return nil
}

func (s *mockStoreForTUI) TrackReadFile(sessionID, path string) error { return nil }
func (s *mockStoreForTUI) HasReadFile(sessionID, path string) (bool, error) {
	return false, nil
}

func newTestModel(store *mockStoreForTUI, sessionID string) *Model {
	cfg := config.DefaultConfig()
	bus := eventbus.NewChannelEventBus()
	client := llm.NewClient(&mockProviderForTUI{}, bus)
	registry := tools.NewRegistry()
	perm := permission.NewService(permission.ModeAutoAllow, nil)

	return NewModel(cfg, client, store, registry, perm, bus, nil, true, sessionID)
}

func TestNewModel_SetsSessionID(t *testing.T) {
	tests := []struct {
		name            string
		sessionID       string
		expectSessionID string
	}{
		{
			name:            "empty session ID",
			sessionID:       "",
			expectSessionID: "",
		},
		{
			name:            "provided session ID",
			sessionID:       "existing-session-123",
			expectSessionID: "existing-session-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStoreForTUI()
			m := newTestModel(store, tt.sessionID)

			if m.sessionID != tt.expectSessionID {
				t.Errorf("expected sessionID %q, got %q", tt.expectSessionID, m.sessionID)
			}
		})
	}
}

func TestInit_CreatesNewSession(t *testing.T) {
	store := newMockStoreForTUI()
	m := newTestModel(store, "")

	m.Init()

	if m.sessionID == "" {
		t.Error("expected session ID to be set after Init")
	}
	if m.state != StateChatting {
		t.Errorf("expected state StateChatting, got %d", m.state)
	}
	if m.err != nil {
		t.Errorf("unexpected error: %v", m.err)
	}

	sess, _ := store.GetSession(m.sessionID)
	if sess == nil {
		t.Error("expected session to be created in store")
	}
}

func TestInit_ResumesExistingSession(t *testing.T) {
	store := newMockStoreForTUI()

	existingSession := &session.Session{
		ID:        "resume-me",
		Title:     "Test Resume",
		Model:     "test-model",
		Provider:  "test-provider",
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
	store.sessions[existingSession.ID] = existingSession

	msgContent := toJSONForTest([]provider.ContentBlock{
		{Type: provider.BlockText, Text: "Hello from history"},
	})
	store.messages[existingSession.ID] = []session.StoredMessage{
		{
			ID:        "msg-1",
			SessionID: existingSession.ID,
			Role:      "user",
			Content:   msgContent,
			CreatedAt: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:        "msg-2",
			SessionID: existingSession.ID,
			Role:      "assistant",
			Content: toJSONForTest([]provider.ContentBlock{
				{Type: provider.BlockText, Text: "Hi there!"},
			}),
			CreatedAt: time.Now().Add(-29 * time.Minute),
		},
	}

	m := newTestModel(store, "resume-me")
	m.Init()

	if m.sessionID != "resume-me" {
		t.Errorf("expected sessionID 'resume-me', got %q", m.sessionID)
	}
	if m.state != StateChatting {
		t.Errorf("expected state StateChatting, got %d", m.state)
	}
	if m.err != nil {
		t.Errorf("unexpected error: %v", m.err)
	}
	if len(m.messages) != 2 {
		t.Fatalf("expected 2 messages loaded, got %d", len(m.messages))
	}
	if m.messages[0].Content != "Hello from history" {
		t.Errorf("expected first message 'Hello from history', got %q", m.messages[0].Content)
	}
	if m.messages[1].Content != "Hi there!" {
		t.Errorf("expected second message 'Hi there!', got %q", m.messages[1].Content)
	}
}

func TestInit_SessionNotFound(t *testing.T) {
	store := newMockStoreForTUI()

	m := newTestModel(store, "nonexistent")
	m.Init()

	if m.state != StateExiting {
		t.Errorf("expected state StateExiting, got %d", m.state)
	}
	if m.err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestInit_EmptySessionIDCreatesNew(t *testing.T) {
	store := newMockStoreForTUI()

	m := newTestModel(store, "")
	m.Init()

	if m.sessionID == "" {
		t.Error("expected new session to be created")
	}
	if m.state != StateChatting {
		t.Errorf("expected state StateChatting, got %d", m.state)
	}
	if len(m.messages) != 0 {
		t.Errorf("expected 0 messages for new session, got %d", len(m.messages))
	}
}

func TestInit_ResumedSessionHasCorrectRoleOrder(t *testing.T) {
	store := newMockStoreForTUI()

	sess := &session.Session{
		ID:       "order-test",
		Title:    "Order Test",
		Model:    "m",
		Provider: "p",
	}
	store.sessions[sess.ID] = sess

	store.messages[sess.ID] = []session.StoredMessage{
		{
			ID: "m1", SessionID: sess.ID, Role: "user",
			Content:   toJSONForTest([]provider.ContentBlock{{Type: provider.BlockText, Text: "q1"}}),
			CreatedAt: time.Now().Add(-3 * time.Minute),
		},
		{
			ID: "m2", SessionID: sess.ID, Role: "assistant",
			Content:   toJSONForTest([]provider.ContentBlock{{Type: provider.BlockText, Text: "a1"}}),
			CreatedAt: time.Now().Add(-2 * time.Minute),
		},
		{
			ID: "m3", SessionID: sess.ID, Role: "user",
			Content:   toJSONForTest([]provider.ContentBlock{{Type: provider.BlockText, Text: "q2"}}),
			CreatedAt: time.Now().Add(-1 * time.Minute),
		},
		{
			ID: "m4", SessionID: sess.ID, Role: "assistant",
			Content:   toJSONForTest([]provider.ContentBlock{{Type: provider.BlockText, Text: "a2"}}),
			CreatedAt: time.Now(),
		},
	}

	m := newTestModel(store, "order-test")
	m.Init()

	expectedRoles := []string{"user", "assistant", "user", "assistant"}
	expectedContents := []string{"q1", "a1", "q2", "a2"}

	for i, msg := range m.messages {
		if msg.Role != expectedRoles[i] {
			t.Errorf("message %d: expected role %q, got %q", i, expectedRoles[i], msg.Role)
		}
		if msg.Content != expectedContents[i] {
			t.Errorf("message %d: expected content %q, got %q", i, expectedContents[i], msg.Content)
		}
	}
}

func toJSONForTest(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return json.RawMessage(data)
}
