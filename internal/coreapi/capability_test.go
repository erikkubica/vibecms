package coreapi

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// stubCoreAPI is a CoreAPI that records every call and returns
// pre-configured zero values. It does NOT implement the database/svc-backed
// behavior — only the interface contract — so the guard test can run
// without DB or services.
type stubCoreAPI struct {
	calls []string // method names invoked, in order
}

func (s *stubCoreAPI) record(name string) {
	s.calls = append(s.calls, name)
}

// --- Nodes ---
func (s *stubCoreAPI) GetNode(_ context.Context, _ uint) (*Node, error) {
	s.record("GetNode")
	return nil, nil
}
func (s *stubCoreAPI) QueryNodes(_ context.Context, _ NodeQuery) (*NodeList, error) {
	s.record("QueryNodes")
	return nil, nil
}
func (s *stubCoreAPI) ListTaxonomyTerms(_ context.Context, _, _ string) ([]string, error) {
	s.record("ListTaxonomyTerms")
	return nil, nil
}
func (s *stubCoreAPI) CreateNode(_ context.Context, _ NodeInput) (*Node, error) {
	s.record("CreateNode")
	return nil, nil
}
func (s *stubCoreAPI) UpdateNode(_ context.Context, _ uint, _ NodeInput) (*Node, error) {
	s.record("UpdateNode")
	return nil, nil
}
func (s *stubCoreAPI) DeleteNode(_ context.Context, _ uint) error {
	s.record("DeleteNode")
	return nil
}

// --- Taxonomies ---
func (s *stubCoreAPI) RegisterTaxonomy(_ context.Context, _ TaxonomyInput) (*Taxonomy, error) {
	s.record("RegisterTaxonomy")
	return nil, nil
}
func (s *stubCoreAPI) GetTaxonomy(_ context.Context, _ string) (*Taxonomy, error) {
	s.record("GetTaxonomy")
	return nil, nil
}
func (s *stubCoreAPI) ListTaxonomies(_ context.Context) ([]*Taxonomy, error) {
	s.record("ListTaxonomies")
	return nil, nil
}
func (s *stubCoreAPI) UpdateTaxonomy(_ context.Context, _ string, _ TaxonomyInput) (*Taxonomy, error) {
	s.record("UpdateTaxonomy")
	return nil, nil
}
func (s *stubCoreAPI) DeleteTaxonomy(_ context.Context, _ string) error {
	s.record("DeleteTaxonomy")
	return nil
}

// --- Terms ---
func (s *stubCoreAPI) ListTerms(_ context.Context, _, _ string) ([]*TaxonomyTerm, error) {
	s.record("ListTerms")
	return nil, nil
}
func (s *stubCoreAPI) GetTerm(_ context.Context, _ uint) (*TaxonomyTerm, error) {
	s.record("GetTerm")
	return nil, nil
}
func (s *stubCoreAPI) CreateTerm(_ context.Context, _ *TaxonomyTerm) (*TaxonomyTerm, error) {
	s.record("CreateTerm")
	return nil, nil
}
func (s *stubCoreAPI) UpdateTerm(_ context.Context, _ uint, _ map[string]interface{}) (*TaxonomyTerm, error) {
	s.record("UpdateTerm")
	return nil, nil
}
func (s *stubCoreAPI) DeleteTerm(_ context.Context, _ uint) error {
	s.record("DeleteTerm")
	return nil
}

// --- Settings ---
func (s *stubCoreAPI) GetSetting(_ context.Context, _ string) (string, error) {
	s.record("GetSetting")
	return "", nil
}
func (s *stubCoreAPI) SetSetting(_ context.Context, _, _ string) error {
	s.record("SetSetting")
	return nil
}
func (s *stubCoreAPI) GetSettings(_ context.Context, _ string) (map[string]string, error) {
	s.record("GetSettings")
	return nil, nil
}
func (s *stubCoreAPI) GetSettingLoc(_ context.Context, _, _ string) (string, error) {
	s.record("GetSettingLoc")
	return "", nil
}
func (s *stubCoreAPI) SetSettingLoc(_ context.Context, _, _, _ string) error {
	s.record("SetSettingLoc")
	return nil
}
func (s *stubCoreAPI) GetSettingsLoc(_ context.Context, _, _ string) (map[string]string, error) {
	s.record("GetSettingsLoc")
	return nil, nil
}

// --- Events ---
func (s *stubCoreAPI) Emit(_ context.Context, _ string, _ map[string]any) error {
	s.record("Emit")
	return nil
}
func (s *stubCoreAPI) Subscribe(_ context.Context, _ string, _ EventHandler) (UnsubscribeFunc, error) {
	s.record("Subscribe")
	return func() {}, nil
}

// --- Email ---
func (s *stubCoreAPI) SendEmail(_ context.Context, _ EmailRequest) error {
	s.record("SendEmail")
	return nil
}

// --- Menus ---
func (s *stubCoreAPI) GetMenu(_ context.Context, _ string) (*Menu, error) {
	s.record("GetMenu")
	return nil, nil
}
func (s *stubCoreAPI) GetMenus(_ context.Context) ([]*Menu, error) {
	s.record("GetMenus")
	return nil, nil
}
func (s *stubCoreAPI) CreateMenu(_ context.Context, _ MenuInput) (*Menu, error) {
	s.record("CreateMenu")
	return nil, nil
}
func (s *stubCoreAPI) UpdateMenu(_ context.Context, _ string, _ MenuInput) (*Menu, error) {
	s.record("UpdateMenu")
	return nil, nil
}
func (s *stubCoreAPI) UpsertMenu(_ context.Context, _ MenuInput) (*Menu, error) {
	s.record("UpsertMenu")
	return nil, nil
}
func (s *stubCoreAPI) DeleteMenu(_ context.Context, _ string) error {
	s.record("DeleteMenu")
	return nil
}

// --- Routes ---
func (s *stubCoreAPI) RegisterRoute(_ context.Context, _, _ string, _ RouteMeta) error {
	s.record("RegisterRoute")
	return nil
}
func (s *stubCoreAPI) RemoveRoute(_ context.Context, _, _ string) error {
	s.record("RemoveRoute")
	return nil
}

// --- Filters ---
func (s *stubCoreAPI) RegisterFilter(_ context.Context, _ string, _ int, _ FilterHandler) (UnsubscribeFunc, error) {
	s.record("RegisterFilter")
	return func() {}, nil
}
func (s *stubCoreAPI) ApplyFilters(_ context.Context, _ string, value any) (any, error) {
	s.record("ApplyFilters")
	return value, nil
}

// --- Media ---
func (s *stubCoreAPI) UploadMedia(_ context.Context, _ MediaUploadRequest) (*MediaFile, error) {
	s.record("UploadMedia")
	return nil, nil
}
func (s *stubCoreAPI) GetMedia(_ context.Context, _ uint) (*MediaFile, error) {
	s.record("GetMedia")
	return nil, nil
}
func (s *stubCoreAPI) QueryMedia(_ context.Context, _ MediaQuery) ([]*MediaFile, error) {
	s.record("QueryMedia")
	return nil, nil
}
func (s *stubCoreAPI) DeleteMedia(_ context.Context, _ uint) error {
	s.record("DeleteMedia")
	return nil
}

// --- Users ---
func (s *stubCoreAPI) GetUser(_ context.Context, _ uint) (*User, error) {
	s.record("GetUser")
	return nil, nil
}
func (s *stubCoreAPI) QueryUsers(_ context.Context, _ UserQuery) ([]*User, error) {
	s.record("QueryUsers")
	return nil, nil
}

// --- HTTP ---
func (s *stubCoreAPI) Fetch(_ context.Context, _ FetchRequest) (*FetchResponse, error) {
	s.record("Fetch")
	return nil, nil
}

// --- Log ---
func (s *stubCoreAPI) Log(_ context.Context, _, _ string, _ map[string]any) error {
	s.record("Log")
	return nil
}

// --- Data Store ---
func (s *stubCoreAPI) DataGet(_ context.Context, _ string, _ uint) (map[string]any, error) {
	s.record("DataGet")
	return nil, nil
}
func (s *stubCoreAPI) DataQuery(_ context.Context, _ string, _ DataStoreQuery) (*DataStoreResult, error) {
	s.record("DataQuery")
	return nil, nil
}
func (s *stubCoreAPI) DataCreate(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	s.record("DataCreate")
	return nil, nil
}
func (s *stubCoreAPI) DataUpdate(_ context.Context, _ string, _ uint, _ map[string]any) error {
	s.record("DataUpdate")
	return nil
}
func (s *stubCoreAPI) DataDelete(_ context.Context, _ string, _ uint) error {
	s.record("DataDelete")
	return nil
}
func (s *stubCoreAPI) DataExec(_ context.Context, _ string, _ ...any) (int64, error) {
	s.record("DataExec")
	return 0, nil
}

// --- Node Types ---
func (s *stubCoreAPI) RegisterNodeType(_ context.Context, _ NodeTypeInput) (*NodeType, error) {
	s.record("RegisterNodeType")
	return nil, nil
}
func (s *stubCoreAPI) GetNodeType(_ context.Context, _ string) (*NodeType, error) {
	s.record("GetNodeType")
	return nil, nil
}
func (s *stubCoreAPI) ListNodeTypes(_ context.Context) ([]*NodeType, error) {
	s.record("ListNodeTypes")
	return nil, nil
}
func (s *stubCoreAPI) UpdateNodeType(_ context.Context, _ string, _ NodeTypeInput) (*NodeType, error) {
	s.record("UpdateNodeType")
	return nil, nil
}
func (s *stubCoreAPI) DeleteNodeType(_ context.Context, _ string) error {
	s.record("DeleteNodeType")
	return nil
}

// --- File Storage ---
func (s *stubCoreAPI) StoreFile(_ context.Context, _ string, _ []byte) (string, error) {
	s.record("StoreFile")
	return "", nil
}
func (s *stubCoreAPI) DeleteFile(_ context.Context, _ string) error {
	s.record("DeleteFile")
	return nil
}

// Compile-time check: stub satisfies the interface.
var _ CoreAPI = (*stubCoreAPI)(nil)

// guardCase parameterizes a single capability check assertion.
type guardCase struct {
	method string
	cap    string
	call   func(ctx context.Context, api CoreAPI) error
}

// ownedTableFor returns the table name a Data* method case operates
// on. Pulled out so the struct literals stay terse — only the Data*
// methods need this and they all use "test_table" as the fixture.
func ownedTableFor(method string) string {
	switch method {
	case "DataGet", "DataQuery", "DataCreate", "DataUpdate", "DataDelete":
		return "test_table"
	}
	return ""
}

// Each entry pairs a method name with the capability that should gate it
// AND the call to invoke. Returning the method's error directly lets
// the test treat them all uniformly.
func guardCases() []guardCase {
	return []guardCase{
		// Nodes — covers all six gated methods.
		{"GetNode", "nodes:read", func(ctx context.Context, a CoreAPI) error { _, e := a.GetNode(ctx, 1); return e }},
		{"QueryNodes", "nodes:read", func(ctx context.Context, a CoreAPI) error { _, e := a.QueryNodes(ctx, NodeQuery{}); return e }},
		{"ListTaxonomyTerms", "nodes:read", func(ctx context.Context, a CoreAPI) error { _, e := a.ListTaxonomyTerms(ctx, "", ""); return e }},
		{"CreateNode", "nodes:write", func(ctx context.Context, a CoreAPI) error { _, e := a.CreateNode(ctx, NodeInput{}); return e }},
		{"UpdateNode", "nodes:write", func(ctx context.Context, a CoreAPI) error { _, e := a.UpdateNode(ctx, 1, NodeInput{}); return e }},
		{"DeleteNode", "nodes:delete", func(ctx context.Context, a CoreAPI) error { return a.DeleteNode(ctx, 1) }},

		// Taxonomies + Terms.
		{"RegisterTaxonomy", "nodetypes:write", func(ctx context.Context, a CoreAPI) error { _, e := a.RegisterTaxonomy(ctx, TaxonomyInput{}); return e }},
		{"GetTaxonomy", "nodetypes:read", func(ctx context.Context, a CoreAPI) error { _, e := a.GetTaxonomy(ctx, ""); return e }},
		{"ListTaxonomies", "nodetypes:read", func(ctx context.Context, a CoreAPI) error { _, e := a.ListTaxonomies(ctx); return e }},
		{"UpdateTaxonomy", "nodetypes:write", func(ctx context.Context, a CoreAPI) error { _, e := a.UpdateTaxonomy(ctx, "", TaxonomyInput{}); return e }},
		{"DeleteTaxonomy", "nodetypes:write", func(ctx context.Context, a CoreAPI) error { return a.DeleteTaxonomy(ctx, "") }},
		{"ListTerms", "nodes:read", func(ctx context.Context, a CoreAPI) error { _, e := a.ListTerms(ctx, "", ""); return e }},
		{"GetTerm", "nodes:read", func(ctx context.Context, a CoreAPI) error { _, e := a.GetTerm(ctx, 1); return e }},
		{"CreateTerm", "nodes:write", func(ctx context.Context, a CoreAPI) error { _, e := a.CreateTerm(ctx, &TaxonomyTerm{}); return e }},
		{"UpdateTerm", "nodes:write", func(ctx context.Context, a CoreAPI) error { _, e := a.UpdateTerm(ctx, 1, nil); return e }},
		{"DeleteTerm", "nodes:write", func(ctx context.Context, a CoreAPI) error { return a.DeleteTerm(ctx, 1) }},

		// Settings.
		{"GetSetting", "settings:read", func(ctx context.Context, a CoreAPI) error { _, e := a.GetSetting(ctx, ""); return e }},
		{"SetSetting", "settings:write", func(ctx context.Context, a CoreAPI) error { return a.SetSetting(ctx, "", "") }},
		{"GetSettings", "settings:read", func(ctx context.Context, a CoreAPI) error { _, e := a.GetSettings(ctx, ""); return e }},

		// Events.
		{"Emit", "events:emit", func(ctx context.Context, a CoreAPI) error { return a.Emit(ctx, "", nil) }},
		{"Subscribe", "events:subscribe", func(ctx context.Context, a CoreAPI) error { _, e := a.Subscribe(ctx, "", nil); return e }},

		// Email.
		{"SendEmail", "email:send", func(ctx context.Context, a CoreAPI) error { return a.SendEmail(ctx, EmailRequest{}) }},

		// Menus.
		{"GetMenu", "menus:read", func(ctx context.Context, a CoreAPI) error { _, e := a.GetMenu(ctx, ""); return e }},
		{"GetMenus", "menus:read", func(ctx context.Context, a CoreAPI) error { _, e := a.GetMenus(ctx); return e }},
		{"CreateMenu", "menus:write", func(ctx context.Context, a CoreAPI) error { _, e := a.CreateMenu(ctx, MenuInput{}); return e }},
		{"UpdateMenu", "menus:write", func(ctx context.Context, a CoreAPI) error { _, e := a.UpdateMenu(ctx, "", MenuInput{}); return e }},
		{"UpsertMenu", "menus:write", func(ctx context.Context, a CoreAPI) error { _, e := a.UpsertMenu(ctx, MenuInput{}); return e }},
		{"DeleteMenu", "menus:delete", func(ctx context.Context, a CoreAPI) error { return a.DeleteMenu(ctx, "") }},

		// Routes.
		{"RegisterRoute", "routes:register", func(ctx context.Context, a CoreAPI) error { return a.RegisterRoute(ctx, "", "", RouteMeta{}) }},
		{"RemoveRoute", "routes:register", func(ctx context.Context, a CoreAPI) error { return a.RemoveRoute(ctx, "", "") }},

		// Filters.
		{"RegisterFilter", "filters:register", func(ctx context.Context, a CoreAPI) error { _, e := a.RegisterFilter(ctx, "", 0, nil); return e }},
		{"ApplyFilters", "filters:apply", func(ctx context.Context, a CoreAPI) error { _, e := a.ApplyFilters(ctx, "", nil); return e }},

		// Media.
		{"UploadMedia", "media:write", func(ctx context.Context, a CoreAPI) error { _, e := a.UploadMedia(ctx, MediaUploadRequest{}); return e }},
		{"GetMedia", "media:read", func(ctx context.Context, a CoreAPI) error { _, e := a.GetMedia(ctx, 1); return e }},
		{"QueryMedia", "media:read", func(ctx context.Context, a CoreAPI) error { _, e := a.QueryMedia(ctx, MediaQuery{}); return e }},
		{"DeleteMedia", "media:delete", func(ctx context.Context, a CoreAPI) error { return a.DeleteMedia(ctx, 1) }},

		// Users.
		{"GetUser", "users:read", func(ctx context.Context, a CoreAPI) error { _, e := a.GetUser(ctx, 1); return e }},
		{"QueryUsers", "users:read", func(ctx context.Context, a CoreAPI) error { _, e := a.QueryUsers(ctx, UserQuery{}); return e }},

		// HTTP.
		{"Fetch", "http:fetch", func(ctx context.Context, a CoreAPI) error { _, e := a.Fetch(ctx, FetchRequest{}); return e }},

		// Log.
		{"Log", "log:write", func(ctx context.Context, a CoreAPI) error { return a.Log(ctx, "", "", nil) }},

		// Data Store. Test caller has "test_table" in OwnedTables so
		// the per-table allowlist sees an explicit grant on top of
		// the data:read/write/delete capability. DataExec has no
		// table arg and stays internal-only.
		{"DataGet", "data:read", func(ctx context.Context, a CoreAPI) error { _, e := a.DataGet(ctx, "test_table", 1); return e }},
		{"DataQuery", "data:read", func(ctx context.Context, a CoreAPI) error { _, e := a.DataQuery(ctx, "test_table", DataStoreQuery{}); return e }},
		{"DataCreate", "data:write", func(ctx context.Context, a CoreAPI) error { _, e := a.DataCreate(ctx, "test_table", nil); return e }},
		{"DataUpdate", "data:write", func(ctx context.Context, a CoreAPI) error { return a.DataUpdate(ctx, "test_table", 1, nil) }},
		{"DataDelete", "data:delete", func(ctx context.Context, a CoreAPI) error { return a.DataDelete(ctx, "test_table", 1) }},
		{"DataExec", "data:write", func(ctx context.Context, a CoreAPI) error { _, e := a.DataExec(ctx, ""); return e }},

		// Node Types.
		{"RegisterNodeType", "nodetypes:write", func(ctx context.Context, a CoreAPI) error { _, e := a.RegisterNodeType(ctx, NodeTypeInput{}); return e }},
		{"GetNodeType", "nodetypes:read", func(ctx context.Context, a CoreAPI) error { _, e := a.GetNodeType(ctx, ""); return e }},
		{"ListNodeTypes", "nodetypes:read", func(ctx context.Context, a CoreAPI) error { _, e := a.ListNodeTypes(ctx); return e }},
		{"UpdateNodeType", "nodetypes:write", func(ctx context.Context, a CoreAPI) error { _, e := a.UpdateNodeType(ctx, "", NodeTypeInput{}); return e }},
		{"DeleteNodeType", "nodetypes:write", func(ctx context.Context, a CoreAPI) error { return a.DeleteNodeType(ctx, "") }},

		// File Storage.
		{"StoreFile", "files:write", func(ctx context.Context, a CoreAPI) error { _, e := a.StoreFile(ctx, "", nil); return e }},
		{"DeleteFile", "files:delete", func(ctx context.Context, a CoreAPI) error { return a.DeleteFile(ctx, "") }},
	}
}

// TestCapabilityGuard_DeniedWithoutCapability verifies every gated method
// returns ErrCapabilityDenied when the caller lacks the required capability.
// This is the most security-critical assertion in the codebase.
func TestCapabilityGuard_DeniedWithoutCapability(t *testing.T) {
	stub := &stubCoreAPI{}
	guarded := NewCapabilityGuard(stub)
	// Caller has no capabilities.
	caller := CallerInfo{Slug: "test", Type: "tengo", Capabilities: map[string]bool{}}
	ctx := WithCaller(context.Background(), caller)

	for _, tc := range guardCases() {
		t.Run(tc.method+"_denied", func(t *testing.T) {
			err := tc.call(ctx, guarded)
			if err == nil {
				t.Fatalf("%s: expected error, got nil", tc.method)
			}
			if !errors.Is(err, ErrCapabilityDenied) && !isCapabilityErr(err) {
				t.Fatalf("%s: expected capability-denied error, got %v", tc.method, err)
			}
		})
	}
	// And the inner stub should never have been invoked — every call
	// got rejected at the guard.
	if len(stub.calls) > 0 {
		t.Fatalf("inner CoreAPI was invoked despite denial: %v", stub.calls)
	}
}

// TestCapabilityGuard_AllowedWithCapability verifies every gated method
// passes through to the inner CoreAPI when the caller has the right cap
// (and, for Data* cases, the per-table ownership entry).
func TestCapabilityGuard_AllowedWithCapability(t *testing.T) {
	for _, tc := range guardCases() {
		t.Run(tc.method+"_allowed", func(t *testing.T) {
			stub := &stubCoreAPI{}
			guarded := NewCapabilityGuard(stub)
			owned := map[string]bool{}
			if tab := ownedTableFor(tc.method); tab != "" {
				owned[tab] = true
			}
			caller := CallerInfo{
				Slug:         "test",
				Type:         "tengo",
				Capabilities: map[string]bool{tc.cap: true},
				OwnedTables:  owned,
			}
			ctx := WithCaller(context.Background(), caller)
			if err := tc.call(ctx, guarded); err != nil {
				t.Fatalf("%s with cap %q: unexpected error %v", tc.method, tc.cap, err)
			}
			if len(stub.calls) != 1 || stub.calls[0] != tc.method {
				t.Fatalf("%s: expected inner %s call, got %v", tc.method, tc.method, stub.calls)
			}
		})
	}
}

// TestCapabilityGuard_InternalCallerBypassesAll verifies internal callers
// (kernel code calling its own CoreAPI) skip every check. This is by
// design — the capability system is for plugins/scripts, not in-process
// services.
func TestCapabilityGuard_InternalCallerBypassesAll(t *testing.T) {
	for _, tc := range guardCases() {
		t.Run(tc.method+"_internal", func(t *testing.T) {
			stub := &stubCoreAPI{}
			guarded := NewCapabilityGuard(stub)
			ctx := WithCaller(context.Background(), InternalCaller())
			if err := tc.call(ctx, guarded); err != nil {
				t.Fatalf("%s as internal: unexpected error %v", tc.method, err)
			}
			if len(stub.calls) != 1 || stub.calls[0] != tc.method {
				t.Fatalf("%s: expected inner %s call, got %v", tc.method, tc.method, stub.calls)
			}
		})
	}
}

// TestCapabilityGuard_MissingContextDefaultsToInternal documents (and
// pins) the fail-OPEN behavior of CallerFromContext when no caller is
// attached. This is a known footgun (see core_dev_guide §1.4) and the
// test exists so a future refactor that flips it to fail-closed is a
// deliberate, reviewed change rather than a silent regression.
func TestCapabilityGuard_MissingContextDefaultsToInternal(t *testing.T) {
	stub := &stubCoreAPI{}
	guarded := NewCapabilityGuard(stub)
	if _, err := guarded.GetNode(context.Background(), 1); err != nil {
		t.Fatalf("expected pass-through for missing-caller context, got %v", err)
	}
	if len(stub.calls) != 1 {
		t.Fatalf("expected inner GetNode call, got %v", stub.calls)
	}
}

// isCapabilityErr accepts wrapped APIError values whose Code matches
// ErrCapabilityDenied. errors.Is below already covers the unwrapped path.
func isCapabilityErr(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return errors.Is(apiErr.Code, ErrCapabilityDenied)
	}
	return false
}

// Compile-time guard: importing io.Discard and time keeps the imports
// stable even if a test below stops using them. Without these references,
// `goimports`-style cleanup would strip the imports and the file
// wouldn't compile when a test is added that needs them.
var _ = io.Discard
var _ = time.Second
