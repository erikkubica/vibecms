# Archived Documentation

These documents are kept for historical reference but **do not reflect the current architecture**. They were authored during early planning (March 30, 2026) when the project was scoped around HTMX/Alpine.js admin UI, Jet/Templ templating, Argon2id passwords, and other choices that have since been replaced.

For current documentation, see `docs/` one directory up.

## What changed since these were written

| Doc | Why archived |
|---|---|
| `overview.md` | Product positioning has shifted from "agency-only fast CMS" to "AI-native MCP-first CMS." Current pitch lives in the project `README.md`. |
| `tech-stack.md` | Lists Jet, Templ, HTMX, Alpine.js, Argon2id — replaced by Go `html/template`, React + Vite SPA, Tailwind v4, bcrypt. See `architecture.md`. |
| `folder-structure.md` | Lists `ui/admin/`, `internal/scheduler/`, `themes/default/blocks/*.jet` — none of these exist. The current structure is documented in `architecture.md` and the project `CLAUDE.md`. |
| `goals.md` | Sub-50ms TTFB and zero-rebuild are still goals, but the document frames the project around agency portfolios, which no longer captures the AI-native intent. |
| `testing.md` | Specifies a Jet/Templ/HTMX test pyramid with Playwright E2E. Current testing strategy is documented inline in `core_dev_guide.md` §8. |
| `observability.md` | References Prometheus metrics + W3C TraceContext spans for Tengo/Jet that were never implemented. Current logging is structured slog with request-id correlation. |
| `deployment.md` | Describes manual SSH deployment with Templ generation and Tailwind v3 CLI. Current deployment is Docker + Coolify (one-click) with a multi-stage Dockerfile; see project `README.md`. |
| `api-spec.md` | (deleted, not archived) Listed REST endpoints for the admin SPA. The system has shifted to MCP-first; admin endpoints live behind VDUS layout trees. See `vdus.md` and the per-domain MCP tools in `internal/mcp/tools_*.go`. |
| `product-requirements.md` | 96 KB greenfield product spec with Jet templates, Argon2, S3 storage abstraction. Useful as a "what was envisioned" artifact only. |
| `development-phases.md` | Phase tracker that already self-admits Phase 3 was rewritten as React. Phases 4-7 are mostly done now (Tengo, media-manager, AI/MCP, monitoring all shipped). |
| `VDUS_HANDOFF.md` | Snapshot of VDUS status as of 2026-04-25. Current state is in `vdus.md`. |
