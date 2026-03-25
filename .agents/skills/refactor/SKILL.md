```yaml
---
name: refactor
description: Use when asked to refactor or clean up code
---

### Workflow

1. **Identify Target:** Use `$ARGUMENTS` to determine the file or component to refactor. If no argument is provided, request the target file path.
2. **Analyze Smells:** Read the file and check for:
    - Duplicate logic or "Copy-Paste" programming.
    - Functions exceeding 40 lines.
    - Deeply nested conditionals (if/else/switch).
    - Hardcoded dependencies (missing interface abstraction).
    - Violations of the three-layer architecture (Handler -> Service -> Repository).
3. **Consult Architecture:** Verify against `architecture.md` and `technology-stack.md`. Ensure consistency with VibeCMS patterns:
    - **Service:** Should handle business logic and Tengo hook orchestration.
    - **Repository:** Should be the only layer interacting with GORM/PostgreSQL.
    - **Controller/Handler:** Should focus on request parsing and response rendering (Jet/Templ).
4. **Apply Minimal Refactor:**
    - Extract helper functions or interfaces.
    - Move data access to repository methods.
    - Apply the "Single Responsibility Principle" so each file does only one thing.
5. **Verify:** Run the test suite:
    - `go test ./internal/...`
6. **Commit:** Provide a descriptive commit message explaining:
    - What smell was removed (e.g., "Extracted DB logic to repository").
    - How it aligns with VibeCMS architectural patterns.

### Rules of Engagement

- **Public API:** Never modify function signatures or exported types. 
- **Verifiction:** Run tests after *every change*. Do not bundle multiple logical refactors into a single step without testing.
- **Scope:** Refactor only the target file and its direct private helpers. Do not refactor unrelated files within the same pass.
- **Pattern Alignment:**
    - Use Go interfaces to decouple services from data storage.
    - Use `internal/` package structure to maintain strict encapsulation.
    - Scripts (Tengo) should remain isolated from business service logic; keep the `tengo_runtime.go` interaction clean.

### Standard Commands
- Run Tests: `go test -v ./...`
- Linting: `golangci-lint run` (if configured)
- Build check: `go build ./cmd/vibecms/main.go`