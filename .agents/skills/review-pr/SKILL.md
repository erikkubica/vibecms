```yaml
---
name: review-pr
description: Use when asked to review a pull request or set of changes
---

## Review Workflow

1.  **Scope Identification**: Use `$ARGUMENTS` to identify the PR, branch, or specific files.
2.  **Code Discovery**: Execute `grep` or `find` to map the scope of changes against the VibeCMS directory structure defined in `docs/folder-structure.md`.
3.  **Convention & Architecture Validation**:
    *   **Core Logic**: Ensure logic in `internal/` follows the "Vibe Loop" pattern (ContentSvc -> ScriptEx -> Jet/Templ).
    *   **Performance**: Verify no blocking I/O is introduced in the primary request path that would violate the <50ms TTFB requirement.
    *   **Scripting**: Check that any additions to `themes/*/scripts/*.tgo` remain stateless and cannot access the host filesystem or OS.
    *   **Templates**: Confirm Jet fragments are placed in `themes/default/blocks/` and Templ components in `ui/admin/`.
4.  **Security & Patterns**:
    *   **Auth/RBAC**: Verify `internal/auth/rbac_middleware.go` is applied to new endpoints.
    *   **Database**: Ensure GORM queries use proper indices and avoid N+1 issues when fetching blocks.
    *   **Secrets**: Ensure no API keys or sensitive configurations are hardcoded; enforce environment variable usage.
    *   **License**: Confirm that new features do not bypass the license verification logic in `internal/auth/license_svc.go`.
5.  **Test Verification**: Confirm existence of unit/integration tests for any new service or helper added to `internal/`.
6.  **Findings Summary**: Categorize all observations as:
    *   **BLOCKER**: Direct violations of performance, security, or architectural mandates (e.g., exposing `os` library to Tengo, blocking I/O in the Vibe Loop, missing RBAC).
    *   **WARNING**: Concerns regarding consistency or maintainability (e.g., misnamed files, deviations from `folder-structure.md`, missing test coverage).
    *   **SUGGESTION**: Optional improvements (e.g., performance tuning, minor refactors, documentation updates).

---

## Reference Checklist

*   **Project Documents**:
    *   `docs/architecture.md`: Defines the "Vibe Loop" and service boundaries.
    *   `docs/folder-structure.md`: Enforces correct placement of `.go`, `.tgo`, `.jet`, and `.templ` files.
    *   `docs/tech-stack.md`: Restricts project libraries (Fiber, GORM, Tengo, Jet, Templ, HTMX/Alpine).
*   **Performance Budget**: Ensure all code paths are audited against the <50ms TTFB threshold.
*   **Deployment**: Ensure changes respect the 1-site-per-binary model (no multi-tenancy hacks).

*When reporting, provide specific file paths and cite the relevant architectural standard from the documents above.*