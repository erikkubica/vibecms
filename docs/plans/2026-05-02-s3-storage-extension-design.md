# S3-Compatible Storage Extension — Design

**Status:** PENDING
**Created:** 2026-05-02
**Worktree:** No

## Problem

Squilla today writes every file directly to local disk via
`coreImpl.StoreFile` (`internal/coreapi/impl_filestorage.go`). For
operators running on managed hosts (Coolify, Hetzner volumes, single-VM
setups) that's fine; for anyone wanting CDN-backed media, durable
object storage, multi-instance deployments, or a clean lift between
hosting providers it's a hard ceiling.

The fix is a bundled `s3-storage` extension that routes media bytes
through any S3-compatible backend (AWS S3, Cloudflare R2, Backblaze B2,
DigitalOcean Spaces, Wasabi, self-hosted MinIO) while keeping the
kernel and the rest of the codebase unaware of S3 as a concept.

## Constraints (non-negotiable)

1. **Hard rule:** uninstalling the extension must not leave dead code
   in core. Kernel cannot learn the word "S3".
2. **Themes and extensions stay local.** Bootstrapping an extension
   stored in the very S3 it provides is a paradox.
3. **No URL rewriting in `media_files`.** Logical paths must remain
   stable so deactivating the extension cleanly degrades. (Solves the
   #1 historical complaint about WordPress S3 plugins.)
4. **No data loss on deactivate or delete.** Tables must persist
   through the standard extension lifecycle. Dropping data is
   opt-in with a loud warning.
5. **Resumable migration.** A 50k-file library must survive crashes,
   restarts, and partial failures.
6. **Self-hosted parity.** The same code path must work against MinIO
   in a Coolify-style docker compose as against AWS S3 in production.

## Architecture

The extension plugs into existing kernel-generic primitives. **Zero
changes to `internal/coreapi/` or `internal/cms/`.**

### Three new event topics (published by `media-manager`, subscribed by
`s3-storage` only when verified)

| Topic | Direction | Payload | Reply |
|---|---|---|---|
| `media.storage.store`  | request/reply | `{path, data}` | `{external_url, etag}` |
| `media.storage.read`   | request/reply | `{path}`       | `{bytes}` |
| `media.storage.delete` | request/reply | `{path}`       | `{deleted: bool}` |

Media-manager publishes via `events.PublishRequest`. On
`events.ErrNoSubscriber` (extension deactivated, never installed, or
not yet verified), media-manager falls back to the existing local
`host.StoreFile` / `os.ReadFile` / `host.DeleteFile` calls.

### One new filter

`media.url.resolve(media_id, default_url) → final_url`

Media-manager applies this filter when emitting media URLs in API
responses. `s3-storage` subscribes; if a row exists in
`s3_file_locations`, returns the CDN-aware external URL; otherwise
passes through. Filter chain is already kernel-generic.

### `provides: ["storage-provider"]` tag

Declared in the manifest for future extension discovery (backups,
exports, etc. could call `pm.GetProvider("storage-provider")`).
Optional but cheap.

### Hard-rule compliance

- Deactivate → no subscribers → media-manager publishes into the
  void → falls back to local. URLs resolve via filter pass-through →
  back to `/media/...`.
- Delete extension (default) → tables persist; bytes in S3 untouched.
- Delete extension with explicit "drop data" toggle → tables gone;
  bytes still in S3 (operator can use lifecycle rules in their cloud
  console to delete).
- Reinstall after delete → migrations idempotent; if data was kept,
  index rehydrates automatically. If data was dropped, "Reconcile from
  bucket" rebuilds the index.

## Component layout

```
extensions/s3-storage/
├── extension.json
├── README.md
├── migrations/
│   └── 0001_init.sql
├── cmd/plugin/
│   ├── main.go              # gRPC plugin entry; loads settings; starts worker
│   ├── client.go            # minio-go client wrapper (endpoint, region, path-style)
│   ├── verify.go            # 5-step connection probe
│   ├── handler.go           # HandleHTTPRequest router
│   ├── events.go            # subscribes media.storage.{store,read,delete} when verified
│   ├── filter.go            # subscribes filter media.url.resolve
│   ├── migration.go         # background worker (forward + backward direction)
│   ├── reconcile.go         # bucket walk → repopulate s3_file_locations
│   └── url.go               # external URL composition (CDN prefix vs bucket URL)
├── admin-ui/
│   ├── vite.config.ts
│   ├── src/index.tsx
│   ├── src/Dashboard.tsx
│   ├── src/SettingsForm.tsx
│   └── src/MigrationProgress.tsx
└── scripts/build.sh
```

### Manifest highlights

```json
{
  "slug": "s3-storage",
  "provides": ["storage-provider"],
  "priority": 50,
  "auto_activate": false,
  "capabilities": [
    "settings:read", "settings:write",
    "data:read", "data:write", "data:delete",
    "events:emit", "events:subscribe",
    "filters:register",
    "files:read", "files:write", "files:delete",
    "log:write"
  ],
  "data_owned_tables": ["s3_file_locations", "s3_migration_jobs"],
  "settings_schema": {
    "endpoint":          { "type": "string", "label": "Endpoint", "placeholder": "leave blank for AWS S3" },
    "region":            { "type": "string", "label": "Region", "default": "us-east-1" },
    "bucket":            { "type": "string", "label": "Bucket", "required": true },
    "access_key_id":     { "type": "string", "label": "Access Key ID", "required": true },
    "secret_access_key": { "type": "string", "label": "Secret Access Key", "required": true, "sensitive": true },
    "path_style":        { "type": "boolean","label": "Path-style addressing", "default": true,
                           "help": "Required for MinIO and many S3-compat providers." },
    "public_url_prefix": { "type": "string", "label": "Public URL prefix",
                           "help": "CDN URL (e.g. https://cdn.example.com). Leave blank to use bucket URL." },
    "keep_local_days":   { "type": "number", "label": "Keep local copy for N days after migration", "default": 7 }
  }
}
```

### Database schema (extension-owned)

```sql
-- s3_file_locations: per-media-row pointer to S3 object
CREATE TABLE IF NOT EXISTS s3_file_locations (
    media_id     BIGINT      PRIMARY KEY REFERENCES media_files(id) ON DELETE CASCADE,
    storage_key  TEXT        NOT NULL,
    external_url TEXT        NOT NULL,
    etag         TEXT,
    uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- s3_migration_jobs: per-row migration state, resumable
CREATE TYPE s3_migration_direction AS ENUM ('upload', 'download');
CREATE TYPE s3_migration_status    AS ENUM ('pending', 'in_progress', 'done', 'failed', 'skipped');

CREATE TABLE IF NOT EXISTS s3_migration_jobs (
    id          BIGSERIAL PRIMARY KEY,
    media_id    BIGINT NOT NULL REFERENCES media_files(id) ON DELETE CASCADE,
    direction   s3_migration_direction NOT NULL,
    status      s3_migration_status    NOT NULL DEFAULT 'pending',
    attempts    INT NOT NULL DEFAULT 0,
    last_error  TEXT,
    started_at  TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    UNIQUE (media_id, direction)
);

CREATE INDEX s3_migration_jobs_status_idx ON s3_migration_jobs(status, id);
```

### Admin endpoints

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/test-connection` | 5-step probe; per-step result |
| `GET`  | `/status` | Dashboard: verified, drift counts, last sync |
| `POST` | `/migration/start` | Enqueue forward (`upload`) or backward (`download`) job |
| `GET`  | `/migration/progress` | SSE stream `{total, done, failed, in_progress, last_error}` |
| `POST` | `/migration/cancel` | Soft-cancel after current batch |
| `POST` | `/migration/retry-failed` | Re-enqueue rows in `failed` |
| `POST` | `/reconcile-from-bucket` | Walk bucket, repopulate `s3_file_locations` |

### Touch points outside the new extension (media-manager only)

| File | Change |
|---|---|
| `extensions/media-manager/cmd/plugin/storage.go` | **New** — `mediaStorage{Store,Read,Delete}` helpers wrapping publish-or-fallback |
| `extensions/media-manager/cmd/plugin/crud.go` | Use new helpers instead of direct host calls |
| `extensions/media-manager/cmd/plugin/optimizer.go` | Use new helpers |
| `extensions/media-manager/cmd/plugin/events.go` | Use new helpers |
| `extensions/media-manager/cmd/plugin/helpers.go` (`handlePublicCacheRequest`) | `mediaStorageRead` for original lookups; cache writes stay local |
| `extensions/media-manager/cmd/plugin/render_url.go` | **New** — apply `media.url.resolve` filter when emitting URLs |

**Kernel: zero changes.**

## Data flows

### Upload
Bytes optimized in-memory → `mediaStorageStore` for original + each
variant → publishes `media.storage.store` → s3-storage uploads,
inserts `s3_file_locations` row → returns external URL. On
`ErrNoSubscriber`, falls back to `host.StoreFile`. `media_files.url`
always stores the logical path.

### Render
Media-manager calls `filters.Apply("media.url.resolve", ...)` →
s3-storage returns external URL if it owns the file, else
pass-through.

### On-demand variant resize
`/media/cache/{size}/{path}` → cache hit serves local; cache miss
calls `mediaStorageRead` (S3 GET if migrated, local read otherwise) →
resize → write to local cache → serve.

### Delete
For each path of the media row, `mediaStorageDelete` → S3 removes
object + DELETE `s3_file_locations` row, OR `host.DeleteFile`
fallback. Then `DELETE FROM media_files`.

### Forward migration (local → S3)
Incremental:
```
SELECT m.id FROM media_files m
LEFT JOIN s3_file_locations s ON s.media_id = m.id
WHERE s.media_id IS NULL
```
Worker uploads original + variants, ETag-verifies, inserts
`s3_file_locations`, deletes local after `keep_local_days` grace.
Resumes from `pending`/stale-`in_progress` rows on plugin restart.

### Backward migration (S3 → local)
Same worker, opposite direction. Downloads, verifies size, deletes
`s3_file_locations` row, removes S3 object. After completion the
filter passes through and URLs revert to local.

### Reconcile from bucket
Walks `minio.ListObjects`, matches keys → `media_files.url`,
re-inserts `s3_file_locations` rows. Handles disaster recovery
(extension reinstalled with data dropped, or external uploads to the
bucket).

### Connection probe (5 steps)
`auth → list bucket → put healthcheck → get healthcheck → delete
healthcheck` + a public-URL fetch step if `public_url_prefix` is set.
Each step's failure includes the IAM action name. On all-pass: set
`verified=true`, subscribe events + filter.

## Drift handling

| Direction | Detection | Resolution |
|---|---|---|
| **Forward** (local files not in S3) | Dashboard counter via the LEFT JOIN above | Run/resume migration — skips already-migrated rows |
| **Backward** (S3 keys not in DB) | "Reconcile from bucket" button | Repopulates `s3_file_locations` |
| **Phantom** (DB row → missing S3 object) | "Audit" action HEADs every row (v2, not v1) | Operator decides per file |

V1 ships forward + backward (reconcile). Audit is a follow-up.

## Error handling & edge cases

| Scenario | Behavior |
|---|---|
| S3 unreachable mid-upload | Falls back to local; drift counter picks it up later. No upload lost. |
| Verified, then bucket policy revoked | Next op fails → unsubscribes self → `verified=false` → red banner; future uploads go local. |
| Migration worker crash | Stale `in_progress` rows (>5 min) reset to `pending` on plugin start. ETag verify guards against partial writes. |
| Local source missing | Job marked `skipped` with reason; surfaced in audit. |
| ETag mismatch on PUT verify | Retry once; second mismatch → `failed`, local kept. |
| Filter race during shutdown | Handler checks `verified` atomically; returns pass-through if unsubscribing. |
| Concurrent migration starts | `INSERT ... ON CONFLICT DO NOTHING` + `SELECT ... FOR UPDATE SKIP LOCKED`. |
| Public bucket misconfigured | Probe includes a public-URL fetch step; explicit error if 403. |
| Bucket changed while files reference old bucket | Settings save blocked unless migrated back or reconciled. |
| `keep_local_days` grace deletion fails | Best-effort; log info, continue. |

## Lifecycle correctness

| Action | What happens to extension data |
|---|---|
| Deactivate | Plugin stops, events/filter unsubscribe. Tables untouched. Filter passes through, URLs revert to local. |
| Reactivate | Plugin starts, re-runs probe, re-subscribes. `s3_file_locations` rehydrates resolution immediately. |
| Delete (default) | Plugin and admin-ui assets removed. Tables preserved by default. Reinstall picks up where it left off. |
| Delete (with explicit "drop data") | Tables dropped after loud confirmation. Bytes in S3 stay; operator can use cloud lifecycle rules to clean up, or run reconcile after reinstall. |

## Testing strategy

### Unit tests (Go)
Per-file coverage with `s3Client` interface mocked by an in-memory
fake. Files: `client_test.go`, `verify_test.go`, `url_test.go`,
`filter_test.go`, `events_test.go`, `migration_test.go`,
`reconcile_test.go`. Target ≥ 90% line coverage.

### Integration tests (testcontainers + MinIO)
Real bytes, real network, real ETags. One MinIO container per test in
`cmd/plugin/integration_test.go`. Gated behind `S3_INTEGRATION=1`.

| Test | Asserts |
|---|---|
| `TestEndToEnd_UploadAndResolve` | Object lands in MinIO; filter returns external URL |
| `TestEndToEnd_MigrationForward` | 50 local files → all migrated; 0 local-only after |
| `TestEndToEnd_MigrationResume` | Worker killed mid-batch resumes without duplicates |
| `TestEndToEnd_MigrationBackward` | Files return to local; `s3_file_locations` empty |
| `TestEndToEnd_ReconcileFromBucket` | Drop table, reconcile rebuilds rows |
| `TestEndToEnd_DeleteCascades` | DB delete → object gone from MinIO |
| `TestEndToEnd_DeactivateFallback` | Verify, upload (S3), un-verify, upload (local) — both URLs resolve |
| `TestEndToEnd_VariantOnDemand` | Cold variant fetches from MinIO, caches locally, second hit served from cache |
| `TestEndToEnd_ConnectionProbe` | All-pass and per-step failure scenarios with correct IAM hints |

### Media-manager regression
- `TestMediaStorage_FallbackWhenNoSubscriber` — proves publish-or-fallback degrades to current behavior with no S3 extension.
- All existing media-manager tests must keep passing.

### Admin UI (Vitest + RTL)
- `Dashboard.test.tsx` — drift counter, button states, banners
- `SettingsForm.test.tsx` — required validation, sensitive masking, test-then-save flow
- `MigrationProgress.test.tsx` — SSE updates, retry-failed UI, cancel state

### Playwright E2E (`tests/e2e/s3-storage.spec.ts`)
Happy-path covering activation → settings → connection test → upload
→ migrate → render via S3 URL → delete → deactivate → upload (local)
→ reactivate → drift visible. Runs against MinIO in
`docker-compose.test.yml`.

### Coverage targets
- Unit ≥ 90%
- Every error-handling row in §"Error handling" has at least one assertion
- CI runs unit on every push; integration gated behind env flag

## Library choice

**`github.com/minio/minio-go/v7`.** S3-compatible-first design. Single
small dep. Same code works against AWS S3, MinIO, R2, B2, Spaces,
Wasabi. Used in production by Plausible, Penpot, Outline, Forgejo.
Smaller plugin binary than `aws-sdk-go-v2`.

## Out of scope (v1 explicit non-goals)

- Signed URLs / private buckets — public buckets only for v1.
- Cross-extension storage-provider consumers (backups, exports). The
  `provides: ["storage-provider"]` tag is declared so they can plug in
  later, but no v1 consumer exists.
- Phantom-row audit (`s3_file_locations` rows pointing to deleted S3
  objects). Reconcile handles the inverse; audit is a follow-up.
- Multi-bucket / multi-provider routing.
- Server-side encryption configuration (use bucket-level defaults).
- Lifecycle rules (out-of-tree; configure in cloud console).

## Open questions

None. All design decisions resolved during brainstorming.

## References

- `internal/coreapi/impl_filestorage.go` — current local-disk implementation
- `internal/cms/plugin_manager.go:381` — `provides`-tag dispatch (`GetProvider`)
- `extensions/media-manager/cmd/plugin/optimizer.go` — current `os.ReadFile` / `host.StoreFile` call sites
- `extensions/media-manager/cmd/plugin/helpers.go:140` — on-demand variant resizer
- `extensions/smtp-provider/extension.json` — reference manifest with `provides`-tag and event subscriber pattern
