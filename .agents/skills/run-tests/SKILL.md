```yaml
---
name: run-tests
description: Use when asked to run tests or verify a feature works
---
```

### Workflow for Running Tests in VibeCMS

1.  **Identify Test Framework & Scope**:
    *   VibeCMS uses the standard Go testing toolchain (`go test`).
    *   If a specific test or package is provided as an argument, use the `-run` flag or target a specific folder (e.g., `go test -v ./internal/cms/...`).

2.  **Run Full Test Suite**:
    Execute the following command from the project root to ensure system-wide integrity:
    ```bash
    go test -v -race -cover ./...
    ```

3.  **Analyze Output**:
    *   Review the output for `FAIL` markers.
    *   Identify the exact package path and test function name that failed.
    *   Examine the error trace to distinguish between integration failures (e.g., DB connection, missing mocks) and logic failures.

4.  **Investigate Root Cause**:
    *   Navigate to the file identified in the stack trace.
    *   If the test failure involves `internal/`, inspect the relevant `_test.go` file and the implementation file (e.g., `content_svc.go` vs `content_svc_test.go`).
    *   Verify if the issue is a regression in logic, a dependency mismatch, or a concurrency issue (since `-race` is used).

5.  **Implement Fixes**:
    *   Apply fixes directly to the implementation code.
    *   **Strict Rule:** Do not modify test assertions to "make them pass." If an assertion is invalid, you must justify changing it, but usually, a failing test indicates a bug in the Go logic, Tengo script, or SQL migration.

6.  **Verify**:
    *   Re-run the failing test specifically to verify the fix:
        `go test -v -run <TestName> ./path/to/package/`
    *   Re-run the full suite to ensure no side effects:
        `go test -v ./...`

7.  **Generate Report**:
    *   Summarize the results: total tests passed, failed, and any lingering edge cases identified during the fix.

---

### Command Reference

| Action | Command |
| :--- | :--- |
| **Run all tests** | `go test -v ./...` |
| **Run with race detection** | `go test -v -race ./...` |
| **Run specific test** | `go test -v -run TestFunction ./internal/package/` |
| **Run with coverage** | `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out` |

*Note: VibeCMS requires a running PostgreSQL instance for integration tests. Ensure `docker-compose.yml` is active (or a local DB is configured) before running tests that interact with the database.*