```yaml
---
name: review-code
description: Review code for correctness, security, and consistency with project conventions
argument-hint: <file path or feature to review>
allowed-tools: Read, Glob, Grep
---
```

# Code Review Guide: VibeCMS

Use this guide to review code in the `VibeCMS` repository. Ensure all contributions adhere to the architecture defined in the project documentation.

## 1. Review Scope
When reviewing `$ARGUMENTS`, execute the following checks based on the specific intent of the code.

## 2. Review Criteria

### A. Correctness & Performance
- **TTFB Strategy:** Verify that DB queries for content nodes are indexed (GIN for JSONB). Ensure no N+1 query patterns exist in the "Vibe Loop" (`internal/cms/content_svc.go`).
- **Memory Efficiency:** Check for unnecessary allocations in loops. Use `fiber.Ctx` properly to avoid heap escapes where possible.
- **Async Execution:** Ensure background tasks (Mail, Optimization) use Go channels or non-blocking routines to keep the Request/Response cycle under 50ms.
- **Rendering:** Confirm usage of `Jet` for public views and `Templ` for Admin UI components.

### B. Security
- **Tengo Sandboxing:** For code in `internal/scripting/`, strictly verify that `os` or `io` imports are blocked. Scripts must only interact with the provided `Context` map.
- **License Logic:** Ensure API endpoints calling external AI/Tengo features verify the license via `internal/auth/license_svc.go` (Ed25519 signature check).
- **Authentication:** All administrative routes must be gated by RBAC middleware. Public routes must trigger no sensitive logic (like re-running Cron tasks).
- **Injection:** Verify that all SQL queries use GORM parameterized inputs; raw SQL strings are forbidden without `?` placeholders.

### C. Consistency with Project Conventions
- **Folder Structure:** Check that files are in the correct namespace (e.g., `internal/models/` for GORM structs, `themes/` for templates/scripts).
- **Naming Conventions:**
  - `*_svc.go` for services.
  - `*_handler.go` for API handlers.
  - `.tgo` for Tengo scripts.
  - `.jet` for theme fragments.
  - `.templ` for Admin UI components.
- **Technology Stack:** Do not introduce non-approved dependencies (e.g., no external task queues, keep strictly to Go standard lib, Fiber, and specified SDKs).

## 3. Step-by-Step Review Instructions

1.  **Read the File:** Inspect the target file `$ARGUMENTS`.
2.  **Verify Imports:** Check for any "forbidden" imports outside the stack defined in `tech-stack.md`.
3.  **Trace Data Flow:** If the code interacts with the DB, ensure the `JSONB` structure matches the field schema documented in `internal/models/`.
4.  **Tengo/Scripting Check:** If reviewing `themes/default/scripts/`, ensure they operate only on the passed `Context` and do not attempt to access local files or global state.
5.  **Performance Audit:** Search for potential blocking I/O calls in the hot path of the "Vibe Loop" (public content delivery).
6.  **Report findings:** Return a summary noting:
    - Passed consistency checks.
    - Potential performance bottlenecks.
    - Security vulnerabilities regarding the Sandbox or RBAC.
    - Suggested refactors for better alignment with VibeCMS architecture.