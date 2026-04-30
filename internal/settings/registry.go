package settings

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is the in-process catalogue of settings schemas. Core
// registers built-ins at boot; extensions register at activation;
// themes (one day) register at theme-load. Concurrent-safe: reads
// (admin UI list/get) are common, writes (lifecycle events) are rare.
type Registry struct {
	mu      sync.RWMutex
	schemas map[string]Schema
}

// NewRegistry constructs an empty registry.
func NewRegistry() *Registry {
	return &Registry{schemas: make(map[string]Schema)}
}

// Register installs a schema, replacing any existing entry with the
// same ID. Returns an error when the schema is structurally invalid
// (missing ID/title or empty section/field declarations); we want bad
// registrations to fail loudly at boot rather than silently produce
// broken admin pages later.
func (r *Registry) Register(s Schema) error {
	if s.ID == "" {
		return fmt.Errorf("settings: schema ID is required")
	}
	if s.Title == "" {
		return fmt.Errorf("settings: schema %q: title is required", s.ID)
	}
	if len(s.Sections) == 0 {
		return fmt.Errorf("settings: schema %q: at least one section required", s.ID)
	}
	seen := make(map[string]struct{})
	for i, sec := range s.Sections {
		if len(sec.Fields) == 0 {
			return fmt.Errorf("settings: schema %q section %d (%q): no fields", s.ID, i, sec.Title)
		}
		for _, f := range sec.Fields {
			if f.Key == "" {
				return fmt.Errorf("settings: schema %q: field with empty key", s.ID)
			}
			if _, dup := seen[f.Key]; dup {
				return fmt.Errorf("settings: schema %q: duplicate field key %q", s.ID, f.Key)
			}
			seen[f.Key] = struct{}{}
			if f.Type == "" {
				return fmt.Errorf("settings: schema %q field %q: type is required", s.ID, f.Key)
			}
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schemas[s.ID] = s
	return nil
}

// MustRegister panics on registration failure. Use for built-ins that
// should never fail at boot — failure indicates a programmer error,
// not a runtime condition.
func (r *Registry) MustRegister(s Schema) {
	if err := r.Register(s); err != nil {
		panic(err)
	}
}

// Unregister removes a schema by ID. No-op when the ID is unknown.
// Used on extension deactivation.
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.schemas, id)
}

// Get returns a schema by ID. The bool is false when no schema with
// that ID is registered.
func (r *Registry) Get(id string) (Schema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.schemas[id]
	return s, ok
}

// List returns every registered schema sorted by ID for stable
// presentation in the admin UI.
func (r *Registry) List() []Schema {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Schema, 0, len(r.schemas))
	for _, s := range r.schemas {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
