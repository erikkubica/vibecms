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
	CreateNode(ctx context.Context, input NodeInput) (*Node, error)
	UpdateNode(ctx context.Context, id uint, input NodeInput) (*Node, error)
	DeleteNode(ctx context.Context, id uint) error

	// Settings
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	GetSettings(ctx context.Context, prefix string) (map[string]string, error)

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
	BlocksData   any               `json:"blocks_data,omitempty"`
	FieldsData   map[string]any    `json:"fields_data,omitempty"`
	SeoSettings  map[string]string `json:"seo_settings,omitempty"`
	PublishedAt  *time.Time        `json:"published_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
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
