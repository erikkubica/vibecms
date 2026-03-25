# VibeCMS Folder Structure

## About This Document

**Purpose:** Directory layout and naming conventions. Establishes where every type of file belongs in the project.

**How AI tools should use this:** Create new files only in directories consistent with this layout; use the naming conventions specified here.

**Consistency requirements:** Directory structure must reflect framework conventions from tech-stack.md and component boundaries from architecture.md.

VibeCMS is organized as a clean, modular Go application that prioritizes performance and ease of deployment. The structure separates the core engine logic from the flexible theme and scripting layers. This layout ensures that while the backend remains a high-performance compiled binary, agencies can easily extend functionality through the designated theme and scripts folders without needing to recompile the entire system.

---

## Directory Tree

```text
.
├── cmd/                          # Application entry points [hand-written]
│   └── vibecms/                  
│       └── main.go               # Main entry point
├── internal/                     # Private application code [hand-written]
│   ├── api/                      # External monitoring and health APIs
│   │   ├── health_handler.go     # example
│   │   └── stats_handler.go      # example
│   ├── auth/                     # RBAC and License verification logic
│   │   ├── license_svc.go        # example
│   │   └── rbac_middleware.go    # example
│   ├── cms/                      # Core CMS Logic (The "Vibe Loop")
│   │   ├── content_svc.go        # example
│   │   ├── node_router.go        # example
│   │   └── seo_engine.go         # example
│   ├── db/                       # Database connection and Migrations
│   │   ├── migrations/           # SQL migration files [hand-written]
│   │   │   └── 0001_nodes.sql    # example
│   │   └── postgres.go           # example
│   ├── integrations/             # Third-party API clients
│   │   ├── ai_provider.go        # example
│   │   ├── mail_svc.go           # example
│   │   └── s3_storage.go         # example
│   ├── models/                   # GORM models and JSONB structures
│   │   ├── content_node.go       # example
│   │   └── media_asset.go        # example
│   ├── rendering/                # Jet and Templ engine wrappers
│   │   ├── jet_loader.go         # example
│   │   └── templ_renderer.go     # example
│   ├── scheduler/                # Internal Cron system
│   │   ├── backup_task.go        # example
│   │   └── cron_runner.go        # example
│   └── scripting/                # Tengo VM integration
│       └── tengo_runtime.go      # example
├── pkg/                          # Shared library code (exported) [hand-written]
│   └── vibeutil/                 
│       └── json_schema.go        # example
├── ui/                           # Admin Portal Frontend [hand-written]
│   ├── admin/                    # Templ components for Admin UI
│   │   ├── layout.templ          # example
│   │   └── editor_form.templ     # example
│   ├── assets/                   # Static assets for Admin UI
│   │   ├── css/                  # Compiled Tailwind output [generated]
│   │   └── js/                   # Minimal Alpine.js scripts
│   └── views/                    # Raw templates for Admin screens
├── themes/                       # Website themes (One per deployment) [hand-written]
│   └── default/                  
│       ├── blocks/               # Jet fragments for JSONB blocks
│       │   ├── hero_section.jet  # example
│       │   └── text_block.jet    # example
│       ├── layouts/              # Main page wrappers
│       │   └── main.jet          # example
│       └── scripts/              # Tengo extensibility scripts
│           ├── on_form_post.tgo  # example
│           └── before_render.tgo # example
├── storage/                      # Local file storage (if not using S3) [generated]
│   ├── media/                    # Processed WebP images
│   └── backups/                  # Database dumps
├── .env.example                  # Environment configuration template
├── docker-compose.yml            # Local dev environment
├── go.mod                        # Go module dependencies
├── go.sum                        # Go module checksums
├── Makefile                      # Build and task shortcuts
└── tailwind.config.js            # Admin UI styling config
```

---

## File Naming Conventions

*   **Go Files (`.go`):** Use `snake_case`. Files in `internal/models` should match the entity name (e.g., `content_node.go`). Services should end in `_svc.go` or `_handler.go`.
*   **Tengo Scripts (`.tgo`):** Use `snake_case`. Naming should reflect the lifecycle hook they attach to (e.g., `after_save.tgo`).
*   **Jet Templates (`.jet`):** Use `snake_case`. Component blocks should be named after the block type they render (e.g., `image_gallery.jet`).
*   **Templ Components (`.templ`):** Use `snake_case`. Logical UI components (e.g., `status_badge.templ`).
*   **SQL Migrations:** Use sequential four-digit prefixing (e.g., `0001_initial_schema.sql`).

---

## Key Files

| File | Purpose |
| :--- | :--- |
| `cmd/vibecms/main.go` | The entry point that initializes the Fiber server, DB connection, and Cron runner. |
| `internal/cms/content_svc.go` | The core "Vibe Loop" logic that fetches JSONB content and prepares it for rendering. |
| `internal/scripting/tengo_runtime.go` | Manages the Tengo VM lifecycle and injects the Go context into sandbox scripts. |
| `internal/models/content_node.go` | Defines the central PostgreSQL schema for universal content (Pages, Posts, Entities). |
| `themes/default/layouts/main.jet` | The master template that defines the HTML shell for the public-facing website. |
| `internal/auth/license_svc.go` | Handles the Ed25519 cryptographic verification of the domain-based license key. |
| `internal/scheduler/cron_runner.go` | Orchestrates internal background tasks like WebP optimization and S3 backups. |
| `ui/admin/editor_form.templ` | The primary Templ component for generating the block-based JSON editor UI. |

---

## Module Organization

*   **internal/**: Contains all business logic. Code here is protected by Go's visibility rules and cannot be imported by external projects.
*   **themes/**: This directory is designed to be modified by developers/designers. It is external to the core logic, allowing for easy updates to the `vibecms` binary without overwriting site design.
*   **ui/admin/**: Houses the server-rendered components for the CMS management interface. It uses **Templ** for type-safety and **Tailwind** for styling.
*   **pkg/vibeutil**: Utility functions that are generic enough to be shared or exported, such as JSON-schema validation helpers.