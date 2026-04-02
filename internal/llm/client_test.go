package llm

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Edcko/techne-code/internal/event"
	pkgEvent "github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/provider"
)

// MockProvider implements provider.Provider for testing.
type MockProvider struct {
	chatFn   func(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error)
	streamFn func(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error)
}

func (m *MockProvider) Name() string { return "mock" }

func (m *MockProvider) Models() []provider.ModelInfo { return nil }

func (m *MockProvider) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	if m.chatFn != nil {
		return m.chatFn(ctx, req)
	}
	return &provider.ChatResponse{}, nil
}

func (m *MockProvider) Stream(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	if m.streamFn != nil {
		return m.streamFn(ctx, req)
	}
	// Default: return empty, immediately closed channel
	ch := make(chan provider.StreamChunk)
	close(ch)
	return ch, nil
}

// newTestClient creates a client with a mock provider and event bus.
func newTestClient(mockProvider *MockProvider) (*Client, *event.ChannelEventBus) {
	bus := event.NewChannelEventBus()
	client := NewClient(mockProvider, bus)
	return client, bus
}

// eventCollector collects events with proper synchronization.
type eventCollector struct {
	mu      sync.Mutex
	events  []pkgEvent.Event
	done    chan struct{}
	handler pkgEvent.EventHandler
}

func newEventCollector(expectedCount int) *eventCollector {
	ec := &eventCollector{
		events: make([]pkgEvent.Event, 0),
		done:   make(chan struct{}),
	}
	var wg sync.WaitGroup
	wg.Add(expectedCount)
	ec.handler = func(evt pkgEvent.Event) {
		ec.mu.Lock()
		ec.events = append(ec.events, evt)
		ec.mu.Unlock()
		wg.Done()
	}
	go func() {
		wg.Wait()
		close(ec.done)
	}()
	return ec
}

func (ec *eventCollector) Handler() pkgEvent.EventHandler {
	return ec.handler
}

func (ec *eventCollector) Events() []pkgEvent.Event {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	result := make([]pkgEvent.Event, len(ec.events))
	copy(result, ec.events)
	return result
}

func (ec *eventCollector) Wait(timeout time.Duration) bool {
	select {
	case <-ec.done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// TestClient_ChatReturnsProviderResponse verifies Chat returns the provider's response.
func TestClient_ChatReturnsProviderResponse(t *testing.T) {
	expectedResponse := &provider.ChatResponse{
		Content: []provider.ContentBlock{
			{Type: provider.BlockText, Text: "Hello, world!"},
		},
		Model:      "test-model",
		StopReason: "end_turn",
		Usage:      provider.Usage{InputTokens: 10, OutputTokens: 5},
	}

	mockProvider := &MockProvider{
		chatFn: func(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
			return expectedResponse, nil
		},
	}

	client, bus := newTestClient(mockProvider)
	defer bus.Close()

	ctx := context.Background()
	req := provider.ChatRequest{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.ContentBlock{{Type: provider.BlockText, Text: "Hello"}}},
		},
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Model != expectedResponse.Model {
		t.Errorf("expected model %s, got %s", expectedResponse.Model, resp.Model)
	}
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(resp.Content))
	}
	if resp.Content[0].Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %s", resp.Content[0].Text)
	}
}

// TestClient_ChatReturnsProviderError verifies Chat propagates provider errors.
func TestClient_ChatReturnsProviderError(t *testing.T) {
	expectedErr := errors.New("provider error")
	mockProvider := &MockProvider{
		chatFn: func(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
			return nil, expectedErr
		},
	}

	client, bus := newTestClient(mockProvider)
	defer bus.Close()

	ctx := context.Background()
	req := provider.ChatRequest{}

	_, err := client.Chat(ctx, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "provider error" {
		t.Errorf("expected error 'provider error', got %s", err.Error())
	}
}

// TestClient_StreamPublishesTextDeltaEvents verifies text delta events are published.
func TestClient_StreamPublishesTextDeltaEvents(t *testing.T) {
	mockProvider := &MockProvider{
		streamFn: func(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
			ch := make(chan provider.StreamChunk, 10)
			ch <- provider.StreamChunk{Type: "text_delta", Text: "Hello"}
			ch <- provider.StreamChunk{Type: "text_delta", Text: " world"}
			ch <- provider.StreamChunk{Type: "text_delta", Text: "!"}
			ch <- provider.StreamChunk{Type: "done"}
			close(ch)
			return ch, nil
		},
	}

	client, bus := newTestClient(mockProvider)
	defer bus.Close()

	// 3 text deltas + 1 done = 4 events
	collector := newEventCollector(4)
	unsubscribe := bus.Subscribe(collector.Handler())
	defer unsubscribe()

	ctx := context.Background()
	req := provider.ChatRequest{}

	resp, err := client.Stream(ctx, "test-session", req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait for all events to be processed
	if !collector.Wait(2 * time.Second) {
		t.Fatal("timeout waiting for events")
	}

	events := collector.Events()

	// Verify events
	messageDeltas := 0
	doneEvents := 0
	for _, evt := range events {
		switch evt.Type {
		case pkgEvent.EventMessageDelta:
			messageDeltas++
		case pkgEvent.EventDone:
			doneEvents++
		}
	}

	if messageDeltas != 3 {
		t.Errorf("expected 3 message delta events, got %d", messageDeltas)
	}
	if doneEvents != 1 {
		t.Errorf("expected 1 done event, got %d", doneEvents)
	}

	// Verify accumulated text
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(resp.Content))
	}
	if resp.Content[0].Text != "Hello world!" {
		t.Errorf("expected accumulated text 'Hello world!', got %s", resp.Content[0].Text)
	}
}

// TestClient_StreamPublishesDoneEvent verifies done event is always published.
func TestClient_StreamPublishesDoneEvent(t *testing.T) {
	mockProvider := &MockProvider{
		streamFn: func(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
			ch := make(chan provider.StreamChunk, 10)
			ch <- provider.StreamChunk{Type: "text_delta", Text: "test"}
			ch <- provider.StreamChunk{Type: "done"}
			close(ch)
			return ch, nil
		},
	}

	client, bus := newTestClient(mockProvider)
	defer bus.Close()

	// 1 text delta + 1 done = 2 events
	collector := newEventCollector(2)
	unsubscribe := bus.Subscribe(collector.Handler())
	defer unsubscribe()

	ctx := context.Background()
	req := provider.ChatRequest{}

	_, err := client.Stream(ctx, "test-session", req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !collector.Wait(2 * time.Second) {
		t.Fatal("timeout waiting for events")
	}

	events := collector.Events()
	foundDone := false
	for _, evt := range events {
		if evt.Type == pkgEvent.EventDone {
			foundDone = true
			break
		}
	}

	if !foundDone {
		t.Error("expected done event to be published")
	}
}

// TestClient_StreamAccumulatesToolCalls verifies tool call delta accumulation.
func TestClient_StreamAccumulatesToolCalls(t *testing.T) {
	mockProvider := &MockProvider{
		streamFn: func(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
			ch := make(chan provider.StreamChunk, 10)
			ch <- provider.StreamChunk{
				Type: "tool_call_delta",
				ToolCall: &provider.ToolCallDelta{
					ID:        "call-1",
					Name:      "read_file",
					InputJSON: `{"path": "`,
				},
			}
			ch <- provider.StreamChunk{
				Type: "tool_call_delta",
				ToolCall: &provider.ToolCallDelta{
					ID:        "call-1",
					InputJSON: `test.txt"}`,
					Done:      true,
				},
			}
			ch <- provider.StreamChunk{Type: "done"}
			close(ch)
			return ch, nil
		},
	}

	client, bus := newTestClient(mockProvider)
	defer bus.Close()

	ctx := context.Background()
	req := provider.ChatRequest{}

	resp, err := client.Stream(ctx, "test-session", req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify tool call was accumulated
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(resp.Content))
	}

	if resp.Content[0].Type != provider.BlockToolCall {
		t.Errorf("expected tool_call block, got %s", resp.Content[0].Type)
	}

	tc := resp.Content[0].ToolCall
	if tc == nil {
		t.Fatal("expected tool call data")
	}
	if tc.ID != "call-1" {
		t.Errorf("expected tool call ID 'call-1', got %s", tc.ID)
	}
	if tc.Name != "read_file" {
		t.Errorf("expected tool name 'read_file', got %s", tc.Name)
	}

	var input map[string]string
	if err := json.Unmarshal(tc.Input, &input); err != nil {
		t.Fatalf("failed to parse tool input: %v", err)
	}
	if input["path"] != "test.txt" {
		t.Errorf("expected path 'test.txt', got %s", input["path"])
	}
}

// TestClient_StreamPublishesErrorOnProviderError verifies error event on Stream() error.
func TestClient_StreamPublishesErrorOnProviderError(t *testing.T) {
	streamErr := errors.New("stream failed")
	mockProvider := &MockProvider{
		streamFn: func(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
			return nil, streamErr
		},
	}

	client, bus := newTestClient(mockProvider)
	defer bus.Close()

	// error + done = 2 events
	collector := newEventCollector(2)
	unsubscribe := bus.Subscribe(collector.Handler())
	defer unsubscribe()

	ctx := context.Background()
	req := provider.ChatRequest{}

	_, err := client.Stream(ctx, "test-session", req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !collector.Wait(2 * time.Second) {
		t.Fatal("timeout waiting for events")
	}

	events := collector.Events()
	var errorEvent *pkgEvent.ErrorData
	for _, evt := range events {
		if evt.Type == pkgEvent.EventError {
			if data, ok := evt.Data.(pkgEvent.ErrorData); ok {
				errorEvent = &data
				break
			}
		}
	}

	if errorEvent == nil {
		t.Fatal("expected error event to be published")
	}
	if errorEvent.Message != "stream failed" {
		t.Errorf("expected error message 'stream failed', got %s", errorEvent.Message)
	}
	if !errorEvent.Fatal {
		t.Error("expected fatal flag to be true")
	}
}

// TestClient_StreamHandlesContextCancellation verifies context cancellation handling.
func TestClient_StreamHandlesContextCancellation(t *testing.T) {
	// Use a channel to control when chunks are sent
	chunkSent := make(chan struct{})
	mockProvider := &MockProvider{
		streamFn: func(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
			ch := make(chan provider.StreamChunk, 10)
			go func() {
				defer close(ch)
				// Send first chunk
				ch <- provider.StreamChunk{Type: "text_delta", Text: "before"}
				close(chunkSent)
				// Wait for context cancellation or timeout
				select {
				case <-ctx.Done():
					ch <- provider.StreamChunk{Type: "error", Error: ctx.Err()}
				case <-time.After(2 * time.Second):
					// Timeout - shouldn't happen in this test
				}
			}()
			return ch, nil
		},
	}

	client, bus := newTestClient(mockProvider)
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after first chunk is sent
	go func() {
		<-chunkSent
		cancel()
	}()

	req := provider.ChatRequest{}

	// Stream should complete without hanging
	done := make(chan struct{})
	go func() {
		_, _ = client.Stream(ctx, "test-session", req)
		close(done)
	}()

	select {
	case <-done:
		// Test passed - Stream returned
	case <-time.After(3 * time.Second):
		t.Fatal("Stream hung - context cancellation not handled")
	}
}

// TestClient_StreamPublishesThinkingDelta verifies thinking delta events.
func TestClient_StreamPublishesThinkingDelta(t *testing.T) {
	mockProvider := &MockProvider{
		streamFn: func(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
			ch := make(chan provider.StreamChunk, 10)
			ch <- provider.StreamChunk{Type: "thinking_delta", Thinking: "Let me think..."}
			ch <- provider.StreamChunk{Type: "thinking_delta", Thinking: " about this."}
			ch <- provider.StreamChunk{Type: "done"}
			close(ch)
			return ch, nil
		},
	}

	client, bus := newTestClient(mockProvider)
	defer bus.Close()

	// 2 thinking + 1 done = 3 events
	collector := newEventCollector(3)
	unsubscribe := bus.Subscribe(collector.Handler())
	defer unsubscribe()

	ctx := context.Background()
	req := provider.ChatRequest{}

	resp, err := client.Stream(ctx, "test-session", req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !collector.Wait(2 * time.Second) {
		t.Fatal("timeout waiting for events")
	}

	events := collector.Events()
	thinkingCount := 0
	for _, evt := range events {
		if evt.Type == pkgEvent.EventMessageDelta {
			thinkingCount++
		}
	}

	if thinkingCount != 2 {
		t.Errorf("expected 2 thinking events, got %d", thinkingCount)
	}

	// Verify accumulated thinking
	if len(resp.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(resp.Content))
	}
	if resp.Content[0].Type != provider.BlockThinking {
		t.Errorf("expected thinking block, got %s", resp.Content[0].Type)
	}
	if resp.Content[0].Thinking == nil || resp.Content[0].Thinking.Text != "Let me think... about this." {
		t.Errorf("expected accumulated thinking text, got %v", resp.Content[0].Thinking)
	}
}

// TestClient_ProviderReturnsUnderlyingProvider verifies Provider() method.
func TestClient_ProviderReturnsUnderlyingProvider(t *testing.T) {
	mockProvider := &MockProvider{}
	client, _ := newTestClient(mockProvider)

	p := client.Provider()
	if p.Name() != "mock" {
		t.Errorf("expected provider name 'mock', got %s", p.Name())
	}
}

// TestClient_StreamHandlesUsageChunks verifies usage chunk handling.
func TestClient_StreamHandlesUsageChunks(t *testing.T) {
	mockProvider := &MockProvider{
		streamFn: func(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
			ch := make(chan provider.StreamChunk, 10)
			ch <- provider.StreamChunk{Type: "text_delta", Text: "Hello"}
			ch <- provider.StreamChunk{
				Type: "usage",
				Usage: &provider.Usage{
					InputTokens:  100,
					OutputTokens: 50,
				},
			}
			ch <- provider.StreamChunk{Type: "done"}
			close(ch)
			return ch, nil
		},
	}

	client, bus := newTestClient(mockProvider)
	defer bus.Close()

	ctx := context.Background()
	req := provider.ChatRequest{}

	resp, err := client.Stream(ctx, "test-session", req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Usage.InputTokens != 100 {
		t.Errorf("expected input tokens 100, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 50 {
		t.Errorf("expected output tokens 50, got %d", resp.Usage.OutputTokens)
	}
}

// TestClient_StreamPublishesErrorOnChunkError verifies error event on chunk error.
func TestClient_StreamPublishesErrorOnChunkError(t *testing.T) {
	mockProvider := &MockProvider{
		streamFn: func(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
			ch := make(chan provider.StreamChunk, 10)
			ch <- provider.StreamChunk{Type: "text_delta", Text: "before error"}
			ch <- provider.StreamChunk{Type: "error", Error: errors.New("chunk error")}
			ch <- provider.StreamChunk{Type: "text_delta", Text: "after error"}
			ch <- provider.StreamChunk{Type: "done"}
			close(ch)
			return ch, nil
		},
	}

	client, bus := newTestClient(mockProvider)
	defer bus.Close()

	// 2 text + 1 error + 1 done = 4 events
	collector := newEventCollector(4)
	unsubscribe := bus.Subscribe(collector.Handler())
	defer unsubscribe()

	ctx := context.Background()
	req := provider.ChatRequest{}

	_, err := client.Stream(ctx, "test-session", req)
	if err != nil {
		t.Fatalf("expected no error from Stream (error in chunk), got %v", err)
	}

	if !collector.Wait(2 * time.Second) {
		t.Fatal("timeout waiting for events")
	}

	events := collector.Events()
	foundError := false
	for _, evt := range events {
		if evt.Type == pkgEvent.EventError {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("expected error event to be published for chunk error")
	}
}
