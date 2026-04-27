package coreapi

import (
	"context"
	"sort"
	"sync"
)

var filtersMu sync.RWMutex

func (c *coreImpl) RegisterFilter(ctx context.Context, name string, priority int, handler FilterHandler) (UnsubscribeFunc, error) {
	if name == "" {
		return nil, NewValidation("filter name is required")
	}
	if handler == nil {
		return nil, NewValidation("filter handler is required")
	}

	filtersMu.Lock()
	c.nextFilterID++
	id := c.nextFilterID
	c.filters[name] = append(c.filters[name], filterEntry{
		id:       id,
		priority: priority,
		handler:  handler,
	})
	sort.Slice(c.filters[name], func(i, j int) bool {
		return c.filters[name][i].priority < c.filters[name][j].priority
	})
	filtersMu.Unlock()

	// Compare by the opaque registration ID rather than the handler's
	// function value (Go forbids ==/!= on funcs) or its address (a stack
	// slot, never matching across calls — the original bug).
	unsub := func() {
		filtersMu.Lock()
		defer filtersMu.Unlock()
		entries := c.filters[name]
		for i, e := range entries {
			if e.id == id {
				c.filters[name] = append(entries[:i], entries[i+1:]...)
				break
			}
		}
	}

	return unsub, nil
}

func (c *coreImpl) ApplyFilters(ctx context.Context, name string, value any) (any, error) {
	filtersMu.RLock()
	entries := make([]filterEntry, len(c.filters[name]))
	copy(entries, c.filters[name])
	filtersMu.RUnlock()

	result := value
	for _, e := range entries {
		result = e.handler(result)
	}
	return result, nil
}
