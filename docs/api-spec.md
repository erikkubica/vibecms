# VibeCMS API Specification

## About This Document

**Purpose:** REST endpoint contracts for frontend and external consumers. Defines the interface between backend services and clients.

**How AI tools should use this:** Use the exact endpoint paths, HTTP methods, and request/response shapes when generating API calls or handlers.

**Consistency requirements:** Every endpoint must correspond to a feature in product-requirements.md; request/response fields must map to tables in database-schema.md; service boundaries must match architecture.md.

This document describes the VibeCMS API, which provides structured access to content management, administrative tasks, and system health monitoring. The API is organized into resource-based groups (Nodes, Users, Media, System) and follows standard REST conventions. Authentication is handled via secure session cookies for the Admin UI and Static Bearer Tokens for the Agency Monitoring API. All JSON responses adhere to a consistent structure to facilitate AI-native integrations and agency automation.

---

### Authentication & Authorization
The API uses two distinct authentication models:
1.  **Session-Based:** Used by the Admin UI (`/admin/api/*`). Requires an active session cookie managed via the `/auth` endpoints.
2.  **Bearer Token:** Used by the Monitoring API (`/api/v1/*`). Requires a static token provided in the `Authorization` header.

## Authentication
Endpoints for managing user identity and session state.
Fulfills requirements: REQ-USER-AUTH, REQ-RBAC

### POST /auth/login
Authenticates a user and establishes a session.
- **Authentication:** Public
- **Request Body:**
  - `email` (string, required): Registered user email.
  - `password` (string, required): Plaintext password.
- **Response Body (200 OK):**
  - `user_id` (int): Unique identifier.
  - `email` (string): User email.
  - `role` (string): Assigned role (admin, editor, agency-manager).
- **Status Codes:** 200 (Success), 401 (Invalid credentials).

### POST /auth/logout
Terminates the current session.
- **Authentication:** Authenticated
- **Request Body:** None
- **Response Body (200 OK):**
  - `message` (string): Success message.
- **Status Codes:** 200 (Success).

---

## User Management
Endpoints for managing administrative accounts.
Fulfills requirements: REQ-RBAC

### GET /users
List all administrative users.
- **Authentication:** Authenticated (Admin only)
- **Parameters:** `page`, `per_page`, `role_filter`
- **Response Body (200 OK):**
  - `users` (array): List of user objects (id, email, full_name, role).
  - `meta` (object): Pagination details.
- **Status Codes:** 200 (Success), 403 (Forbidden).

### GET /users/{id}
Retrieve details for a specific user.
- **Authentication:** Authenticated
- **Response Body (200 OK):**
  - `id` (int): Unique identifier.
  - `email` (string): Email address.
  - `full_name` (string): Display name.
  - `role` (string): Role identifier.
  - `last_login_at` (string/iso8601): Timestamp.

### GET /me
Retrieve the profile of the currently authenticated user.
- **Authentication:** Authenticated
- **Response Body (200 OK):** Same as GET /users/{id}.

### POST /users
Create a new administrative user.
- **Authentication:** Authenticated (Admin only)
- **Request Body:**
  - `email` (string, required)
  - `password` (string, required)
  - `full_name` (string)
  - `role` (string, default: 'editor')
- **Response Body (201 Created):** User object with `id`.

### PATCH /users/{id}
Update user details or role.
- **Authentication:** Authenticated (Admin/Self)
- **Request Body:** Partial fields (`email`, `full_name`, `role`, `password`).
- **Response Body (200 OK):** Updated user object.

### DELETE /users/{id}
Remove a user account.
- **Authentication:** Authenticated (Admin only)
- **Status Codes:** 204 (No Content), 403 (Forbidden).

---

## Content Nodes
The primary resource for website content (Pages, Posts, Custom Entities).
Fulfills requirements: REQ-NODE-ARCH, REQ-BLOCK-EDITOR

### GET /nodes
List all content nodes with filtering.
- **Authentication:** Authenticated
- **Parameters:** `page`, `per_page`, `status`, `node_type`, `language_code`, `search` (on title/slug).
- **Response Body (200 OK):**
  - `data` (array): List of nodes excluding detailed `blocks_data`.
  - `meta` (object): Pagination and totals.

### GET /nodes/{id}
Retrieve a complete content node.
- **Authentication:** Authenticated
- **Response Body (200 OK):** Includes `id`, `uuid`, `title`, `slug`, `full_url`, `status`, `blocks_data` (JSONB), `seo_settings` (JSONB).

### POST /nodes
Create a new content node.
- **Authentication:** Authenticated
- **Request Body:** `title`, `node_type`, `language_code`, `parent_id` (optional).
- **Response Body (201 Created):** New node object.

### PATCH /nodes/{id}
Update content, status, or SEO settings.
- **Authentication:** Authenticated
- **Request Body:** Partial JSON containing `title`, `slug`, `status`, `blocks_data`, `seo_settings`.
- **Response Body (200 OK):** Updated node object.

### DELETE /nodes/{id}
Soft-delete a content node.
- **Authentication:** Authenticated
- **Note:** Sets `deleted_at` timestamp; removes node from public routing.

---

## Media Manager
Endpoints for binary asset management and WebP optimization.
Fulfills requirements: REQ-MEDIA-MGMT

### GET /media
List uploaded assets.
- **Authentication:** Authenticated
- **Parameters:** `page`, `per_page`, `mime_type`, `search` (on filename).
- **Response Body (200 OK):** Array of media objects (id, filename, webp_path, byte_size, dimensions).

### GET /media/{id}
Retrieve asset metadata.
- **Authentication:** Authenticated
- **Response Body (200 OK):** Full asset object including `focal_point` and `alt_text`.

### POST /media
Upload a new asset.
- **Authentication:** Authenticated
- **Request:** `multipart/form-data` with `file` field.
- **Response Body (201 Created):** Media object with background optimization status.

### PATCH /media/{id}
Update asset metadata (Alt text, Focal point).
- **Authentication:** Authenticated
- **Request Body:** `alt_text`, `focal_point` (x/y).

### DELETE /media/{id}
Permanently delete a media asset and its optimized derivatives.
- **Authentication:** Authenticated

---

## System Statistics & Monitoring
Agency-facing endpoints for remote instance management.
Fulfills requirements: REQ-MONITOR-API

### GET /api/v1/stats
Detailed telemetry for external dashboards.
- **Authentication:** Static Bearer Token
- **Response Body (200 OK):**
  - `performance` (object): Average TTFB, Memory usage.
  - `health` (object): DB connectivity, backup status.
  - `license` (object): Validity, expiration, domain.
  - `comms` (object): Mail queue status, failed logs.

### GET /api/v1/health
Simple liveness check for load balancers.
- **Authentication:** Public / Internal
- **Response Body (200 OK):** `{"status": "up"}`

---

## System Tasks (Cron)
Management of background jobs and automation.
Fulfills requirements: REQ-TASK-SCHED

### GET /tasks
List all scheduled tasks and status.
- **Authentication:** Authenticated (Admin/Agency)
- **Response Body:** Array of task objects (id, name, cron_expression, next_run_at).

### GET /tasks/{id}
Detail of task and recent execution logs.
- **Authentication:** Authenticated
- **Response Body:** Task object + `logs` (array of statuses and duration).

### POST /tasks/{id}/run
Manually trigger a task execution (e.g., immediate S3 backup).
- **Authentication:** Authenticated (Admin/Agency)
- **Response Body:** `{"message": "Task queued"}`

---

## Error Response Format
All non-2xx responses follow this standard envelope.

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "The provided slug is already in use by another node.",
    "fields": {
      "slug": "must be unique"
    },
    "request_id": "req_88921xabc"
  }
}
```

## HTTP Status Code Summary

| Status Code | Meaning in This API |
|-------------|---------------------|
| 200 | Success. Response body contains requested data. |
| 201 | Created. Resource was successfully generated (e.g., Media/Node). |
| 204 | No Content. Action succeeded (e.g., Delete) with no return body. |
| 400 | Bad Request. Request body or parameters are malformed. |
| 401 | Unauthorized. Session expired or Bearer token invalid. |
| 402 | Payment Required. License signature verification failed (for AI/Tengo). |
| 403 | Forbidden. Authenticated but lacks required role (e.g., Editor trying to edit Tasks). |
| 404 | Not Found. The requested resource ID or URL does not exist. |
| 409 | Conflict. Version mismatch or duplicate slug detected. |
| 429 | Too Many Requests. Rate limit exceeded (Agency API/AI usage). |
| 500 | Internal Server Error. Critical system failure (DB loss, panic). |
| 503 | Service Unavailable. System is in maintenance mode or performing migrations. |