package events

import (
	"log"
	"sync"
)

// Payload is the data carried by an event.
type Payload map[string]interface{}

// Handler is a callback that processes an event.
type Handler func(action string, payload Payload)

// ResultHandler is a callback that processes an event and returns a string
// (typically rendered HTML) to be collected by PublishCollect callers.
// Used for sync request/reply patterns (e.g. extensions rendering form HTML
// for templates via {{event "forms:render" ...}}).
type ResultHandler func(action string, payload Payload) string

// EventBus is a thread-safe publish/subscribe event dispatcher.
type EventBus struct {
	mu             sync.RWMutex
	handlers       map[string][]Handler
	resultHandlers map[string][]ResultHandler
	allHandlers    []Handler
}

// New creates and returns a new EventBus.
func New() *EventBus {
	return &EventBus{
		handlers:       make(map[string][]Handler),
		resultHandlers: make(map[string][]ResultHandler),
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

// HasHandlers returns true if there are any registered handlers for the given action.
func (b *EventBus) HasHandlers(action string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[action]) > 0 || len(b.allHandlers) > 0
}

// PublishSync fires an event and waits for all handlers to complete.
// Use for cases where the caller needs to know delivery succeeded (e.g. SendEmail).
func (b *EventBus) PublishSync(action string, payload Payload) {
	b.mu.RLock()
	specific := make([]Handler, len(b.handlers[action]))
	copy(specific, b.handlers[action])
	all := make([]Handler, len(b.allHandlers))
	copy(all, b.allHandlers)
	b.mu.RUnlock()

	var wg sync.WaitGroup
	for _, h := range specific {
		wg.Add(1)
		go func(fn Handler) {
			defer wg.Done()
			safeCall(fn, action, payload)
		}(h)
	}
	for _, h := range all {
		wg.Add(1)
		go func(fn Handler) {
			defer wg.Done()
			safeCall(fn, action, payload)
		}(h)
	}
	wg.Wait()
}

func safeCall(h Handler, action string, payload Payload) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[events] panic in handler for %q: %v", action, r)
		}
	}()
	h(action, payload)
}

// SubscribeResult registers a handler that returns a string result. Used by
// extensions that render content for templates via PublishCollect (e.g. the
// forms extension returning rendered form HTML for {{event "forms:render"}}).
// Result handlers run synchronously, separately from regular Subscribe handlers.
func (b *EventBus) SubscribeResult(action string, handler ResultHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.resultHandlers[action] = append(b.resultHandlers[action], handler)
}

// PublishCollect runs all result handlers for an action synchronously and
// returns their non-empty results in registration order. Regular fire-and-forget
// handlers (Subscribe) are NOT invoked — callers that need both should call
// Publish in addition.
func (b *EventBus) PublishCollect(action string, payload Payload) []string {
	b.mu.RLock()
	handlers := make([]ResultHandler, len(b.resultHandlers[action]))
	copy(handlers, b.resultHandlers[action])
	b.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}
	results := make([]string, 0, len(handlers))
	for _, h := range handlers {
		if r := safeCallResult(h, action, payload); r != "" {
			results = append(results, r)
		}
	}
	return results
}

func safeCallResult(h ResultHandler, action string, payload Payload) (result string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[events] panic in result handler for %q: %v", action, r)
			result = ""
		}
	}()
	return h(action, payload)
}
