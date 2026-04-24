package sdui

// BootManifest is the single source of truth returned by GET /admin/api/boot
type BootManifest struct {
	Version    string         `json:"version"`
	User       BootUser       `json:"user"`
	Extensions []BootExt      `json:"extensions"`
	Navigation []NavItem      `json:"navigation"`
	NodeTypes  []BootNodeType `json:"node_types"`
}

// BootUser represents the authenticated user in the boot manifest.
type BootUser struct {
	ID           int                    `json:"id"`
	Email        string                 `json:"email"`
	FullName     string                 `json:"full_name,omitempty"`
	Role         string                 `json:"role"`
	Capabilities map[string]interface{} `json:"capabilities"`
}

// BootExt represents an active extension in the boot manifest.
type BootExt struct {
	Slug       string   `json:"slug"`
	Name       string   `json:"name"`
	Entry      string   `json:"entry,omitempty"`
	Components []string `json:"components,omitempty"`
}

// NavItem represents a single item in the admin navigation sidebar.
type NavItem struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Icon      string    `json:"icon,omitempty"`
	IsSection bool      `json:"is_section,omitempty"` // true = non-clickable section header
	Section   string    `json:"section,omitempty"`    // content, design, development, settings — for grouping extension items into sections
	Path      string    `json:"path,omitempty"`
	Children  []NavItem `json:"children,omitempty"`
}

// BootNodeType represents a content type in the boot manifest.
type BootNodeType struct {
	Slug           string `json:"slug"`
	Label          string `json:"label"`
	LabelPlural    string `json:"label_plural"`
	Icon           string `json:"icon"`
	SupportsBlocks bool   `json:"supports_blocks"`
}

// LayoutNode is a node in the SDUI layout tree.
type LayoutNode struct {
	Type     string                 `json:"type"`
	Props    map[string]interface{} `json:"props,omitempty"`
	Children []LayoutNode           `json:"children,omitempty"`
	Actions  map[string]ActionDef   `json:"actions,omitempty"`
}

// ActionDef defines a side-effect triggered by user interaction.
type ActionDef struct {
	Type    string                 `json:"type"`
	Method  string                 `json:"method,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
	Message string                 `json:"message,omitempty"`
	Title   string                 `json:"title,omitempty"`
	Variant string                 `json:"variant,omitempty"`
	To      string                 `json:"to,omitempty"`
	Keys    []string               `json:"keys,omitempty"`
	Key     string                 `json:"key,omitempty"`
	Value   interface{}            `json:"value,omitempty"`

	// For SEQUENCE actions
	Steps []ActionDef `json:"steps,omitempty"`

	// For CORE_API
	Bind string `json:"bind,omitempty"`

	// For CONFIRM
	Then *ActionDef `json:"then,omitempty"`

	// For CORE_API mutations: override the default "Saved." / "Failed: <err>"
	// toasts. Set Silent=true to suppress all user-facing feedback.
	SuccessMessage string `json:"success_message,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	Silent         bool   `json:"silent,omitempty"`
}

// SSEEvent is sent over the Server-Sent Events stream.
//
// Type values:
//   - CONNECTED         — sent once on stream open
//   - NAV_STALE         — sidebar / boot manifest needs a refetch
//     (extension toggled, theme activated, node type changed)
//   - ENTITY_CHANGED    — a CRUD event on a specific entity; Entity + ID + Op are set
//   - SETTING_CHANGED   — a setting was written; Key is set
//   - NOTIFY            — pass-through notification payload for toasts
//   - UI_STALE          — coarse fallback, equivalent to "refetch boot + every layout"
//
// The client maps these to TanStack Query invalidations via a routing table in
// admin-ui/src/hooks/use-sse.ts. Keep the shape stable; prefer extending Data
// over renaming top-level fields.
type SSEEvent struct {
	Type   string      `json:"type"`
	Entity string      `json:"entity,omitempty"` // e.g. "user", "node", "menu", "layout", "layout_block", "block_type", "node_type", "term", "taxonomy", "template", "role"
	ID     interface{} `json:"id,omitempty"`     // numeric or string id from events.Payload
	Op     string      `json:"op,omitempty"`     // created | updated | deleted | published | unpublished | login | registered
	Key    string      `json:"key,omitempty"`    // SETTING_CHANGED: the setting key
	Data   interface{} `json:"data,omitempty"`   // pass-through payload (NOTIFY, or extra context)
}
