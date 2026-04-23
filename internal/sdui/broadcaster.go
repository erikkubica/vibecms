package sdui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"vibecms/internal/events"

	"github.com/gofiber/fiber/v2"
)

// Broadcaster manages Server-Sent Events connections and pushes events
// to connected admin clients. It wires into the event bus to automatically
// forward relevant state changes.
type Broadcaster struct {
	mu      sync.RWMutex
	clients map[chan SSEEvent]bool
}

// NewBroadcaster creates a new Broadcaster and subscribes to relevant
// event bus actions to forward state changes as SSE events.
func NewBroadcaster(eventBus *events.EventBus) *Broadcaster {
	b := &Broadcaster{
		clients: make(map[chan SSEEvent]bool),
	}

	// UI_STALE: extension lifecycle changes invalidate cached layouts.
	eventBus.Subscribe("extension.activated", b.handleEvent("UI_STALE"))
	eventBus.Subscribe("extension.deactivated", b.handleEvent("UI_STALE"))

	// NODE_TYPE_CHANGED: node type CRUD changes navigation and list layouts.
	eventBus.Subscribe("node_type.created", b.handleEvent("NODE_TYPE_CHANGED"))
	eventBus.Subscribe("node_type.updated", b.handleEvent("NODE_TYPE_CHANGED"))
	eventBus.Subscribe("node_type.deleted", b.handleEvent("NODE_TYPE_CHANGED"))

	// Forward notification-type events to connected clients.
	eventBus.SubscribeAll(b.handleAllEvents)

	return b
}

// handleEvent returns an events.Handler that wraps the payload into an
// SSEEvent with the given event type.
func (b *Broadcaster) handleEvent(eventType string) events.Handler {
	return func(action string, payload events.Payload) {
		b.Broadcast(SSEEvent{
			Type: eventType,
			Data: map[string]interface{}{
				"action":  action,
				"payload": payload,
			},
		})
	}
}

// handleAllEvents forwards notification-type events to connected clients.
func (b *Broadcaster) handleAllEvents(action string, payload events.Payload) {
	if action == "notify" || action == "user.notification" {
		b.Broadcast(SSEEvent{
			Type: "NOTIFY",
			Data: payload,
		})
	}
}

// Subscribe creates a new client channel that will receive broadcast events.
func (b *Broadcaster) Subscribe() chan SSEEvent {
	ch := make(chan SSEEvent, 32)
	b.mu.Lock()
	b.clients[ch] = true
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a client channel from the broadcaster.
// The channel is NOT closed — the stream writer exits via heartbeat
// timeout or write error, and the channel is garbage collected.
func (b *Broadcaster) Unsubscribe(ch chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, ch)
}

// Broadcast sends an event to all connected clients. Events are dropped
// for slow clients rather than blocking the broadcaster.
func (b *Broadcaster) Broadcast(event SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- event:
		default:
			// Client is slow, drop event (don't block the broadcaster).
		}
	}
}

// ClientCount returns the number of connected SSE clients (for diagnostics).
func (b *Broadcaster) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Handler returns a Fiber handler that serves a Server-Sent Events stream.
//
// Key insight: c.Context().SetBodyStreamWriter() is asynchronous in
// fasthttp — it registers a callback that runs later during response
// writing. The handler returns nil immediately. Therefore:
//   - We must NOT defer Unsubscribe at the handler level (it would close
//     the channel before the stream writer starts).
//   - All cleanup lives inside the stream writer callback.
//   - A heartbeat ticker keeps the TCP connection alive and lets us detect
//     disconnected clients within 30 seconds.
func (b *Broadcaster) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")
		c.Set("X-Accel-Buffering", "no") // Disable nginx buffering

		// Subscribe creates a channel — we clean it up inside the
		// stream writer, NOT via defer, because SetBodyStreamWriter
		// is async and the handler returns before streaming begins.
		ch := b.Subscribe()

		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			// Cleanup when the stream ends (client disconnect or error).
			defer b.Unsubscribe(ch)

			// Send initial connected event so the client knows the stream is live.
			writeSSE(w, SSEEvent{Type: "CONNECTED", Data: nil})

			// Heartbeat: send an SSE comment every 15 seconds to keep the
			// connection alive through proxies, load balancers, and browser
			// idle timeouts. SSE comments start with ":" and are ignored by
			// the EventSource API.
			heartbeat := time.NewTicker(15 * time.Second)
			defer heartbeat.Stop()

			for {
				select {
				case event, ok := <-ch:
					if !ok {
						// Channel was removed — shouldn't happen since we
						// don't close channels in Unsubscribe, but handle it.
						return
					}
					if !writeSSE(w, event) {
						return // write error — client gone
					}

				case <-heartbeat.C:
					// SSE comment (ignored by EventSource) keeps the TCP
					// connection warm and lets us detect dead clients.
					if !writeSSEComment(w) {
						return // write error — client gone
					}
				}
			}
		})

		return nil
	}
}

// writeSSE serializes an event and writes it to the stream.
// Returns false if the write failed (client disconnected).
func writeSSE(w *bufio.Writer, event SSEEvent) bool {
	data, err := json.Marshal(event)
	if err != nil {
		return true // skip malformed events, don't kill the stream
	}
	_, err = fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
	if err != nil {
		return false
	}
	return w.Flush() == nil
}

// writeSSEComment sends an SSE comment line. Comments are ignored by
// the EventSource API but keep the connection alive.
func writeSSEComment(w *bufio.Writer) bool {
	_, err := w.WriteString(": heartbeat\n\n")
	if err != nil {
		return false
	}
	return w.Flush() == nil
}
