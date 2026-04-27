package events

import (
	"log"
	"sync"
	"sync/atomic"
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

// UnsubscribeFunc removes the corresponding subscription. Calling it more
// than once is safe and a no-op.
type UnsubscribeFunc func()

// handlerEntry pairs a handler with an opaque registration id so the
// returned UnsubscribeFunc can locate the right slot to remove. Without
// this, comparing function values (Go forbids ==/!= on funcs) or their
// stack-slot addresses (the original filter-unsub bug) doesn't work.
type handlerEntry struct {
	id uint64
	fn Handler
}

type resultEntry struct {
	id uint64
	fn ResultHandler
}

// EventBus is a thread-safe publish/subscribe event dispatcher.
type EventBus struct {
	mu             sync.RWMutex
	nextID         atomic.Uint64
	handlers       map[string][]handlerEntry
	resultHandlers map[string][]resultEntry
	allHandlers    []handlerEntry
}

// New creates and returns a new EventBus.
func New() *EventBus {
	return &EventBus{
		handlers:       make(map[string][]handlerEntry),
		resultHandlers: make(map[string][]resultEntry),
	}
}

// Subscribe registers a handler for a specific action. The returned
// UnsubscribeFunc removes the subscription — call it during teardown
// (theme/extension reload, test cleanup) so handler dispatch doesn't
// multiply on every reload.
func (b *EventBus) Subscribe(action string, handler Handler) UnsubscribeFunc {
	id := b.nextID.Add(1)
	b.mu.Lock()
	b.handlers[action] = append(b.handlers[action], handlerEntry{id: id, fn: handler})
	b.mu.Unlock()
	return func() { b.removeAction(action, id) }
}

// SubscribeAll registers a handler that receives ALL events.
func (b *EventBus) SubscribeAll(handler Handler) UnsubscribeFunc {
	id := b.nextID.Add(1)
	b.mu.Lock()
	b.allHandlers = append(b.allHandlers, handlerEntry{id: id, fn: handler})
	b.mu.Unlock()
	return func() { b.removeAll(id) }
}

// SubscribeResult registers a handler that returns a string result. Used by
// extensions that render content for templates via PublishCollect (e.g. the
// forms extension returning rendered form HTML for {{event "forms:render"}}).
// Result handlers run synchronously, separately from regular Subscribe handlers.
func (b *EventBus) SubscribeResult(action string, handler ResultHandler) UnsubscribeFunc {
	id := b.nextID.Add(1)
	b.mu.Lock()
	b.resultHandlers[action] = append(b.resultHandlers[action], resultEntry{id: id, fn: handler})
	b.mu.Unlock()
	return func() { b.removeResult(action, id) }
}

func (b *EventBus) removeAction(action string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	list := b.handlers[action]
	for i, e := range list {
		if e.id == id {
			b.handlers[action] = append(list[:i], list[i+1:]...)
			return
		}
	}
}

func (b *EventBus) removeAll(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, e := range b.allHandlers {
		if e.id == id {
			b.allHandlers = append(b.allHandlers[:i], b.allHandlers[i+1:]...)
			return
		}
	}
}

func (b *EventBus) removeResult(action string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	list := b.resultHandlers[action]
	for i, e := range list {
		if e.id == id {
			b.resultHandlers[action] = append(list[:i], list[i+1:]...)
			return
		}
	}
}

// Publish fires an event. Handlers run in goroutines (non-blocking).
// Panics in handlers are recovered and logged.
func (b *EventBus) Publish(action string, payload Payload) {
	b.mu.RLock()
	specific := make([]handlerEntry, len(b.handlers[action]))
	copy(specific, b.handlers[action])
	all := make([]handlerEntry, len(b.allHandlers))
	copy(all, b.allHandlers)
	b.mu.RUnlock()

	for _, e := range specific {
		go safeCall(e.fn, action, payload)
	}
	for _, e := range all {
		go safeCall(e.fn, action, payload)
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
	specific := make([]handlerEntry, len(b.handlers[action]))
	copy(specific, b.handlers[action])
	all := make([]handlerEntry, len(b.allHandlers))
	copy(all, b.allHandlers)
	b.mu.RUnlock()

	var wg sync.WaitGroup
	for _, e := range specific {
		wg.Add(1)
		go func(fn Handler) {
			defer wg.Done()
			safeCall(fn, action, payload)
		}(e.fn)
	}
	for _, e := range all {
		wg.Add(1)
		go func(fn Handler) {
			defer wg.Done()
			safeCall(fn, action, payload)
		}(e.fn)
	}
	wg.Wait()
}

// PublishCollect runs all result handlers for an action synchronously and
// returns their non-empty results in registration order. Regular fire-and-forget
// handlers (Subscribe) are NOT invoked — callers that need both should call
// Publish in addition.
func (b *EventBus) PublishCollect(action string, payload Payload) []string {
	b.mu.RLock()
	entries := make([]resultEntry, len(b.resultHandlers[action]))
	copy(entries, b.resultHandlers[action])
	b.mu.RUnlock()

	if len(entries) == 0 {
		return nil
	}
	results := make([]string, 0, len(entries))
	for _, e := range entries {
		if r := safeCallResult(e.fn, action, payload); r != "" {
			results = append(results, r)
		}
	}
	return results
}

func safeCall(h Handler, action string, payload Payload) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[events] panic in handler for %q: %v", action, r)
		}
	}()
	h(action, payload)
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
