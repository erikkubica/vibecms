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

// NewBroadcaster creates a new Broadcaster and subscribes to the event bus.
// A single SubscribeAll handler routes every backend event to one of the
// typed SSE events consumed by the admin client.
func NewBroadcaster(eventBus *events.EventBus) *Broadcaster {
	b := &Broadcaster{
		clients: make(map[chan SSEEvent]bool),
	}
	eventBus.SubscribeAll(b.route)
	return b
}

// route maps a backend event.Bus action to the SSE event(s) the admin UI
// expects. See the SSEEvent type comment in types.go for the taxonomy.
//
// A single backend event may fan out into multiple SSE events — e.g. a
// node_type.updated both flips the sidebar (NAV_STALE) and invalidates the
// entity's own query caches (ENTITY_CHANGED).
func (b *Broadcaster) route(action string, payload events.Payload) {
	// Pass-through notifications for toasts.
	if action == "notify" || action == "user.notification" {
		b.Broadcast(SSEEvent{Type: "NOTIFY", Data: payload})
		return
	}

	// Settings have their own channel because query keys are scoped by key.
	if action == "setting.updated" {
		key, _ := payload["key"].(string)
		b.Broadcast(SSEEvent{Type: "SETTING_CHANGED", Key: key, Data: payload})
		return
	}

	// Navigation-affecting events: sidebar + boot manifest need a refetch.
	// These ALSO fan out to ENTITY_CHANGED so per-entity list/detail queries
	// (e.g. the node-types list page) refresh too.
	switch action {
	case "extension.activated", "extension.deactivated",
		"theme.activated", "theme.deactivated",
		"taxonomies:register":
		b.Broadcast(SSEEvent{Type: "NAV_STALE", Data: payload})
		return
	}

	// Everything else is an entity CRUD event of the form "<entity>.<op>".
	entity, op := splitEntityAction(action)
	if entity == "" {
		return // unknown shape — drop silently
	}
	ev := SSEEvent{
		Type:   "ENTITY_CHANGED",
		Entity: entity,
		Op:     op,
		ID:     extractID(payload),
		Data:   payload,
	}
	b.Broadcast(ev)

	// node_type changes also rebuild the sidebar.
	if entity == "node_type" {
		b.Broadcast(SSEEvent{Type: "NAV_STALE", Data: payload})
	}
}

// splitEntityAction splits "layout_block.updated" → ("layout_block", "updated").
// Returns ("", "") if the action is not a dotted entity.op form.
func splitEntityAction(action string) (string, string) {
	for i := 0; i < len(action); i++ {
		if action[i] == '.' {
			return action[:i], action[i+1:]
		}
	}
	return "", ""
}

// extractID pulls an id-shaped field out of a payload. Backend publishers use
// a few different field names — "id" is canonical, but some older publishers
// emit "user_id", "node_id", etc. We accept any of them.
func extractID(p events.Payload) interface{} {
	for _, k := range []string{"id", "user_id", "node_id", "menu_id", "layout_id",
		"layout_block_id", "block_type_id", "template_id", "taxonomy_id", "term_id",
		"role_id", "node_type_id", "extension_id"} {
		if v, ok := p[k]; ok {
			return v
		}
	}
	return nil
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
