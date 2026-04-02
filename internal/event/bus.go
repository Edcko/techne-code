// Package event provides the concrete implementation of the EventBus interface.
// It uses Go channels and goroutines for thread-safe event distribution.
package event

import (
	"sync"

	"github.com/Edcko/techne-code/pkg/event"
)

// ChannelEventBus implements event.EventBus using Go channels.
// It provides thread-safe publish/subscribe functionality with goroutine-based
// handler invocation to prevent slow handlers from blocking the publisher.
type ChannelEventBus struct {
	handlers []handlerEntry
	mu       sync.RWMutex
	closed   bool
	nextID   int
}

// handlerEntry stores a handler with its unique ID for unsubscribe operations.
type handlerEntry struct {
	id      int
	handler event.EventHandler
}

// NewChannelEventBus creates a new EventBus instance.
func NewChannelEventBus() *ChannelEventBus {
	return &ChannelEventBus{
		handlers: make([]handlerEntry, 0),
	}
}

// Publish sends an event to all subscribed handlers.
// Each handler is invoked in its own goroutine to prevent blocking.
// If the bus is closed, this is a no-op.
func (b *ChannelEventBus) Publish(evt event.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	// Call each handler in a goroutine for non-blocking delivery
	for _, entry := range b.handlers {
		go entry.handler(evt)
	}
}

// Subscribe registers a handler and returns an unsubscribe function.
// The returned function removes the handler when called.
// If the bus is closed, the handler is not added and a no-op unsubscribe is returned.
func (b *ChannelEventBus) Subscribe(handler event.EventHandler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return func() {} // no-op if already closed
	}

	id := b.nextID
	b.nextID++
	b.handlers = append(b.handlers, handlerEntry{
		id:      id,
		handler: handler,
	})

	return func() {
		b.unsubscribe(id)
	}
}

// unsubscribe removes a handler by ID.
// This is called by the unsubscribe function returned by Subscribe.
func (b *ChannelEventBus) unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Find and remove the handler with this ID
	for i, entry := range b.handlers {
		if entry.id == id {
			b.handlers = append(b.handlers[:i], b.handlers[i+1:]...)
			return
		}
	}
}

// Close shuts down the event bus.
// After closing, Publish becomes a no-op and Subscribe returns a no-op unsubscribe.
// Close is idempotent and safe to call multiple times.
func (b *ChannelEventBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	b.handlers = nil // release handler references
}
