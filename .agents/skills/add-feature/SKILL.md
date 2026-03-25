```yaml
---
name: add-feature
description: Use when asked to add a new feature not covered by existing task files
---
```

### Workflow

1.  **Analyze Architecture and Scope**: 
    Read `architecture.md` and `product-requirements.md` to understand system constraints. Ensure the feature aligns with the "1-site-per-binary" and "Zero-rebuild" principles.
2.  **Identify Change Set**:
    Locate the files requiring modification based on `folder-structure.md`:
    *   **Model**: Update `internal/models/` for DB schema changes.
    *   **Repository/Service**: Update `internal/cms/` or `internal/integrations/` for business logic.
    *   **Handler**: Update `internal/api/` or add a specific handler if the feature needs an API endpoint.
    *   **Frontend**: If adding UI elements, update `ui/admin/` using *Templ* and *Alpine.js*.
    *   **Database**: If a migration is needed, add an SQL file to `internal/db/migrations/` using the next sequential number (e.g., `0002_new_feature.sql`).
3.  **Implement**:
    *   Write clean, idiomatic Go code.
    *   Follow the dependency injection patterns established in `cmd/vibecms/main.go`.
    *   Use `vibeutil` from `pkg/vibeutil/` for shared logic.
4.  **Testing**:
    *   Write unit tests in the same package (e.g., `feature_test.go`).
    *   If end-to-end testing is required, leverage the existing testing environment.
    *   Command: `go test ./...`
5.  **Final Verification**:
    *   Run the full test suite.
    *   Ensure no regressions are introduced in the "Vibe Loop" (Core CMS rendering).
    *   Commit changes with a clear, concise message.

---

### Implementation Guidelines

*   **Database**: Use `gorm` model struct tags for JSONB field mappings.
*   **Performance**: If the feature touches the public request cycle, ensure it remains within the 50ms TTFB budget. Avoid blocking I/O; use Goroutines where appropriate.
*   **Security**: Always validate user session/RBAC via `internal/auth/` middleware for any admin-facing features.
*   **Extensibility**: If the feature allows site-specific logic, expose a hook for the `internal/scripting/` (Tengo) engine.
*   **Templating**: If updating the UI, ensure forms follow the `editor_form.templ` pattern for consistency.