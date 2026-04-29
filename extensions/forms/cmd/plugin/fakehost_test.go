package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"squilla/internal/coreapi"
)

// FakeHost implements coreapi.CoreAPI in-memory for plugin unit tests.
type FakeHost struct {
	mu          sync.Mutex
	Tables      map[string]map[uint]map[string]any
	NextID      map[string]uint
	Settings    map[string]string
	StoredFiles map[string][]byte
	Emitted     []EmittedEvent
	Logs        []LogLine
	Sent        []coreapi.EmailRequest
	FetchStub   func(req coreapi.FetchRequest) (*coreapi.FetchResponse, error)
	UserStub    map[uint]*coreapi.User
}

type EmittedEvent struct {
	Action  string
	Payload map[string]any
}

type LogLine struct {
	Level, Message string
	Fields         map[string]any
}

func NewFakeHost() *FakeHost {
	return &FakeHost{
		Tables:      map[string]map[uint]map[string]any{},
		NextID:      map[string]uint{},
		Settings:    map[string]string{},
		StoredFiles: map[string][]byte{},
		UserStub:    map[uint]*coreapi.User{},
	}
}

// newPlugin creates a FormsPlugin wired to the given FakeHost.
func newPlugin(h *FakeHost) *FormsPlugin {
	return &FormsPlugin{
		host:        h,
		rateLimiter: NewRateLimiter(10000),
	}
}

// ---- DataStore ----

func (h *FakeHost) DataGet(_ context.Context, table string, id uint) (map[string]any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if t, ok := h.Tables[table]; ok {
		if r, ok := t[id]; ok {
			return cloneMap(r), nil
		}
	}
	return nil, fmt.Errorf("not found: %s/%d", table, id)
}

func (h *FakeHost) DataCreate(_ context.Context, table string, data map[string]any) (map[string]any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.Tables[table] == nil {
		h.Tables[table] = map[uint]map[string]any{}
	}
	h.NextID[table]++
	id := h.NextID[table]
	row := cloneMap(data)
	row["id"] = float64(id)
	h.Tables[table][id] = row
	return cloneMap(row), nil
}

func (h *FakeHost) DataQuery(_ context.Context, table string, q coreapi.DataStoreQuery) (*coreapi.DataStoreResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := &coreapi.DataStoreResult{}
	for _, r := range h.Tables[table] {
		if matchesWhere(r, q.Where) && matchesRaw(r, q.Raw, q.Args) {
			out.Rows = append(out.Rows, cloneMap(r))
		}
	}
	out.Total = int64(len(out.Rows))
	if q.Limit > 0 && int(out.Total) > q.Limit {
		end := q.Offset + q.Limit
		if end > len(out.Rows) {
			end = len(out.Rows)
		}
		start := q.Offset
		if start > len(out.Rows) {
			start = len(out.Rows)
		}
		out.Rows = out.Rows[start:end]
	}
	return out, nil
}

func (h *FakeHost) DataUpdate(_ context.Context, table string, id uint, data map[string]any) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if t, ok := h.Tables[table]; ok {
		if r, ok := t[id]; ok {
			for k, v := range data {
				r[k] = v
			}
			return nil
		}
	}
	return fmt.Errorf("not found: %s/%d", table, id)
}

func (h *FakeHost) DataDelete(_ context.Context, table string, id uint) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if t, ok := h.Tables[table]; ok {
		delete(t, id)
	}
	return nil
}

func (h *FakeHost) DataExec(_ context.Context, _ string, _ ...any) (int64, error) {
	return 0, nil
}

// ---- File Storage ----

func (h *FakeHost) StoreFile(_ context.Context, path string, data []byte) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.StoredFiles[path] = data
	return "/" + path, nil
}

func (h *FakeHost) DeleteFile(_ context.Context, path string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.StoredFiles, path)
	return nil
}

// ---- Events ----

func (h *FakeHost) Emit(_ context.Context, action string, payload map[string]any) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Emitted = append(h.Emitted, EmittedEvent{action, cloneMap(payload)})
	return nil
}

func (h *FakeHost) Subscribe(_ context.Context, _ string, _ coreapi.EventHandler) (coreapi.UnsubscribeFunc, error) {
	return func() {}, nil
}

// ---- Logging ----

func (h *FakeHost) Log(_ context.Context, level, message string, fields map[string]any) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Logs = append(h.Logs, LogLine{level, message, fields})
	return nil
}

// ---- Email ----

func (h *FakeHost) SendEmail(_ context.Context, req coreapi.EmailRequest) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Sent = append(h.Sent, req)
	return nil
}

// ---- HTTP (outbound) ----

func (h *FakeHost) Fetch(_ context.Context, req coreapi.FetchRequest) (*coreapi.FetchResponse, error) {
	if h.FetchStub != nil {
		return h.FetchStub(req)
	}
	return &coreapi.FetchResponse{StatusCode: 200, Body: `{"success":true}`}, nil
}

// ---- Users ----

func (h *FakeHost) GetUser(_ context.Context, id uint) (*coreapi.User, error) {
	if u, ok := h.UserStub[id]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user not found: %d", id)
}

func (h *FakeHost) QueryUsers(_ context.Context, _ coreapi.UserQuery) ([]*coreapi.User, error) {
	panic("not used by forms tests")
}

// ---- Settings ----

func (h *FakeHost) GetSetting(_ context.Context, key string) (string, error) {
	if v, ok := h.Settings[key]; ok {
		return v, nil
	}
	return "", nil
}

func (h *FakeHost) SetSetting(_ context.Context, key, value string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Settings[key] = value
	return nil
}

func (h *FakeHost) GetSettings(_ context.Context, _ string) (map[string]string, error) {
	return h.Settings, nil
}

func (h *FakeHost) GetSettingLoc(ctx context.Context, key, _ string) (string, error) {
	return h.GetSetting(ctx, key)
}

func (h *FakeHost) SetSettingLoc(ctx context.Context, key, _, value string) error {
	return h.SetSetting(ctx, key, value)
}

func (h *FakeHost) GetSettingsLoc(ctx context.Context, prefix, _ string) (map[string]string, error) {
	return h.GetSettings(ctx, prefix)
}

// ---- Stubs for interface methods not used by forms ----

func (h *FakeHost) GetNode(_ context.Context, _ uint) (*coreapi.Node, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) QueryNodes(_ context.Context, _ coreapi.NodeQuery) (*coreapi.NodeList, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) ListTaxonomyTerms(_ context.Context, _ string, _ string) ([]string, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) CreateNode(_ context.Context, _ coreapi.NodeInput) (*coreapi.Node, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) UpdateNode(_ context.Context, _ uint, _ coreapi.NodeInput) (*coreapi.Node, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) DeleteNode(_ context.Context, _ uint) error {
	panic("not used by forms tests")
}
func (h *FakeHost) RegisterTaxonomy(_ context.Context, _ coreapi.TaxonomyInput) (*coreapi.Taxonomy, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) GetTaxonomy(_ context.Context, _ string) (*coreapi.Taxonomy, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) ListTaxonomies(_ context.Context) ([]*coreapi.Taxonomy, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) UpdateTaxonomy(_ context.Context, _ string, _ coreapi.TaxonomyInput) (*coreapi.Taxonomy, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) DeleteTaxonomy(_ context.Context, _ string) error {
	panic("not used by forms tests")
}
func (h *FakeHost) ListTerms(_ context.Context, _ string, _ string) ([]*coreapi.TaxonomyTerm, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) GetTerm(_ context.Context, _ uint) (*coreapi.TaxonomyTerm, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) CreateTerm(_ context.Context, _ *coreapi.TaxonomyTerm) (*coreapi.TaxonomyTerm, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) UpdateTerm(_ context.Context, _ uint, _ map[string]interface{}) (*coreapi.TaxonomyTerm, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) DeleteTerm(_ context.Context, _ uint) error {
	panic("not used by forms tests")
}
func (h *FakeHost) GetMenu(_ context.Context, _ string) (*coreapi.Menu, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) GetMenus(_ context.Context) ([]*coreapi.Menu, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) CreateMenu(_ context.Context, _ coreapi.MenuInput) (*coreapi.Menu, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) UpdateMenu(_ context.Context, _ string, _ coreapi.MenuInput) (*coreapi.Menu, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) UpsertMenu(_ context.Context, _ coreapi.MenuInput) (*coreapi.Menu, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) DeleteMenu(_ context.Context, _ string) error {
	panic("not used by forms tests")
}
func (h *FakeHost) RegisterRoute(_ context.Context, _ string, _ string, _ coreapi.RouteMeta) error {
	panic("not used by forms tests")
}
func (h *FakeHost) RemoveRoute(_ context.Context, _ string, _ string) error {
	panic("not used by forms tests")
}
func (h *FakeHost) RegisterFilter(_ context.Context, _ string, _ int, _ coreapi.FilterHandler) (coreapi.UnsubscribeFunc, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) ApplyFilters(_ context.Context, _ string, value any) (any, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) UploadMedia(_ context.Context, _ coreapi.MediaUploadRequest) (*coreapi.MediaFile, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) GetMedia(_ context.Context, _ uint) (*coreapi.MediaFile, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) QueryMedia(_ context.Context, _ coreapi.MediaQuery) ([]*coreapi.MediaFile, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) DeleteMedia(_ context.Context, _ uint) error {
	panic("not used by forms tests")
}
func (h *FakeHost) RegisterNodeType(_ context.Context, _ coreapi.NodeTypeInput) (*coreapi.NodeType, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) GetNodeType(_ context.Context, _ string) (*coreapi.NodeType, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) ListNodeTypes(_ context.Context) ([]*coreapi.NodeType, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) UpdateNodeType(_ context.Context, _ string, _ coreapi.NodeTypeInput) (*coreapi.NodeType, error) {
	panic("not used by forms tests")
}
func (h *FakeHost) DeleteNodeType(_ context.Context, _ string) error {
	panic("not used by forms tests")
}

// ---- Test helpers ----

// ctx returns a background context for use in tests.
func ctx() context.Context { return context.Background() }

// ---- Internal helpers ----

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}

// matchesWhere returns true if every key-value in where matches the row.
func matchesWhere(row map[string]any, where map[string]any) bool {
	for k, v := range where {
		rowVal, ok := row[k]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", rowVal) != fmt.Sprintf("%v", v) {
			return false
		}
	}
	return true
}

// matchesRaw provides minimal support for retention's "created_at < ?" filter.
// Only handles the pattern: "created_at < ?" used by runRetention.
func matchesRaw(row map[string]any, raw string, args []any) bool {
	if raw == "" {
		return true
	}
	// Only support the retention pattern for tests: "created_at < ?"
	if strings.Contains(raw, "created_at < ?") && len(args) > 0 {
		// If the row doesn't have created_at, it matches (include in deletion set)
		_, ok := row["created_at"]
		if !ok {
			return true
		}
	}
	// For all other raw queries (search, date filters etc.) — include all rows.
	// Tests that care about precise filtering should use Where instead.
	return true
}
