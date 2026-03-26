# Users, Roles, Email & Actions Implementation Plan

> **For agentic workers:** Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add RBAC roles, system event bus, modular email (SMTP/Resend), email templates/rules/logs, and admin UI for all of the above.

**Architecture:** Event bus publishes CRUD actions across all entities. Email dispatcher subscribes, matches rules, renders templates, sends via configured provider. JSONB capability-based roles control admin access and per-node-type permissions.

**Tech Stack:** Go 1.22+, Fiber, GORM, PostgreSQL JSONB, React + TypeScript + Tailwind + shadcn/ui

---

## Task 1: Database Migration

**Files:**
- Create: `internal/db/migrations/0009_roles_actions_email.sql`

- [ ] **Step 1: Write migration SQL**

```sql
-- Roles
CREATE TABLE IF NOT EXISTS roles (
    id           SERIAL PRIMARY KEY,
    slug         VARCHAR(50) UNIQUE NOT NULL,
    name         VARCHAR(100) NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    is_system    BOOLEAN NOT NULL DEFAULT false,
    capabilities JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default roles
INSERT INTO roles (slug, name, description, is_system, capabilities) VALUES
('admin', 'Administrator', 'Full system access', true, '{"admin_access":true,"manage_users":true,"manage_roles":true,"manage_settings":true,"manage_menus":true,"manage_layouts":true,"manage_email":true,"default_node_access":{"access":"write","scope":"all"},"email_subscriptions":["user.registered","user.deleted","node.created","node.updated","node.published","node.deleted"]}'),
('editor', 'Editor', 'Can manage all content', true, '{"admin_access":true,"manage_users":false,"manage_roles":false,"manage_settings":false,"manage_menus":true,"manage_layouts":false,"manage_email":false,"default_node_access":{"access":"write","scope":"all"},"email_subscriptions":["node.created","node.published"]}'),
('author', 'Author', 'Can manage own content', true, '{"admin_access":true,"manage_users":false,"manage_roles":false,"manage_settings":false,"manage_menus":false,"manage_layouts":false,"manage_email":false,"default_node_access":{"access":"write","scope":"own"},"email_subscriptions":["node.published"]}'),
('member', 'Member', 'Public member, no admin access', true, '{"admin_access":false,"manage_users":false,"manage_roles":false,"manage_settings":false,"manage_menus":false,"manage_layouts":false,"manage_email":false,"default_node_access":{"access":"read","scope":"all"},"email_subscriptions":[]}')
ON CONFLICT (slug) DO NOTHING;

-- Migrate users.role string → role_id
ALTER TABLE users ADD COLUMN IF NOT EXISTS role_id INT REFERENCES roles(id);
UPDATE users SET role_id = (SELECT id FROM roles WHERE slug = users.role) WHERE role_id IS NULL;
UPDATE users SET role_id = (SELECT id FROM roles WHERE slug = 'editor') WHERE role_id IS NULL;
ALTER TABLE users ALTER COLUMN role_id SET NOT NULL;
ALTER TABLE users DROP COLUMN IF EXISTS role;

-- Author tracking on content nodes
ALTER TABLE content_nodes ADD COLUMN IF NOT EXISTS author_id INT REFERENCES users(id);

-- System actions
CREATE TABLE IF NOT EXISTS system_actions (
    id              SERIAL PRIMARY KEY,
    slug            VARCHAR(100) UNIQUE NOT NULL,
    label           VARCHAR(150) NOT NULL,
    category        VARCHAR(50) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    payload_schema  JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO system_actions (slug, label, category, description) VALUES
('user.registered', 'User Registered', 'user', 'Fired when a new user registers'),
('user.updated', 'User Updated', 'user', 'Fired when a user profile is updated'),
('user.deleted', 'User Deleted', 'user', 'Fired when a user is deleted'),
('user.login', 'User Login', 'user', 'Fired on successful login'),
('node.created', 'Node Created', 'node', 'Fired when a content node is created'),
('node.updated', 'Node Updated', 'node', 'Fired when a content node is updated'),
('node.published', 'Node Published', 'node', 'Fired when a node status changes to published'),
('node.unpublished', 'Node Unpublished', 'node', 'Fired when a published node is unpublished'),
('node.deleted', 'Node Deleted', 'node', 'Fired when a content node is deleted'),
('layout.created', 'Layout Created', 'layout', 'Fired when a layout is created'),
('layout.updated', 'Layout Updated', 'layout', 'Fired when a layout is updated'),
('layout.deleted', 'Layout Deleted', 'layout', 'Fired when a layout is deleted'),
('layout_block.created', 'Layout Block Created', 'layout_block', 'Fired when a layout block is created'),
('layout_block.updated', 'Layout Block Updated', 'layout_block', 'Fired when a layout block is updated'),
('layout_block.deleted', 'Layout Block Deleted', 'layout_block', 'Fired when a layout block is deleted'),
('block_type.created', 'Block Type Created', 'block_type', 'Fired when a block type is created'),
('block_type.updated', 'Block Type Updated', 'block_type', 'Fired when a block type is updated'),
('block_type.deleted', 'Block Type Deleted', 'block_type', 'Fired when a block type is deleted'),
('menu.created', 'Menu Created', 'menu', 'Fired when a menu is created'),
('menu.updated', 'Menu Updated', 'menu', 'Fired when a menu is updated'),
('menu.deleted', 'Menu Deleted', 'menu', 'Fired when a menu is deleted')
ON CONFLICT (slug) DO NOTHING;

-- Email templates
CREATE TABLE IF NOT EXISTS email_templates (
    id               SERIAL PRIMARY KEY,
    slug             VARCHAR(100) UNIQUE NOT NULL,
    name             VARCHAR(150) NOT NULL,
    subject_template TEXT NOT NULL DEFAULT '',
    body_template    TEXT NOT NULL DEFAULT '',
    test_data        JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Email rules
CREATE TABLE IF NOT EXISTS email_rules (
    id              SERIAL PRIMARY KEY,
    action          VARCHAR(100) NOT NULL,
    node_type       VARCHAR(50),
    template_id     INT NOT NULL REFERENCES email_templates(id),
    recipient_type  VARCHAR(20) NOT NULL,
    recipient_value VARCHAR(500) NOT NULL DEFAULT '',
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Email logs
CREATE TABLE IF NOT EXISTS email_logs (
    id              SERIAL PRIMARY KEY,
    rule_id         INT REFERENCES email_rules(id) ON DELETE SET NULL,
    template_slug   VARCHAR(100) NOT NULL,
    action          VARCHAR(100) NOT NULL,
    recipient_email VARCHAR(255) NOT NULL,
    subject         TEXT NOT NULL,
    rendered_body   TEXT NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message   TEXT,
    provider        VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_logs_status ON email_logs(status);
CREATE INDEX IF NOT EXISTS idx_email_logs_action ON email_logs(action);
CREATE INDEX IF NOT EXISTS idx_email_logs_created_at ON email_logs(created_at DESC);
```

- [ ] **Step 2: Commit**

---

## Task 2: Go Models

**Files:**
- Create: `internal/models/role.go`
- Create: `internal/models/system_action.go`
- Create: `internal/models/email_template.go`
- Create: `internal/models/email_rule.go`
- Create: `internal/models/email_log.go`
- Modify: `internal/models/user.go` — replace `Role string` with `RoleID int` + `Role Role` (belongs-to)
- Modify: `internal/models/content_node.go` — add `AuthorID *int` + `Author *User`

Each model follows existing GORM patterns in the codebase. See existing models for style reference.

- [ ] **Step 1: Create all model files**
- [ ] **Step 2: Update user.go and content_node.go**
- [ ] **Step 3: Verify `go build ./...` passes**
- [ ] **Step 4: Commit**

---

## Task 3: Event Bus

**Files:**
- Create: `internal/events/bus.go`

Implement `EventBus` with `Publish`, `Subscribe`, `SubscribeAll`. Handlers run in goroutines with panic recovery. Thread-safe with sync.RWMutex.

- [ ] **Step 1: Implement event bus**
- [ ] **Step 2: Verify `go build ./...` passes**
- [ ] **Step 3: Commit**

---

## Task 4: RBAC — Role Service & Middleware

**Files:**
- Create: `internal/rbac/role_svc.go` — CRUD for roles, capability helpers
- Create: `internal/rbac/middleware.go` — `CapabilityRequired()`, `NodeAccess()` helpers
- Create: `internal/rbac/handler.go` — admin API: list/get/create/update/delete roles
- Modify: `internal/auth/rbac_middleware.go` — `AuthRequired` loads user + role with capabilities
- Modify: `internal/auth/user_handler.go` — use role_id, add role info to responses

Role service: List, GetByID, GetBySlug, Create, Update, Delete (prevent deleting system roles).
Capability helpers: `HasCapability(role, cap) bool`, `NodeAccess(role, nodeType) (access, scope)`.
Middleware: `CapabilityRequired("manage_users")` checks loaded role capabilities.

- [ ] **Step 1: Implement role service**
- [ ] **Step 2: Implement capability helpers and middleware**
- [ ] **Step 3: Implement role handler (admin API)**
- [ ] **Step 4: Update auth middleware to load role**
- [ ] **Step 5: Update user handler for role_id**
- [ ] **Step 6: Verify `go build ./...` passes**
- [ ] **Step 7: Commit**

---

## Task 5: Email Provider Interface + Implementations

**Files:**
- Create: `internal/email/provider.go` — Provider interface + factory
- Create: `internal/email/smtp.go` — SMTP provider
- Create: `internal/email/resend.go` — Resend HTTP API provider

Provider interface: `Send(to []string, subject string, html string) error`, `Name() string`.
Factory: `NewProvider(name string, settings map[string]string) (Provider, error)` — reads config from site_settings map.

- [ ] **Step 1: Implement provider interface and factory**
- [ ] **Step 2: Implement SMTP provider (net/smtp with TLS)**
- [ ] **Step 3: Implement Resend provider (HTTP POST)**
- [ ] **Step 4: Verify `go build ./...` passes**
- [ ] **Step 5: Commit**

---

## Task 6: Email Services

**Files:**
- Create: `internal/email/template_svc.go` — CRUD for email templates
- Create: `internal/email/rule_svc.go` — CRUD for email rules
- Create: `internal/email/log_svc.go` — list/filter logs, resend

Standard GORM CRUD following existing service patterns (see `internal/cms/layout_svc.go` as reference).
Log service: `List(filters)` with status/action/date/recipient filtering, `Resend(logID)`, `GetRenderedBody(logID)`.

- [ ] **Step 1: Implement template service**
- [ ] **Step 2: Implement rule service**
- [ ] **Step 3: Implement log service**
- [ ] **Step 4: Verify `go build ./...` passes**
- [ ] **Step 5: Commit**

---

## Task 7: Email Dispatcher

**Files:**
- Create: `internal/email/dispatcher.go`

Subscribes to EventBus via `SubscribeAll`. On each event:
1. Query enabled email_rules matching action (+ node_type if applicable)
2. For each rule, resolve recipients (actor, role → expand to user emails filtered by email_subscriptions, node_author, fixed)
3. Render subject + body templates with Go html/template using event payload + site settings
4. Wrap body in base email layout (from site_settings `email_base_layout`)
5. Send via active provider (from site_settings `email_provider`)
6. Log to email_logs

- [ ] **Step 1: Implement dispatcher**
- [ ] **Step 2: Verify `go build ./...` passes**
- [ ] **Step 3: Commit**

---

## Task 8: Email Admin API Handler

**Files:**
- Create: `internal/email/handler.go`

Endpoints under `/admin/api`:
- `GET/POST /email-templates`, `GET/PATCH/DELETE /email-templates/:id`
- `GET/POST /email-rules`, `GET/PATCH/DELETE /email-rules/:id`
- `GET /email-logs` (with query params: status, action, date_from, date_to, recipient), `GET /email-logs/:id`, `POST /email-logs/:id/resend`
- `GET/POST /email-settings` (get/save provider config from site_settings)
- `POST /email-settings/test` (send test email to current user)

- [ ] **Step 1: Implement handler**
- [ ] **Step 2: Verify `go build ./...` passes**
- [ ] **Step 3: Commit**

---

## Task 9: Fire Events from Existing Services

**Files:**
- Modify: `internal/cms/content_svc.go` — fire node.created/updated/published/unpublished/deleted, set author_id
- Modify: `internal/cms/layout_svc.go` — fire layout.* events
- Modify: `internal/cms/layout_block_svc.go` — fire layout_block.* events
- Modify: `internal/cms/block_type_svc.go` — fire block_type.* events
- Modify: `internal/cms/menu_svc.go` — fire menu.* events
- Modify: `internal/auth/page_handler.go` — fire user.registered, user.login
- Modify: `internal/auth/user_handler.go` — fire user.updated, user.deleted

Each service needs an `eventBus *events.EventBus` field added to its struct. Pass via constructor.
Events fire AFTER successful DB operations using `go eventBus.Publish(...)`.

- [ ] **Step 1: Add eventBus to all service/handler structs**
- [ ] **Step 2: Fire events after DB operations**
- [ ] **Step 3: Verify `go build ./...` passes**
- [ ] **Step 4: Commit**

---

## Task 10: Wire Everything in main.go

**Files:**
- Modify: `cmd/vibecms/main.go`

1. Create EventBus
2. Create email dispatcher (subscribes to event bus)
3. Create role service + handler
4. Create email template/rule/log services + handler
5. Pass eventBus to all existing services that fire events
6. Register new admin API routes: `/admin/api/roles/*`, `/admin/api/email-templates/*`, `/admin/api/email-rules/*`, `/admin/api/email-logs/*`, `/admin/api/email-settings/*`
7. Apply capability middleware where needed

- [ ] **Step 1: Wire all new services and routes**
- [ ] **Step 2: Verify `go build ./...` passes**
- [ ] **Step 3: Commit**

---

## Task 11: Seed Data

**Files:**
- Modify: `internal/db/seed.go`

Add `seedEmailTemplates()` and `seedEmailRules()` functions. Templates:
- welcome, user-registered-admin, password-reset, node-published, node-created-admin

Rules:
- user.registered → welcome → actor
- user.registered → user-registered-admin → role:admin
- node.published → node-published → node_author
- node.created → node-created-admin → role:admin

Also seed default `email_base_layout` in site_settings.

- [ ] **Step 1: Implement seed functions**
- [ ] **Step 2: Verify `go build ./...` passes**
- [ ] **Step 3: Commit**

---

## Task 12: Admin UI — API Client

**Files:**
- Modify: `admin-ui/src/api/client.ts`

Add API functions:
- Roles: getRoles, getRole, createRole, updateRole, deleteRole
- Email templates: getEmailTemplates, getEmailTemplate, createEmailTemplate, updateEmailTemplate, deleteEmailTemplate
- Email rules: getEmailRules, getEmailRule, createEmailRule, updateEmailRule, deleteEmailRule
- Email logs: getEmailLogs, getEmailLog, resendEmail
- Email settings: getEmailSettings, saveEmailSettings, sendTestEmail
- Users: update getUsers/createUser/updateUser/deleteUser to use role_id
- System actions: getSystemActions (for dropdowns)

- [ ] **Step 1: Add all API functions**
- [ ] **Step 2: Commit**

---

## Task 13: Admin UI — Users Page

**Files:**
- Create: `admin-ui/src/pages/users.tsx`

Table with columns: name, email, role badge, last login, created. Create/edit dialog with name, email, password (optional on edit), role selector dropdown. Delete confirmation. Admin-only page.

- [ ] **Step 1: Implement users page**
- [ ] **Step 2: Commit**

---

## Task 14: Admin UI — Roles Page

**Files:**
- Create: `admin-ui/src/pages/roles.tsx`

List view: name, slug, user count, system badge. Edit view: name, description, capability toggles (admin_access, manage_users, manage_roles, manage_settings, manage_menus, manage_layouts, manage_email), node access matrix (rows=node types, columns=none/read/write + all/own), email subscriptions checklist. System roles show lock but are editable.

- [ ] **Step 1: Implement roles list + edit page**
- [ ] **Step 2: Commit**

---

## Task 15: Admin UI — Email Templates Page

**Files:**
- Create: `admin-ui/src/pages/email-templates.tsx`

List view: name, slug, subject preview. Edit view: slug, name, subject input, split pane — left: HTML code editor (textarea with monospace), right: rendered preview using test_data. Test data JSON editor below.

- [ ] **Step 1: Implement email templates page**
- [ ] **Step 2: Commit**

---

## Task 16: Admin UI — Email Rules Page

**Files:**
- Create: `admin-ui/src/pages/email-rules.tsx`

List view: action, node_type, template name, recipient info, enabled toggle. Create/edit dialog: action dropdown (from system_actions), node_type dropdown (optional), template selector, recipient type (actor/role/node_author/fixed), recipient value input, enabled toggle.

- [ ] **Step 1: Implement email rules page**
- [ ] **Step 2: Commit**

---

## Task 17: Admin UI — Email Logs Page

**Files:**
- Create: `admin-ui/src/pages/email-logs.tsx`

Table: date, action, recipient, subject, status badge (sent=green/failed=red). Filters: status, action, date range, recipient search. Row actions: View (modal with rendered HTML in iframe/sandbox), Resend button.

- [ ] **Step 1: Implement email logs page**
- [ ] **Step 2: Commit**

---

## Task 18: Admin UI — Email Settings Page

**Files:**
- Create: `admin-ui/src/pages/email-settings.tsx`

Provider picker (None/SMTP/Resend). Dynamic config form: SMTP shows host/port/user/password/from/from_name. Resend shows api_key/from/from_name. Send Test Email button.

- [ ] **Step 1: Implement email settings page**
- [ ] **Step 2: Commit**

---

## Task 19: Admin UI — Sidebar + Routes

**Files:**
- Modify: `admin-ui/src/components/layout/admin-layout.tsx` — add Users, Roles, Email group
- Modify: `admin-ui/src/App.tsx` (or router file) — add routes for all new pages

Sidebar additions:
- Top level: Users (after Posts)
- Schema group: add Roles
- New Email group: Templates, Rules, Logs, Settings

- [ ] **Step 1: Update sidebar navigation**
- [ ] **Step 2: Add routes**
- [ ] **Step 3: Build frontend: `npm run build`**
- [ ] **Step 4: Commit**

---

## Task 20: Verification & Docker Rebuild

- [ ] **Step 1: `go build ./...` passes**
- [ ] **Step 2: `docker compose up -d --build` succeeds**
- [ ] **Step 3: Run seed: `docker compose exec app ./vibecms seed`**
- [ ] **Step 4: Verify container stays running: `docker compose ps`**
- [ ] **Step 5: Verify admin UI loads at /admin/dashboard**
- [ ] **Step 6: Verify new API endpoints respond (roles, email-templates, etc.)**
