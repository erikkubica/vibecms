```yaml
---
name: write-tests
description: Write tests for a feature or component
argument-hint: <feature, function, or component to test>
allowed-tools: Read, Write, Edit, Bash, Glob, Grep
---

# Write Tests Command

Use this command to generate or update tests for $ARGUMENTS.

## Guidelines
- **Framework:** This project uses the standard Go `testing` package.
- **Location:** Place test files in the same directory as the source code, named `[feature]_test.go`.
- **Patterns:**
    - Use `testify/assert` or `testify/require` for assertions.
    - Mock external services (like S3 or OpenAI) using standard interface substitution in Go.
    - Database tests in `internal/db` must clean up after themselves.
    - Integration tests for `internal/cms` should use a temporary database connection.

## Step-by-Step Instructions

1. **Understand the Target:** 
   Identify the component in `internal/` or `pkg/`. Review `internal/models/` if the component relies on data structures.

2. **Locate/Create Test File:**
   - If testing `internal/cms/content_svc.go`, the test file is `internal/cms/content_svc_test.go`.
   - If testing `pkg/vibeutil/json_schema.go`, the test file is `pkg/vibeutil/json_schema_test.go`.

3. **Write the Test:**
   - Use table-driven tests for Go functions where possible.
   - For `internal/api` handlers, use `net/http/httptest` to record and assert responses.
   - For `internal/cms` logic, check if you need to initialize a test DB. The project uses GORM; you can set up a `test_db` locally via `docker-compose.yml` if needed.

4. **Run the Test:**
   Execute the following command in your terminal:
   ```bash
   go test -v ./internal/$PACKAGE_PATH/...
   ```

5. **Best Practices Checklist:**
   - [ ] Are external side effects (mail/API calls) mocked?
   - [ ] Is the `Tengo` sandbox tested by feeding it a sample `context`?
   - [ ] Do tests clean up stale records in Postgres?
   - [ ] Does the test run within the environment constraints (no reliance on external services)?

## Example Helper
To test a specific file path use:
```bash
# Example for content service tests
go test -v ./internal/cms/content_svc_test.go
```

If you are unsure where to start, search for existing test patterns:
```bash
grep -r "_test.go" internal/
```