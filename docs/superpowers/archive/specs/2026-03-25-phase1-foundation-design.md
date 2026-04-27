# Phase 1: Foundation, Auth & Universal Nodes — Design

## Overview
Build the foundational layer of VibeCMS: Go server, PostgreSQL connectivity, GORM models, authentication with RBAC, and Content Node CRUD. This is the bedrock all subsequent phases build on.

## Scope (from development-phases.md)
- Go (Fiber) project scaffolding
- PostgreSQL connection + GORM models for `users`, `sessions`, `content_nodes`
- Auth Middleware + RBAC logic
- Endpoints: `POST /auth/login`, `POST /auth/logout`, `GET /me`
- Node CRUD: `GET /nodes`, `POST /nodes`, `PATCH /nodes/{id}`, `DELETE /nodes/{id}`

## Architecture Decisions

### Module Structure
- Module: `vibecms` (short, clean)
- Entry: `cmd/vibecms/main.go` — initializes Fiber, DB, registers routes
- Follows folder-structure.md exactly

### Database
- PostgreSQL 16 via GORM with connection pooling
- Initial migration: `0001_initial_schema.sql` covering users, sessions, content_nodes, content_node_revisions, redirects, site_settings
- GORM AutoMigrate disabled in favor of explicit SQL migrations run at startup

### Auth
- Session-based auth via secure HTTP-only cookies
- `bcrypt` for password hashing
- RBAC middleware checks role from session (admin, editor, agency-manager)
- Sessions stored in DB with expiry

### Content Nodes
- Universal content model: pages, posts, custom entities
- `blocks_data` as JSONB array
- `seo_settings` as JSONB object
- Soft-delete via `deleted_at`
- Slug auto-generation from title

### API Response Format
- Success: `{"data": {...}, "meta": {...}}`
- Error: `{"error": {"code": "...", "message": "...", "fields": {...}}}`

### Error Handling
- DB connectivity failure → fatal halt
- Validation errors → 400 with field details
- Auth failures → 401/403

## Acceptance Criteria
- User can log in and receive a secure session cookie
- Node can be created with valid JSONB blocks_data
- Request to /nodes returns in <10ms on localhost
- RBAC restricts endpoints by role

## Out of Scope
- Block rendering (Jet templates)
- Media management
- Tengo scripting
- AI features

## References
- docs/architecture.md, docs/database-schema.md, docs/api-spec.md, docs/folder-structure.md, docs/tech-stack.md
