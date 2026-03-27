-- Migration 0009: Roles, System Actions, Email System
-- Creates roles, system_actions, email_templates, email_rules, email_logs tables.
-- Migrates users.role varchar to users.role_id FK.
-- Adds author_id to content_nodes.

-- ============================================================
-- 1. Roles table
-- ============================================================
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
(
    'admin', 'Admin', 'Full access to all features', true,
    '{
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
    }'::jsonb
),
(
    'editor', 'Editor', 'Can edit all content and manage menus', true,
    '{
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
    }'::jsonb
),
(
    'author', 'Author', 'Can create and edit own content', true,
    '{
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
    }'::jsonb
),
(
    'member', 'Member', 'Front-end member with read access', true,
    '{
        "admin_access": false,
        "manage_users": false,
        "manage_roles": false,
        "manage_settings": false,
        "manage_menus": false,
        "manage_layouts": false,
        "manage_email": false,
        "default_node_access": { "access": "read", "scope": "all" },
        "email_subscriptions": []
    }'::jsonb
)
ON CONFLICT (slug) DO NOTHING;

-- ============================================================
-- 2. Migrate users.role varchar -> users.role_id FK
-- ============================================================

-- Add role_id column (nullable initially)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'role_id'
    ) THEN
        ALTER TABLE users ADD COLUMN role_id INT REFERENCES roles(id);
    END IF;
END $$;

-- Map existing role values to role IDs (only if old role column still exists)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'role'
    ) THEN
        UPDATE users SET role_id = (SELECT id FROM roles WHERE slug = 'admin')
        WHERE role = 'admin' AND role_id IS NULL;

        UPDATE users SET role_id = (SELECT id FROM roles WHERE slug = 'editor')
        WHERE role IS NOT NULL AND role != 'admin' AND role_id IS NULL;
    END IF;

    -- Default any remaining NULL role_id to editor
    UPDATE users SET role_id = (SELECT id FROM roles WHERE slug = 'editor')
    WHERE role_id IS NULL;

    -- Make role_id NOT NULL if not already
    ALTER TABLE users ALTER COLUMN role_id SET NOT NULL;
END $$;

-- Drop old role column if it exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'role'
    ) THEN
        ALTER TABLE users DROP COLUMN role;
    END IF;
END $$;

-- ============================================================
-- 3. Add author_id to content_nodes
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'content_nodes' AND column_name = 'author_id'
    ) THEN
        ALTER TABLE content_nodes ADD COLUMN author_id INT REFERENCES users(id);
    END IF;
END $$;

-- ============================================================
-- 4. System Actions table
-- ============================================================
CREATE TABLE IF NOT EXISTS system_actions (
    id              SERIAL PRIMARY KEY,
    slug            VARCHAR(100) UNIQUE NOT NULL,
    label           VARCHAR(150) NOT NULL,
    category        VARCHAR(50) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    payload_schema  JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed system actions
INSERT INTO system_actions (slug, label, category, description, payload_schema) VALUES
-- User actions
('user.registered', 'User Registered', 'user', 'Fired when a new user registers or is created', '["user.*"]'::jsonb),
('user.updated', 'User Updated', 'user', 'Fired when a user profile is updated', '["user.*"]'::jsonb),
('user.deleted', 'User Deleted', 'user', 'Fired when a user is deleted', '["user.*"]'::jsonb),
('user.login', 'User Login', 'user', 'Fired when a user logs in', '["user.*", "ip_address", "user_agent"]'::jsonb),
-- Node actions
('node.created', 'Node Created', 'node', 'Fired when a content node is created', '["node.*", "node_type", "author.*"]'::jsonb),
('node.updated', 'Node Updated', 'node', 'Fired when a content node is updated', '["node.*", "node_type", "author.*", "editor.*"]'::jsonb),
('node.published', 'Node Published', 'node', 'Fired when a content node is published', '["node.*", "node_type", "author.*"]'::jsonb),
('node.unpublished', 'Node Unpublished', 'node', 'Fired when a content node is unpublished', '["node.*", "node_type", "author.*"]'::jsonb),
('node.deleted', 'Node Deleted', 'node', 'Fired when a content node is deleted', '["node.*", "node_type", "author.*"]'::jsonb),
-- Layout actions
('layout.created', 'Layout Created', 'layout', 'Fired when a layout is created', '["layout.*", "editor.*"]'::jsonb),
('layout.updated', 'Layout Updated', 'layout', 'Fired when a layout is updated', '["layout.*", "editor.*"]'::jsonb),
('layout.deleted', 'Layout Deleted', 'layout', 'Fired when a layout is deleted', '["layout.*", "editor.*"]'::jsonb),
-- Layout block actions
('layout_block.created', 'Layout Block Created', 'layout_block', 'Fired when a layout block is created', '["layout_block.*", "editor.*"]'::jsonb),
('layout_block.updated', 'Layout Block Updated', 'layout_block', 'Fired when a layout block is updated', '["layout_block.*", "editor.*"]'::jsonb),
('layout_block.deleted', 'Layout Block Deleted', 'layout_block', 'Fired when a layout block is deleted', '["layout_block.*", "editor.*"]'::jsonb),
-- Block type actions
('block_type.created', 'Block Type Created', 'block_type', 'Fired when a block type is created', '["block_type.*", "editor.*"]'::jsonb),
('block_type.updated', 'Block Type Updated', 'block_type', 'Fired when a block type is updated', '["block_type.*", "editor.*"]'::jsonb),
('block_type.deleted', 'Block Type Deleted', 'block_type', 'Fired when a block type is deleted', '["block_type.*", "editor.*"]'::jsonb),
-- Menu actions
('menu.created', 'Menu Created', 'menu', 'Fired when a menu is created', '["menu.*", "editor.*"]'::jsonb),
('menu.updated', 'Menu Updated', 'menu', 'Fired when a menu is updated', '["menu.*", "editor.*"]'::jsonb),
('menu.deleted', 'Menu Deleted', 'menu', 'Fired when a menu is deleted', '["menu.*", "editor.*"]'::jsonb)
ON CONFLICT (slug) DO NOTHING;

-- ============================================================
-- 5. Email Templates table
-- ============================================================
CREATE TABLE IF NOT EXISTS email_templates (
    id               SERIAL PRIMARY KEY,
    slug             VARCHAR(100) NOT NULL,
    name             VARCHAR(150) NOT NULL,
    subject_template TEXT NOT NULL DEFAULT '',
    body_template    TEXT NOT NULL DEFAULT '',
    test_data        JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default email templates (using NOT EXISTS for idempotency)
INSERT INTO email_templates (slug, name, subject_template, body_template, test_data)
SELECT 'welcome', 'Welcome Email', 'Welcome to {{.site.name}}',
    E'Hi {{.user.full_name}},\n\nWelcome to {{.site.name}}! Your account has been created.\n\nYou can log in at: {{.site.url}}/login',
    '{"user": {"full_name": "Jane Doe", "email": "jane@example.com"}, "site": {"name": "My Site", "url": "https://example.com"}}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM email_templates WHERE slug = 'welcome');

INSERT INTO email_templates (slug, name, subject_template, body_template, test_data)
SELECT 'user-registered-admin', 'Admin: New User Registered', 'New user registered: {{.user.full_name}}',
    E'A new user has registered on {{.site.name}}.\n\nName: {{.user.full_name}}\nEmail: {{.user.email}}',
    '{"user": {"full_name": "Jane Doe", "email": "jane@example.com"}, "site": {"name": "My Site"}}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM email_templates WHERE slug = 'user-registered-admin');

INSERT INTO email_templates (slug, name, subject_template, body_template, test_data)
SELECT 'password-reset', 'Password Reset', 'Reset your password',
    E'Hi {{.user.full_name}},\n\nYou requested a password reset for your account on {{.site.name}}.\n\nClick here to reset your password: {{.reset_url}}\n\nIf you didn''t request this, you can safely ignore this email.',
    '{"user": {"full_name": "Jane Doe"}, "site": {"name": "My Site"}, "reset_url": "https://example.com/reset?token=abc123"}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM email_templates WHERE slug = 'password-reset');

INSERT INTO email_templates (slug, name, subject_template, body_template, test_data)
SELECT 'node-published', 'Node Published', '{{.node.title}} has been published',
    E'Hi {{.user.full_name}},\n\n"{{.node.title}}" ({{.node.node_type}}) has been published on {{.site.name}}.\n\nView it at: {{.site.url}}{{.node.full_url}}',
    '{"user": {"full_name": "Jane Doe"}, "node": {"title": "Hello World", "node_type": "post", "full_url": "/hello-world"}, "site": {"name": "My Site", "url": "https://example.com"}}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM email_templates WHERE slug = 'node-published');

INSERT INTO email_templates (slug, name, subject_template, body_template, test_data)
SELECT 'node-created-admin', 'Admin: New Node Created', 'New {{.node.node_type}} created: {{.node.title}}',
    E'A new {{.node.node_type}} has been created on {{.site.name}}.\n\nTitle: {{.node.title}}\nAuthor: {{.author.full_name}}\nURL: {{.site.url}}{{.node.full_url}}',
    '{"node": {"title": "Hello World", "node_type": "post", "full_url": "/hello-world"}, "author": {"full_name": "Jane Doe"}, "site": {"name": "My Site", "url": "https://example.com"}}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM email_templates WHERE slug = 'node-created-admin');

-- ============================================================
-- 6. Email Rules table
-- ============================================================
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

-- Seed default email rules
INSERT INTO email_rules (action, node_type, template_id, recipient_type, recipient_value, enabled)
SELECT 'user.registered', NULL, et.id, 'actor', '', true
FROM email_templates et WHERE et.slug = 'welcome'
AND NOT EXISTS (SELECT 1 FROM email_rules er WHERE er.action = 'user.registered' AND er.recipient_type = 'actor');

INSERT INTO email_rules (action, node_type, template_id, recipient_type, recipient_value, enabled)
SELECT 'user.registered', NULL, et.id, 'role', 'admin', true
FROM email_templates et WHERE et.slug = 'user-registered-admin'
AND NOT EXISTS (SELECT 1 FROM email_rules er WHERE er.action = 'user.registered' AND er.recipient_type = 'role');

INSERT INTO email_rules (action, node_type, template_id, recipient_type, recipient_value, enabled)
SELECT 'node.published', NULL, et.id, 'node_author', '', true
FROM email_templates et WHERE et.slug = 'node-published'
AND NOT EXISTS (SELECT 1 FROM email_rules er WHERE er.action = 'node.published' AND er.recipient_type = 'node_author');

INSERT INTO email_rules (action, node_type, template_id, recipient_type, recipient_value, enabled)
SELECT 'node.created', NULL, et.id, 'role', 'admin', true
FROM email_templates et WHERE et.slug = 'node-created-admin'
AND NOT EXISTS (SELECT 1 FROM email_rules er WHERE er.action = 'node.created' AND er.recipient_type = 'role');

-- ============================================================
-- 7. Email Logs table
-- ============================================================
CREATE TABLE IF NOT EXISTS email_logs (
    id              SERIAL PRIMARY KEY,
    rule_id         INT REFERENCES email_rules(id) ON DELETE SET NULL,
    template_slug   VARCHAR(100) NOT NULL,
    action          VARCHAR(100) NOT NULL,
    recipient_email VARCHAR(255) NOT NULL,
    subject         TEXT NOT NULL,
    rendered_body   TEXT NOT NULL,
    status          VARCHAR(20) NOT NULL,
    error_message   TEXT,
    provider        VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_logs_status ON email_logs(status);
CREATE INDEX IF NOT EXISTS idx_email_logs_action ON email_logs(action);
CREATE INDEX IF NOT EXISTS idx_email_logs_created_at ON email_logs(created_at DESC);
