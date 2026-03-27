package coreapi

import (
	"context"
	"vibecms/internal/events"
)

func (c *coreImpl) Emit(_ context.Context, action string, payload map[string]any) error {
	p := make(events.Payload, len(payload))
	for k, v := range payload {
		p[k] = v
	}
	c.eventBus.Publish(action, p)
	return nil
}

func (c *coreImpl) Subscribe(_ context.Context, action string, handler EventHandler) (UnsubscribeFunc, error) {
	wrapped := func(a string, p events.Payload) {
		m := make(map[string]any, len(p))
		for k, v := range p {
			m[k] = v
		}
		handler(a, m)
	}
	c.eventBus.Subscribe(action, wrapped)
	return func() {}, nil // EventBus doesn't support unsubscribe yet
}
