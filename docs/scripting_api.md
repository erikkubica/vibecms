# VibeCMS Theme Scripting API

Complete developer reference for building themes with VibeCMS's embedded scripting system.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Getting Started](#2-getting-started)
3. [Module Import System](#3-module-import-system)
4. [cms/events](#4-cmsevents)
5. [cms/filters](#5-cmsfilters)
6. [cms/routing](#6-cmsrouting)
7. [cms/nodes](#7-cmsnodes)
8. [cms/settings](#8-cmssettings)
9. [cms/http](#9-cmshttp)
10. [cms/email](#10-cmsemail)
11. [cms/menus](#11-cmsmenus)
12. [cms/helpers](#12-cmshelpers)
13. [cms/log](#13-cmslog)
14. [Standard Library](#14-standard-library)
15. [Script Execution Model](#15-script-execution-model)
16. [Writing Theme Modules](#16-writing-theme-modules)
17. [Complete Examples](#17-complete-examples)

---

## 1. Overview

VibeCMS includes a sandboxed scripting engine that lets theme developers add custom logic without modifying the Go core. Scripts are written in **[Tengo](https://github.com/d5/tengo)**, a fast, secure, embeddable scripting language with Go-like syntax.

### What You Can Do

- **Inject HTML** into template hook points (banners, alerts, widgets)
- **Transform values** through filter chains (modify titles, content, metadata)
- **Expose REST APIs** for frontend AJAX calls (search, filtering, custom endpoints)
- **React to lifecycle events** (node published, user registered, content deleted)
- **Send emails** by triggering email rules or sending directly
- **Query and modify content** programmatically (CRUD operations on nodes)
- **Read and write site settings**

### Key Properties

- **Sandboxed**: No filesystem or network access. Scripts cannot escape the VM.
- **Stateless**: Every script execution gets a fresh VM. No global state persists between calls.
- **Thread-safe**: Multiple scripts can execute concurrently without interference.
- **Fail-safe**: Script errors are logged but never crash the server. The public site continues operating.

---

## 2. Getting Started

### Entry Point

Every theme's scripting starts with a single file:

```
themes/<your-theme>/scripts/theme.tengo
```

This file is executed **once at server startup**. Its only job is to **register** event handlers, filters, and HTTP routes. It does not render anything itself.

### Directory Structure

```
themes/your-theme/
  scripts/
    theme.tengo              # Entry point (required)
    hooks/                   # Event handler scripts (render-time)
      hello_world.tengo
      banner.tengo
    handlers/                # Lifecycle event handlers
      on_node_published.tengo
      on_user_registered.tengo
    filters/                 # Filter chain scripts
      site_title_suffix.tengo
      uppercase_titles.tengo
    api/                     # HTTP endpoint handler scripts
      search.tengo
      nodes_by_type.tengo
    lib/                     # Shared importable modules
      helpers.tengo
      formatters.tengo
```

The directory names (`hooks/`, `handlers/`, `filters/`, `api/`, `lib/`) are conventions, not requirements. You can organize files however you like. Script paths in registrations are relative to the `scripts/` directory and omit the `.tengo` extension.

### How Scripts Are Loaded

1. On server startup, the engine looks for `scripts/theme.tengo` in the active theme directory.
2. If found, it compiles and executes `theme.tengo` in a sandboxed Tengo VM.
3. During execution, calls to `events.on()`, `filters.add()`, and `http.get()` (etc.) register handlers.
4. After `theme.tengo` completes, all registered event handlers are wired to the internal event bus, and HTTP routes are mounted on the web server.
5. When a registered event fires or an HTTP request arrives, the corresponding handler script is compiled and executed in a **new, fresh VM**.

### Minimal theme.tengo

```tengo
log := import("cms/log")
events := import("cms/events")

log.info("My theme is loading!")

// Register a hook that runs on every page render
events.on("before_main_content", "hooks/my_banner")

log.info("My theme is ready!")
```

---

## 3. Module Import System

Tengo uses `import()` to load modules. VibeCMS provides three categories of importable modules:

### CMS API Modules (`cms/*`)

Built-in modules providing access to CMS functionality:

```tengo
events  := import("cms/events")    // Event registration and emission
filters := import("cms/filters")   // Filter registration
routing := import("cms/routing")   // Current page context (render-time only)
nodes   := import("cms/nodes")     // Content CRUD operations
settings := import("cms/settings") // Site settings read/write
http    := import("cms/http")      // REST endpoint registration
email   := import("cms/email")     // Email triggering
menus   := import("cms/menus")     // Menu retrieval
helpers := import("cms/helpers")   // String/text utility functions
log     := import("cms/log")       // Logging
```

### Theme Source Modules (`./*`)

Your own `.tengo` files in the `scripts/` directory, imported with a `./` prefix. The path is relative to `scripts/` and omits the `.tengo` extension:

```tengo
// Import scripts/lib/helpers.tengo
my_helpers := import("./lib/helpers")

// Import scripts/lib/formatters.tengo
fmt := import("./lib/formatters")

// Use exported functions
result := my_helpers.truncate("Hello world", 5)
```

Theme modules must use `export` to expose their API (see [Writing Theme Modules](#16-writing-theme-modules)).

### Standard Library Modules

A safe subset of the Tengo standard library is available:

```tengo
fmt    := import("fmt")     // String formatting (sprintf, etc.)
math   := import("math")    // Math operations
text   := import("text")    // Regex, string manipulation
times  := import("times")   // Date/time operations
rand   := import("rand")    // Random number generation
json   := import("json")    // JSON encode/decode
base64 := import("base64")  // Base64 encode/decode
hex    := import("hex")     // Hex encode/decode
enum   := import("enum")    // Enumeration helpers (map, filter, etc.)
```

**Not available**: `os` and any file/network access modules. This is intentional for security.

---

## 4. cms/events

The event system is the primary way scripts interact with VibeCMS's rendering pipeline and lifecycle. Events are used for two distinct purposes: **template hooks** (injecting HTML during page renders) and **lifecycle events** (reacting to CMS actions like publishing a node).

### API

#### `events.on(name, script_path, priority?)`

Registers a handler script for the given event name.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Event name to listen for |
| `script_path` | string | yes | Path to handler script, relative to `scripts/`, without `.tengo` |
| `priority` | int | no | Execution order. Lower = runs first. Default: `50` |

```tengo
events := import("cms/events")

// Register with default priority (50)
events.on("node.published", "handlers/on_node_published")

// Register with explicit priority (10 runs before 50)
events.on("before_main_content", "hooks/banner", 10)
events.on("before_main_content", "hooks/hello_world", 20)
```

#### `events.emit(name, payload?, args?)`

Fires a custom event, triggering any registered handlers (both script-based and Go-based like email rules).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Event name to fire |
| `payload` | map | no | Key-value data passed to handlers |
| `args` | array | no | Extra arguments (stored in `payload._args`) |

```tengo
events := import("cms/events")

// Fire a simple event
events.emit("my_theme.initialized")

// Fire with payload
events.emit("contact.submitted", {
    name: "John Doe",
    email: "john@example.com",
    message: "Hello!"
})

// Fire with payload and extra args
events.emit("custom.action", {key: "value"}, ["arg1", "arg2"])
```

### Priority System

When multiple handlers are registered for the same event, they execute in **priority order** (lowest number first):

| Priority | Runs |
|----------|------|
| 1-20 | First (high priority) |
| 50 | Default |
| 80-100 | Last (low priority) |

For template events, each handler's HTML output is **concatenated** in priority order. For lifecycle events, handlers run sequentially but their return values are independent.

### Template Events (Render-Time Hooks)

Template events are triggered from Go templates using the `{{event}}` function. They run during page rendering and can inject HTML into the page.

**In a Go template:**
```html
<main>
  {{event "before_main_content" .}}
  {{.node.blocks_html}}
  {{event "after_main_content" .}}
</main>
```

**In the handler script**, the full render context is available via `cms/routing`, and the script sets `response` to return HTML:

```tengo
// hooks/banner.tengo
routing := import("cms/routing")

if routing.is_homepage() {
    response = {
        html: `<div class="banner">Welcome to our site!</div>`
    }
}
```

The handler can also return a plain string instead of a map:

```tengo
response = "<p>Hello from a script!</p>"
```

**Passing arguments from templates:**

Templates can pass extra arguments to event handlers:

```html
{{event "sidebar_widget" . "featured" 5}}
```

In the handler script, these are available via the `args` variable:

```tengo
// hooks/sidebar_widget.tengo
// args[0] = "featured", args[1] = 5
widget_type := args[0]
count := args[1]
```

### Lifecycle Events

Lifecycle events are fired by the CMS core when actions occur. Handler scripts receive an `event` variable:

```tengo
// handlers/on_node_published.tengo
// The `event` variable is automatically injected
log := import("cms/log")

log.info("Node published: " + string(event.payload.node_id) + " -- " + event.payload.node_title)
```

The `event` variable has this structure:

```tengo
event.action   // string: the event name (e.g., "node.published")
event.payload  // map: event-specific data
```

#### Built-in Lifecycle Events

| Event Name | Payload Fields | Description |
|------------|---------------|-------------|
| `node.created` | `node_id`, `node_title`, `node_type`, `slug` | A content node was created |
| `node.updated` | `node_id`, `node_title`, `node_type`, `slug` | A content node was updated |
| `node.published` | `node_id`, `node_title`, `node_type`, `slug` | A content node was published |
| `node.deleted` | `node_id`, `node_title`, `node_type` | A content node was deleted |
| `user.registered` | `user_id`, `email`, `name` | A new user registered |
| `user.deleted` | `user_id`, `email` | A user was deleted |

You can also listen for **custom events** emitted by other scripts via `events.emit()`.

---

## 5. cms/filters

Filters transform values through a priority-ordered chain of scripts. They are the scripting equivalent of WordPress filters -- each script in the chain receives a value, optionally modifies it, and passes it along.

### API

#### `filters.add(name, script_path, priority?)`

Registers a filter script for the given filter name.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Filter name to hook into |
| `script_path` | string | yes | Path to filter script, relative to `scripts/`, without `.tengo` |
| `priority` | int | no | Execution order. Lower = runs first. Default: `50` |

```tengo
filters := import("cms/filters")

// Append site name to all page titles
filters.add("node.title", "filters/site_title_suffix", 90)

// Run a different filter first (priority 10)
filters.add("node.title", "filters/capitalize_title", 10)
```

### How Filter Chains Work

1. The CMS calls a filter with an initial value.
2. The value is passed to the first filter script (lowest priority) as the `value` variable.
3. The script sets `response` to the modified value.
4. The output becomes the `value` input for the next script in the chain.
5. The final `response` is returned to the CMS.

If a filter script errors or does not set `response`, the value passes through unchanged.

### Writing a Filter Script

Filter scripts receive a `value` variable and must set `response`:

```tengo
// filters/site_title_suffix.tengo
// Appends the site name to page titles for SEO

settings := import("cms/settings")

site_name := settings.get("site_name")
if site_name != undefined && value != undefined {
    response = value + " | " + site_name
} else {
    response = value
}
```

### Template Usage

Filters are invoked from Go templates using the `{{filter}}` function:

```html
<title>{{filter "node.title" .node.title}}</title>
<meta name="description" content="{{filter "meta.description" .node.seo_description}}">
```

### Common Filter Names

You can register filters for any name. These are conventional names used by the default templates:

| Filter Name | Input Type | Description |
|-------------|-----------|-------------|
| `node.title` | string | Page/post title before rendering |
| `meta.description` | string | SEO meta description |
| `node.content` | string | Rendered content HTML |

---

## 6. cms/routing

Provides information about the current page being rendered. This module is **context-aware** -- its functions return meaningful data only during template rendering (inside event hooks and filter scripts triggered by page renders). Outside of render context, functions return `undefined` or `false`.

### API

#### Page Detection

##### `routing.is_homepage()` -> bool

Returns `true` if the current page is the homepage. **Language-aware**: all translations of the homepage also return `true` (resolved via translation groups).

```tengo
routing := import("cms/routing")
if routing.is_homepage() {
    response = {html: "<div>Welcome home!</div>"}
}
```

##### `routing.is_404()` -> bool

Returns `true` if the current page is a 404 error page (slug is "404" or node ID is 0).

```tengo
if routing.is_404() {
    response = {html: "<div>Sorry, page not found.</div>"}
}
```

##### `routing.is_node_type(type)` -> bool

Returns `true` if the current page matches the given node type.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `type` | string | yes | Node type to check (e.g., `"page"`, `"post"`, `"product"`) |

```tengo
if routing.is_node_type("post") {
    // Show blog-specific sidebar
}
```

##### `routing.is_language(code)` -> bool

Returns `true` if the current page is in the specified language.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `code` | string | yes | Language code (e.g., `"en"`, `"fr"`, `"de"`) |

```tengo
if routing.is_language("fr") {
    response = {html: "<div>Bienvenue!</div>"}
}
```

##### `routing.is_slug(slug)` -> bool

Returns `true` if the current page's slug matches.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Slug to check (e.g., `"about"`, `"contact"`) |

```tengo
if routing.is_slug("contact") {
    // Inject contact form widget
}
```

##### `routing.is_logged_in()` -> bool

Returns `true` if the current visitor is a logged-in user.

```tengo
if routing.is_logged_in() {
    response = {html: "<div>Welcome back!</div>"}
}
```

#### Data Accessors

##### `routing.get_node()` -> map | undefined

Returns the full current node (page/post) as a map. Returns `undefined` outside render context.

```tengo
node := routing.get_node()
if node != undefined {
    log.info("Rendering: " + node.title)
}
```

The returned map contains:

| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Node ID |
| `uuid` | string | UUID |
| `node_type` | string | `"page"`, `"post"`, etc. |
| `status` | string | `"draft"`, `"published"`, etc. |
| `language_code` | string | Language code (e.g., `"en"`) |
| `slug` | string | URL slug |
| `full_url` | string | Full URL path |
| `title` | string | Node title |
| `version` | int | Content version |
| `parent_id` | int or undefined | Parent node ID |
| `author_id` | int or undefined | Author user ID |
| `created_at` | string | ISO 8601 timestamp |
| `updated_at` | string | ISO 8601 timestamp |
| `published_at` | string or undefined | ISO 8601 timestamp |
| `blocks_data` | map/array | Parsed JSONB block data |
| `fields_data` | map | Parsed JSONB custom fields |
| `seo_settings` | map | Parsed JSONB SEO settings |

##### `routing.get_user()` -> map | undefined

Returns the current user data, or `undefined` if not logged in.

```tengo
user := routing.get_user()
if user != undefined {
    log.info("User: " + user.email)
}
```

##### `routing.current_url()` -> string | undefined

Returns the full URL path of the current page (e.g., `"/en/about"`).

##### `routing.current_language()` -> string | undefined

Returns the language code of the current page (e.g., `"en"`).

##### `routing.current_node_type()` -> string | undefined

Returns the node type of the current page (e.g., `"page"`, `"post"`).

##### `routing.site_setting(key)` -> string | undefined

Returns a site setting by key. Tries the cached request context first, then falls back to the database.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `key` | string | yes | Setting key name |

```tengo
site_name := routing.site_setting("site_name")
```

---

## 7. cms/nodes

Full CRUD access to content nodes (pages, posts, and any custom node types). Available in all script contexts (not limited to render-time).

### API

#### `nodes.list(options?)` -> {items, total, page}

Lists content nodes with filtering and pagination.

**Options map:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `status` | string | all | Filter by status (`"published"`, `"draft"`, etc.) |
| `node_type` | string | all | Filter by node type (`"page"`, `"post"`, etc.) |
| `language_code` | string | all | Filter by language code |
| `search` | string | none | Search in title/content |
| `page` | int | `1` | Page number (1-based) |
| `per_page` | int | `50` | Results per page (max 200) |

**Returns:**

```tengo
{
    items: [...],  // Array of node maps
    total: 42,     // Total matching count (int)
    page: 1        // Current page (int)
}
```

```tengo
nodes := import("cms/nodes")

// List all published posts
result := nodes.list({
    status: "published",
    node_type: "post",
    per_page: 10
})

for item in result.items {
    log.info(item.title)
}
```

#### `nodes.get(id)` -> node map | undefined

Retrieves a single node by its numeric ID. Returns `undefined` if not found.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | int | yes | Node ID |

```tengo
node := nodes.get(42)
if node != undefined {
    log.info("Found: " + node.title)
}
```

#### `nodes.get_by_slug(full_url)` -> node map | undefined

Retrieves a single node by its full URL path. Returns `undefined` if not found.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `full_url` | string | yes | Full URL path (e.g., `"/en/about"`) |

```tengo
node := nodes.get_by_slug("/en/about")
```

#### `nodes.create(data)` -> node map

Creates a new content node and returns it.

**Data map fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `title` | string | `""` | Node title |
| `slug` | string | `""` | URL slug |
| `node_type` | string | `"page"` | Node type |
| `status` | string | `"draft"` | Status |
| `language_code` | string | `"en"` | Language code |
| `parent_id` | int | none | Parent node ID |
| `blocks_data` | map/array | none | JSONB block content |
| `fields_data` | map | none | JSONB custom fields |
| `seo_settings` | map | none | JSONB SEO settings |

```tengo
new_node := nodes.create({
    title: "My New Page",
    slug: "my-new-page",
    node_type: "page",
    status: "draft",
    language_code: "en",
    fields_data: {
        custom_field: "value"
    }
})
log.info("Created node ID: " + string(new_node.id))
```

#### `nodes.update(id, data)` -> node map

Updates an existing node and returns the updated version. Only fields present in `data` are modified.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | int | yes | Node ID to update |
| `data` | map | yes | Fields to update |

```tengo
updated := nodes.update(42, {
    title: "Updated Title",
    status: "published"
})
```

#### `nodes.delete(id)` -> bool

Deletes a node by ID. Returns `true` on success, `false` on failure.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | int | yes | Node ID to delete |

```tengo
if nodes.delete(42) {
    log.info("Node deleted")
}
```

#### `nodes.query(options)` -> [node maps]

A flexible query builder for advanced queries. Returns an array of node maps (not paginated like `list`).

**Options map:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `where` | map | none | Field-value equality conditions |
| `order` | string | none | SQL ORDER BY clause (e.g., `"created_at DESC"`) |
| `limit` | int | `50` | Max results (max 500) |
| `offset` | int | `0` | Skip N results |

```tengo
// Get the 5 most recently published posts
recent_posts := nodes.query({
    where: {
        node_type: "post",
        status: "published"
    },
    order: "published_at DESC",
    limit: 5
})

for post in recent_posts {
    log.info(post.title + " - " + post.published_at)
}
```

### Node Map Shape

All node-returning functions return maps with these fields:

| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Numeric ID |
| `uuid` | string | UUID string |
| `node_type` | string | `"page"`, `"post"`, etc. |
| `status` | string | `"draft"`, `"published"`, etc. |
| `language_code` | string | Language code |
| `slug` | string | URL slug |
| `full_url` | string | Full URL path |
| `title` | string | Title |
| `version` | int | Content version |
| `parent_id` | int or undefined | Parent node ID |
| `author_id` | int or undefined | Author user ID |
| `created_at` | string | ISO 8601 timestamp |
| `updated_at` | string | ISO 8601 timestamp |
| `published_at` | string or undefined | ISO 8601 timestamp |
| `blocks_data` | map/array or undefined | Parsed JSONB block content |
| `fields_data` | map or undefined | Parsed JSONB custom fields |
| `seo_settings` | map or undefined | Parsed JSONB SEO settings |

---

## 8. cms/settings

Read and write site-level settings (key-value pairs stored in the database).

### API

#### `settings.get(key)` -> string | undefined

Retrieves a setting value by key. Returns `undefined` if the key does not exist or has a null value.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `key` | string | yes | Setting key |

```tengo
settings := import("cms/settings")

site_name := settings.get("site_name")
if site_name != undefined {
    log.info("Site: " + site_name)
}
```

#### `settings.set(key, value)` -> bool

Creates or updates a setting. Returns `true` on success.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `key` | string | yes | Setting key |
| `value` | string | yes | Setting value |

```tengo
settings.set("my_theme_color", "#4f46e5")
```

**Protected keys** -- the following keys cannot be modified by scripts and will return `false`:

- `email_smtp_password`
- `license_key`
- `monitor_bearer_token`

#### `settings.all()` -> map

Returns all non-encrypted settings as a `{key: value}` map. Encrypted settings are excluded for security.

```tengo
all_settings := settings.all()
// all_settings.site_name, all_settings.site_url, etc.
```

---

## 9. cms/http

Register custom REST API endpoints that are handled by Tengo scripts. All routes are mounted under the `/api/theme/` prefix.

### API

#### `http.get(path, script_path)`
#### `http.post(path, script_path)`
#### `http.put(path, script_path)`
#### `http.patch(path, script_path)`
#### `http.delete(path, script_path)`

Each function registers a route for the corresponding HTTP method.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | yes | URL path (appended to `/api/theme`). Supports `:param` placeholders. |
| `script_path` | string | yes | Path to handler script, relative to `scripts/`, without `.tengo` |

```tengo
http := import("cms/http")

http.get("/search", "api/search")              // GET /api/theme/search
http.get("/nodes/:type", "api/nodes_by_type")   // GET /api/theme/nodes/post
http.post("/contact", "api/contact_form")        // POST /api/theme/contact
http.delete("/cache/:key", "api/clear_cache")    // DELETE /api/theme/cache/main
```

### The `request` Object

Handler scripts receive a `request` variable with this structure:

```tengo
request.method   // string: "GET", "POST", etc.
request.path     // string: full request path
request.query    // map: query string parameters {key: value}
request.params   // map: URL parameters from :param placeholders {key: value}
request.headers  // map: HTTP headers {key: value}
request.body     // map or string: parsed JSON body (POST/PUT/PATCH) or raw string
request.ip       // string: client IP address
```

### The `response` Object

Handler scripts set the `response` variable to control the HTTP response. There are several response formats:

#### JSON Response (most common)

```tengo
response = {
    status: 200,        // HTTP status code (default: 200)
    body: {             // Any value -- serialized as JSON
        items: [...],
        total: 42
    }
}
```

#### HTML Response

```tengo
response = {
    status: 200,
    html: "<h1>Hello World</h1>"
}
```

#### Plain Text Response

```tengo
response = {
    status: 200,
    text: "OK"
}
```

#### Custom Headers

```tengo
response = {
    status: 200,
    headers: {
        "X-Custom-Header": "my-value",
        "Cache-Control": "public, max-age=3600"
    },
    body: {data: "value"}
}
```

#### No Content

If `response` is not set (or set to `nil`), a `204 No Content` response is returned.

### Complete HTTP Handler Example

```tengo
// api/search.tengo -- Theme search API endpoint
// GET /api/theme/search?q=keyword&type=page&limit=10

nodes := import("cms/nodes")

q := ""
if request.query.q != undefined {
    q = request.query.q
}

node_type := ""
if request.query.type != undefined {
    node_type = request.query.type
}

limit := 10
if request.query.limit != undefined {
    limit = int(request.query.limit)
}
if limit > 50 {
    limit = 50
}

result := nodes.list({
    search: q,
    node_type: node_type,
    status: "published",
    per_page: limit
})

items := []
for item in result.items {
    items = append(items, {
        id: item.id,
        title: item.title,
        slug: item.slug,
        full_url: item.full_url,
        node_type: item.node_type,
        language_code: item.language_code
    })
}

response = {
    status: 200,
    body: {
        query: q,
        total: result.total,
        items: items
    }
}
```

### URL Parameters Example

```tengo
// api/nodes_by_type.tengo
// GET /api/theme/nodes/:type?page=1&per_page=20

nodes := import("cms/nodes")

node_type := ""
if request.params.type != undefined {
    node_type = request.params.type
}

page := 1
if request.query.page != undefined {
    page = int(request.query.page)
}

per_page := 20
if request.query.per_page != undefined {
    per_page = int(request.query.per_page)
}
if per_page > 100 {
    per_page = 100
}

result := nodes.list({
    node_type: node_type,
    status: "published",
    page: page,
    per_page: per_page
})

items := []
for item in result.items {
    items = append(items, {
        id: item.id,
        title: item.title,
        slug: item.slug,
        full_url: item.full_url,
        language_code: item.language_code
    })
}

response = {
    status: 200,
    body: {
        node_type: node_type,
        total: result.total,
        page: page,
        per_page: per_page,
        items: items
    }
}
```

---

## 10. cms/email

Send emails by integrating with VibeCMS's email rule system. Emails are not sent directly from scripts; instead, scripts trigger events that match configured email rules in the admin panel.

### API

#### `email.trigger(action, payload?)`

Publishes an event that may trigger matching email rules. This is the **preferred** way to send emails from scripts.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | Action name that email rules match against |
| `payload` | map | no | Data passed to the email template |

```tengo
email := import("cms/email")

email.trigger("contact.submitted", {
    to_email: "user@example.com",
    name: "John Doe",
    message: "Hello from the contact form!"
})
```

To make this work, configure an email rule in the admin panel that matches the `"contact.submitted"` action.

#### `email.send(options)`

Sends a direct email by triggering the special `"script.email.send"` event. You must configure an email rule matching `"script.email.send"` in the admin.

**Options map:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `to` | string | yes | Recipient email address |
| `subject` | string | no | Email subject line |
| `template` | string | no | Email template slug (from admin templates) |
| `data` | map | no | Template variables |

```tengo
email := import("cms/email")

email.send({
    to: "admin@site.com",
    subject: "New Contact Form Submission",
    template: "contact-notify",
    data: {
        name: "John Doe",
        message: "Hello!"
    }
})
```

### How It Works

1. Your script calls `email.trigger("action_name", payload)` or `email.send(options)`.
2. The engine publishes an event to the internal event bus.
3. The email rule engine checks if any configured rules match the action name.
4. If a match is found, the rule's email template is rendered with the payload data and sent via the configured email provider (e.g., Resend).

All script-triggered emails include `_source: "theme_script"` in the payload for traceability.

---

## 11. cms/menus

Retrieve navigation menus configured in the admin panel.

### API

#### `menus.get(slug, language_id?)` -> menu map | undefined

Retrieves a menu by its slug, with resolved items. Returns `undefined` if not found.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | yes | Menu slug (e.g., `"main-menu"`, `"footer"`) |
| `language_id` | int | no | Language ID to filter menu items |

```tengo
menus := import("cms/menus")

main_menu := menus.get("main-menu")
if main_menu != undefined {
    for item in main_menu.items {
        log.info(item.title + " -> " + item.url)
    }
}

// Language-specific menu
fr_menu := menus.get("main-menu", 2)
```

#### `menus.list()` -> [menu maps]

Returns all menus as an array.

```tengo
all_menus := menus.list()
for menu in all_menus {
    log.info(menu.name + " (" + menu.slug + ")")
}
```

### Menu Map Shape

```tengo
{
    id: 1,                  // int: menu ID
    slug: "main-menu",      // string: menu slug
    name: "Main Menu",      // string: display name
    language_id: 1,         // int or undefined: associated language
    items: [...]            // array: menu items (see below)
}
```

### Menu Item Shape

```tengo
{
    id: 1,                   // int: item ID
    title: "Home",           // string: display title
    item_type: "node",       // string: "node", "url", "custom"
    url: "/",                // string: resolved URL
    target: "",              // string: link target ("_blank", etc.)
    css_class: "",           // string: custom CSS class
    node_id: 5,              // int or undefined: linked node ID
    children: [...]          // array: nested child items (recursive)
}
```

Menu items support arbitrary nesting through the `children` field.

---

## 12. cms/helpers

A collection of string and text utility functions commonly needed in theme scripts. These are pure functions with no side effects.

### API Reference

#### `helpers.slugify(text)` -> string

Converts a string to a URL-safe slug. Normalizes unicode, lowercases, replaces non-alphanumeric sequences with hyphens.

```tengo
helpers := import("cms/helpers")
helpers.slugify("Hello World!")     // "hello-world"
helpers.slugify("Cafe Creme")       // "cafe-creme"
```

#### `helpers.truncate(text, max_len, suffix?)` -> string

Truncates a string to `max_len` characters, appending a suffix if truncated.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `text` | string | required | Input string |
| `max_len` | int | required | Maximum character length |
| `suffix` | string | `"..."` | Appended when truncated |

```tengo
helpers.truncate("Hello World", 5)          // "Hello..."
helpers.truncate("Hello World", 5, " >>")   // "Hello >>"
helpers.truncate("Hi", 5)                   // "Hi" (no truncation)
```

#### `helpers.excerpt(text, word_count)` -> string

Extracts a word-aware excerpt. Strips HTML tags first, then limits to the specified number of words.

```tengo
helpers.excerpt("Hello beautiful world of Go", 3)  // "Hello beautiful world..."
helpers.excerpt("<p>Hello <b>world</b></p>", 2)     // "Hello world"
```

#### `helpers.strip_html(text)` -> string

Removes all HTML tags from a string.

```tengo
helpers.strip_html("<p>Hello <b>world</b></p>")  // "Hello world"
```

#### `helpers.escape_html(text)` -> string

Escapes HTML special characters (`&`, `<`, `>`, `"`, `'`).

```tengo
helpers.escape_html("<script>alert('xss')</script>")
// "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
```

#### `helpers.lower(text)` -> string

Converts to lowercase.

```tengo
helpers.lower("Hello World")  // "hello world"
```

#### `helpers.upper(text)` -> string

Converts to uppercase.

```tengo
helpers.upper("Hello World")  // "HELLO WORLD"
```

#### `helpers.title_case(text)` -> string

Capitalizes the first letter of each word.

```tengo
helpers.title_case("hello world")  // "Hello World"
```

#### `helpers.contains(text, substr)` -> bool

Returns `true` if `text` contains `substr`.

```tengo
helpers.contains("Hello World", "World")  // true
helpers.contains("Hello World", "world")  // false (case-sensitive)
```

#### `helpers.starts_with(text, prefix)` -> bool

Returns `true` if `text` starts with `prefix`.

```tengo
helpers.starts_with("/en/about", "/en/")  // true
```

#### `helpers.ends_with(text, suffix)` -> bool

Returns `true` if `text` ends with `suffix`.

```tengo
helpers.ends_with("image.png", ".png")  // true
```

#### `helpers.replace(text, old, new)` -> string

Replaces all occurrences of `old` with `new`.

```tengo
helpers.replace("hello world", "world", "tengo")  // "hello tengo"
```

#### `helpers.split(text, separator)` -> [strings]

Splits a string into an array.

```tengo
helpers.split("a,b,c", ",")  // ["a", "b", "c"]
```

#### `helpers.join(array, separator)` -> string

Joins an array of strings with a separator.

```tengo
helpers.join(["a", "b", "c"], ", ")  // "a, b, c"
```

#### `helpers.trim(text)` -> string

Removes leading and trailing whitespace.

```tengo
helpers.trim("  hello  ")  // "hello"
```

#### `helpers.md5(text)` -> string

Returns the MD5 hex digest of a string. Useful for Gravatar URLs and cache keys.

```tengo
helpers.md5("user@example.com")  // "b58996c504c5638798eb6b511e6f49af"
```

#### `helpers.repeat(text, count)` -> string

Repeats a string `count` times (max 1000).

```tengo
helpers.repeat("ha", 3)  // "hahaha"
```

#### `helpers.word_count(text)` -> int

Counts the number of words in a string.

```tengo
helpers.word_count("Hello beautiful world")  // 3
```

#### `helpers.pluralize(count, singular, plural)` -> string

Returns the singular or plural form based on count.

```tengo
helpers.pluralize(1, "item", "items")  // "item"
helpers.pluralize(5, "item", "items")  // "items"
helpers.pluralize(0, "item", "items")  // "items"
```

#### `helpers.default(value, fallback, ...)` -> any

Returns the first non-empty, non-undefined, non-falsy argument. Useful for providing fallback values.

```tengo
helpers.default(undefined, "fallback")       // "fallback"
helpers.default("", "fallback")              // "fallback"
helpers.default("hello", "fallback")         // "hello"
helpers.default(undefined, "", "last")       // "last"
```

---

## 13. cms/log

Write messages to the server log. All messages are prefixed with `[script]` and the log level.

### API

#### `log.info(message)`

Logs an informational message.

```tengo
log := import("cms/log")
log.info("Theme initialized successfully")
// Output: [script] INFO: Theme initialized successfully
```

#### `log.warn(message)`

Logs a warning.

```tengo
log.warn("Deprecated feature used in theme")
// Output: [script] WARN: Deprecated feature used in theme
```

#### `log.error(message)`

Logs an error.

```tengo
log.error("Failed to fetch external data")
// Output: [script] ERROR: Failed to fetch external data
```

#### `log.debug(message)`

Logs a debug message.

```tengo
log.debug("Processing node ID: 42")
// Output: [script] DEBUG: Processing node ID: 42
```

All functions accept a single string argument. To log complex data, use `fmt.sprintf()` or string concatenation:

```tengo
fmt := import("fmt")
log := import("cms/log")

log.info(fmt.sprintf("Found %d nodes of type %s", count, node_type))
```

---

## 13.5 core/assets — Read theme/extension files

Read-only access to files inside the calling theme or extension's own root
directory. Use it to ship templates, fixtures, default content, or per-form
HTML layouts as plain `.html` / `.json` / `.txt` files instead of inlining
multi-line strings in `theme.tengo`.

```tengo
assets := import("core/assets")

// Read a file relative to the theme/extension root.
html := assets.read("forms/trip-order.html")
if is_error(html) {
    log.warn("layout missing: " + string(html))
    html = ""
}

// Existence check (true/false; never returns an error).
if assets.exists("forms/contact.html") {
    // ...
}
```

### API

| Function | Returns | Notes |
|----------|---------|-------|
| `assets.read(path)` | `string` or `error` | Reads UTF-8. Returns an error value if the file is missing or the path escapes the theme root — wrap with `is_error()`. |
| `assets.exists(path)` | `bool` | `true` if the path resolves to a real file inside the root, `false` otherwise. Never returns an error. |

### Path Rules

- Paths are **relative to the theme/extension root** (the parent of the
  `scripts/` directory). For a theme that means `themes/<theme>/<path>`; for
  an extension it means `extensions/<slug>/<path>`.
- Absolute paths (`/etc/passwd`) are rejected.
- Path traversal that escapes the root (`../../...`) is rejected.
- Empty path → error.

### Common Patterns

**Ship a form layout from the theme** (e.g. `themes/<theme>/forms/<slug>.html`)
and seed it via `forms:upsert`:

```tengo
events  := import("core/events")
assets  := import("core/assets")

layout := assets.read("forms/trip-order.html")
if is_error(layout) { layout = "" }

events.emit("forms:upsert", {
    slug:   "trip-order",
    name:   "Trip Booking",
    force:  true,
    layout: layout,
    fields: [ /* … */ ]
})
```

**Bundle a JSON fixture** for default content:

```tengo
helpers := import("core/helpers")
raw     := assets.read("data/regions.json")
regions := helpers.json_decode(raw)
for r in regions { /* … */ }
```

---

## 14. Standard Library

VibeCMS exposes a safe subset of the [Tengo standard library](https://github.com/d5/tengo/blob/master/docs/stdlib.md). The following modules are available:

### `fmt` -- String Formatting

```tengo
fmt := import("fmt")
s := fmt.sprintf("Hello %s, you have %d items", name, count)
```

### `math` -- Mathematical Operations

```tengo
math := import("math")
result := math.sqrt(16)    // 4.0
max := math.max(10, 20)    // 20
```

### `text` -- Regular Expressions and String Tools

```tengo
text := import("text")
matched := text.match("^[a-z]+$", "hello")  // true
replaced := text.re_replace("\\d+", "NUM", "abc123")  // "abcNUM"
```

### `times` -- Date and Time

```tengo
times := import("times")
now := times.now()
formatted := times.time_format(now, "2006-01-02")
```

### `rand` -- Random Numbers

```tengo
rand := import("rand")
n := rand.intn(100)  // Random int 0-99
```

### `json` -- JSON Encoding/Decoding

```tengo
json := import("json")
encoded := json.encode({name: "John", age: 30})
decoded := json.decode(encoded)
```

### `base64` -- Base64 Encoding

```tengo
base64 := import("base64")
encoded := base64.encode("Hello")
decoded := base64.decode(encoded)
```

### `hex` -- Hexadecimal Encoding

```tengo
hex := import("hex")
encoded := hex.encode("Hello")
```

### `enum` -- Enumeration Helpers

```tengo
enum := import("enum")
doubled := enum.map([1, 2, 3], func(k, v) { return v * 2 })  // [2, 4, 6]
filtered := enum.filter([1, 2, 3, 4], func(k, v) { return v > 2 })  // [3, 4]
```

### Restricted Modules

The following standard library modules are **not available** for security:

- `os` -- No filesystem access
- Any module providing network, process, or file I/O access

Scripts are fully sandboxed and cannot interact with the host system beyond the provided CMS API modules.

---

## 15. Script Execution Model

### Sandboxing

Every script runs in a sandboxed Tengo VM with strict resource limits:

| Constraint | Limit | Description |
|-----------|-------|-------------|
| Max allocations | 50,000 | Maximum memory allocations per execution |
| Timeout | 10 seconds | Maximum wall-clock execution time |
| File access | None | No `os` module, no filesystem operations |
| Network access | None | No outbound HTTP, sockets, or DNS |
| Process access | None | No ability to spawn processes or read env vars |

If a script exceeds the allocation limit or timeout, execution is terminated and an error is logged. The CMS continues operating normally.

### Fresh VM Per Execution

Each script execution creates a **brand new VM**. There is no shared state between executions:

- `theme.tengo` runs once at startup to register handlers.
- Each event handler, filter, or HTTP handler runs in its own fresh VM.
- Global variables set in one execution are not visible to the next.
- This means scripts are inherently stateless. Use `cms/settings` for persistent state.

### Thread Safety

Multiple scripts can execute concurrently (e.g., two page renders triggering the same event handler simultaneously). Because each gets its own VM, there are no race conditions or shared mutable state. The CMS API modules (`cms/nodes`, `cms/settings`, etc.) handle their own concurrency internally.

### Error Handling

- **Script compilation errors**: Logged as errors. The handler is skipped.
- **Runtime errors**: Logged as errors. The handler is skipped, and the next handler in the chain runs.
- **Filter errors**: The value passes through unchanged. The next filter in the chain runs.
- **HTTP handler errors**: Returns a `500` status with `{"error": "script execution error"}`.
- **theme.tengo errors**: Logged at startup. Scripting is disabled, but the site continues.

No script error can crash the server.

---

## 16. Writing Theme Modules

Create reusable Tengo modules that can be imported by other scripts in your theme.

### Creating a Module

Place `.tengo` files anywhere inside your theme's `scripts/` directory. Use the `export` keyword to expose functions and values:

```tengo
// scripts/lib/helpers.tengo

fmt := import("fmt")

truncate := func(s, max_len) {
    if len(s) <= max_len {
        return s
    }
    return s[:max_len] + "..."
}

slugify := func(s) {
    result := ""
    for c in s {
        if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
            result += string(c)
        } else if c >= 'A' && c <= 'Z' {
            result += string(c + 32)
        } else if c == ' ' || c == '_' {
            result += "-"
        }
    }
    return result
}

format_date := func(iso_date) {
    if len(iso_date) < 10 {
        return iso_date
    }
    return iso_date[:10]
}

export {
    truncate: truncate,
    slugify: slugify,
    format_date: format_date
}
```

### Importing a Module

Use the `./` prefix followed by the path relative to `scripts/`, without the `.tengo` extension:

```tengo
// In any script inside scripts/
my_helpers := import("./lib/helpers")

short := my_helpers.truncate("Hello World", 5)
slug := my_helpers.slugify("My Blog Post")
date := my_helpers.format_date("2026-01-15T10:30:00Z")
```

### Module Naming Rules

| File Path | Import Path |
|-----------|-------------|
| `scripts/lib/helpers.tengo` | `"./lib/helpers"` |
| `scripts/lib/formatters.tengo` | `"./lib/formatters"` |
| `scripts/utils/date.tengo` | `"./utils/date"` |
| `scripts/hooks/banner.tengo` | `"./hooks/banner"` |

The entry file `scripts/theme.tengo` is **not importable** -- it is excluded from the module registry.

### Module Tips

- Modules can import other modules (both `cms/*` API modules and other `./` theme modules).
- Modules can import standard library modules (`fmt`, `json`, etc.).
- The `export` statement must be at the top level of the module, not inside a function or conditional.
- Module source is loaded once and cached. Changes require a server restart.

---

## 17. Complete Examples

### Full theme.tengo Example

```tengo
// theme.tengo -- Complete theme scripting entry point.
// Registers all event handlers, filters, and API routes.

log := import("cms/log")
events := import("cms/events")
filters := import("cms/filters")
http := import("cms/http")

log.info("My Theme scripts initializing...")

// ---- Events ---------------------------------------------------------------

// Template hooks (render-time, inject HTML)
events.on("before_main_content", "hooks/banner", 10)
events.on("before_main_content", "hooks/hello_world", 20)
events.on("after_main_content", "hooks/related_posts", 50)
events.on("head_scripts", "hooks/analytics", 99)

// Lifecycle handlers (background, no HTML output)
events.on("node.published", "handlers/on_node_published")
events.on("node.created", "handlers/on_node_created")
events.on("user.registered", "handlers/on_user_registered")

// ---- Filters ---------------------------------------------------------------

// SEO: append site name to page titles
filters.add("node.title", "filters/site_title_suffix", 90)

// Content processing
filters.add("node.content", "filters/auto_link_headings", 50)

// ---- REST API ---------------------------------------------------------------

// Public search API
http.get("/search", "api/search")

// List nodes by type
http.get("/nodes/:type", "api/nodes_by_type")

// Contact form submission
http.post("/contact", "api/contact_form")

// Newsletter signup
http.post("/newsletter/subscribe", "api/newsletter_subscribe")

log.info("My Theme scripts loaded!")
```

### Contact Form Handler

```tengo
// api/contact_form.tengo
// POST /api/theme/contact
// Body: {name: "...", email: "...", message: "..."}

email := import("cms/email")
helpers := import("cms/helpers")
log := import("cms/log")

// Validate required fields
if request.body == undefined {
    response = {status: 400, body: {error: "Missing request body"}}
    return
}

name := ""
if request.body.name != undefined {
    name = helpers.trim(request.body.name)
}

email_addr := ""
if request.body.email != undefined {
    email_addr = helpers.trim(request.body.email)
}

message := ""
if request.body.message != undefined {
    message = helpers.trim(request.body.message)
}

if name == "" || email_addr == "" || message == "" {
    response = {
        status: 400,
        body: {error: "Name, email, and message are required"}
    }
    return
}

// Trigger email rule
email.trigger("contact.submitted", {
    to_email: email_addr,
    name: name,
    message: message,
    ip: request.ip
})

log.info("Contact form submitted by " + name + " (" + email_addr + ")")

response = {
    status: 200,
    body: {success: true, message: "Thank you for your message!"}
}
```

### Conditional Banner with Routing

```tengo
// hooks/banner.tengo
// Displays different banners based on page context.

routing := import("cms/routing")
helpers := import("cms/helpers")
settings := import("cms/settings")

if routing.is_homepage() {
    hero_text := helpers.default(settings.get("hero_text"), "Welcome to our site!")
    response = {
        html: `<section class="hero bg-indigo-600 text-white py-16 text-center">
            <h1 class="text-4xl font-bold">` + helpers.escape_html(hero_text) + `</h1>
        </section>`
    }
} else if routing.is_node_type("post") {
    node := routing.get_node()
    if node != undefined {
        response = {
            html: `<div class="post-header bg-gray-100 py-4 px-6">
                <span class="text-sm text-gray-500">Published: ` + node.published_at + `</span>
            </div>`
        }
    }
} else if routing.is_404() {
    response = {
        html: `<div class="bg-red-50 border-l-4 border-red-500 p-4 mb-6">
            <p class="text-red-700">The page you're looking for doesn't exist.</p>
        </div>`
    }
}
```

### Related Posts Widget

```tengo
// hooks/related_posts.tengo
// Shows related posts after the main content on blog posts.

routing := import("cms/routing")
nodes := import("cms/nodes")
helpers := import("cms/helpers")

if !routing.is_node_type("post") {
    return
}

current := routing.get_node()
if current == undefined {
    return
}

// Get recent posts, excluding the current one
recent := nodes.query({
    where: {
        node_type: "post",
        status: "published",
        language_code: current.language_code
    },
    order: "published_at DESC",
    limit: 4
})

// Filter out current post and limit to 3
items := []
for post in recent {
    if post.id != current.id && len(items) < 3 {
        items = append(items, post)
    }
}

if len(items) == 0 {
    return
}

html := `<section class="related-posts mt-12 border-t pt-8">
    <h2 class="text-2xl font-bold mb-6">Related Posts</h2>
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">`

for post in items {
    excerpt := helpers.excerpt(post.title, 10)
    html += `<a href="` + post.full_url + `" class="block p-4 border rounded hover:shadow-lg transition">
        <h3 class="font-semibold text-lg mb-2">` + helpers.escape_html(post.title) + `</h3>
        <p class="text-sm text-gray-500">` + helpers.default(post.published_at, "") + `</p>
    </a>`
}

html += `</div></section>`

response = {html: html}
```

### Reusable Module with CMS Access

```tengo
// scripts/lib/content_utils.tengo
// Shared content utility functions.

nodes := import("cms/nodes")
helpers := import("cms/helpers")

// Get featured posts for a language
get_featured := func(lang, count) {
    result := nodes.query({
        where: {
            node_type: "post",
            status: "published",
            language_code: lang
        },
        order: "published_at DESC",
        limit: count
    })
    return result
}

// Generate a post card HTML snippet
post_card := func(post) {
    title := helpers.escape_html(post.title)
    excerpt := helpers.excerpt(post.title, 15)
    return `<div class="post-card">
        <a href="` + post.full_url + `">
            <h3>` + title + `</h3>
        </a>
    </div>`
}

export {
    get_featured: get_featured,
    post_card: post_card
}
```

Using it from a handler:

```tengo
// hooks/featured_sidebar.tengo
routing := import("cms/routing")
content := import("./lib/content_utils")

lang := routing.current_language()
if lang == undefined {
    lang = "en"
}

posts := content.get_featured(lang, 5)

html := `<aside class="featured-posts">`
for post in posts {
    html += content.post_card(post)
}
html += `</aside>`

response = {html: html}
```

---

## Quick Reference Card

| Task | Module | Function |
|------|--------|----------|
| Register event handler | `cms/events` | `events.on(name, script, priority?)` |
| Fire custom event | `cms/events` | `events.emit(name, payload?, args?)` |
| Register filter | `cms/filters` | `filters.add(name, script, priority?)` |
| Register API endpoint | `cms/http` | `http.get(path, script)`, `.post()`, `.put()`, `.patch()`, `.delete()` |
| Check page type | `cms/routing` | `routing.is_homepage()`, `.is_404()`, `.is_node_type()`, `.is_slug()` |
| Check language | `cms/routing` | `routing.is_language(code)` |
| Check login status | `cms/routing` | `routing.is_logged_in()` |
| Get current node | `cms/routing` | `routing.get_node()` |
| Get current user | `cms/routing` | `routing.get_user()` |
| List nodes | `cms/nodes` | `nodes.list(options)` |
| Get node by ID | `cms/nodes` | `nodes.get(id)` |
| Get node by URL | `cms/nodes` | `nodes.get_by_slug(url)` |
| Create node | `cms/nodes` | `nodes.create(data)` |
| Update node | `cms/nodes` | `nodes.update(id, data)` |
| Delete node | `cms/nodes` | `nodes.delete(id)` |
| Advanced query | `cms/nodes` | `nodes.query(options)` |
| Read setting | `cms/settings` | `settings.get(key)` |
| Write setting | `cms/settings` | `settings.set(key, value)` |
| All settings | `cms/settings` | `settings.all()` |
| Send email via rule | `cms/email` | `email.trigger(action, payload)` |
| Send direct email | `cms/email` | `email.send(options)` |
| Get menu | `cms/menus` | `menus.get(slug, language_id?)` |
| List menus | `cms/menus` | `menus.list()` |
| Log message | `cms/log` | `log.info()`, `.warn()`, `.error()`, `.debug()` |
| Slugify text | `cms/helpers` | `helpers.slugify(text)` |
| Truncate text | `cms/helpers` | `helpers.truncate(text, len, suffix?)` |
| Word excerpt | `cms/helpers` | `helpers.excerpt(text, words)` |
| Strip HTML | `cms/helpers` | `helpers.strip_html(text)` |
| Escape HTML | `cms/helpers` | `helpers.escape_html(text)` |
| String case | `cms/helpers` | `helpers.lower()`, `.upper()`, `.title_case()` |
| String search | `cms/helpers` | `helpers.contains()`, `.starts_with()`, `.ends_with()` |
| String ops | `cms/helpers` | `helpers.replace()`, `.split()`, `.join()`, `.trim()` |
| MD5 hash | `cms/helpers` | `helpers.md5(text)` |
| Repeat string | `cms/helpers` | `helpers.repeat(text, n)` |
| Word count | `cms/helpers` | `helpers.word_count(text)` |
| Pluralize | `cms/helpers` | `helpers.pluralize(n, singular, plural)` |
| Default value | `cms/helpers` | `helpers.default(val, fallback, ...)` |
