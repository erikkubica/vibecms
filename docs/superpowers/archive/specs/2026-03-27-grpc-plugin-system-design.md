# gRPC Plugin System Design

**Date:** 2026-03-27
**Status:** Approved
**Goal:** Enable extensions to ship compiled Go (or any language) binaries that communicate with VibeCMS over gRPC, subscribing to events like any other handler. First use case: move SMTP email sending out of core into a plugin extension.

---

## 1. Overview

VibeCMS extensions currently support:
- **Tengo scripts** — for hooks, filters, simple logic
- **React micro-frontends** — for admin UI components

This design adds a third capability:
- **gRPC plugins** — compiled binaries (any language) that subscribe to EventBus events and handle them natively

The three layers are complementary. An extension can use any combination.

## 2. Architecture

### Plugin Lifecycle

1. Extension manifest declares plugins: `"plugins": [{"binary": "bin/smtp-sender", ...}]`
2. On extension activate: VibeCMS starts each plugin binary as a child process using HashiCorp go-plugin
3. Plugin connects over gRPC on a local Unix socket (managed by go-plugin)
4. Plugin tells core which events it subscribes to via `GetSubscriptions()` RPC
5. Core registers plugin's subscriptions in the EventBus
6. On event fire: EventBus calls plugin's `HandleEvent()` RPC
7. On extension deactivate: Core sends shutdown, kills process, removes subscriptions

### Protocol (protobuf)

```protobuf
syntax = "proto3";
package plugin;

service ExtensionPlugin {
  rpc GetSubscriptions(Empty) returns (SubscriptionList);
  rpc HandleEvent(EventRequest) returns (EventResponse);
  rpc Shutdown(Empty) returns (Empty);
}

message Empty {}

message Subscription {
  string event_name = 1;
  int32 priority = 2;
}

message SubscriptionList {
  repeated Subscription subscriptions = 1;
}

message EventRequest {
  string action = 1;
  bytes payload = 2; // JSON-encoded event payload
}

message EventResponse {
  bool handled = 1;
  string error = 2;
  bytes result = 3; // JSON-encoded result (optional)
}
```

### Plugin Manager

New `internal/cms/plugin_manager.go`:
- Starts plugin processes via `go-plugin`
- Queries subscriptions
- Registers handlers in EventBus
- Manages lifecycle (start/stop/health)
- Called by ExtensionHandler on activate/deactivate

### Event Flow for Email

**Before (current):**
Core dispatcher → hardcoded SMTP/Resend Go code

**After:**
Core dispatcher publishes `email.send` event → EventBus → Plugin's HandleEvent gRPC call → Plugin sends email via SMTP/HTTP → returns result

## 3. Manifest Changes

```json
{
  "name": "SMTP Provider",
  "plugins": [
    {
      "binary": "bin/smtp-provider",
      "events": ["email.send"]
    }
  ],
  "admin_ui": { ... }
}
```

- `plugins` is an array — extension can ship multiple binaries
- Each plugin declares which events it handles
- `binary` path is relative to extension root

## 4. Email System Refactor

### Core changes:
- Remove `internal/email/smtp.go` and `internal/email/resend.go`
- Remove `internal/email/provider.go` (Provider interface, factory)
- Simplify `internal/email/dispatcher.go`:
  - No longer instantiates providers
  - Instead publishes `email.send` event with full payload
  - Payload: `{to, subject, html, from_email, from_name, action, rule_id, template_slug}`
  - Logging moves to: dispatcher logs "sent" if event handled, "failed" if error returned

### SMTP Extension (gRPC plugin):
- `extensions/smtp-provider/cmd/plugin/main.go` — Go binary implementing ExtensionPlugin service
- Subscribes to `email.send`
- Reads its settings from payload or config file
- Sends via Go's `net/smtp`

### Resend Extension (Tengo script):
- `extensions/resend-provider/scripts/extension.tengo` — subscribes to `email.send`
- Uses new `cms/fetch` Tengo module for outbound HTTP
- POSTs to `https://api.resend.com/emails`

## 5. New Tengo Module: cms/fetch

For Tengo-based extensions that need outbound HTTP:

```
result := fetch.post(url, {
  headers: {"Authorization": "Bearer " + api_key, "Content-Type": "application/json"},
  body: json_string
})
// result.status_code, result.body, result.error
```

Functions: `fetch.get(url, options)`, `fetch.post(url, options)`, `fetch.put(url, options)`, `fetch.delete(url, options)`

Timeout: 30 seconds. Sandboxed to HTTP/HTTPS only.

## 6. Extension Settings Access for Plugins

Plugins need to read their settings (SMTP host, API keys, etc.). Two approaches:

**Chosen: Settings passed in event payload.** The dispatcher includes `provider_settings` in the email.send payload — a map of the active provider extension's settings. The plugin reads them from the event. Simple, no additional RPC needed.

The dispatcher reads `site_settings.email_provider` to know which extension is the active provider, loads that extension's settings (`ext.<slug>.*`), and includes them in the payload.

## 7. Safety

| Scenario | Handling |
|----------|----------|
| Plugin binary crashes | go-plugin detects, logs error. Event returns error. Email logged as failed. |
| Plugin hangs | go-plugin timeout (30s). Process killed. Event returns error. |
| Plugin not found (binary missing) | Logged on activate. Extension stays active (UI still works) but plugin events fail. |
| Multiple plugins claim same event | All get called, like any EventBus subscriber. For email.send, first to return handled=true wins. |
| Deactivate while handling event | go-plugin handles graceful shutdown. In-flight RPCs complete or timeout. |

## 8. Build Process

SMTP plugin binary must be compiled during Docker build:

```dockerfile
# Build SMTP plugin
RUN cd extensions/smtp-provider && CGO_ENABLED=0 go build -o bin/smtp-provider ./cmd/plugin/
```

The extension's Go code lives in the same module as VibeCMS (shares go.mod) so it can import the shared plugin interface from `pkg/plugin/`.

## 9. Out of Scope

- Plugin auto-restart on crash (future)
- Plugin resource limits (CPU/memory)
- Plugin marketplace
- Bidirectional streaming (plugins calling back into core)
