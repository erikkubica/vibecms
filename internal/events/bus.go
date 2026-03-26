package events

import (
	"log"
	"sync"
)

// Payload is the data carried by an event.
type Payload map[string]interface{}

// Handler is a callback that processes an event.
type Handler func(action string, payload Payload)

// EventBus is a thread-safe publish/subscribe event dispatcher.
type EventBus struct {
	mu          sync.RWMutex
	handlers    map[string][]Handler
	allHandlers []Handler
}

// New creates and returns a new EventBus.
func New() *EventBus {
	return &EventBus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe registers a handler for a specific action.
func (b *EventBus) Subscribe(action string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[action] = append(b.handlers[action], handler)
}

// SubscribeAll registers a handler that receives ALL events.
func (b *EventBus) SubscribeAll(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.allHandlers = append(b.allHandlers, handler)
}

// Publish fires an event. Handlers run in goroutines (non-blocking).
// Panics in handlers are recovered and logged.
func (b *EventBus) Publish(action string, payload Payload) {
	b.mu.RLock()
	// Copy handler slices under the lock to avoid races.
	specific := make([]Handler, len(b.handlers[action]))
	copy(specific, b.handlers[action])
	all := make([]Handler, len(b.allHandlers))
	copy(all, b.allHandlers)
	b.mu.RUnlock()

	for _, h := range specific {
		go safeCall(h, action, payload)
	}
	for _, h := range all {
		go safeCall(h, action, payload)
	}
}

func safeCall(h Handler, action string, payload Payload) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[events] panic in handler for %q: %v", action, r)
		}
	}()
	h(action, payload)
}
