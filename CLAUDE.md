# VibeCMS

A high-performance, AI-native Go-based CMS optimized for sub-50ms TTFB, featuring a block-based JSON editor, zero-rebuild extension architecture (via Tengo), and automated SEO for agency-managed independent site deployments.

## Tech Stack
- **Languages:** Go 1.22+
- **Frameworks:** Fiber (routing, middleware), GORM (PostgreSQL ORM)
- **Database:** PostgreSQL 16+ (leveraging JSONB and GIN indexes)
- **Frontend/Admin:** HTMX, Alpine.js, Tailwind CSS
- **Templating:** Jet (public themes), Templ (admin interface)
- **Scripting:** Tengo (embedded sandboxed VM for hooks)
- **Storage:** AWS S3/Cloudflare R2 (S3-compatible) & Local Disk
- **Integrations:** Resend (email), OpenAI/Anthropic (AI), Ahrefs/Semrush (SEO)
- **Security:** Ed25519 (license verification)

## Architecture Overview
VibeCMS utilizes a "single-binary, single-site" deployment model. The "Vibe Loop" renders content by fetching JSONB blocks from Postgres, passing them through Tengo scripts for logic-injection, and streaming HTML via the Jet template engine. Assets are optimized to WebP via internal background workers. Admin UI is server-side rendered (SSR) using HTMX, offloading interaction logic to the backend to maintain a sub-50ms TTFB. Internal health monitoring APIs allow agency-level aggregation via static bearer tokens.

## Folder Structure
- `cmd/vibecms/`: Application entry point.
- `internal/`: Private core logic:
    - `cms/`: The core rendering loop and node management.
    - `scripting/`: Tengo VM runtime and hook management.
    - `models/`: GORM models, specifically `content_node` with JSONB.
    - `db/`: Migrations and connection pooling.
- `themes/`: Theme repository containing `.jet` templates and `.tgo` extension scripts.
- `ui/`: Admin portal frontend (Templ components).
- `pkg/`: Shared utility libraries (JSON-schema helpers).
- `storage/`: Local asset storage and backup cache.

## Key Conventions
- **Zero-Rebuild Hooks:** Use `.tgo` scripts in `themes/{theme}/scripts/` for custom logic.
- **Node-Based Content:** All pages, posts, and entities are treated as `content_nodes` with `blocks_data` storage.
- **Admin UI:** All admin interactions must be performed via HTMX fragments; avoid client-side state management outside of simple Alpine.js transitions.
- **Hard-Fail vs. Soft-Fail:** 
    - Database connectivity failures should trigger a fatal server halt.
    - Missing themes or Tengo script errors should log warnings but continue execution.
    - Invalid licenses should disable AI and Tengo features but keep the public site active.
- **Security:** Ensure scripts are executed within a sandboxed `tengo.VM` with restricted I/O.
- **Naming:** Follow `snake_case` for Go files, `.jet` for templates, and `.tgo` for scripting hooks.
- **Performance:** Always prefer atomic operations for hot-swapped configuration maps and cache management.