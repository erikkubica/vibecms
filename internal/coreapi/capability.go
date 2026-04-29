package coreapi

import "context"

// compile-time check
var _ CoreAPI = (*capabilityGuard)(nil)

type capabilityGuard struct {
	inner CoreAPI
}

// NewCapabilityGuard wraps a CoreAPI implementation with per-caller capability checks.
func NewCapabilityGuard(inner CoreAPI) CoreAPI {
	return &capabilityGuard{inner: inner}
}

func checkCapability(ctx context.Context, cap string) error {
	caller := CallerFromContext(ctx)
	if caller.Type == "internal" {
		return nil
	}
	if caller.Capabilities[cap] {
		return nil
	}
	return NewCapabilityDenied(cap)
}

// --- Nodes ---

func (g *capabilityGuard) GetNode(ctx context.Context, id uint) (*Node, error) {
	if err := checkCapability(ctx, "nodes:read"); err != nil {
		return nil, err
	}
	return g.inner.GetNode(ctx, id)
}

func (g *capabilityGuard) QueryNodes(ctx context.Context, query NodeQuery) (*NodeList, error) {
	if err := checkCapability(ctx, "nodes:read"); err != nil {
		return nil, err
	}
	return g.inner.QueryNodes(ctx, query)
}

func (g *capabilityGuard) ListTaxonomyTerms(ctx context.Context, nodeType string, taxonomy string) ([]string, error) {
	if err := checkCapability(ctx, "nodes:read"); err != nil {
		return nil, err
	}
	return g.inner.ListTaxonomyTerms(ctx, nodeType, taxonomy)
}

func (g *capabilityGuard) CreateNode(ctx context.Context, input NodeInput) (*Node, error) {
	if err := checkCapability(ctx, "nodes:write"); err != nil {
		return nil, err
	}
	return g.inner.CreateNode(ctx, input)
}

func (g *capabilityGuard) UpdateNode(ctx context.Context, id uint, input NodeInput) (*Node, error) {
	if err := checkCapability(ctx, "nodes:write"); err != nil {
		return nil, err
	}
	return g.inner.UpdateNode(ctx, id, input)
}

func (g *capabilityGuard) DeleteNode(ctx context.Context, id uint) error {
	if err := checkCapability(ctx, "nodes:delete"); err != nil {
		return err
	}
	return g.inner.DeleteNode(ctx, id)
}

// --- Taxonomies ---

func (g *capabilityGuard) ListTerms(ctx context.Context, nodeType string, taxonomy string) ([]*TaxonomyTerm, error) {
	if err := checkCapability(ctx, "nodes:read"); err != nil {
		return nil, err
	}
	return g.inner.ListTerms(ctx, nodeType, taxonomy)
}

func (g *capabilityGuard) GetTerm(ctx context.Context, id uint) (*TaxonomyTerm, error) {
	if err := checkCapability(ctx, "nodes:read"); err != nil {
		return nil, err
	}
	return g.inner.GetTerm(ctx, id)
}

func (g *capabilityGuard) CreateTerm(ctx context.Context, term *TaxonomyTerm) (*TaxonomyTerm, error) {
	if err := checkCapability(ctx, "nodes:write"); err != nil {
		return nil, err
	}
	return g.inner.CreateTerm(ctx, term)
}

func (g *capabilityGuard) UpdateTerm(ctx context.Context, id uint, updates map[string]interface{}) (*TaxonomyTerm, error) {
	if err := checkCapability(ctx, "nodes:write"); err != nil {
		return nil, err
	}
	return g.inner.UpdateTerm(ctx, id, updates)
}

func (g *capabilityGuard) DeleteTerm(ctx context.Context, id uint) error {
	if err := checkCapability(ctx, "nodes:write"); err != nil {
		return err
	}
	return g.inner.DeleteTerm(ctx, id)
}

// --- Taxonomy Definitions ---

func (g *capabilityGuard) RegisterTaxonomy(ctx context.Context, input TaxonomyInput) (*Taxonomy, error) {
	if err := checkCapability(ctx, "nodetypes:write"); err != nil {
		return nil, err
	}
	return g.inner.RegisterTaxonomy(ctx, input)
}

func (g *capabilityGuard) GetTaxonomy(ctx context.Context, slug string) (*Taxonomy, error) {
	if err := checkCapability(ctx, "nodetypes:read"); err != nil {
		return nil, err
	}
	return g.inner.GetTaxonomy(ctx, slug)
}

func (g *capabilityGuard) ListTaxonomies(ctx context.Context) ([]*Taxonomy, error) {
	if err := checkCapability(ctx, "nodetypes:read"); err != nil {
		return nil, err
	}
	return g.inner.ListTaxonomies(ctx)
}

func (g *capabilityGuard) UpdateTaxonomy(ctx context.Context, slug string, input TaxonomyInput) (*Taxonomy, error) {
	if err := checkCapability(ctx, "nodetypes:write"); err != nil {
		return nil, err
	}
	return g.inner.UpdateTaxonomy(ctx, slug, input)
}

func (g *capabilityGuard) DeleteTaxonomy(ctx context.Context, slug string) error {
	if err := checkCapability(ctx, "nodetypes:write"); err != nil {
		return err
	}
	return g.inner.DeleteTaxonomy(ctx, slug)
}

// --- Settings ---

func (g *capabilityGuard) GetSetting(ctx context.Context, key string) (string, error) {
	if err := checkCapability(ctx, "settings:read"); err != nil {
		return "", err
	}
	return g.inner.GetSetting(ctx, key)
}

func (g *capabilityGuard) SetSetting(ctx context.Context, key, value string) error {
	if err := checkCapability(ctx, "settings:write"); err != nil {
		return err
	}
	return g.inner.SetSetting(ctx, key, value)
}

func (g *capabilityGuard) GetSettings(ctx context.Context, prefix string) (map[string]string, error) {
	if err := checkCapability(ctx, "settings:read"); err != nil {
		return nil, err
	}
	return g.inner.GetSettings(ctx, prefix)
}

func (g *capabilityGuard) GetSettingLoc(ctx context.Context, key, locale string) (string, error) {
	if err := checkCapability(ctx, "settings:read"); err != nil {
		return "", err
	}
	return g.inner.GetSettingLoc(ctx, key, locale)
}

func (g *capabilityGuard) SetSettingLoc(ctx context.Context, key, locale, value string) error {
	if err := checkCapability(ctx, "settings:write"); err != nil {
		return err
	}
	return g.inner.SetSettingLoc(ctx, key, locale, value)
}

func (g *capabilityGuard) GetSettingsLoc(ctx context.Context, prefix, locale string) (map[string]string, error) {
	if err := checkCapability(ctx, "settings:read"); err != nil {
		return nil, err
	}
	return g.inner.GetSettingsLoc(ctx, prefix, locale)
}

// --- Events ---

func (g *capabilityGuard) Emit(ctx context.Context, action string, payload map[string]any) error {
	if err := checkCapability(ctx, "events:emit"); err != nil {
		return err
	}
	return g.inner.Emit(ctx, action, payload)
}

func (g *capabilityGuard) Subscribe(ctx context.Context, action string, handler EventHandler) (UnsubscribeFunc, error) {
	if err := checkCapability(ctx, "events:subscribe"); err != nil {
		return nil, err
	}
	return g.inner.Subscribe(ctx, action, handler)
}

// --- Email ---

func (g *capabilityGuard) SendEmail(ctx context.Context, req EmailRequest) error {
	if err := checkCapability(ctx, "email:send"); err != nil {
		return err
	}
	return g.inner.SendEmail(ctx, req)
}

// --- Menus ---

func (g *capabilityGuard) GetMenu(ctx context.Context, slug string) (*Menu, error) {
	if err := checkCapability(ctx, "menus:read"); err != nil {
		return nil, err
	}
	return g.inner.GetMenu(ctx, slug)
}

func (g *capabilityGuard) GetMenus(ctx context.Context) ([]*Menu, error) {
	if err := checkCapability(ctx, "menus:read"); err != nil {
		return nil, err
	}
	return g.inner.GetMenus(ctx)
}

func (g *capabilityGuard) CreateMenu(ctx context.Context, input MenuInput) (*Menu, error) {
	if err := checkCapability(ctx, "menus:write"); err != nil {
		return nil, err
	}
	return g.inner.CreateMenu(ctx, input)
}

func (g *capabilityGuard) UpdateMenu(ctx context.Context, slug string, input MenuInput) (*Menu, error) {
	if err := checkCapability(ctx, "menus:write"); err != nil {
		return nil, err
	}
	return g.inner.UpdateMenu(ctx, slug, input)
}

func (g *capabilityGuard) UpsertMenu(ctx context.Context, input MenuInput) (*Menu, error) {
	if err := checkCapability(ctx, "menus:write"); err != nil {
		return nil, err
	}
	return g.inner.UpsertMenu(ctx, input)
}

func (g *capabilityGuard) DeleteMenu(ctx context.Context, slug string) error {
	if err := checkCapability(ctx, "menus:delete"); err != nil {
		return err
	}
	return g.inner.DeleteMenu(ctx, slug)
}

// --- Routes ---

func (g *capabilityGuard) RegisterRoute(ctx context.Context, method, path string, meta RouteMeta) error {
	if err := checkCapability(ctx, "routes:register"); err != nil {
		return err
	}
	return g.inner.RegisterRoute(ctx, method, path, meta)
}

func (g *capabilityGuard) RemoveRoute(ctx context.Context, method, path string) error {
	if err := checkCapability(ctx, "routes:register"); err != nil {
		return err
	}
	return g.inner.RemoveRoute(ctx, method, path)
}

// --- Filters ---

func (g *capabilityGuard) RegisterFilter(ctx context.Context, name string, priority int, handler FilterHandler) (UnsubscribeFunc, error) {
	if err := checkCapability(ctx, "filters:register"); err != nil {
		return nil, err
	}
	return g.inner.RegisterFilter(ctx, name, priority, handler)
}

func (g *capabilityGuard) ApplyFilters(ctx context.Context, name string, value any) (any, error) {
	if err := checkCapability(ctx, "filters:apply"); err != nil {
		return nil, err
	}
	return g.inner.ApplyFilters(ctx, name, value)
}

// --- Media ---

func (g *capabilityGuard) UploadMedia(ctx context.Context, req MediaUploadRequest) (*MediaFile, error) {
	if err := checkCapability(ctx, "media:write"); err != nil {
		return nil, err
	}
	return g.inner.UploadMedia(ctx, req)
}

func (g *capabilityGuard) GetMedia(ctx context.Context, id uint) (*MediaFile, error) {
	if err := checkCapability(ctx, "media:read"); err != nil {
		return nil, err
	}
	return g.inner.GetMedia(ctx, id)
}

func (g *capabilityGuard) QueryMedia(ctx context.Context, query MediaQuery) ([]*MediaFile, error) {
	if err := checkCapability(ctx, "media:read"); err != nil {
		return nil, err
	}
	return g.inner.QueryMedia(ctx, query)
}

func (g *capabilityGuard) DeleteMedia(ctx context.Context, id uint) error {
	if err := checkCapability(ctx, "media:delete"); err != nil {
		return err
	}
	return g.inner.DeleteMedia(ctx, id)
}

// --- Users ---

func (g *capabilityGuard) GetUser(ctx context.Context, id uint) (*User, error) {
	if err := checkCapability(ctx, "users:read"); err != nil {
		return nil, err
	}
	return g.inner.GetUser(ctx, id)
}

func (g *capabilityGuard) QueryUsers(ctx context.Context, query UserQuery) ([]*User, error) {
	if err := checkCapability(ctx, "users:read"); err != nil {
		return nil, err
	}
	return g.inner.QueryUsers(ctx, query)
}

// --- HTTP ---

func (g *capabilityGuard) Fetch(ctx context.Context, req FetchRequest) (*FetchResponse, error) {
	if err := checkCapability(ctx, "http:fetch"); err != nil {
		return nil, err
	}
	return g.inner.Fetch(ctx, req)
}

// --- Log ---

func (g *capabilityGuard) Log(ctx context.Context, level, message string, fields map[string]any) error {
	if err := checkCapability(ctx, "log:write"); err != nil {
		return err
	}
	return g.inner.Log(ctx, level, message, fields)
}

// --- Data Store ---
//
// Two-layer gate:
//
//  1. checkCapability — does the extension hold the data:read /
//     data:write / data:delete cap at all?
//  2. checkTableAccess — is this specific table in the extension's
//     OwnedTables (and not in the kernel-private list)?
//
// Both must pass. Internal callers bypass both. DataExec stays
// internal-only — the impl already enforces that, and there's no
// per-table check to run on a free-form SQL string.

func (g *capabilityGuard) DataGet(ctx context.Context, table string, id uint) (map[string]any, error) {
	if err := checkCapability(ctx, "data:read"); err != nil {
		return nil, err
	}
	if err := checkTableAccess(ctx, table); err != nil {
		return nil, err
	}
	return g.inner.DataGet(ctx, table, id)
}

func (g *capabilityGuard) DataQuery(ctx context.Context, table string, query DataStoreQuery) (*DataStoreResult, error) {
	if err := checkCapability(ctx, "data:read"); err != nil {
		return nil, err
	}
	if err := checkTableAccess(ctx, table); err != nil {
		return nil, err
	}
	return g.inner.DataQuery(ctx, table, query)
}

func (g *capabilityGuard) DataCreate(ctx context.Context, table string, data map[string]any) (map[string]any, error) {
	if err := checkCapability(ctx, "data:write"); err != nil {
		return nil, err
	}
	if err := checkTableAccess(ctx, table); err != nil {
		return nil, err
	}
	return g.inner.DataCreate(ctx, table, data)
}

func (g *capabilityGuard) DataUpdate(ctx context.Context, table string, id uint, data map[string]any) error {
	if err := checkCapability(ctx, "data:write"); err != nil {
		return err
	}
	if err := checkTableAccess(ctx, table); err != nil {
		return err
	}
	return g.inner.DataUpdate(ctx, table, id, data)
}

func (g *capabilityGuard) DataDelete(ctx context.Context, table string, id uint) error {
	if err := checkCapability(ctx, "data:delete"); err != nil {
		// Fall back to data:write for backwards compatibility.
		if err2 := checkCapability(ctx, "data:write"); err2 != nil {
			return err
		}
	}
	if err := checkTableAccess(ctx, table); err != nil {
		return err
	}
	return g.inner.DataDelete(ctx, table, id)
}

func (g *capabilityGuard) DataExec(ctx context.Context, sql string, args ...any) (int64, error) {
	if err := checkCapability(ctx, "data:write"); err != nil {
		return 0, err
	}
	return g.inner.DataExec(ctx, sql, args...)
}

// --- Node Types ---

func (g *capabilityGuard) RegisterNodeType(ctx context.Context, input NodeTypeInput) (*NodeType, error) {
	if err := checkCapability(ctx, "nodetypes:write"); err != nil {
		return nil, err
	}
	return g.inner.RegisterNodeType(ctx, input)
}

func (g *capabilityGuard) GetNodeType(ctx context.Context, slug string) (*NodeType, error) {
	if err := checkCapability(ctx, "nodetypes:read"); err != nil {
		return nil, err
	}
	return g.inner.GetNodeType(ctx, slug)
}

func (g *capabilityGuard) ListNodeTypes(ctx context.Context) ([]*NodeType, error) {
	if err := checkCapability(ctx, "nodetypes:read"); err != nil {
		return nil, err
	}
	return g.inner.ListNodeTypes(ctx)
}

func (g *capabilityGuard) UpdateNodeType(ctx context.Context, slug string, input NodeTypeInput) (*NodeType, error) {
	if err := checkCapability(ctx, "nodetypes:write"); err != nil {
		return nil, err
	}
	return g.inner.UpdateNodeType(ctx, slug, input)
}

func (g *capabilityGuard) DeleteNodeType(ctx context.Context, slug string) error {
	if err := checkCapability(ctx, "nodetypes:write"); err != nil {
		return err
	}
	return g.inner.DeleteNodeType(ctx, slug)
}

// --- File Storage ---

func (g *capabilityGuard) StoreFile(ctx context.Context, path string, data []byte) (string, error) {
	if err := checkCapability(ctx, "files:write"); err != nil {
		return "", err
	}
	return g.inner.StoreFile(ctx, path, data)
}

func (g *capabilityGuard) DeleteFile(ctx context.Context, path string) error {
	if err := checkCapability(ctx, "files:delete"); err != nil {
		return err
	}
	return g.inner.DeleteFile(ctx, path)
}
