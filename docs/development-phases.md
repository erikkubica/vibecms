# Development Phases: VibeCMS

## About This Document
**Purpose:** A high-level implementation roadmap broken into sequential phases. Each phase defines a deliverable milestone with clear acceptance criteria, dependencies, and scope.
**How AI tools should use this:** Use these phases to understand the build order and scope boundaries; break each phase into granular tasks based on the actual codebase state when starting that phase.
**Consistency requirements:** Phases must reference actual endpoints from api-spec.md, actual tables from database-schema.md, actual components from architecture.md, and actual file paths from folder-structure.md.

This document outlines the sequential construction of VibeCMS, starting from the core high-performance Go engine and concluding with production-ready deployment strategies. Each phase is designed to produce a functional increment of the system, ensuring that the sub-50ms TTFB goal is monitored and maintained through every architectural addition.

---

## Progress Tracker

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 1: Foundation, Auth & Universal Nodes | DONE | Go/Fiber, PostgreSQL, GORM, auth, RBAC, node CRUD API |
| Phase 2: The "Vibe Loop" Rendering Engine | PARTIAL | Public pages render via html/template (Jet planned). Route cache not yet implemented. |
| Phase 3: Admin UI & Block Editor | DONE (rewritten) | **Rewritten as React SPA** (Vite + TypeScript + Tailwind v4 + shadcn/ui). Block editor is JSON textarea (visual editor planned). |
| Phase 4: Zero-Rebuild Extensions & Cron | NOT STARTED | Tengo VM, hooks, cron runner |
| Phase 5: Media Manager & Communications | NOT STARTED | S3 storage, WebP optimization, mail engine |
| Phase 6: AI-Native Layer & Advanced SEO | NOT STARTED | AI bridge, sitemap, SEO automation, multi-language |
| Phase 7: Agency Monitoring & Production | NOT STARTED | License verification, health API (basic version exists), hardening |

**Architecture change:** Admin UI was originally planned as server-rendered HTMX/Alpine.js (Phase 3 spec) but was rewritten as a React SPA for extensibility — plugins can register their own admin UI routes/components.

---

### Phase 1: Foundation, Auth & Universal Nodes
- **Goal:** Establish the Go server environment, database connectivity, and the core "Content Node" data model with secure administrative access.
- **Dependencies:** None. Reference `architecture.md` and `database-schema.md`.
- **Scope:** 
    - Initialize Go (Fiber/Echo) project scaffolding.
    - Setup PostgreSQL connection and GORM models for `users`, `sessions`, and `content_nodes`.
    - Implement `Auth Middleware` and RBAC logic.
    - Build endpoints: `POST /auth/login`, `POST /auth/logout`, `GET /me`.
    - Basic Node CRUD: `GET /nodes`, `POST /nodes`, `PATCH /nodes/{id}`.
- **Acceptance Criteria:**
    - User can successfully log in and receive a secure session cookie.
    - A "node" can be created in the `content_nodes` table with valid `JSONB` for `blocks_data`.
    - Request to `/nodes` returns in <10ms on localhost.
- **Out of Scope:** Block rendering (Jet Templates) and Media management.

### Phase 2: The "Vibe Loop" Rendering Engine
- **Goal:** Implement the high-performance rendering pipeline that transforms JSONB content into HTML via Jet templates.
- **Dependencies:** Phase 1. Reference `Section 3` of `product-requirements.md`.
- **Scope:**
    - Integrate `Jet Rendering Engine` with pre-compilation on startup.
    - Implement the `In-Memory Route Cache` (Radix Tree) to resolve slugs to Node IDs.
    - Develop the "Vibe Loop" logic to iterate through `blocks_data` and map to `/themes/{active}/blocks/{type}.jet`.
    - Implement `Section 5` routing optimizations for sub-50ms TTFB.
- **Acceptance Criteria:**
    - Navigating to a Node's `full_url` renders the correct Jet block templates.
    - TTFB for a basic page remains below 30ms on local benchmarks.
    - Missing templates fail-soft with an HTML comment.
- **Out of Scope:** Tengo scripting hooks and localized routing.

### Phase 3: Admin UI & Block Editor
- **Goal:** Build the server-rendered administrative interface using HTMX and Alpine.js for structured content management.
- **Dependencies:** Phase 2. Reference `architecture.md` and `api-spec.md`.
- **Scope:**
    - Build the Admin Dashboard Layout using Jet fragments.
    - Implement the "Form-to-JSON" generator using HTMX for block fragments.
    - Endpoints: `GET /admin/api/blocks/render-form/{type}`, `PATCH /nodes/{id}` (to save block data).
    - Alpine.js logic for local UI state (modals, block reordering).
- **Acceptance Criteria:**
    - Editor can add/remove/reorder blocks in a Page and save changes to the DB.
    - Slug uniqueness is validated via HTMX during typing (`GET /admin/api/validate-slug`).
- **Out of Scope:** AI-assisted content generation.

### Phase 4: Zero-Rebuild Extensions & Cron
- **Goal:** Enable business logic extensibility via Tengo scripting and establish the internal task scheduler.
- **Dependencies:** Phase 1 & 3. Reference `Section 4` and `Section 13` of `product-requirements.md`.
- **Scope:**
    - Initialize `Tengo VM Runner` with strict sandboxing (stripping `os`/`io`).
    - Implement `before_page_render` and `on_form_submit` Tengo hooks.
    - Create `system_tasks` and `task_logs` tables.
    - Build the internal `Cron Runner` for background tasks.
    - Endpoints: `GET /tasks`, `POST /tasks/{id}/run`.
- **Acceptance Criteria:**
    - A `.tgo` script in the theme folder can modify a Jet template's variable context.
    - A scheduled task successfully logs its execution in the `task_logs` table.
- **Out of Scope:** External S3 backups.

### Phase 5: Media Manager & Communications
- **Goal:** Integrate a high-performance asset pipeline and the SMTP/Resend mail engine.
- **Dependencies:** Phase 1 & 4. Reference `Section 10` and `Section 11` of `product-requirements.md`.
- **Scope:**
    - Implement `StorageDriver` for Local and S3 providers.
    - Build the image optimization pipeline (WebP conversion via `libvips`).
    - Create `media_assets` and `mail_logs` tables.
    - Implement the `Communications Engine` with asynchronous worker pools.
    - Endpoints: `POST /media`, `GET /media`, `DELETE /media/{id}`.
- **Acceptance Criteria:**
    - Uploaded PNG/JPG is automatically stored and a WebP version is generated.
    - A Tengo script can trigger `mail.send()` and the result appears in the `mail_logs` table.
- **Out of Scope:** AI-vision for alt-text generation.

### Phase 6: AI-Native Layer & Advanced SEO
- **Goal:** Deploy the provider-agnostic AI bridge and the global SEO automation suite.
- **Dependencies:** Phase 2, 3, & 5. Reference `Section 7` and `Section 8` of `product-requirements.md`.
- **Scope:**
    - Build `AISvc` bridge for OpenAI/Anthropic.
    - Implement real-time in-memory `sitemap.xml` and `robots.txt` generation.
    - Develop the "Suggest SEO" feature in Admin (Meta title/desc generation).
    - Logic for `Schema.org` JSON-LD injection in Jet templates.
    - Native Multi-language routing support (`Section 9`).
- **Acceptance Criteria:**
    - Admin can generate SEO meta tags via AI with one click.
    - `/sitemap.xml` updates instantly when a new node is published.
    - Path-based routing (e.g., `/en/` and `/sk/`) resolves to the same node with different locale data.
- **Out of Scope:** License migration tools.

### Phase 7: Agency Monitoring & Production Readiness
- **Goal:** Secure the instance with license verification, expose the health API, and finalize DevOps workflows.
- **Dependencies:** All previous phases. Reference `Section 12` and `Section 14` of `product-requirements.md`.
- **Scope:**
    - Implement `Ed25519 License Verification` middleware (Soft-fail policy).
    - Deploy the `Agency Monitoring API` with Static Bearer Token auth.
    - Endpoint: `GET /api/v1/stats`.
    - Setup automated DB migrations and GFS backup rotation.
    - Final security hardening (Header security, rate limiting).
- **Acceptance Criteria:**
    - External dashboard can retrieve instance stats using a Bearer token.
    - System detects an invalid license but continues to serve public content.
    - Full binary bootstrap works on a clean OS with one command.
- **Out of Scope:** Centralized agency master dashboard (external app).