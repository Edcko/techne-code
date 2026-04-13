//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Edcko/techne-code/internal/agent"
	"github.com/Edcko/techne-code/internal/db"
	"github.com/Edcko/techne-code/internal/llm"
	"github.com/Edcko/techne-code/internal/llm/providers/ollama"
	"github.com/Edcko/techne-code/internal/permission"
	"github.com/Edcko/techne-code/internal/tools"
	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/session"
	"github.com/google/uuid"
)

const (
	modelName     = "qwen2.5-coder"
	ollamaBaseURL = "http://localhost:11434"
	testMaxTokens = 2048
	llmTimeout    = 60 * time.Second
)

func ollamaAvailable() bool {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ollamaBaseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func skipWithoutOllama(t *testing.T) {
	t.Helper()
	if !ollamaAvailable() {
		t.Skip("Ollama not available at " + ollamaBaseURL)
	}
}

func waitForOllama(t *testing.T) {
	t.Helper()
	for i := 0; i < 10; i++ {
		if ollamaAvailable() {
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Skip("Ollama not available after waiting")
}

type testEnv struct {
	tmpDir string
	store  session.SessionStore
	bus    *agent.SilentEventBus
	client *llm.Client
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "techne-e2e-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	gormDB, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	store := db.NewSessionStore(gormDB)
	bus := &agent.SilentEventBus{}
	prov := ollama.New(ollamaBaseURL + "/v1")
	client := llm.NewClient(prov, bus)

	return &testEnv{
		tmpDir: tmpDir,
		store:  store,
		bus:    bus,
		client: client,
	}
}

func setupStoreOnly(t *testing.T) session.SessionStore {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "techne-e2e-store-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	gormDB, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	return db.NewSessionStore(gormDB)
}

func (e *testEnv) newSession(t *testing.T) string {
	t.Helper()

	id := uuid.New().String()
	sess := &session.Session{
		ID:       id,
		Title:    "e2e-test",
		Model:    modelName,
		Provider: "ollama",
	}
	if err := e.store.CreateSession(sess); err != nil {
		t.Fatal(err)
	}
	return id
}

func (e *testEnv) newAgent(registry *tools.Registry) *agent.Agent {
	perm := permission.NewService(permission.ModeAutoAllow, nil)
	return agent.New(e.client, e.store, registry, perm, e.bus)
}

func newToolRegistry() *tools.Registry {
	r := tools.NewRegistry()
	r.Register(&tools.ReadFileTool{})
	r.Register(&tools.ListDirTool{})
	r.Register(&tools.WriteFileTool{})
	r.Register(&tools.GlobTool{})
	r.Register(&tools.GrepTool{})
	return r
}

func getAssistantText(t *testing.T, store session.SessionStore, sessionID string) string {
	t.Helper()

	messages, err := store.GetMessages(sessionID)
	if err != nil {
		t.Fatal(err)
	}

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != string(provider.RoleAssistant) {
			continue
		}
		var blocks []provider.ContentBlock
		if err := json.Unmarshal(messages[i].Content, &blocks); err != nil {
			continue
		}
		var parts []string
		for _, b := range blocks {
			if b.Type == provider.BlockText && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return ""
}

func hasToolCall(t *testing.T, store session.SessionStore, sessionID string, toolName string) bool {
	t.Helper()

	messages, err := store.GetMessages(sessionID)
	if err != nil {
		t.Fatal(err)
	}

	for _, msg := range messages {
		if msg.Role != string(provider.RoleAssistant) {
			continue
		}
		var blocks []provider.ContentBlock
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type == provider.BlockToolCall && b.ToolCall != nil {
				if b.ToolCall.Name == toolName {
					return true
				}
			}
		}
	}
	return false
}

func TestSessionPersistence(t *testing.T) {
	store := setupStoreOnly(t)

	sessionID := uuid.New().String()
	sess := &session.Session{
		ID:       sessionID,
		Title:    "e2e-persistence",
		Model:    modelName,
		Provider: "ollama",
	}
	if err := store.CreateSession(sess); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.GetSession(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil {
		t.Fatal("Session should exist")
	}
	if loaded.ID != sessionID {
		t.Fatalf("Expected session ID %s, got %s", sessionID, loaded.ID)
	}

	if err := store.UpdateSessionTitle(sessionID, "E2E Test Session"); err != nil {
		t.Fatal(err)
	}

	loaded, err = store.GetSession(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "E2E Test Session" {
		t.Fatalf("Expected title 'E2E Test Session', got '%s'", loaded.Title)
	}

	msg := &session.StoredMessage{
		SessionID: sessionID,
		Role:      string(provider.RoleUser),
		Content:   json.RawMessage(`[{"type":"text","text":"hello from persistence test"}]`),
	}
	if err := store.SaveMessage(msg); err != nil {
		t.Fatal(err)
	}

	messages, err := store.GetMessages(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	if messages[0].Role != string(provider.RoleUser) {
		t.Fatalf("Expected role 'user', got '%s'", messages[0].Role)
	}

	var storedBlocks []provider.ContentBlock
	if err := json.Unmarshal(messages[0].Content, &storedBlocks); err != nil {
		t.Fatal(err)
	}
	if len(storedBlocks) == 0 || storedBlocks[0].Text != "hello from persistence test" {
		t.Fatalf("Message content not persisted correctly: %v", storedBlocks)
	}

	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range sessions {
		if s.ID == sessionID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Session not found in list")
	}
}

func TestBasicChat(t *testing.T) {
	skipWithoutOllama(t)

	env := setupTestEnv(t)
	sessionID := env.newSession(t)
	ag := env.newAgent(newToolRegistry())

	ctx, cancel := context.WithTimeout(context.Background(), llmTimeout)
	defer cancel()

	err := ag.Run(ctx, sessionID, "What is 2+2? Reply with just the number.", agent.Config{
		Model:         modelName,
		MaxTokens:     testMaxTokens,
		MaxIterations: 5,
		ToolsEnabled:  false,
	})
	if err != nil {
		t.Fatalf("Agent Run failed: %v", err)
	}

	response := getAssistantText(t, env.store, sessionID)
	t.Logf("LLM Response: %s", response)

	if response == "" {
		t.Fatal("Expected non-empty response from LLM")
	}
	if !strings.Contains(response, "4") {
		t.Fatalf("Expected response to contain '4', got: %s", response)
	}
}

func TestToolCalling(t *testing.T) {
	skipWithoutOllama(t)

	env := setupTestEnv(t)
	sessionID := env.newSession(t)

	testFile := filepath.Join(env.tmpDir, "hello.txt")
	if err := os.WriteFile(testFile, []byte("Hello from E2E test!"), 0644); err != nil {
		t.Fatal(err)
	}

	registry := newToolRegistry()
	ag := env.newAgent(registry)

	ctx, cancel := context.WithTimeout(context.Background(), llmTimeout)
	defer cancel()

	prompt := fmt.Sprintf("Please use the read_file tool to read the file at path %s and then tell me what is in it.", testFile)
	err := ag.Run(ctx, sessionID, prompt, agent.Config{
		Model:         modelName,
		MaxTokens:     testMaxTokens,
		MaxIterations: 10,
		ToolsEnabled:  true,
	})
	if err != nil {
		t.Fatalf("Agent Run failed: %v", err)
	}

	if !hasToolCall(t, env.store, sessionID, "read_file") {
		msgs, _ := env.store.GetMessages(sessionID)
		for _, m := range msgs {
			t.Logf("  [%s] %s", m.Role, string(m.Content)[:min(200, len(string(m.Content)))])
		}
		t.Fatal("Expected read_file tool to be called")
	}

	response := getAssistantText(t, env.store, sessionID)
	t.Logf("LLM Response: %s", response)

	if !strings.Contains(response, "Hello from E2E test") {
		t.Fatalf("Expected response to contain file content, got: %s", response)
	}
}

func TestMultiTurn(t *testing.T) {
	waitForOllama(t)

	env := setupTestEnv(t)
	sessionID := env.newSession(t)
	ag := env.newAgent(newToolRegistry())

	ctx, cancel := context.WithTimeout(context.Background(), llmTimeout)
	defer cancel()

	err := ag.Run(ctx, sessionID, "My favorite programming language is Go. Remember this.", agent.Config{
		Model:         modelName,
		MaxTokens:     testMaxTokens,
		MaxIterations: 5,
		ToolsEnabled:  false,
	})
	if err != nil {
		t.Fatalf("First turn failed: %v", err)
	}

	firstResponse := getAssistantText(t, env.store, sessionID)
	t.Logf("First turn response: %s", firstResponse)

	err = ag.Run(ctx, sessionID, "What is my favorite programming language?", agent.Config{
		Model:         modelName,
		MaxTokens:     testMaxTokens,
		MaxIterations: 5,
		ToolsEnabled:  false,
	})
	if err != nil {
		t.Fatalf("Second turn failed: %v", err)
	}

	secondResponse := getAssistantText(t, env.store, sessionID)
	t.Logf("Second turn response: %s", secondResponse)

	if !strings.Contains(strings.ToLower(secondResponse), "go") {
		t.Fatalf("Expected second response to mention 'Go', got: %s", secondResponse)
	}
}

type smallContextProvider struct {
	provider.Provider
}

func (p *smallContextProvider) Models() []provider.ModelInfo {
	return []provider.ModelInfo{
		{
			ID:             modelName,
			MaxTokens:      2048,
			SupportsTools:  true,
			SupportsVision: false,
			ContextWindow:  500,
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func hasSummaryInMessages(t *testing.T, store session.SessionStore, sessionID string) bool {
	t.Helper()

	sess, _ := store.GetSession(sessionID)
	if sess != nil && sess.SummaryMessageID != nil {
		t.Logf("Session has summary message ID: %s", *sess.SummaryMessageID)
		return true
	}

	messages, err := store.GetMessages(sessionID)
	if err != nil {
		return false
	}

	for _, msg := range messages {
		if msg.Role == string(provider.RoleSystem) {
			return true
		}
		var blocks []provider.ContentBlock
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type == provider.BlockText && strings.Contains(b.Text, "[Conversation Summary]") {
				return true
			}
		}
	}
	return false
}

func TestContextManagement(t *testing.T) {
	waitForOllama(t)

	env := setupTestEnv(t)
	sessionID := env.newSession(t)

	smallCtxProvider := &smallContextProvider{Provider: ollama.New(ollamaBaseURL + "/v1")}
	bus := &agent.SilentEventBus{}
	client := llm.NewClient(smallCtxProvider, bus)
	registry := newToolRegistry()
	perm := permission.NewService(permission.ModeAutoAllow, nil)
	ag := agent.New(client, env.store, registry, perm, bus)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	longText := strings.Repeat("This is a test message for context compression. ", 50)

	for i := 0; i < 10; i++ {
		prompt := fmt.Sprintf("Message %d: %s Just acknowledge briefly.", i+1, longText)
		err := ag.Run(ctx, sessionID, prompt, agent.Config{
			Model:         modelName,
			MaxTokens:     512,
			MaxIterations: 3,
			ToolsEnabled:  false,
		})
		if err != nil {
			if hasSummaryInMessages(t, env.store, sessionID) {
				t.Logf("Run %d failed but compression already triggered: %v", i+1, err)
				break
			}
			t.Fatalf("Run %d failed: %v", i+1, err)
		}
		t.Logf("Completed turn %d", i+1)

		if hasSummaryInMessages(t, env.store, sessionID) {
			t.Logf("Context compression detected after turn %d", i+1)
			break
		}
	}

	messages, err := env.store.GetMessages(sessionID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Total messages: %d", len(messages))

	if !hasSummaryInMessages(t, env.store, sessionID) {
		t.Fatal("Expected context compression to trigger")
	}
	t.Log("Context compression triggered successfully")
}
