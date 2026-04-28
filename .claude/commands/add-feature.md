```yaml
---
name: add-feature
description: Add a new feature to the project
argument-hint: <feature description>
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Add New Feature: $ARGUMENTS

This command provides a standardized workflow for implementing features in **VibeCMS** while adhering to our architecture, performance requirements (sub-50ms TTFB), and "Zero-Rebuild" ethos.

## Step 1: Analyze & Plan
1.  **Review existing structure:** Check `internal/` directory to determine which business service should host the logic (e.g., `cms/`, `integrations/`, or `scheduler/`).
2.  **Schema Check:** If the feature requires persistent data, define new GORM fields in the relevant model within `internal/models/`. If content-related, consider the `JSONB` structure in `content_node.go`.
3.  **Endpoint Design:** Consult `VibeCMS API Specification` (api-spec.md) to define new REST routes if an API surface is needed.

## Step 2: Implementation Strategy
- **Go Logic:** Implement core services in `internal/`. Keep the critical path (The "Vibe Loop") free of heavy blocking calls. If the logic is asynchronous, use Go channels or the `internal/scheduler` background tasks.
- **Admin UI:** If a UI component is required, create a `.templ` file in `ui/admin/`. Use `HTMX` attributes for server-side interaction to maintain lightness.
- **Extensibility:** If the feature should be configurable by agencies without recompilation, expose the necessary hooks to the `internal/scripting/tengo_runtime.go` VM.

## Step 3: Implementation Steps
1.  **Draft Implementation:**
    *   Create or update files in `internal/` or `ui/admin/` as planned.
    *   Add necessary database migrations in `internal/db/migrations/` using the `000X_name.sql` pattern.
2.  **Connect the Service:** Register your new service in `cmd/vibecms/main.go` to ensure lifecycle management (init, health checks).
3.  **Test:** Create a test file in the relevant `internal/` sub-package. VibeCMS prioritizes unit tests for Core CMS Logic and Integration handlers.
    *   `go test ./internal/...`
4.  **Verify:**
    *   Start the local dev environment: `docker-compose up`.
    *   Check logs for any startup errors or migration failures.
    *   Verify TTFB remains performant by curling the endpoint: `curl -w "@curl-format.txt" -o /dev/null -s "http://localhost:8080/..."`.

## Step 4: Verification Checklist
- [ ] Code follows `snake_case` naming conventions.
- [ ] Database migrations are idempotent.
- [ ] Exposed API endpoints are secured by the appropriate Middleware (RBAC or Bearer Token).
- [ ] All new logic is covered by unit tests in the respective `internal` package.
- [ ] Docs are updated: If you added an API, ensure `api-spec.md` is updated.

### Execution
Proceed by creating your implementation files, running `go mod tidy` if new dependencies were added, and verifying against the `cmd/vibecms/main.go` entry point.