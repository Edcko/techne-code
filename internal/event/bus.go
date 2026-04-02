package event

import (
	"sync"

	"github.com/Edcko/techne-code/pkg/event"
)

type ChannelEventBus struct {
	handlers []handlerEntry
	mu       sync.RWMutex
	closed   bool
	nextID   int
	seq      chan func()
}

type handlerEntry struct {
	id      int
	handler event.EventHandler
}

func NewChannelEventBus() *ChannelEventBus {
	b := &ChannelEventBus{
		handlers: make([]handlerEntry, 0),
		seq:      make(chan func(), 1024),
	}
	go b.processSequential()
	return b
}

func (b *ChannelEventBus) processSequential() {
	for fn := range b.seq {
		fn()
	}
}

func (b *ChannelEventBus) Publish(evt event.Event) {
	b.mu.RLock()
	handlers := make([]handlerEntry, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()

	if b.closed {
		return
	}

	for _, entry := range handlers {
		h := entry.handler
		b.seq <- func() { h(evt) }
	}
}

func (b *ChannelEventBus) Subscribe(handler event.EventHandler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return func() {}
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

func (b *ChannelEventBus) unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, entry := range b.handlers {
		if entry.id == id {
			b.handlers = append(b.handlers[:i], b.handlers[i+1:]...)
			return
		}
	}
}

func (b *ChannelEventBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.closed {
		b.closed = true
		close(b.seq)
	}
	b.handlers = nil
}
