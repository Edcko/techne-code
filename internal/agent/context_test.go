package agent

import (
	"context"
	"testing"
	"time"

	"github.com/Edcko/techne-code/internal/event"
	"github.com/Edcko/techne-code/internal/llm"
	pkgevent "github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/session"
)

type SummarizeMockProvider struct {
	lastRequest *provider.ChatRequest
	response    *provider.ChatResponse
}

func (m *SummarizeMockProvider) Name() string { return "mock" }

func (m *SummarizeMockProvider) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	m.lastRequest = &req
	if m.response != nil {
		return m.response, nil
	}
	return &provider.ChatResponse{
		Content: []provider.ContentBlock{
			{Type: provider.BlockText, Text: "This is a summary of the conversation."},
		},
		Usage:      provider.Usage{InputTokens: 100, OutputTokens: 50},
		Model:      "mock-model",
		StopReason: "end_turn",
	}, nil
}

func (m *SummarizeMockProvider) Stream(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	ch := make(chan provider.StreamChunk)
	go func() {
		defer close(ch)
		ch <- provider.StreamChunk{Type: "text_delta", Text: "response"}
		ch <- provider.StreamChunk{Type: "done"}
	}()
	return ch, nil
}

func (m *SummarizeMockProvider) Models() []provider.ModelInfo {
	return []provider.ModelInfo{
		{ID: "mock-model", ContextWindow: 1000, MaxTokens: 256, SupportsTools: true},
	}
}

func TestSummarizer_Summarize(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	mockProvider := &SummarizeMockProvider{}
	client := llm.NewClient(mockProvider, bus)

	summarizer := NewSummarizer(client, bus)

	messages := []provider.Message{
		{
			Role:    provider.RoleUser,
			Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hello, help me with Go"}},
		},
		{
			Role:    provider.RoleAssistant,
			Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Sure, what do you need help with?"}},
		},
	}

	summary, err := summarizer.Summarize(context.Background(), "session-1", "mock-model", messages)
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	if summary == "" {
		t.Error("summary should not be empty")
	}

	if mockProvider.lastRequest == nil {
		t.Fatal("should have called the provider")
	}

	if mockProvider.lastRequest.System == "" {
		t.Error("should include system prompt for summarization")
	}

	if len(mockProvider.lastRequest.Messages) != 1 {
		t.Error("should send exactly 1 message with the conversation text")
	}
}

func TestSummarizer_Summarize_EmptyMessages(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	mockProvider := &SummarizeMockProvider{}
	client := llm.NewClient(mockProvider, bus)

	summarizer := NewSummarizer(client, bus)

	summary, err := summarizer.Summarize(context.Background(), "session-1", "mock-model", nil)
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	if summary != "" {
		t.Error("empty messages should return empty summary")
	}
}

func TestContextManager_CheckAndCompress_NoCompressionNeeded(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	mockProvider := &SummarizeMockProvider{}
	client := llm.NewClient(mockProvider, bus)

	store := NewMockStore()
	cm := NewContextManager(store, bus, client)

	messages := []provider.Message{
		{
			Role:    provider.RoleUser,
			Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "short message"}},
		},
		{
			Role:    provider.RoleAssistant,
			Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "short response"}},
		},
	}

	result, err := cm.CheckAndCompress(context.Background(), "session-1", "mock-model", messages, "system")
	if err != nil {
		t.Fatalf("CheckAndCompress returned error: %v", err)
	}

	if len(result) != len(messages) {
		t.Error("should return same messages when not approaching limit")
	}
}

func TestContextManager_CheckAndCompress_TriggersCompression(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	mockProvider := &SummarizeMockProvider{}
	client := llm.NewClient(mockProvider, bus)

	store := NewMockStore()
	cm := NewContextManager(store, bus, client)

	var messages []provider.Message
	for i := 0; i < 20; i++ {
		text := ""
		for j := 0; j < 100; j++ {
			text += "This is a long message to consume tokens. "
		}
		role := provider.RoleUser
		if i%2 == 1 {
			role = provider.RoleAssistant
		}
		messages = append(messages, provider.Message{
			Role:    role,
			Content: []provider.ContentBlock{{Type: provider.BlockText, Text: text}},
		})
	}

	result, err := cm.CheckAndCompress(context.Background(), "session-1", "mock-model", messages, "system prompt that is also somewhat long to use more tokens")
	if err != nil {
		t.Fatalf("CheckAndCompress returned error: %v", err)
	}

	if len(result) >= len(messages) {
		t.Errorf("expected compression to reduce messages, got %d vs original %d", len(result), len(messages))
	}

	firstBlock := result[0].Content[0]
	if firstBlock.Type != provider.BlockText {
		t.Error("first message should be a text summary block")
	}

	if mockProvider.lastRequest == nil {
		t.Error("should have called provider for summarization")
	}
}

func TestContextManager_CheckAndCompress_TooFewMessages(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	mockProvider := &SummarizeMockProvider{}
	client := llm.NewClient(mockProvider, bus)

	store := NewMockStore()
	cm := NewContextManager(store, bus, client)

	messages := []provider.Message{
		{
			Role:    provider.RoleUser,
			Content: []provider.ContentBlock{{Type: provider.BlockText, Text: makeLongText(50000)}},
		},
	}

	result, err := cm.CheckAndCompress(context.Background(), "session-1", "mock-model", messages, "system")
	if err != nil {
		t.Fatalf("CheckAndCompress returned error: %v", err)
	}

	if len(result) != len(messages) {
		t.Error("should not compress when too few messages even if over limit")
	}
}

func TestContextManager_TrackUsage(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	mockProvider := &SummarizeMockProvider{}
	client := llm.NewClient(mockProvider, bus)

	store := NewMockStore()
	cm := NewContextManager(store, bus, client)

	var capturedEvents []pkgevent.Event
	bus.Subscribe(func(e pkgevent.Event) {
		capturedEvents = append(capturedEvents, e)
	})

	cm.TrackUsage("session-1", provider.Usage{
		InputTokens:  100,
		OutputTokens: 50,
	})

	usage := cm.GetTokenUsage("session-1")
	if usage.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 50 {
		t.Errorf("expected 50 output tokens, got %d", usage.OutputTokens)
	}

	cm.TrackUsage("session-1", provider.Usage{
		InputTokens:  200,
		OutputTokens: 100,
	})

	usage = cm.GetTokenUsage("session-1")
	if usage.InputTokens != 300 {
		t.Errorf("expected 300 cumulative input tokens, got %d", usage.InputTokens)
	}
	if usage.TotalTokens != 450 {
		t.Errorf("expected 450 total tokens, got %d", usage.TotalTokens)
	}
}

func TestContextManager_TrackUsage_CacheTokens(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	mockProvider := &SummarizeMockProvider{}
	client := llm.NewClient(mockProvider, bus)

	store := NewMockStore()
	cm := NewContextManager(store, bus, client)

	cm.TrackUsage("session-1", provider.Usage{
		InputTokens:     100,
		OutputTokens:    50,
		CacheReadTokens: 30,
	})

	usage := cm.GetTokenUsage("session-1")
	if usage.CachedTokens != 30 {
		t.Errorf("expected 30 cached tokens, got %d", usage.CachedTokens)
	}
}

func TestContextManager_GetTokenUsage_NoSession(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	mockProvider := &SummarizeMockProvider{}
	client := llm.NewClient(mockProvider, bus)

	store := NewMockStore()
	cm := NewContextManager(store, bus, client)

	usage := cm.GetTokenUsage("nonexistent")
	if usage.InputTokens != 0 || usage.OutputTokens != 0 || usage.TotalTokens != 0 {
		t.Error("nonexistent session should return zero usage")
	}
}

func TestContextManager_EstimateCurrentUsage(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	mockProvider := &SummarizeMockProvider{}
	client := llm.NewClient(mockProvider, bus)

	store := NewMockStore()
	cm := NewContextManager(store, bus, client)

	messages := []provider.Message{
		{
			Role:    provider.RoleUser,
			Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hello world"}},
		},
	}

	usage := cm.EstimateCurrentUsage(messages, "system prompt")
	if usage <= 0 {
		t.Error("should return positive token estimate")
	}

	expectedMsg := EstimateMessagesTokens(messages)
	expectedSys := EstimateSystemPromptTokens("system prompt")
	if usage != expectedMsg+expectedSys {
		t.Errorf("expected %d, got %d", expectedMsg+expectedSys, usage)
	}
}

func TestContextManager_PersistSummary(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	mockProvider := &SummarizeMockProvider{}
	client := llm.NewClient(mockProvider, bus)

	store := NewMockStore()
	err := store.CreateSession(&session.Session{
		ID:        "session-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	cm := NewContextManager(store, bus, client)

	cm.persistSummary("session-1", "This is a conversation summary")

	msgs, err := store.GetMessages("session-1")
	if err != nil {
		t.Fatalf("GetMessages returned error: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Role != string(provider.RoleSystem) {
		t.Errorf("expected system role, got %s", msgs[0].Role)
	}

	sess, err := store.GetSession("session-1")
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}

	if sess.SummaryMessageID == nil {
		t.Error("session should have summary message ID set")
	}
}

func TestSummarizer_PublishesEvent(t *testing.T) {
	bus := event.NewChannelEventBus()
	defer bus.Close()

	mockProvider := &SummarizeMockProvider{}
	client := llm.NewClient(mockProvider, bus)

	var capturedEvents []pkgevent.Event
	bus.Subscribe(func(evt pkgevent.Event) {
		capturedEvents = append(capturedEvents, evt)
	})

	summarizer := NewSummarizer(client, bus)

	messages := []provider.Message{
		{
			Role:    provider.RoleUser,
			Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Test message"}},
		},
	}

	_, err := summarizer.Summarize(context.Background(), "session-1", "mock-model", messages)
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	foundSummarize := false
	for _, e := range capturedEvents {
		if e.Type == pkgevent.EventSummarize {
			foundSummarize = true
			data, ok := e.Data.(SummarizeData)
			if !ok {
				t.Error("expected SummarizeData in event")
			}
			if data.OriginalMessages != 1 {
				t.Errorf("expected 1 original message, got %d", data.OriginalMessages)
			}
			if data.SummaryTokens <= 0 {
				t.Error("expected positive summary token count")
			}
		}
	}

	if !foundSummarize {
		t.Error("expected EventSummarize to be published")
	}
}

func TestFormatMessageForSummary(t *testing.T) {
	msg := provider.Message{
		Role: provider.RoleUser,
		Content: []provider.ContentBlock{
			{Type: provider.BlockText, Text: "Hello"},
		},
	}

	result := formatMessageForSummary(msg)
	if result == "" {
		t.Error("should produce non-empty text")
	}
}

func TestTruncateForSummary(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"long", "hello world", 5, "hello... [truncated]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateForSummary(tt.input, tt.maxLen)
			if result != tt.want {
				t.Errorf("truncateForSummary(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.want)
			}
		})
	}
}

func makeLongText(charCount int) string {
	result := make([]byte, charCount)
	for i := range result {
		result[i] = 'x'
	}
	return string(result)
}
