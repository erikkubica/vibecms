```yaml
---
name: implement-task
description: Use when asked to implement a specific task from .claude/tasks/
---

# implement-task Workflow

To implement a task, follow these steps strictly:

1.  **Read Task:** Read the task definition file from `.claude/tasks/[task-id].md`.
    *   Example: `cat .claude/tasks/001-setup-project.md`
2.  **Analyze Requirements:**
    *   Review the "Context" and "What to implement" sections.
    *   Verify against `VibeCMS` architecture (`.agents/docs/architecture.md`) and technical stack requirements (`.agents/docs/tech-stack.md`).
3.  **Execute Implementation:**
    *   Write or modify the necessary files in the project structure provided in `.agents/docs/folder-structure.md`.
    *   Use the existing tech stack (Go, Fiber, GORM, Jet/Templ, Tengo).
    *   Follow the referenced naming conventions.
4.  **Verify:**
    *   Run the verification command defined in the task file.
    *   Example: `go test ./internal/...` or `make verify`.
5.  **Commit:**
    *   Commit the changes atomically.
    *   Git message format: `git commit -m "feat(task-[id]): [brief description of implementation]"`

## Quality Assurance Rules
*   **No Scope Creep:** Only implement the features defined in the task file.
*   **Strict Adherence:** Use the file paths exactly as specified in the folder structure documentation.
*   **Performance First:** Ensure implementations comply with the sub-50ms TTFB requirement (e.g., avoid blocking I/O in the primary request path).
*   **Sandboxing:** Any extensions must utilize the `internal/scripting/tengo_runtime.go` wrapper.
```