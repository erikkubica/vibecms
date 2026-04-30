# Presigned-URL Upload Mechanism

**Status:** COMPLETE
**Created:** 2026-05-01
**Worktree:** No

## Problem

Today's MCP tools that accept binary payloads (`core.media.upload`,
`core.theme.deploy`, `core.extension.deploy`) take `body_base64`. Every
client has to read the file, base64-encode it, and stuff it into a JSON
argument. This:

1. Burns 33% bandwidth (base64 overhead).
2. Forces the entire file through the JSON parser into memory before
   the handler sees it.
3. Hits an artificial 50 MB cap that exists to keep the JSON envelope
   sane, not because the disk can't hold more.
4. Makes shell-driven uploads via curl awkward — the encoded blob can
   exceed shell argument limits, forcing the workarounds we hit while
   redeploying themes (stream-build the JSON envelope, manage temp
   files, etc.).

The protocol itself can't carry binary — JSON has no binary type, and
MCP is JSON-RPC. The fix is to take binaries OUT of the MCP envelope
and through a normal HTTP PUT, with the MCP envelope carrying only
metadata.

## Design — three-step token flow

```
1. core.<kind>.upload_init { filename, mime_type, size? }
   → { upload_url, upload_token, expires_at, max_bytes }

2. PUT <upload_url>  (binary body, no JSON, no base64)
   ← { size, sha256 }

3. core.<kind>.upload_finalize { upload_token, sha256? }
   → { id, url, slug, ... }   # same shape as the legacy tool returned
```

Where `<kind>` is one of `media`, `theme`, `extension`.

The upload URL is `/api/uploads/<token>` — an unauthenticated route
where the **token is the auth**. Single-use, 15-minute TTL, scoped to
one kind, bound to the user who issued the init.

The legacy `core.media.upload` / `core.theme.deploy` /
`core.extension.deploy` tools KEEP working with `body_base64` so simple
clients with small payloads (favicons, small assets) don't pay the
3-round-trip cost. Their tool descriptions get a "for files >5 MB
prefer upload_init + upload_finalize" hint.

## Storage — `pending_uploads` table

| col | type | notes |
|---|---|---|
| `token` | varchar(64) PK | 32-byte random hex (~128 bits entropy) |
| `kind` | varchar(16) | `media`, `theme`, `extension` |
| `user_id` | bigint | who issued the init |
| `filename` | text | from init args |
| `mime_type` | text | optional |
| `max_bytes` | bigint | server-imposed cap per kind |
| `state` | varchar(16) | `pending` → `uploaded` → `finalized` |
| `size_bytes` | bigint NULL | populated on PUT |
| `sha256` | char(64) NULL | populated on PUT |
| `temp_path` | text NULL | `data/pending/<token>.bin` |
| `created_at` | timestamptz | |
| `expires_at` | timestamptz | created + 15 min |
| `finalized_at` | timestamptz NULL | |

A background ticker (every 5 min) deletes rows where
`expires_at < now() AND state != 'finalized'` and removes the matching
temp files. Idempotent — surviving rows are visible in the admin if we
ever want a "pending uploads" UI (out of scope for v1).

## Plan — phases

### Phase 1: foundation (~20–30 min)

New package `internal/uploads/`:

- `schema.go` — types (`Kind`, `State`, `PendingUpload`)
- `store.go` — `Issue`, `Validate`, `MarkUploaded`, `MarkFinalized`,
  `Cleanup`. Store wraps `*gorm.DB`.
- `token.go` — `crypto/rand` 32-byte hex
- `handler.go` — `PUT /api/uploads/:token`. Validates token, streams
  body to `data/pending/<token>.bin`, computes SHA-256 on the fly,
  enforces `max_bytes`. Returns `{ size, sha256 }`. Returns 413 on
  oversize, 404 on bad token, 410 on expired, 409 on already-uploaded.
- `cleanup.go` — background ticker, deletes expired-not-finalized rows
  + their temp files.

Migration — new file under `internal/db/migrations/` creating the
`pending_uploads` table and a btree index on `(state, expires_at)`.

Wiring in `cmd/squilla/main.go`:
- Construct `uploads.Store`, `uploads.Handler`, start cleanup ticker
- Mount `app.Put("/api/uploads/:token", uploadsHandler.Receive)`
  OUTSIDE the `adminAPI` group — token IS the auth, no session
  middleware.

### Phase 2: wire MCP tools (~15 min)

Six new tools, two per kind, sharing `pending_uploads`:

**Media:**
- `core.media.upload_init { filename, mime_type, size? }` — issues
  row with `kind=media, max_bytes=envIntDefault("SQUILLA_MEDIA_MAX_MB",50)*1MB`,
  `user_id=current`. Returns `{upload_url, upload_token, expires_at,
  max_bytes}`.
- `core.media.upload_finalize { upload_token, sha256? }` — opens
  `temp_path`, calls existing `cms.MediaService.Upload(io.Reader,
  filename, mime_type, size)` exactly the way the manual UI upload
  does (so media-manager extension's optimisation pipeline runs),
  deletes the temp file, marks the row finalized, returns
  `{id, url, slug, ...}` — same shape as the legacy
  `core.media.upload`.

**Theme:**
- `core.theme.deploy_init { activate?: bool }` — returns same envelope.
  `kind=theme, max_bytes=200MB default`.
- `core.theme.deploy_finalize { upload_token, activate?: bool }` —
  reads temp file, runs the existing theme-deploy logic (unzip,
  validate `theme.json`, atomic `themes/<slug>/` swap, register row,
  optionally activate), deletes temp file, marks row finalized.

**Extension:**
- `core.extension.deploy_init`, `core.extension.deploy_finalize` —
  same as theme but `extensions/<slug>/`.

The legacy three tools keep working unchanged.

### Phase 3: tests (~20 min)

In `internal/uploads/uploads_test.go` and
`internal/cms/uploads_e2e_test.go`:

- **E2E happy path per kind:** init → PUT → finalize → assert
  the real DB row / file / theme exists. Uses the existing
  `internal/testutil` SQLite helper plus a Fiber `app.Test()`
  for the PUT.
- **Token security:**
  - Expired token → 410 Gone on PUT, BAD_REQUEST on finalize
  - Wrong-kind finalize (token issued for media, finalize as theme)
    → rejected
  - Double-finalize → second call rejected
  - Missing token → 404 on PUT
- **Size enforcement:** PUT body > `max_bytes` → 413, no temp file
  left behind, row state stays `pending`.
- **SHA-256 mismatch:** finalize with `sha256` arg that doesn't match
  the stored hash → rejected.
- **Cleanup:** create expired row + temp file, run cleanup tick,
  assert both gone.
- **Concurrency:** two concurrent PUTs to the same token — only one
  succeeds, no half-written file (`os.O_EXCL` on the temp file open
  is the simplest guard).

### Phase 4: docs (~5 min)

- `core.media.upload` description: "For files >5 MB prefer
  `core.media.upload_init` + `core.media.upload_finalize` — direct
  binary PUT, no base64 overhead."
- `docs/extension_api.md` — add a "Presigned uploads" section.
- `docs/architecture.md` — short note on the upload mechanism.
- `CLAUDE.md` — convention note: "MCP tools that take binaries
  bigger than ~5 MB SHOULD provide an `_init`/`_finalize` pair
  alongside the inline-base64 form."

### Phase 5 — optional, defer

- Resumable uploads (Range-PUT with offset). Add only if a real-world
  need surfaces.
- Direct-to-S3 mode for hosted Squilla — `upload_url` points at an S3
  presigned URL, finalize fetches the object from S3 and processes
  locally. Same protocol shape.

## Design decisions

1. **Two MCP tools per kind, not three.** The PUT endpoint is REST,
   not MCP — the only piece that doesn't fit JSON-RPC, so it lives as
   a plain HTTP route. Keeps the MCP surface clean.

2. **Token > session for the PUT.** The PUT endpoint sees no MCP
   session and no admin cookie. The unguessable token IS the auth.
   Cleaner separation, easier to reason about, easier to scale to
   multiple Squilla nodes sharing the DB.

3. **Server picks the temp path, not the client.** Eliminates a
   path-traversal class of bugs.

4. **SHA-256 returned by PUT, optionally verified on finalize.** The
   client can compare against its own hash to detect corruption
   before calling finalize.

5. **Per-kind size cap, declared at init time, written into the row.**
   Media 50 MB, theme/extension 200 MB by default, both env-tunable.
   The PUT route just enforces what the row says — no cross-table
   lookup, no per-route config drift.

6. **No multipart, no chunked transfer.** Single PUT, single file.
   Simpler. Resumability becomes a later opt-in via Range headers
   if anyone actually asks for it.

7. **Backwards compat is forever.** `body_base64` tools stay. They're
   useful for tiny payloads where 3 round-trips is more friction than
   33% bandwidth overhead.

8. **Single commit, not phased PRs.** Lands all three kinds at once.
   Smoke-test by uploading a real big image / theme / extension via
   the new flow; anything broken gets fixed in the same session in
   minutes. The "phased rollout" framing was vestigial team-of-humans
   thinking.

## Files touched

**New:**
- `internal/uploads/schema.go`
- `internal/uploads/store.go`
- `internal/uploads/token.go`
- `internal/uploads/handler.go`
- `internal/uploads/cleanup.go`
- `internal/uploads/uploads_test.go`
- `internal/cms/uploads_e2e_test.go`
- `internal/db/migrations/00XX_pending_uploads.sql`

**Modified:**
- `cmd/squilla/main.go` — wire store/handler/cleanup ticker, mount route
- `internal/mcp/tools_media.go` — add `_init` / `_finalize`
- `internal/mcp/tools_deploy.go` — add `_init` / `_finalize` for theme + extension
- Tool descriptions on the legacy three tools — add the "for big files use init/finalize" hint
- `docs/extension_api.md`, `docs/architecture.md`, `CLAUDE.md`

## Total effort

~1 hour of session time. Real wall-clock dominated by the user
exercising the new flow through a real MCP client (Cursor / Claude
Code) with a big file after deploy.

## Resume instructions for a fresh session

1. Read this file end-to-end.
2. Confirm the design holds — anything to tweak before starting?
3. Implement Phase 1 → 2 → 3 → 4 in one commit.
4. Push when smoke-tested.

## Open questions

- Should the legacy tools auto-detect "big" body and reject with a
  helpful error pointing at `_init`/`_finalize`? (Probably yes — if
  `body_base64` decodes to >5 MB, return error suggesting the new
  flow. Simple, helpful, no breaking change.)
- Token format: hex (64 chars, universal) vs base64url (43 chars,
  shorter URLs). Hex picked for now; nobody's typing these by hand
  so length doesn't matter much.
- Do we need an admin "pending uploads" table view? Probably not v1.
  The cleanup ticker handles housekeeping.
