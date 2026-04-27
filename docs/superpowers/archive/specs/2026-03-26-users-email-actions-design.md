# Users, Roles, Email System & System Actions — Design Spec

## Overview

Four interconnected subsystems that make VibeCMS a fully-featured, extensible CMS:

1. **System Actions** — in-process event bus for CRUD events across all entities
2. **Roles & Permissions** — JSONB capability-based RBAC with per-node-type granularity
3. **Email System** — modular providers (SMTP/Resend), templates, rules, and logs
4. **Admin UI** — management pages for users, roles, email templates/rules/logs

---

## 1. System Actions (Event Bus)

### 1.1 `system_actions` Table

```sql
CREATE TABLE system_actions (
    id          SERIAL PRIMARY KEY,
    slug        VARCHAR(100) UNIQUE NOT NULL,   -- e.g. "node.created"
    label       VARCHAR(150) NOT NULL,          -- e.g. "Node Created"
    category    VARCHAR(50) NOT NULL,           -- user, node, layout, layout_block, block_type, menu, system
    description TEXT NOT NULL DEFAULT '',
    payload_schema JSONB NOT NULL DEFAULT '[]', -- describes available template variables
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 1.2 Seeded Actions

**User actions:**
- `user.registered` — payload: `user.*`
- `user.updated` — payload: `user.*`
- `user.deleted` — payload: `user.*`
- `user.login` — payload: `user.*`, `ip_address`, `user_agent`

**Node actions:**
- `node.created` — payload: `node.*`, `node_type`, `author.*`
- `node.updated` — payload: `node.*`, `node_type`, `author.*`, `editor.*`
- `node.published` — payload: `node.*`, `node_type`, `author.*`
- `node.unpublished` — payload: `node.*`, `node_type`, `author.*`
- `node.deleted` — payload: `node.*`, `node_type`, `author.*`

**Layout actions:**
- `layout.created`, `layout.updated`, `layout.deleted` — payload: `layout.*`, `editor.*`

**Layout block actions:**
- `layout_block.created`, `layout_block.updated`, `layout_block.deleted` — payload: `layout_block.*`, `editor.*`

**Block type actions:**
- `block_type.created`, `block_type.updated`, `block_type.deleted` — payload: `block_type.*`, `editor.*`

**Menu actions:**
- `menu.created`, `menu.updated`, `menu.deleted` — payload: `menu.*`, `editor.*`

### 1.3 Go Event Bus

```go
// internal/events/bus.go

type EventPayload map[string]interface{}

type EventHandler func(action string, payload EventPayload)

type EventBus struct {
    mu       sync.RWMutex
    handlers map[string][]EventHandler
}

func (b *EventBus) Publish(action string, payload EventPayload)
func (b *EventBus) Subscribe(action string, handler EventHandler)
func (b *EventBus) SubscribeAll(handler EventHandler)  // wildcard listener
```

- Handlers run in goroutines (non-blocking to the triggering request)
- Panics in handlers are recovered and logged
- `SubscribeAll` is for the email dispatcher — it checks rules for every event

### 1.4 Integration Points

Events are fired from existing services after successful DB operations:

- `ContentService.Create()` → `eventBus.Publish("node.created", ...)`
- `ContentService.Update()` → `eventBus.Publish("node.updated", ...)` (+ "node.published"/"node.unpublished" if status changed)
- `ContentService.Delete()` → `eventBus.Publish("node.deleted", ...)`
- `UserHandler.CreateUser()` → `eventBus.Publish("user.registered", ...)`
- `PageAuthHandler.ProcessRegister()` → `eventBus.Publish("user.registered", ...)`
- `PageAuthHandler.ProcessLogin()` → `eventBus.Publish("user.login", ...)`
- `LayoutService`, `LayoutBlockService`, `BlockTypeService`, `MenuService` — same pattern for their CRUD ops

---

## 2. Roles & Permissions

### 2.1 `roles` Table

```sql
CREATE TABLE roles (
    id           SERIAL PRIMARY KEY,
    slug         VARCHAR(50) UNIQUE NOT NULL,
    name         VARCHAR(100) NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    is_system    BOOLEAN NOT NULL DEFAULT false,
    capabilities JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 2.2 Capabilities JSONB Schema

```json
{
  "admin_access": true,
  "manage_users": false,
  "manage_roles": false,
  "manage_settings": false,
  "manage_menus": true,
  "manage_layouts": true,
  "manage_email": false,
  "nodes": {
    "page":     { "access": "write", "scope": "all" },
    "post":     { "access": "write", "scope": "own" },
    "property": { "access": "none" }
  },
  "default_node_access": { "access": "read", "scope": "all" },
  "email_subscriptions": ["node.published", "user.registered"]
}
```

**Access levels:** `none`, `read`, `write` (write implies read)
**Scope:** `all` (see/edit everything), `own` (only nodes where author_id = user.id)
**`default_node_access`:** applies to any node type not explicitly listed in `nodes`
**`email_subscriptions`:** which action emails this role is eligible to receive

### 2.3 Seeded Default Roles

**admin** (is_system: true)
```json
{
  "admin_access": true,
  "manage_users": true,
  "manage_roles": true,
  "manage_settings": true,
  "manage_menus": true,
  "manage_layouts": true,
  "manage_email": true,
  "default_node_access": { "access": "write", "scope": "all" },
  "email_subscriptions": [
    "user.registered", "user.deleted",
    "node.created", "node.updated", "node.published", "node.deleted"
  ]
}
```

**editor** (is_system: true)
```json
{
  "admin_access": true,
  "manage_users": false,
  "manage_roles": false,
  "manage_settings": false,
  "manage_menus": true,
  "manage_layouts": false,
  "manage_email": false,
  "default_node_access": { "access": "write", "scope": "all" },
  "email_subscriptions": [
    "node.created", "node.published"
  ]
}
```

**author** (is_system: true)
```json
{
  "admin_access": true,
  "manage_users": false,
  "manage_roles": false,
  "manage_settings": false,
  "manage_menus": false,
  "manage_layouts": false,
  "manage_email": false,
  "default_node_access": { "access": "write", "scope": "own" },
  "email_subscriptions": [
    "node.published"
  ]
}
```

**member** (is_system: true)
```json
{
  "admin_access": false,
  "manage_users": false,
  "manage_roles": false,
  "manage_settings": false,
  "manage_menus": false,
  "manage_layouts": false,
  "manage_email": false,
  "default_node_access": { "access": "read", "scope": "all" },
  "email_subscriptions": []
}
```

### 2.4 User Model Change

Current `role VARCHAR(50)` field becomes `role_id INT` foreign key to `roles.id`.

Migration:
1. Create `roles` table and seed default roles
2. Add `role_id` column to `users` (nullable initially)
3. Map existing `role` values: `"admin"` → admin role id, everything else → editor role id
4. Make `role_id` NOT NULL, drop old `role` column

### 2.5 Middleware Changes

- `AuthRequired` loads user + role (with capabilities) in one join query
- New helper: `HasCapability(user, "manage_users") bool` — checks capabilities JSON
- New helper: `NodeAccess(user, nodeType) (access string, scope string)` — resolves per-node-type access from capabilities, falling back to `default_node_access`
- Node list endpoints filter by scope: if `scope = "own"`, add `WHERE author_id = ?`
- Node write endpoints check access level before allowing create/update/delete

### 2.6 Content Node Change

Add `author_id INT` foreign key to `content_nodes` table referencing `users.id`. Set to current user on node creation. Needed for `scope: "own"` filtering.

---

## 3. Email System

### 3.1 Email Provider Interface

```go
// internal/email/provider.go

type Email struct {
    To      []string
    Subject string
    HTML    string
}

type Provider interface {
    Name() string
    Send(email Email) error
}
```

**Built-in providers:**

- `SMTPProvider` — uses Go `net/smtp` with STARTTLS/TLS support
- `ResendProvider` — uses Resend HTTP API (`POST https://api.resend.com/emails`)

### 3.2 Provider Configuration (site_settings)

```
email_provider        = "smtp" | "resend" | ""  (empty = disabled)

-- SMTP settings
email_smtp_host       = "smtp.example.com"
email_smtp_port       = "587"
email_smtp_user       = "user@example.com"
email_smtp_password   = "secret"              (is_encrypted = true)
email_smtp_from       = "noreply@example.com"
email_smtp_from_name  = "VibeCMS"

-- Resend settings
email_resend_api_key  = "re_..."              (is_encrypted = true)
email_resend_from     = "noreply@example.com"
email_resend_from_name = "VibeCMS"
```

Provider is resolved at send time from site_settings — no restart needed to switch.

### 3.3 `email_templates` Table

```sql
CREATE TABLE email_templates (
    id               SERIAL PRIMARY KEY,
    slug             VARCHAR(100) UNIQUE NOT NULL,
    name             VARCHAR(150) NOT NULL,
    subject_template TEXT NOT NULL DEFAULT '',
    body_template    TEXT NOT NULL DEFAULT '',
    test_data        JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Templates use Go `html/template` syntax: `{{.user.full_name}}`, `{{.node.title}}`, `{{.site.name}}`.

A base email layout is stored in site_settings (`email_base_layout`) and wraps all templates. Default base layout provides a simple branded wrapper with header/footer.

### 3.4 Seeded Email Templates

**welcome** — "Welcome to {{.site.name}}"
```
Hi {{.user.full_name}},

Welcome to {{.site.name}}! Your account has been created.

You can log in at: {{.site.url}}/login
```

**user-registered-admin** — "New user registered: {{.user.full_name}}"
```
A new user has registered on {{.site.name}}.

Name: {{.user.full_name}}
Email: {{.user.email}}
Role: {{.user.role}}
```

**password-reset** — "Reset your password"
```
Hi {{.user.full_name}},

You requested a password reset for your account on {{.site.name}}.

Click here to reset your password: {{.reset_url}}

If you didn't request this, you can safely ignore this email.
```

**node-published** — "{{.node.title}} has been published"
```
Hi {{.user.full_name}},

"{{.node.title}}" ({{.node.node_type}}) has been published on {{.site.name}}.

View it at: {{.site.url}}{{.node.full_url}}
```

**node-created-admin** — "New {{.node.node_type}} created: {{.node.title}}"
```
A new {{.node.node_type}} has been created on {{.site.name}}.

Title: {{.node.title}}
Author: {{.author.full_name}}
URL: {{.site.url}}{{.node.full_url}}
```

### 3.5 `email_rules` Table

```sql
CREATE TABLE email_rules (
    id              SERIAL PRIMARY KEY,
    action          VARCHAR(100) NOT NULL,      -- e.g. "node.created"
    node_type       VARCHAR(50),                -- NULL = all node types
    template_id     INT NOT NULL REFERENCES email_templates(id),
    recipient_type  VARCHAR(20) NOT NULL,        -- "actor", "role", "node_author", "fixed"
    recipient_value VARCHAR(500) NOT NULL DEFAULT '', -- role slug, or comma-separated emails
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Recipient types:**
- `actor` — the user who triggered the action (value ignored)
- `role` — all users with this role slug who have the action in their `email_subscriptions` capability (value = role slug)
- `node_author` — the author of the node (value ignored, only for node.* actions)
- `fixed` — comma-separated email addresses (value = "a@b.com,c@d.com")

### 3.6 Seeded Email Rules

| Action | Node Type | Template | Recipient | Value |
|--------|-----------|----------|-----------|-------|
| user.registered | — | welcome | actor | — |
| user.registered | — | user-registered-admin | role | admin |
| node.published | — | node-published | node_author | — |
| node.created | — | node-created-admin | role | admin |

### 3.7 `email_logs` Table

```sql
CREATE TABLE email_logs (
    id              SERIAL PRIMARY KEY,
    rule_id         INT REFERENCES email_rules(id) ON DELETE SET NULL,
    template_slug   VARCHAR(100) NOT NULL,
    action          VARCHAR(100) NOT NULL,
    recipient_email VARCHAR(255) NOT NULL,
    subject         TEXT NOT NULL,
    rendered_body   TEXT NOT NULL,        -- full rendered HTML for "view" feature
    status          VARCHAR(20) NOT NULL, -- "sent", "failed", "pending"
    error_message   TEXT,
    provider        VARCHAR(50),          -- "smtp", "resend", or NULL if no provider
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_logs_status ON email_logs(status);
CREATE INDEX idx_email_logs_action ON email_logs(action);
CREATE INDEX idx_email_logs_created_at ON email_logs(created_at DESC);
```

### 3.8 Email Dispatch Flow

1. Event fires via EventBus
2. Email dispatcher (subscribed to all events) queries `email_rules` for matching action + node_type
3. For each matching enabled rule:
   a. Resolve recipients (expand role → user emails, filter by `email_subscriptions` capability)
   b. Build template variables from event payload + site settings (site.name, site.url)
   c. Render subject + body templates with Go `html/template`
   d. Wrap body in base email layout
   e. Send via active provider (or log as failed if none configured)
   f. Write to `email_logs` with full rendered HTML, status, and any error

**Resend from logs:** Load the log entry's `rendered_body` and `subject`, send via current provider. Creates a new log entry linked to the original rule.

---

## 4. Admin UI

### 4.1 New Pages

**Users** (`/admin/users`)
- Table: name, email, role badge, last login, created date
- Create/edit: name, email, password (optional on edit), role selector
- Admin-only (requires `manage_users` capability)

**Roles** (`/admin/roles`)
- Table: name, slug, user count, is_system badge
- Edit view: name, description, capability toggles:
  - Top section: admin_access, manage_users, manage_roles, manage_settings, manage_menus, manage_layouts, manage_email checkboxes
  - Node access matrix: rows = all node types, columns = access (none/read/write radio) + scope (all/own toggle)
  - Email subscriptions: checklist of all system actions
- System roles show lock icon, can edit capabilities but not delete
- Requires `manage_roles` capability

**Email Templates** (`/admin/email-templates`)
- Table: name, slug, subject preview
- Edit view: slug, name, subject template input, body HTML code editor (left) + rendered preview (right), test_data JSON editor
- Requires `manage_email` capability

**Email Rules** (`/admin/email-rules`)
- Table: action, node_type filter, template name, recipient, enabled toggle
- Create/edit: action dropdown, optional node_type dropdown, template selector, recipient type selector + value input, enabled toggle
- Requires `manage_email` capability

**Email Logs** (`/admin/email-logs`)
- Table: date, action, recipient, subject, status badge (sent=green, failed=red)
- Filters: status dropdown, action dropdown, date range, recipient search
- Row actions: "View" (modal with rendered HTML iframe), "Resend" button
- Requires `manage_email` capability

**Email Settings** (`/admin/email-settings`)
- Provider picker: None / SMTP / Resend
- Dynamic config form per provider (host/port/user/pass/from for SMTP, api_key/from for Resend)
- "Send Test Email" button (sends to current user's email)
- Requires `manage_settings` capability

### 4.2 Sidebar Changes

```
Dashboard
Pages
Posts
[custom node types]
Users                    ← new, top-level
Menus
Design ▾
  Layouts
  Layout Blocks
  Templates
Schema ▾
  Content Types
  Block Types
  Roles                  ← new, under Schema
Email ▾                  ← new group
  Templates
  Rules
  Logs
  Settings
Languages
Media (soon)
Settings (soon)
```

---

## 5. Database Migration Summary

Single migration `0009_users_roles_email_actions.sql`:

1. Create `roles` table + seed 4 default roles
2. Create `system_actions` table + seed all actions
3. Create `email_templates` table + seed 5 default templates
4. Create `email_rules` table + seed 4 default rules
5. Create `email_logs` table
6. Add `author_id` to `content_nodes` (nullable, FK to users)
7. Add `role_id` to `users` (nullable initially)
8. Migrate existing user roles: map `role` string → `role_id`
9. Make `role_id` NOT NULL, drop `role` column

---

## 6. New Go Packages

```
internal/events/       — EventBus, event payload builders
internal/email/        — Provider interface, SMTP + Resend providers, dispatcher, template renderer
internal/rbac/         — Role service, capability helpers, middleware
```

---

## 7. File Impact Summary

**New files:**
- `internal/events/bus.go` — event bus
- `internal/email/provider.go` — provider interface
- `internal/email/smtp.go` — SMTP provider
- `internal/email/resend.go` — Resend provider
- `internal/email/dispatcher.go` — listens to events, resolves rules, sends emails
- `internal/email/template_svc.go` — template CRUD service
- `internal/email/rule_svc.go` — rule CRUD service
- `internal/email/log_svc.go` — log service (list, resend)
- `internal/email/handler.go` — admin API handlers for templates, rules, logs, settings
- `internal/rbac/role_svc.go` — role CRUD service
- `internal/rbac/capability.go` — capability check helpers
- `internal/rbac/middleware.go` — permission middleware
- `internal/rbac/handler.go` — admin API handler for roles
- `internal/models/role.go`, `system_action.go`, `email_template.go`, `email_rule.go`, `email_log.go`
- `internal/db/migrations/0009_users_roles_email_actions.sql`
- `admin-ui/src/pages/` — users, roles, email-templates, email-rules, email-logs, email-settings pages

**Modified files:**
- `internal/models/user.go` — role_id FK, drop role string
- `internal/models/content_node.go` — add author_id
- `internal/auth/rbac_middleware.go` — load role capabilities, new helpers
- `internal/auth/user_handler.go` — use role_id, capability checks
- `internal/auth/page_handler.go` — fire user.registered/login events
- `internal/cms/content_svc.go` — fire node.* events, set author_id
- `internal/cms/layout_svc.go`, `layout_block_svc.go`, `block_type_svc.go`, `menu_svc.go` — fire events
- `internal/db/seed.go` — seed roles, actions, templates, rules
- `cmd/vibecms/main.go` — wire EventBus, email providers, new handlers
- `admin-ui/src/components/layout/admin-layout.tsx` — sidebar updates
- `admin-ui/src/api/client.ts` — new API calls
