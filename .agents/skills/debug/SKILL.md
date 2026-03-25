```yaml
---
name: debug
description: Use when a bug is reported or tests are failing
---

1. **Reproduce:** 
   - Run the specific test case: `go test -v ./internal/... -run Test[NameOfFailingTest]`
   - Reproduce the edge case manually using the local dev environment: `docker-compose up` or `make run`.
   - Inspect local logs in `storage/logs/` (or stdout) for stack traces.

2. **Read:** 
   - Examine relevant source files based on the stack trace (e.g., `internal/cms/content_svc.go` for rendering issues or `internal/db/postgres.go` for data persistence).
   - Check `go.mod` if the error relates to package versions.

3. **Identify:** 
   - Pinpoint whether the bug is in the **Vibe Loop** (Rendering), **Tengo VM** (Scripting), or **Auth layer** (License/RBAC).
   - Ensure the root cause is addressed (e.g., fixing a GORM JSONB query vs. just sanitizing output).

4. **Implement:** 
   - Apply a minimal fix. 
   - If modifying `internal/`, respect the strict module boundaries defined in `architecture.md`.
   - **Crucial:** Scripts in `themes/default/scripts/` should be treated as user-land code; fix the core engine to handle script failures gracefully (e.g., adding timeouts or safe-mode execution).

5. **Verify:** 
   - Run the test suite: `go test ./...`
   - Verify performance in the browser (Network tab) to ensure the bug fix didn't exceed the 50ms TTFB threshold.
   - Run `go vet` and `golangci-lint` to ensure code quality compliance.

6. **Commit:** 
   - Use imperative mood: `git commit -m "fix: [component] [short description of bug]"`
   - Example: `fix: content_svc cache invalidation when blocks are updated`
```