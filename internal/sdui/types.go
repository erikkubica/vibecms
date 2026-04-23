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
}

// SSEEvent is sent over the Server-Sent Events stream.
type SSEEvent struct {
	Type string      `json:"type"` // UI_STALE, NODE_TYPE_CHANGED, NOTIFY
	Data interface{} `json:"data"`
}
