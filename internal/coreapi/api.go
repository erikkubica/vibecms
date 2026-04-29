package coreapi

import (
	"context"
	"io"
	"time"
)

type CoreAPI interface {
	// Nodes
	GetNode(ctx context.Context, id uint) (*Node, error)
	QueryNodes(ctx context.Context, query NodeQuery) (*NodeList, error)
	ListTaxonomyTerms(ctx context.Context, nodeType string, taxonomy string) ([]string, error)
	CreateNode(ctx context.Context, input NodeInput) (*Node, error)
	UpdateNode(ctx context.Context, id uint, input NodeInput) (*Node, error)
	DeleteNode(ctx context.Context, id uint) error

	// Taxonomies (Definitions)
	RegisterTaxonomy(ctx context.Context, input TaxonomyInput) (*Taxonomy, error)
	GetTaxonomy(ctx context.Context, slug string) (*Taxonomy, error)
	ListTaxonomies(ctx context.Context) ([]*Taxonomy, error)
	UpdateTaxonomy(ctx context.Context, slug string, input TaxonomyInput) (*Taxonomy, error)
	DeleteTaxonomy(ctx context.Context, slug string) error

	// Taxonomy Terms
	ListTerms(ctx context.Context, nodeType string, taxonomy string) ([]*TaxonomyTerm, error)
	GetTerm(ctx context.Context, id uint) (*TaxonomyTerm, error)
	CreateTerm(ctx context.Context, term *TaxonomyTerm) (*TaxonomyTerm, error)
	UpdateTerm(ctx context.Context, id uint, updates map[string]interface{}) (*TaxonomyTerm, error)
	DeleteTerm(ctx context.Context, id uint) error

	// Settings
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	GetSettings(ctx context.Context, prefix string) (map[string]string, error)
	// Locale-aware variants. Pass "" for locale to read/write the fallback
	// row that applies across all languages. GetSettingLoc / GetSettingsLoc
	// fall back to "" when no per-locale row exists.
	GetSettingLoc(ctx context.Context, key, locale string) (string, error)
	SetSettingLoc(ctx context.Context, key, locale, value string) error
	GetSettingsLoc(ctx context.Context, prefix, locale string) (map[string]string, error)

	// Events
	Emit(ctx context.Context, action string, payload map[string]any) error
	Subscribe(ctx context.Context, action string, handler EventHandler) (UnsubscribeFunc, error)

	// Email
	SendEmail(ctx context.Context, req EmailRequest) error

	// Menus
	GetMenu(ctx context.Context, slug string) (*Menu, error)
	GetMenus(ctx context.Context) ([]*Menu, error)
	CreateMenu(ctx context.Context, input MenuInput) (*Menu, error)
	UpdateMenu(ctx context.Context, slug string, input MenuInput) (*Menu, error)
	// UpsertMenu creates the menu if it doesn't exist and then replaces its
	// items with input.Items. Idempotent — safe to call from theme seeds on
	// every activation. Name/Slug are taken from input; items tree may nest
	// one level deep (Children).
	UpsertMenu(ctx context.Context, input MenuInput) (*Menu, error)
	DeleteMenu(ctx context.Context, slug string) error

	// Routes
	RegisterRoute(ctx context.Context, method, path string, meta RouteMeta) error
	RemoveRoute(ctx context.Context, method, path string) error

	// Filters
	RegisterFilter(ctx context.Context, name string, priority int, handler FilterHandler) (UnsubscribeFunc, error)
	ApplyFilters(ctx context.Context, name string, value any) (any, error)

	// Media
	UploadMedia(ctx context.Context, req MediaUploadRequest) (*MediaFile, error)
	GetMedia(ctx context.Context, id uint) (*MediaFile, error)
	QueryMedia(ctx context.Context, query MediaQuery) ([]*MediaFile, error)
	DeleteMedia(ctx context.Context, id uint) error

	// Users (read-only for extensions)
	GetUser(ctx context.Context, id uint) (*User, error)
	QueryUsers(ctx context.Context, query UserQuery) ([]*User, error)

	// HTTP (outbound)
	Fetch(ctx context.Context, req FetchRequest) (*FetchResponse, error)

	// Log
	Log(ctx context.Context, level, message string, fields map[string]any) error

	// Data Store — extension-scoped table operations
	DataGet(ctx context.Context, table string, id uint) (map[string]any, error)
	DataQuery(ctx context.Context, table string, query DataStoreQuery) (*DataStoreResult, error)
	DataCreate(ctx context.Context, table string, data map[string]any) (map[string]any, error)
	DataUpdate(ctx context.Context, table string, id uint, data map[string]any) error
	DataDelete(ctx context.Context, table string, id uint) error
	DataExec(ctx context.Context, sql string, args ...any) (int64, error)

	// Node Types
	RegisterNodeType(ctx context.Context, input NodeTypeInput) (*NodeType, error)
	GetNodeType(ctx context.Context, slug string) (*NodeType, error)
	ListNodeTypes(ctx context.Context) ([]*NodeType, error)
	UpdateNodeType(ctx context.Context, slug string, input NodeTypeInput) (*NodeType, error)
	DeleteNodeType(ctx context.Context, slug string) error

	// File Storage
	StoreFile(ctx context.Context, path string, data []byte) (string, error)
	DeleteFile(ctx context.Context, path string) error
}

type EventHandler func(action string, payload map[string]any)
type FilterHandler func(value any) any
type UnsubscribeFunc func()

type Node struct {
	ID           uint              `json:"id"`
	UUID         string            `json:"uuid"`
	ParentID     *uint             `json:"parent_id,omitempty"`
	NodeType     string            `json:"node_type"`
	Status       string            `json:"status"`
	LanguageCode string            `json:"language_code"`
	Slug         string            `json:"slug"`
	FullURL      string            `json:"full_url"`
	Title        string            `json:"title"`
	FeaturedImage any               `json:"featured_image,omitempty"`
	Excerpt       string            `json:"excerpt,omitempty"`
	Taxonomies    map[string][]string `json:"taxonomies,omitempty"`
	BlocksData   any               `json:"blocks_data,omitempty"`
	FieldsData   map[string]any    `json:"fields_data,omitempty"`
	SeoSettings  map[string]string `json:"seo_settings,omitempty"`
	PublishedAt  *time.Time        `json:"published_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Translations []map[string]interface{} `json:"translations,omitempty"`
}

type NodeQuery struct {
	NodeType     string `json:"node_type,omitempty"`
	Status       string `json:"status,omitempty"`
	ParentID     *uint  `json:"parent_id,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
	Slug         string `json:"slug,omitempty"`
	Search       string `json:"search,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
	OrderBy      string `json:"order_by,omitempty"`
	Category     string `json:"category,omitempty"`
	TaxQuery     map[string][]string `json:"tax_query,omitempty"`
}

type NodeList struct {
	Nodes []*Node `json:"nodes"`
	Total int64   `json:"total"`
}

type NodeInput struct {
	ParentID     *uint             `json:"parent_id,omitempty"`
	NodeType     string            `json:"node_type,omitempty"`
	Status       string            `json:"status,omitempty"`
	LanguageCode string            `json:"language_code,omitempty"`
	Slug         string            `json:"slug,omitempty"`
	Title        string            `json:"title,omitempty"`
	LayoutSlug    string             `json:"layout_slug,omitempty"`
	FeaturedImage any               `json:"featured_image,omitempty"`
	Excerpt       string            `json:"excerpt,omitempty"`
	Taxonomies    map[string][]string `json:"taxonomies,omitempty"`
	BlocksData   any               `json:"blocks_data,omitempty"`
	FieldsData   map[string]any    `json:"fields_data,omitempty"`
	SeoSettings  map[string]string `json:"seo_settings,omitempty"`
}

type Menu struct {
	ID        uint       `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Items     []MenuItem `json:"items"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type MenuItem struct {
	ID       uint       `json:"id"`
	Label    string     `json:"label"`
	URL      string     `json:"url"`
	Target   string     `json:"target,omitempty"`
	ParentID *uint      `json:"parent_id,omitempty"`
	Position int        `json:"position"`
	Children []MenuItem `json:"children,omitempty"`
	// ItemType is "custom" (URL is authoritative) or "node" (URL is derived
	// from NodeID's current full_url at render time).
	ItemType string `json:"item_type,omitempty"`
	NodeID   *uint  `json:"node_id,omitempty"`
}

type MenuInput struct {
	Name  string     `json:"name"`
	Slug  string     `json:"slug,omitempty"`
	Items []MenuItem `json:"items,omitempty"`
}

type EmailRequest struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

type RouteMeta struct {
	ScriptPath string `json:"script_path,omitempty"`
	Handler    any    `json:"-"`
}

type MediaFile struct {
	ID        uint      `json:"id"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type MediaUploadRequest struct {
	Filename string    `json:"filename"`
	MimeType string    `json:"mime_type"`
	Body     io.Reader `json:"-"`
}

type MediaQuery struct {
	MimeType string `json:"mime_type,omitempty"`
	Search   string `json:"search,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

type User struct {
	ID         uint   `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	RoleID     *uint  `json:"role_id,omitempty"`
	RoleSlug   string `json:"role_slug,omitempty"`
	LanguageID *int   `json:"language_id,omitempty"`
}

type UserQuery struct {
	RoleSlug string `json:"role_slug,omitempty"`
	Search   string `json:"search,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

type FetchRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
}

type FetchResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

type DataStoreQuery struct {
	Where   map[string]any `json:"where,omitempty"`
	Search  string         `json:"search,omitempty"`
	OrderBy string         `json:"order_by,omitempty"`
	Limit   int            `json:"limit,omitempty"`
	Offset  int            `json:"offset,omitempty"`
	Raw     string         `json:"raw,omitempty"`
	Args    []any          `json:"args,omitempty"`
}

type DataStoreResult struct {
	Rows  []map[string]any `json:"rows"`
	Total int64            `json:"total"`
}

type NodeType struct {
	ID             int                  `json:"id"`
	Slug           string               `json:"slug"`
	Label          string               `json:"label"`
	LabelPlural    string               `json:"label_plural"`
	Icon           string               `json:"icon"`
	Description    string               `json:"description"`
	Taxonomies     []TaxonomyDefinition `json:"taxonomies,omitempty"`
	FieldSchema    []NodeTypeField      `json:"field_schema"`
	URLPrefixes    map[string]string    `json:"url_prefixes"`
	SupportsBlocks bool                 `json:"supports_blocks"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// NodeTypeField — note: both `name` and `key` JSON tags resolve to the same
// Go field. The admin UI uses `key` (the block-type convention); Tengo theme
// scripts and legacy code use `name`. NormalizeFieldSchema mirrors one into
// the other so either naming works and consumers always see both.
type NodeTypeField struct {
	Name      string           `json:"name"`
	Key       string           `json:"key,omitempty"`
	Label     string           `json:"label"`
	Type      string           `json:"type"`
	Required  bool             `json:"required"`
	Options   []interface{}    `json:"options,omitempty"`
	SubFields []NodeTypeField  `json:"sub_fields,omitempty"`
	Default   interface{}      `json:"default,omitempty"`
	Help      string           `json:"help,omitempty"`

	// Type-specific config carried through to the admin UI and templates.
	// Kept as explicit fields so JSON round-trips cleanly.
	NodeTypeFilter string   `json:"node_type_filter,omitempty"` // `node` field: single node_type slug to filter by
	NodeTypes      []string `json:"node_types,omitempty"`       // `node` field: alt multi-slug filter
	Multiple       bool     `json:"multiple,omitempty"`         // `node` / `term` / `gallery` multi-select toggle
	Taxonomy       string   `json:"taxonomy,omitempty"`         // `term` field: taxonomy slug (e.g. "trip_tag")
	TermNodeType   string   `json:"term_node_type,omitempty"`   // `term` field: owning node_type slug for the taxonomy
}

// NormalizeFieldSchema mirrors Name↔Key on every field (including recursively
// inside sub_fields) so the admin UI and template code can both rely on
// either accessor being populated.
func NormalizeFieldSchema(fields []NodeTypeField) []NodeTypeField {
	for i := range fields {
		if fields[i].Key == "" && fields[i].Name != "" {
			fields[i].Key = fields[i].Name
		} else if fields[i].Name == "" && fields[i].Key != "" {
			fields[i].Name = fields[i].Key
		}
		if len(fields[i].SubFields) > 0 {
			fields[i].SubFields = NormalizeFieldSchema(fields[i].SubFields)
		}
	}
	return fields
}

// OptionsToStrings coerces the polymorphic Options slice to strings for
// gRPC proto wire compatibility. Map-shaped {label,value} options are
// reduced to their string value.
func (f NodeTypeField) OptionsToStrings() []string {
	if len(f.Options) == 0 {
		return nil
	}
	out := make([]string, 0, len(f.Options))
	for _, o := range f.Options {
		switch v := o.(type) {
		case string:
			out = append(out, v)
		case map[string]any:
			if s, ok := v["value"].(string); ok {
				out = append(out, s)
			} else if s, ok := v["label"].(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

// OptionsFromStrings re-hydrates a []string into the polymorphic slice
// (used when decoding from gRPC proto wire).
func OptionsFromStrings(in []string) []interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make([]interface{}, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

type NodeTypeInput struct {
	Slug           string               `json:"slug,omitempty"`
	Label          string               `json:"label,omitempty"`
	LabelPlural    string               `json:"label_plural,omitempty"`
	Icon           string               `json:"icon,omitempty"`
	Description    string               `json:"description,omitempty"`
	Taxonomies     []TaxonomyDefinition `json:"taxonomies,omitempty"`
	FieldSchema    []NodeTypeField      `json:"field_schema,omitempty"`
	URLPrefixes    map[string]string    `json:"url_prefixes,omitempty"`
	SupportsBlocks *bool                `json:"supports_blocks,omitempty"`
}

type TaxonomyDefinition struct {
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Multiple bool   `json:"multiple"` // Allow multiple terms per node
}

type Taxonomy struct {
	ID           uint                 `json:"id"`
	Slug         string               `json:"slug"`
	Label        string               `json:"label"`
	LabelPlural  string               `json:"label_plural"`
	Description  string               `json:"description"`
	Hierarchical bool                 `json:"hierarchical"`
	ShowUI       bool                 `json:"show_ui"`
	NodeTypes    []string             `json:"node_types"`
	FieldSchema  []NodeTypeField      `json:"field_schema,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

type TaxonomyInput struct {
	Slug         string               `json:"slug,omitempty"`
	Label        string               `json:"label,omitempty"`
	LabelPlural  string               `json:"label_plural,omitempty"`
	Description  string               `json:"description,omitempty"`
	Hierarchical *bool                `json:"hierarchical,omitempty"`
	ShowUI       *bool                `json:"show_ui,omitempty"`
	NodeTypes    []string             `json:"node_types,omitempty"`
	FieldSchema  []NodeTypeField      `json:"field_schema,omitempty"`
}

type TaxonomyTerm struct {
	ID                 uint                   `json:"id"`
	NodeType           string                 `json:"node_type"`
	Taxonomy           string                 `json:"taxonomy"`
	LanguageCode       string                 `json:"language_code"`
	TranslationGroupID *string                `json:"translation_group_id,omitempty"`
	Slug               string                 `json:"slug"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	ParentID           *uint                  `json:"parent_id,omitempty"`
	Count              int                    `json:"count"`
	FieldsData         map[string]interface{} `json:"fields_data,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}
