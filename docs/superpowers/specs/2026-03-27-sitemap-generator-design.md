# Sitemap Generator Extension — Design Spec

**Date:** 2026-03-27
**Status:** Approved

## Overview

A hybrid extension using both a gRPC plugin (Go binary for XML generation) and Tengo scripts (event hooks + HTTP route). Exercises both CoreAPI adapters bidirectionally.

## Architecture

```
Tengo script (extension.tengo)          gRPC Plugin (Go binary)
├─ events.on("node.published")  ──────> HandleEvent("sitemap.rebuild")
├─ events.on("node.deleted")    ──────> HandleEvent("sitemap.rebuild")
└─ registers GET /sitemap.xml   ──────> HandleEvent("sitemap.get")
                                              │
                                              ├─ host.QueryNodes() ← CoreAPI
                                              ├─ host.GetSetting("site_url") ← CoreAPI
                                              ├─ host.Log() ← CoreAPI
                                              └─ returns cached XML
```

## gRPC Plugin (Go Binary)

### Event Handling

- `sitemap.rebuild`: Calls `host.QueryNodes(status=published, limit=10000)` and `host.GetSetting("site_url")`. Generates XML sitemap string. Caches in memory (string variable). Logs rebuild via `host.Log()`.
- `sitemap.get`: Returns the cached XML in the EventResponse result bytes. If cache is empty, triggers a rebuild first.

### Initialize

On `Initialize(hostConn)`: creates GRPCHostClient, immediately does first sitemap build so it's ready on first request.

### XML Format

Standard sitemap protocol:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/about</loc>
    <lastmod>2026-03-27T12:00:00Z</lastmod>
  </url>
</urlset>
```

- `<loc>`: site_url + node.full_url
- `<lastmod>`: node.updated_at in RFC3339 format
- Excludes nodes with empty full_url

## Tengo Scripts

### extension.tengo

Registers two event handlers:
- `events.on("node.published", "handlers/rebuild_sitemap", 10)` — emits `sitemap.rebuild`
- `events.on("node.deleted", "handlers/rebuild_sitemap", 10)` — same handler, emits `sitemap.rebuild`

Registers one HTTP route:
- `GET /sitemap.xml` → `handlers/serve_sitemap`

### handlers/rebuild_sitemap.tengo

Emits `sitemap.rebuild` event (triggers the gRPC plugin to rebuild its cache).

### handlers/serve_sitemap.tengo

Emits `sitemap.get` event synchronously. The gRPC plugin handles this and returns XML in the event response. Script sets Content-Type to `application/xml` and returns the XML body.

Note: Since the current EventBus.Publish is async (fire-and-forget), the serve handler will use a simpler approach — it emits `sitemap.get` which triggers the plugin to rebuild/cache, then the handler itself generates a minimal redirect or the plugin writes to a shared setting.

**Revised approach:** The gRPC plugin stores the generated XML in a site setting (`sitemap_xml_cache`). The Tengo route handler reads it via `core/settings.get("sitemap_xml_cache")` and returns it. This avoids the async event response problem entirely.

## Extension Manifest

```json
{
  "name": "Sitemap Generator",
  "slug": "sitemap-generator",
  "version": "1.0.0",
  "author": "VibeCMS",
  "description": "Automatic XML sitemap generation. Rebuilds when content changes.",
  "provides": ["sitemap"],
  "capabilities": [
    "nodes:read",
    "settings:read",
    "settings:write",
    "events:subscribe",
    "events:emit",
    "routes:register",
    "log:write"
  ],
  "plugins": [
    {
      "binary": "bin/sitemap-generator",
      "events": ["sitemap.rebuild", "sitemap.get"]
    }
  ]
}
```

## File Structure

```
extensions/sitemap-generator/
├── extension.json
├── cmd/plugin/main.go
├── scripts/
│   ├── extension.tengo
│   └── handlers/
│       ├── rebuild_sitemap.tengo
│       └── serve_sitemap.tengo
└── bin/                        # compiled binary
```

## Data Flow

### Content Published → Sitemap Rebuild

```
1. User publishes a node
2. CMS emits "node.published" event
3. Tengo handler (rebuild_sitemap.tengo) fires
4. Handler emits "sitemap.rebuild" event
5. gRPC plugin HandleEvent("sitemap.rebuild") fires
6. Plugin calls host.QueryNodes(status=published) via CoreAPI
7. Plugin calls host.GetSetting("site_url") via CoreAPI
8. Plugin generates XML, stores via host.SetSetting("sitemap_xml_cache", xml)
9. Plugin calls host.Log("info", "sitemap rebuilt", {urls: count})
```

### Browser Requests /sitemap.xml

```
1. GET /sitemap.xml hits Tengo route handler
2. serve_sitemap.tengo calls core/settings.get("sitemap_xml_cache")
3. Returns XML with Content-Type: application/xml
4. If cache is empty, returns minimal empty sitemap
```

## Capabilities Used

| Capability | Used By | For |
|---|---|---|
| nodes:read | gRPC plugin | QueryNodes to list all published pages |
| settings:read | gRPC plugin + Tengo | Read site_url, read cached XML |
| settings:write | gRPC plugin | Store generated XML in sitemap_xml_cache |
| events:subscribe | Tengo script | Listen for node.published, node.deleted |
| events:emit | Tengo script | Trigger sitemap.rebuild |
| routes:register | Tengo script | Register GET /sitemap.xml |
| log:write | gRPC plugin | Log rebuild events |
