```yaml
---
name: fix-bug
description: Fix a bug in the project
argument-hint: <bug description or error message>
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---
```

## Bug Diagnosis and Fix Workflow

To fix a bug in VibeCMS, follow this structured process. Ensure you are working on a clean branch and verifying all changes locally.

### 1. Reproduce the Issue
Before applying any fix, create a reproduction case to ensure the bug is isolated.
*   **Identify the scope**: If it is a rendering issue, check `internal/cms/content_svc.go` and the `themes/` folder. If it is an API/Database issue, check `internal/api/` or `internal/models/`.
*   **Write a Test**: 
    *   For Go logic: Create a test file in the corresponding `internal/` package (e.g., `internal/cms/content_svc_test.go`).
    *   For API issues: Use the existing Go test suite to mock the request that triggers the failure.
*   **Verify Failure**: Run the test:
    ```bash
    go test ./internal/... -v -run Test[NameOfYourTest]
    ```

### 2. Identify Root Cause
Use the following commands to trace the fault:
*   **Search logs**: Check `storage/logs/` (if enabled) or the console output.
*   **Code Search**: Use `grep` to find relevant definitions:
    ```bash
    grep -r "error_message_or_context" internal/
    ```
*   **Inspect Models**: If the issue involves data state, inspect `internal/models/` to check if JSONB parsing or GORM schema mapping is correct.

### 3. Implement the Fix
Apply changes according to the project's architectural constraints:
*   **Strictly follow patterns**:
    *   Do not add new dependencies; use existing libraries (Go standard library, Fiber, GORM, Jet, Tengo).
    *   If fixing a UI bug, modify `ui/admin/*.templ`.
    *   If fixing an extension issue, check `internal/scripting/tengo_runtime.go` for sandbox constraints.
*   **Consistency**: Ensure the file follows the naming conventions defined in `directory-structure.md`.

### 4. Verify the Fix
*   **Run existing tests**: Ensure no regressions by running the package suite:
    ```bash
    go test ./internal/... -v
    ```
*   **UI Verification**: If the fix involves the Admin UI, ensure formatting is consistent with `tailwind.config.js`.
*   **Build Check**: Verify the binary still compiles:
    ```bash
    go build -o bin/vibecms cmd/vibecms/main.go
    ```

### 5. Finalize
*   Document the fix in your commit message following the format: `fix(component): $ARGUMENTS`.
*   If the bug required an architectural change, ensure you update any relevant documentation files in the root.

---
**Bug Description for this session:**
$ARGUMENTS