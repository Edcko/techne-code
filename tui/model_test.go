package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Edcko/techne-code/internal/config"
	eventbus "github.com/Edcko/techne-code/internal/event"
	"github.com/Edcko/techne-code/internal/llm"
	"github.com/Edcko/techne-code/internal/permission"
	"github.com/Edcko/techne-code/internal/tools"
	"github.com/Edcko/techne-code/pkg/event"
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

func initTestModel() *Model {
	store := newMockStoreForTUI()
	m := newTestModel(store, "")
	m.Init()
	return m
}

func TestInputBuffer_InsertChar(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("h")
	buf.InsertChar("i")

	if buf.Text() != "hi" {
		t.Errorf("expected 'hi', got %q", buf.Text())
	}
	line, col := buf.CursorPos()
	if line != 0 || col != 2 {
		t.Errorf("expected cursor at (0,2), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_InsertNewline(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertChar("b")
	buf.InsertNewline()
	buf.InsertChar("c")

	if buf.Text() != "ab\nc" {
		t.Errorf("expected 'ab\\nc', got %q", buf.Text())
	}
	if buf.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", buf.LineCount())
	}
	line, col := buf.CursorPos()
	if line != 1 || col != 1 {
		t.Errorf("expected cursor at (1,1), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_Backspace(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertChar("b")
	buf.Backspace()

	if buf.Text() != "a" {
		t.Errorf("expected 'a', got %q", buf.Text())
	}
	line, col := buf.CursorPos()
	if line != 0 || col != 1 {
		t.Errorf("expected cursor at (0,1), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_BackspaceJoinsLines(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertNewline()
	buf.InsertChar("b")
	buf.Backspace()
	buf.Backspace()

	if buf.Text() != "a" {
		t.Errorf("expected 'a', got %q", buf.Text())
	}
	if buf.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", buf.LineCount())
	}
	line, col := buf.CursorPos()
	if line != 0 || col != 1 {
		t.Errorf("expected cursor at (0,1), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_Delete(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertChar("b")
	buf.MoveLeft()
	buf.Delete()

	if buf.Text() != "a" {
		t.Errorf("expected 'a', got %q", buf.Text())
	}
}

func TestInputBuffer_DeleteJoinsLines(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertNewline()
	buf.InsertChar("b")
	buf.MoveLeft()
	buf.MoveLeft()
	buf.Delete()

	if buf.Text() != "ab" {
		t.Errorf("expected 'ab', got %q", buf.Text())
	}
	if buf.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", buf.LineCount())
	}
}

func TestInputBuffer_MoveLeft(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertChar("b")
	buf.MoveLeft()

	line, col := buf.CursorPos()
	if line != 0 || col != 1 {
		t.Errorf("expected cursor at (0,1), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_MoveLeftWrapsToPrevLine(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertNewline()
	buf.MoveLeft()

	line, col := buf.CursorPos()
	if line != 0 || col != 1 {
		t.Errorf("expected cursor at (0,1), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_MoveRight(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertChar("b")
	buf.MoveLeft()
	buf.MoveRight()

	line, col := buf.CursorPos()
	if line != 0 || col != 2 {
		t.Errorf("expected cursor at (0,2), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_MoveRightWrapsToNextLine(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertNewline()
	buf.InsertChar("b")
	buf.MoveLeft()
	buf.MoveLeft()
	buf.MoveRight()

	line, col := buf.CursorPos()
	if line != 1 || col != 0 {
		t.Errorf("expected cursor at (1,0), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_MoveUp(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertChar("b")
	buf.InsertNewline()
	buf.InsertChar("c")
	buf.InsertChar("d")
	moved := buf.MoveUp()

	if !moved {
		t.Error("expected MoveUp to return true")
	}
	line, col := buf.CursorPos()
	if line != 0 || col != 2 {
		t.Errorf("expected cursor at (0,2), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_MoveUpAtTop(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	moved := buf.MoveUp()

	if moved {
		t.Error("expected MoveUp at top to return false")
	}
}

func TestInputBuffer_MoveDown(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertNewline()
	buf.InsertChar("b")
	buf.MoveUp()
	moved := buf.MoveDown()

	if !moved {
		t.Error("expected MoveDown to return true")
	}
	line, col := buf.CursorPos()
	if line != 1 || col != 1 {
		t.Errorf("expected cursor at (1,1), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_MoveDownAtBottom(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	moved := buf.MoveDown()

	if moved {
		t.Error("expected MoveDown at bottom to return false")
	}
}

func TestInputBuffer_MoveHome(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertChar("b")
	buf.MoveHome()

	line, col := buf.CursorPos()
	if line != 0 || col != 0 {
		t.Errorf("expected cursor at (0,0), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_MoveEnd(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertChar("b")
	buf.MoveHome()
	buf.MoveEnd()

	line, col := buf.CursorPos()
	if line != 0 || col != 2 {
		t.Errorf("expected cursor at (0,2), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_MoveUpClampsCol(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertNewline()
	buf.InsertChar("b")
	buf.InsertChar("c")
	buf.InsertChar("d")
	buf.MoveUp()

	line, col := buf.CursorPos()
	if line != 0 || col != 1 {
		t.Errorf("expected cursor at (0,1), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_MoveUpClampsColWhenShorter(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertNewline()
	buf.InsertChar("b")
	buf.InsertChar("c")
	buf.InsertChar("d")
	buf.MoveUp()

	line, col := buf.CursorPos()
	if line != 0 || col != 1 {
		t.Errorf("expected cursor at (0,1), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_InsertPaste(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertPaste("hello\nworld")

	if buf.Text() != "hello\nworld" {
		t.Errorf("expected 'hello\\nworld', got %q", buf.Text())
	}
	if buf.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", buf.LineCount())
	}
	line, col := buf.CursorPos()
	if line != 1 || col != 5 {
		t.Errorf("expected cursor at (1,5), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_InsertPasteIntoExisting(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertChar("b")
	buf.InsertPaste("x\ny")

	if buf.Text() != "abx\ny" {
		t.Errorf("expected 'abx\\ny', got %q", buf.Text())
	}
}

func TestInputBuffer_InsertPasteEmpty(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertPaste("")

	if buf.Text() != "a" {
		t.Errorf("expected 'a', got %q", buf.Text())
	}
}

func TestInputBuffer_SetText(t *testing.T) {
	buf := newInputBuffer()
	buf.SetText("line1\nline2\nline3")

	if buf.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", buf.LineCount())
	}
	line, col := buf.CursorPos()
	if line != 2 || col != 5 {
		t.Errorf("expected cursor at (2,5), got (%d,%d)", line, col)
	}
}

func TestInputBuffer_Clear(t *testing.T) {
	buf := newInputBuffer()
	buf.InsertChar("a")
	buf.InsertNewline()
	buf.InsertChar("b")
	buf.Clear()

	if !buf.IsEmpty() {
		t.Error("expected buffer to be empty after Clear")
	}
	if buf.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", buf.LineCount())
	}
}

func TestInputBuffer_CursorIsAtTop(t *testing.T) {
	buf := newInputBuffer()
	if !buf.CursorIsAtTop() {
		t.Error("expected CursorIsAtTop for empty buffer")
	}
	buf.InsertChar("a")
	if buf.CursorIsAtTop() {
		t.Error("expected not CursorIsAtTop after typing")
	}
	buf.MoveHome()
	if !buf.CursorIsAtTop() {
		t.Error("expected CursorIsAtTop after MoveHome on first line")
	}
}

func TestInputBuffer_CursorIsAtBottom(t *testing.T) {
	buf := newInputBuffer()
	if !buf.CursorIsAtBottom() {
		t.Error("expected CursorIsAtBottom for empty buffer")
	}
	buf.InsertNewline()
	buf.InsertChar("a")
	if !buf.CursorIsAtBottom() {
		t.Error("expected CursorIsAtBottom after typing on last line")
	}
}

func TestInputBuffer_VisibleLines(t *testing.T) {
	buf := newInputBuffer()
	for i := 0; i < 10; i++ {
		buf.InsertChar(string('a' + rune(i)))
		if i < 9 {
			buf.InsertNewline()
		}
	}

	visible := buf.VisibleLines(3)
	if len(visible) != 3 {
		t.Errorf("expected 3 visible lines, got %d", len(visible))
	}
}

func TestHandleKey_EnterAddsNewline(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.inputBuf.Text() != "\n" {
		t.Errorf("expected newline after Enter, got %q", m.inputBuf.Text())
	}
}

func TestHandleKey_CtrlEnterSendsMessage(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Text: "h"})
	m.Update(tea.KeyPressMsg{Text: "i"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})

	if m.state != StateStreaming {
		t.Errorf("expected StateStreaming after ctrl+enter, got %d", m.state)
	}
	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	if m.messages[0].Content != "hi" {
		t.Errorf("expected message 'hi', got %q", m.messages[0].Content)
	}
	if !m.inputBuf.IsEmpty() {
		t.Error("expected input to be cleared after submit")
	}
}

func TestHandleKey_AltEnterSendsMessage(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Text: "h"})
	m.Update(tea.KeyPressMsg{Text: "i"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt})

	if m.state != StateStreaming {
		t.Errorf("expected StateStreaming after alt+enter, got %d", m.state)
	}
}

func TestHandleKey_CtrlSSendsMessage(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Text: "h"})
	m.Update(tea.KeyPressMsg{Text: "i"})
	m.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})

	if m.state != StateStreaming {
		t.Errorf("expected StateStreaming after ctrl+s, got %d", m.state)
	}
}

func TestHandleKey_EnterDoesNotSendEmpty(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.state != StateChatting {
		t.Errorf("expected StateChatting, got %d", m.state)
	}
	if len(m.messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(m.messages))
	}
}

func TestHandleKey_CursorMovement(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Text: "a"})
	m.Update(tea.KeyPressMsg{Text: "b"})
	m.Update(tea.KeyPressMsg{Text: "c"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})

	line, col := m.inputBuf.CursorPos()
	if line != 0 || col != 2 {
		t.Errorf("expected cursor at (0,2) after left, got (%d,%d)", line, col)
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	line, col = m.inputBuf.CursorPos()
	if line != 0 || col != 3 {
		t.Errorf("expected cursor at (0,3) after right, got (%d,%d)", line, col)
	}
}

func TestHandleKey_HomeEnd(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Text: "a"})
	m.Update(tea.KeyPressMsg{Text: "b"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyHome})

	line, col := m.inputBuf.CursorPos()
	if line != 0 || col != 0 {
		t.Errorf("expected cursor at (0,0) after home, got (%d,%d)", line, col)
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	line, col = m.inputBuf.CursorPos()
	if line != 0 || col != 2 {
		t.Errorf("expected cursor at (0,2) after end, got (%d,%d)", line, col)
	}
}

func TestHandleKey_CtrlAMovesToStart(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Text: "a"})
	m.Update(tea.KeyPressMsg{Text: "b"})
	m.Update(tea.KeyPressMsg{Text: "c"})
	m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})

	line, col := m.inputBuf.CursorPos()
	if line != 0 || col != 0 {
		t.Errorf("expected cursor at (0,0) after ctrl+a, got (%d,%d)", line, col)
	}
}

func TestHandleKey_Backspace(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Text: "a"})
	m.Update(tea.KeyPressMsg{Text: "b"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})

	if m.inputBuf.Text() != "a" {
		t.Errorf("expected 'a' after backspace, got %q", m.inputBuf.Text())
	}
}

func TestHandleKey_BackspaceJoinsLines(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Text: "a"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Update(tea.KeyPressMsg{Text: "b"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})

	if m.inputBuf.Text() != "a" {
		t.Errorf("expected 'a' after backspace joins lines, got %q", m.inputBuf.Text())
	}
	if m.inputBuf.LineCount() != 1 {
		t.Errorf("expected 1 line, got %d", m.inputBuf.LineCount())
	}
}

func TestHandleKey_Delete(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Text: "a"})
	m.Update(tea.KeyPressMsg{Text: "b"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m.Update(tea.KeyPressMsg{Code: tea.KeyDelete})

	if m.inputBuf.Text() != "a" {
		t.Errorf("expected 'a' after delete, got %q", m.inputBuf.Text())
	}
}

func TestHandleKey_InputHistoryNavigation(t *testing.T) {
	m := initTestModel()

	for _, ch := range "first" {
		m.Update(tea.KeyPressMsg{Text: string(ch)})
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	m.state = StateChatting

	for _, ch := range "second" {
		m.Update(tea.KeyPressMsg{Text: string(ch)})
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	m.state = StateChatting

	if len(m.inputHistory) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(m.inputHistory))
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})

	if m.inputBuf.Text() != "second" {
		t.Errorf("expected 'second' after up, got %q", m.inputBuf.Text())
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})

	if m.inputBuf.Text() != "first" {
		t.Errorf("expected 'first' after second up, got %q", m.inputBuf.Text())
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	if m.inputBuf.Text() != "second" {
		t.Errorf("expected 'second' after down, got %q", m.inputBuf.Text())
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	if m.inputBuf.Text() != "" {
		t.Errorf("expected empty after down past history, got %q", m.inputBuf.Text())
	}
}

func TestHandleKey_HistoryPreservesDraft(t *testing.T) {
	m := initTestModel()

	for _, ch := range "sent" {
		m.Update(tea.KeyPressMsg{Text: string(ch)})
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	m.state = StateChatting

	for _, ch := range "draft" {
		m.Update(tea.KeyPressMsg{Text: string(ch)})
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})

	if m.inputBuf.Text() != "sent" {
		t.Errorf("expected 'sent' after up, got %q", m.inputBuf.Text())
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	if m.inputBuf.Text() != "draft" {
		t.Errorf("expected 'draft' restored after down, got %q", m.inputBuf.Text())
	}
}

func TestHandleKey_HistoryLimitedTo100(t *testing.T) {
	m := initTestModel()

	for i := 0; i < 110; i++ {
		m.inputBuf.SetText(fmt.Sprintf("msg-%d", i))
		m.handleSubmit()
		m.state = StateChatting
	}

	if len(m.inputHistory) > 100 {
		t.Errorf("expected history limited to 100, got %d", len(m.inputHistory))
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})

	if m.inputBuf.Text() != "msg-109" {
		t.Errorf("expected most recent 'msg-109', got %q", m.inputBuf.Text())
	}
}

func TestHandleKey_CtrlCClearsInput(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Text: "h"})
	m.Update(tea.KeyPressMsg{Text: "i"})
	m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	if m.state != StateChatting {
		t.Errorf("expected StateChatting after ctrl+c with input, got %d", m.state)
	}
	if !m.inputBuf.IsEmpty() {
		t.Error("expected input to be cleared after ctrl+c")
	}
}

func TestHandleKey_CtrlCQuitsWhenEmpty(t *testing.T) {
	m := initTestModel()

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	if m.state != StateExiting {
		t.Errorf("expected StateExiting after ctrl+c with empty input, got %d", m.state)
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestHandleKey_CtrlCCancelsStreaming(t *testing.T) {
	m := initTestModel()
	m.state = StateStreaming

	m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	if m.state != StateChatting {
		t.Errorf("expected StateChatting after cancel, got %d", m.state)
	}
}

func TestHandleKey_PasteHandling(t *testing.T) {
	m := initTestModel()

	m.Update(tea.PasteMsg{Content: "line1\nline2\nline3"})

	if m.inputBuf.Text() != "line1\nline2\nline3" {
		t.Errorf("expected multiline paste, got %q", m.inputBuf.Text())
	}
	if m.inputBuf.LineCount() != 3 {
		t.Errorf("expected 3 lines, got %d", m.inputBuf.LineCount())
	}
}

func TestHandleKey_PasteIgnoredDuringStreaming(t *testing.T) {
	m := initTestModel()
	m.state = StateStreaming

	m.Update(tea.PasteMsg{Content: "should be ignored"})

	if !m.inputBuf.IsEmpty() {
		t.Error("expected paste to be ignored during streaming")
	}
}

func TestHandleKey_SpaceKey(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Code: tea.KeySpace})

	if m.inputBuf.Text() != " " {
		t.Errorf("expected space, got %q", m.inputBuf.Text())
	}
}

func TestHandleKey_MultilineTyping(t *testing.T) {
	m := initTestModel()

	m.Update(tea.KeyPressMsg{Text: "h"})
	m.Update(tea.KeyPressMsg{Text: "e"})
	m.Update(tea.KeyPressMsg{Text: "l"})
	m.Update(tea.KeyPressMsg{Text: "l"})
	m.Update(tea.KeyPressMsg{Text: "o"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Update(tea.KeyPressMsg{Text: "w"})
	m.Update(tea.KeyPressMsg{Text: "o"})
	m.Update(tea.KeyPressMsg{Text: "r"})
	m.Update(tea.KeyPressMsg{Text: "l"})
	m.Update(tea.KeyPressMsg{Text: "d"})

	expected := "hello\nworld"
	if m.inputBuf.Text() != expected {
		t.Errorf("expected %q, got %q", expected, m.inputBuf.Text())
	}
	if m.inputBuf.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", m.inputBuf.LineCount())
	}

	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})

	if m.state != StateStreaming {
		t.Errorf("expected StateStreaming, got %d", m.state)
	}
	if m.messages[0].Content != expected {
		t.Errorf("expected message %q, got %q", expected, m.messages[0].Content)
	}
}

func TestStatusBar_RendersModelName(t *testing.T) {
	m := initTestModel()
	m.width = 80
	m.height = 24

	view := m.View()
	rendered := view.Content

	if !contains(rendered, m.cfg.DefaultProvider) {
		t.Errorf("expected status bar to contain provider %q, got: %s", m.cfg.DefaultProvider, rendered)
	}
	if !contains(rendered, m.cfg.DefaultModel) {
		t.Errorf("expected status bar to contain model %q, got: %s", m.cfg.DefaultModel, rendered)
	}
}

func TestStatusBar_RendersSessionID(t *testing.T) {
	m := initTestModel()
	m.width = 80
	m.height = 24

	view := m.View()
	rendered := view.Content

	if m.sessionID == "" {
		t.Fatal("expected session ID to be set after Init")
	}

	expectedShort := m.sessionID
	if len(m.sessionID) >= 8 {
		expectedShort = m.sessionID[:8]
	}

	if !contains(rendered, expectedShort) {
		t.Errorf("expected status bar to contain session short ID %q, got: %s", expectedShort, rendered)
	}
}

func TestStatusBar_UpdatesOnTokenEvent(t *testing.T) {
	m := initTestModel()
	m.width = 80
	m.height = 24

	view := m.View()
	rendered := view.Content

	if !contains(rendered, "tokens: --") {
		t.Errorf("expected status bar to show 'tokens: --' before token event, got: %s", rendered)
	}

	m.Update(tokenUsageMsg{
		data: event.TokenUsageData{
			InputTokens:           100,
			OutputTokens:          50,
			TotalTokens:           150,
			EstimatedContextUsage: 1000,
			ContextWindow:         128000,
		},
	})

	view = m.View()
	rendered = view.Content

	if !contains(rendered, "tokens: 150") {
		t.Errorf("expected status bar to show 'tokens: 150' after token event, got: %s", rendered)
	}
}

func TestStatusBar_ShowsContextUsage(t *testing.T) {
	m := initTestModel()
	m.width = 80
	m.height = 24

	m.Update(tokenUsageMsg{
		data: event.TokenUsageData{
			TotalTokens:           64000,
			EstimatedContextUsage: 64000,
			ContextWindow:         128000,
		},
	})

	view := m.View()
	rendered := view.Content

	if !contains(rendered, "ctx: 50%") {
		t.Errorf("expected status bar to show 'ctx: 50%%', got: %s", rendered)
	}
}

func TestStatusBar_ShowsNoContextWhenZero(t *testing.T) {
	m := initTestModel()
	m.width = 80
	m.height = 24

	view := m.View()
	rendered := view.Content

	if !contains(rendered, "ctx: --") {
		t.Errorf("expected status bar to show 'ctx: --' when no token data, got: %s", rendered)
	}
}

func TestStatusBar_StreamingIndicator(t *testing.T) {
	m := initTestModel()
	m.width = 80
	m.height = 24

	view := m.View()
	rendered := view.Content

	if contains(rendered, "●") {
		t.Errorf("expected no streaming indicator in chatting state, got: %s", rendered)
	}

	m.state = StateStreaming

	view = m.View()
	rendered = view.Content

	if !contains(rendered, "●") {
		t.Errorf("expected streaming indicator '●' in streaming state, got: %s", rendered)
	}

	if !contains(rendered, "ctrl+c:cancel") {
		t.Errorf("expected 'ctrl+c:cancel' help text during streaming, got: %s", rendered)
	}
}

func TestStatusBar_SessionIDShort(t *testing.T) {
	store := newMockStoreForTUI()
	sess := &session.Session{
		ID:       "abcdefgh-1234-5678-abcd-1234567890ab",
		Title:    "Long ID",
		Model:    "test",
		Provider: "test",
	}
	store.sessions[sess.ID] = sess

	m := newTestModel(store, sess.ID)
	m.Init()
	m.width = 80
	m.height = 24

	view := m.View()
	rendered := view.Content

	if !contains(rendered, "abcdefgh") {
		t.Errorf("expected status bar to show first 8 chars of session ID, got: %s", rendered)
	}
}

func TestStatusBar_SessionIDShortWhenLessThan8(t *testing.T) {
	store := newMockStoreForTUI()
	sess := &session.Session{
		ID:       "abc",
		Title:    "Short ID",
		Model:    "test",
		Provider: "test",
	}
	store.sessions[sess.ID] = sess

	m := newTestModel(store, sess.ID)
	m.Init()
	m.width = 80
	m.height = 24

	view := m.View()
	rendered := view.Content

	if !contains(rendered, "abc") {
		t.Errorf("expected status bar to show full short session ID, got: %s", rendered)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
