# CoreAPI Design — Unified Extension Interface

**Date:** 2026-03-27
**Status:** Approved

## Overview

A single Go interface (`CoreAPI`) providing every CMS capability to all extension types equally. Three adapters wrap it: Tengo modules (`core/*`), gRPC host service (via go-plugin's GRPCBroker), and internal Go calls. Manifest-declared capabilities gate access per extension.

This is a clean-break redesign. Tengo modules move from `cms/*` to `core/*` namespace. No backward compatibility with the old API.

## Architecture

```
┌─────────────────────────────────────────────────┐
│              CoreAPI (Go interface)              │
│  Nodes · Settings · Events · Email · Menus      │
│  Routes · Filters · Media · Users · Log · HTTP  │
├─────────┬──────────────┬────────────────────────┤
│ Tengo   │ gRPC Host    │ Internal (Go)          │
│ Adapter │ Service      │ Direct calls           │
│ core/*  │ VibeCMSHost  │ (theme renderer, etc.) │
└────┬────┴──────┬───────┴────────────┬───────────┘
     │           │                    │
  .tgo scripts  gRPC plugins    CMS internals
```

## CoreAPI Go Interface

```go
type CoreAPI interface {
    // Nodes
    GetNode(ctx context.Context, id uint) (*Node, error)
    QueryNodes(ctx context.Context, query NodeQuery) ([]*Node, error)
    CreateNode(ctx context.Context, input NodeInput) (*Node, error)
    UpdateNode(ctx context.Context, id uint, input NodeInput) (*Node, error)
    DeleteNode(ctx context.Context, id uint) error

    // Settings
    GetSetting(ctx context.Context, key string) (string, error)
    SetSetting(ctx context.Context, key, value string) error
    GetSettings(ctx context.Context, prefix string) (map[string]string, error)

    // Events
    Emit(ctx context.Context, action string, payload map[string]any) error
    Subscribe(ctx context.Context, action string, handler EventHandler) error

    // Email
    SendEmail(ctx context.Context, req EmailRequest) error

    // Menus
    GetMenu(ctx context.Context, slug string) (*Menu, error)
    GetMenus(ctx context.Context) ([]*Menu, error)
    CreateMenu(ctx context.Context, input MenuInput) (*Menu, error)
    UpdateMenu(ctx context.Context, slug string, input MenuInput) (*Menu, error)
    DeleteMenu(ctx context.Context, slug string) error

    // Routes
    RegisterRoute(ctx context.Context, method, path string, handler RouteHandler) error
    RemoveRoute(ctx context.Context, method, path string) error

    // Filters
    RegisterFilter(ctx context.Context, name string, handler FilterHandler) error
    ApplyFilters(ctx context.Context, name string, value any) (any, error)

    // Media
    UploadMedia(ctx context.Context, req MediaUploadRequest) (*MediaFile, error)
    GetMedia(ctx context.Context, id uint) (*MediaFile, error)
    QueryMedia(ctx context.Context, query MediaQuery) ([]*MediaFile, error)
    DeleteMedia(ctx context.Context, id uint) error

    // Users (read-only for extensions)
    GetUser(ctx context.Context, id uint) (*User, error)
    QueryUsers(ctx context.Context, query UserQuery) ([]*User, error)
    GetCurrentUser(ctx context.Context) (*User, error)

    // HTTP (outbound)
    Fetch(ctx context.Context, req FetchRequest) (*FetchResponse, error)

    // Log
    Log(ctx context.Context, level, message string, fields map[string]any) error
}
```

## Capability System

### Manifest Declaration

Extensions declare required capabilities in `extension.json`:

```json
{
    "name": "My Extension",
    "slug": "my-extension",
    "capabilities": [
        "nodes:read",
        "nodes:write",
        "settings:read",
        "email:send",
        "events:emit"
    ]
}
```

### Capability Keys

| Domain   | Capabilities                           |
|----------|----------------------------------------|
| Nodes    | `nodes:read`, `nodes:write`, `nodes:delete` |
| Settings | `settings:read`, `settings:write`      |
| Events   | `events:emit`, `events:subscribe`      |
| Email    | `email:send`                           |
| Menus    | `menus:read`, `menus:write`, `menus:delete` |
| Routes   | `routes:register`                      |
| Filters  | `filters:register`, `filters:apply`    |
| Media    | `media:read`, `media:write`, `media:delete` |
| Users    | `users:read`                           |
| HTTP     | `http:fetch`                           |
| Log      | `log:write`                            |

### Enforcement

A `capabilityGuard` wraps the CoreAPI implementation. Each method checks the caller's allowed capabilities before executing. Caller identity is embedded in `context.Context` via a `CallerInfo` struct:

```go
type CallerInfo struct {
    Slug         string           // extension slug
    Type         string           // "tengo", "grpc", "internal"
    Capabilities map[string]bool  // allowed capabilities from manifest
}
```

- Unauthorized calls return `ErrCapabilityDenied`
- Internal callers (Type="internal") bypass all checks
- Capabilities are loaded from the extension manifest on activation and cached

## Adapter Details

### Tengo Adapter (`core/*` modules)

Replaces the current 11 `cms/*` modules in `internal/scripting/api_*.go`. Each Tengo module becomes a thin wrapper:

1. Convert Tengo values to Go types
2. Call CoreAPI method
3. Convert Go return values back to Tengo values

Module mapping (old -> new):
- `cms/nodes` -> `core/nodes`
- `cms/settings` -> `core/settings`
- `cms/events` -> `core/events`
- `cms/email` -> `core/email`
- `cms/menus` -> `core/menus`
- `cms/routing` -> `core/routes`
- `cms/filters` -> `core/filters`
- `cms/fetch` + `cms/http` -> `core/http`
- `cms/helpers` -> `core/helpers` (utility functions, may not need CoreAPI)
- `cms/log` -> `core/log`

CallerInfo injected per-script execution context with the extension's slug and capabilities.

### gRPC Host Service (VibeCMSHost)

New proto service exposed to plugins via go-plugin's `GRPCBroker`:

- Plugin's `GRPCClient` receives a broker connection to the host on startup
- Methods mirror CoreAPI 1:1, payloads as protobuf messages
- CallerInfo derived from the plugin's registered slug
- Streaming not needed initially — all request/response

The plugin interface changes from:
```go
// Old: plugins can only handle events
type ExtensionPlugin interface {
    GetSubscriptions() ([]*pb.Subscription, error)
    HandleEvent(action string, payload []byte) (*pb.EventResponse, error)
    Shutdown() error
}
```

To:
```go
// New: plugins handle events AND receive a host API client
type ExtensionPlugin interface {
    GetSubscriptions() ([]*pb.Subscription, error)
    HandleEvent(action string, payload []byte) (*pb.EventResponse, error)
    Initialize(host CoreAPIClient) error  // receives host API
    Shutdown() error
}
```

### Internal Go Adapter

Direct struct implementing CoreAPI backed by GORM, event bus, and email dispatcher. Used by theme renderer, admin handlers, and any internal code. Bypasses capability checks (trusted internal caller).

## Error Types

```go
var (
    ErrCapabilityDenied = errors.New("capability denied")
    ErrNotFound         = errors.New("not found")
    ErrValidation       = errors.New("validation error")
    ErrInternal         = errors.New("internal error")
)
```

All errors carry a code + message. Mapped to:
- gRPC status codes (PermissionDenied, NotFound, InvalidArgument, Internal)
- Tengo error values (returned as error objects scripts can check)

## File Structure

```
internal/
  coreapi/
    api.go             # CoreAPI interface + all data types (Node, NodeQuery, etc.)
    impl.go            # Implementation backed by GORM/bus/email
    impl_nodes.go      # Node methods
    impl_settings.go   # Settings methods
    impl_events.go     # Events methods
    impl_email.go      # Email methods
    impl_menus.go      # Menu methods
    impl_routes.go     # Route methods
    impl_filters.go    # Filter methods
    impl_media.go      # Media methods
    impl_users.go      # User methods
    impl_http.go       # Outbound HTTP methods
    impl_log.go        # Logging methods
    capability.go      # Capability guard wrapper
    errors.go          # Error types
    context.go         # CallerInfo context helpers
proto/
  coreapi/
    vibecms_coreapi.proto  # VibeCMSHost service definition
internal/
  coreapi/
    grpc_server.go     # gRPC server implementing VibeCMSHost
    grpc_client.go     # gRPC client for plugin-side usage
    tengo_adapter.go   # Tengo module factory (core/* modules)
```

## Changes to Existing Code

1. **`internal/scripting/api_*.go`** (11 files) — Deleted, replaced by `coreapi/tengo_adapter.go`
2. **`internal/scripting/engine.go`** — Module map updated from `cms/*` to `core/*`, receives CoreAPI instance
3. **`pkg/plugin/plugin.go`** — ExtensionPlugin interface gains `Initialize(host CoreAPIClient)` method
4. **`pkg/plugin/grpc.go`** — Updated to broker host service connection
5. **`internal/cms/plugin_manager.go`** — Passes CoreAPI instance when starting plugins via broker
6. **`internal/cms/extension_loader.go`** — Validates `capabilities` from manifest, stores in DB
7. **`internal/cms/extension_handler.go`** — Returns capabilities in manifest API response
8. **`proto/plugin/vibecms_plugin.proto`** — Updated plugin interface with Initialize RPC
9. **Extension manifests** (`extension.json`) — New `capabilities` array field
10. **Existing `.tgo` scripts** — Update imports from `cms/*` to `core/*`

## Data Flow: gRPC Plugin Reads a Node

```
Plugin binary                    VibeCMS Host
    │                                │
    ├─ GetNode(id=42) ──────────────>│
    │   (proto request via broker)   │
    │                                ├─ capabilityGuard.check("nodes:read")
    │                                ├─ coreImpl.GetNode(ctx, 42)
    │                                ├─ GORM query -> PostgreSQL
    │                                │
    │<── NodeResponse ──────────────┤
    │   (proto response)             │
```

## Data Flow: Tengo Script Sends Email

```
.tgo script                      VibeCMS Host
    │                                │
    ├─ core/email.send({...}) ──────>│
    │   (Tengo adapter)              │
    │                                ├─ capabilityGuard.check("email:send")
    │                                ├─ coreImpl.SendEmail(ctx, req)
    │                                ├─ email dispatcher -> plugin provider
    │                                │
    │<── true/error ────────────────┤
    │   (Tengo return value)         │
```
