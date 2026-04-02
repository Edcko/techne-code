package event

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Edcko/techne-code/pkg/event"
)

func TestChannelEventBus_PublishCallsAllSubscribers(t *testing.T) {
	bus := NewChannelEventBus()
	defer bus.Close()

	var callCount int32
	var wg sync.WaitGroup

	// Subscribe 3 handlers
	for i := 0; i < 3; i++ {
		wg.Add(1)
		bus.Subscribe(func(evt event.Event) {
			atomic.AddInt32(&callCount, 1)
			wg.Done()
		})
	}

	// Publish an event
	evt := event.NewEvent(event.EventMessageDelta, "test-session", event.MessageDeltaData{Text: "hello"})
	bus.Publish(evt)

	// Wait for all handlers to complete
	wg.Wait()

	if atomic.LoadInt32(&callCount) != 3 {
		t.Errorf("expected 3 handler calls, got %d", callCount)
	}
}

func TestChannelEventBus_UnsubscribeStopsReceiving(t *testing.T) {
	bus := NewChannelEventBus()
	defer bus.Close()

	var callCount int32
	var wg sync.WaitGroup

	// Subscribe and then unsubscribe
	wg.Add(1)
	unsubscribe := bus.Subscribe(func(evt event.Event) {
		atomic.AddInt32(&callCount, 1)
		wg.Done()
	})

	// Publish first event
	evt := event.NewEvent(event.EventMessageDelta, "test-session", event.MessageDeltaData{Text: "first"})
	bus.Publish(evt)
	wg.Wait()

	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected 1 handler call before unsubscribe, got %d", callCount)
	}

	// Unsubscribe
	unsubscribe()

	// Publish second event - should not be received
	var wg2 sync.WaitGroup
	wg2.Add(1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		wg2.Done()
	}()

	evt2 := event.NewEvent(event.EventMessageDelta, "test-session", event.MessageDeltaData{Text: "second"})
	bus.Publish(evt2)
	wg2.Wait()

	// callCount should still be 1 (no new calls)
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected handler to not be called after unsubscribe, but callCount is %d", callCount)
	}
}

func TestChannelEventBus_ClosePreventsNewPublishes(t *testing.T) {
	bus := NewChannelEventBus()

	var callCount int32
	var wg sync.WaitGroup

	wg.Add(1)
	bus.Subscribe(func(evt event.Event) {
		atomic.AddInt32(&callCount, 1)
		wg.Done()
	})

	// Publish before close - should work
	evt := event.NewEvent(event.EventMessageDelta, "test-session", event.MessageDeltaData{Text: "before close"})
	bus.Publish(evt)
	wg.Wait()

	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected 1 handler call before close, got %d", callCount)
	}

	// Close the bus
	bus.Close()

	// Publish after close - should not be delivered
	var wg2 sync.WaitGroup
	wg2.Add(1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		wg2.Done()
	}()

	evt2 := event.NewEvent(event.EventMessageDelta, "test-session", event.MessageDeltaData{Text: "after close"})
	bus.Publish(evt2)
	wg2.Wait()

	// callCount should still be 1
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected no handler calls after close, got %d", callCount)
	}
}

func TestChannelEventBus_SubscribeAfterCloseReturnsNoOp(t *testing.T) {
	bus := NewChannelEventBus()
	bus.Close()

	// Subscribe after close should return a no-op unsubscribe
	unsubscribe := bus.Subscribe(func(evt event.Event) {
		t.Error("handler should not be called for closed bus")
	})

	// Should be safe to call
	unsubscribe()
}

func TestChannelEventBus_ConcurrentPublishSubscribe(t *testing.T) {
	bus := NewChannelEventBus()
	defer bus.Close()

	const numOperations = 100
	var callCount int32

	// Subscribe multiple handlers first
	for i := 0; i < 10; i++ {
		bus.Subscribe(func(evt event.Event) {
			atomic.AddInt32(&callCount, 1)
		})
	}

	var wg sync.WaitGroup

	// Concurrently publish
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			evt := event.NewEvent(event.EventMessageDelta, "test-session", event.MessageDeltaData{Text: "concurrent"})
			bus.Publish(evt)
		}()
	}

	wg.Wait()

	// Wait a bit for all goroutines to process
	time.Sleep(50 * time.Millisecond)

	// Should have received many events (10 handlers * up to 100 events each)
	if atomic.LoadInt32(&callCount) == 0 {
		t.Error("expected some handler calls during concurrent operations")
	}
}

func TestChannelEventBus_ConcurrentUnsubscribe(t *testing.T) {
	bus := NewChannelEventBus()
	defer bus.Close()

	var wg sync.WaitGroup
	unsubscribes := make([]func(), 0, 10)

	// Subscribe 10 handlers
	for i := 0; i < 10; i++ {
		unsub := bus.Subscribe(func(evt event.Event) {})
		unsubscribes = append(unsubscribes, unsub)
	}

	// Concurrently unsubscribe all
	for _, unsub := range unsubscribes {
		wg.Add(1)
		go func(u func()) {
			defer wg.Done()
			u()
		}(unsub)
	}

	wg.Wait()

	// Should complete without race condition (verified with -race flag)
}

func TestChannelEventBus_CloseIsIdempotent(t *testing.T) {
	bus := NewChannelEventBus()

	// Close multiple times should not panic
	bus.Close()
	bus.Close()
	bus.Close()
}

func TestChannelEventBus_EventDataIsPassedCorrectly(t *testing.T) {
	bus := NewChannelEventBus()
	defer bus.Close()

	var receivedEvent event.Event
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(func(evt event.Event) {
		receivedEvent = evt
		wg.Done()
	})

	expectedData := event.MessageDeltaData{Text: "test message"}
	evt := event.NewEvent(event.EventMessageDelta, "session-123", expectedData)
	bus.Publish(evt)

	wg.Wait()

	if receivedEvent.Type != event.EventMessageDelta {
		t.Errorf("expected type %s, got %s", event.EventMessageDelta, receivedEvent.Type)
	}
	if receivedEvent.SessionID != "session-123" {
		t.Errorf("expected session ID 'session-123', got %s", receivedEvent.SessionID)
	}

	data, ok := receivedEvent.Data.(event.MessageDeltaData)
	if !ok {
		t.Fatalf("expected MessageDeltaData, got %T", receivedEvent.Data)
	}
	if data.Text != "test message" {
		t.Errorf("expected text 'test message', got %s", data.Text)
	}
}
