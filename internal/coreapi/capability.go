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

func (g *capabilityGuard) DataGet(ctx context.Context, table string, id uint) (map[string]any, error) {
	if err := checkCapability(ctx, "data:read"); err != nil {
		return nil, err
	}
	return g.inner.DataGet(ctx, table, id)
}

func (g *capabilityGuard) DataQuery(ctx context.Context, table string, query DataStoreQuery) (*DataStoreResult, error) {
	if err := checkCapability(ctx, "data:read"); err != nil {
		return nil, err
	}
	return g.inner.DataQuery(ctx, table, query)
}

func (g *capabilityGuard) DataCreate(ctx context.Context, table string, data map[string]any) (map[string]any, error) {
	if err := checkCapability(ctx, "data:write"); err != nil {
		return nil, err
	}
	return g.inner.DataCreate(ctx, table, data)
}

func (g *capabilityGuard) DataUpdate(ctx context.Context, table string, id uint, data map[string]any) error {
	if err := checkCapability(ctx, "data:write"); err != nil {
		return err
	}
	return g.inner.DataUpdate(ctx, table, id, data)
}

func (g *capabilityGuard) DataDelete(ctx context.Context, table string, id uint) error {
	if err := checkCapability(ctx, "data:write"); err != nil {
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
