## Section 1: Content Node Architecture and Post-Type Configuration

VibeCMS utilizes a unified "Content Node" architecture, where every routable entity—whether a static page, a blog post, a real-estate listing, or a team member profile—is stored within the `content_nodes` table. This approach eliminates the relational overhead of joining disparate tables for different content types, facilitating the system's sub-50ms TTFB target while maintaining strict structural integrity through a JSON-schema-first approach.

### 1.1 The Universal `content_nodes` Schema

The core of VibeCMS is the `content_nodes` table. Unlike traditional CMS platforms that split "Pages" and "Posts" into different architectural buckets, VibeCMS differentiates them via a `node_type` identifier and associated configuration logic.

#### Database Definition (PostgreSQL/GORM)
```go
type ContentNode struct {
    ID              uint32         `gorm:"primaryKey;autoIncrement"`
    UUID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();index"`
    ParentID        *uint32        `gorm:"index"` // Support for hierarchical URIs
    NodeType        string         `gorm:"type:varchar(50);index;default:'page'"`
    Status          string         `gorm:"type:varchar(20);index;default:'draft'"` // draft, published, archived
    Slug            string         `gorm:"type:varchar(255);index"`
    FullURL         string         `gorm:"type:text;uniqueIndex"` // Computed: /parent/child-slug
    LanguageCode    string         `gorm:"type:varchar(10);index;default:'en'"`
    Title           string         `gorm:"type:varchar(255)"`
    
    // Core Data Blobs
    BlocksData      datatypes.JSON `gorm:"type:jsonb"` // The block-based content
    MetaSettings    datatypes.JSON `gorm:"type:jsonb"` // SEO, NoIndex, Canonical, OG Tags
    
    // Performance Attributes
    RenderCacheHash string         `gorm:"type:varchar(64)"`
    
    // Timestamps for Versioning & Audit
    CreatedAt       time.Time
    UpdatedAt       time.Time
    DeletedAt       gorm.DeletedAt `gorm:"index"`
    PublishedAt     *time.Time
    Version         int            `gorm:"default:1"`
}
```

### 1.2 Node Type Configuration (The Blueprint)

Developers define how a `NodeType` behaves using a JSON-based configuration file (or the Admin UI). This configuration dictates which fields are available, which blocks are "allowed" for that specific type, and how the node is routed.

#### Configuration Schema Example (`/config/node_types/property.json`):
```json
{
  "handle": "property",
  "singular_name": "Property",
  "plural_name": "Properties",
  "hierarchical": false,
  "route_pattern": "/rentals/{slug}",
  "allowed_blocks": ["hero_header", "image_gallery", "amenities_list", "contact_form"],
  "custom_fields": [
    {
      "name": "price",
      "type": "Number",
      "required": true
    },
    {
      "name": "location_ref",
      "type": "Relationship",
      "target": "location_node"
    }
  ]
}
```

### 1.3 Block-Type Constraints and Field Definitions

Each Content Node is a collection of "Blocks" stored in the `blocks_data` JSONB column. To ensure the AI-native generation remains consistent, every block must adhere to a strict definition.

#### Supported Field Types (`Vibe-Fields`)
1.  **Text:** Supports plain text, markdown, or limited HTML.
2.  **Repeater:** A list of nested fields (e.g., a "Team" block containing multiple "Person" items).
3.  **Relationship:** A pointer to another ID in the `content_nodes` table (supports 1:1 and 1:N).
4.  **Media:** A reference to the Media Manager ID, ensuring WebP delivery and S3 pathing is handled correctly.

#### The JSONB Structure of `blocks_data`
```json
[
  {
    "id": "blk_8291a",
    "type": "hero_header",
    "order": 0,
    "settings": { "padding_top": "large" },
    "content": {
      "headline": "Luxury Villa in Tuscany",
      "subline": "Experience authentic Italian living.",
      "background_image": 1042
    }
  },
  {
    "id": "blk_4412z",
    "type": "property_stats",
    "order": 1,
    "content": {
      "bedrooms": 4,
      "bathrooms": 2.5,
      "sq_ft": 3200
    }
  }
]
```

### 1.4 Versioning and Point-in-Time Recovery

VibeCMS implements a shadow-tree versioning strategy. Every save operation on a `content_node` triggers a trigger-based or application-level "snapshot" of the `blocks_data` and `MetaSettings`.

*   **Revision Table:** `content_node_revisions` stores the full JSONB delta.
*   **Enforcement:** Revisions are capped at 50 versions per node by default to prevent PostgreSQL bloat.
*   **Recovery:** The Admin UI (HTMX-driven) allows a developer to "Preview Revision." This temporarily injects the historical JSONB into the Jet rendering engine without overwriting the production `content_node` record until "Restore" is confirmed.

### 1.5 Localization & Routing Constraints

Localization is baked into the Node architecture. Rather than using "translation strings," VibeCMS uses a "Parallel Node" approach.

*   **Node Linking:** A `translation_group_uuid` links the English version of a page to its Slovak or Spanish counterpart.
*   **Routing Logic:** If a request comes in for `/sk/domov`, the router queries for `Slug: 'domov'` AND `LanguageCode: 'sk'`.
*   **Fallback Mechanism:** If a specific block is missing content in a localized node, the system can be configured to "Inherit" from the default language node dynamically during the Jet rendering loop.

### 1.6 Edge Case: Dynamic URI Collision Resolution

Since VibeCMS allows Custom Post Types to define custom `route_patterns` (e.g., `/blog/{slug}`), the system runs a pre-save validation hook:
1.  **Regex Check:** Ensures the slug doesn't conflict with reserved system paths (`/admin`, `/api`, `/assets`).
2.  **Breadth-First Slug Resolution:** If two nodes share the same generated `FullURL`, the system appends a numerical suffix (`-1`, `-2`) automatically.
3.  **Cache Invalidation:** On every update to a Node's slug or its parent ID, the `FullURL` of all child nodes is recalculated and the in-memory routing trie is hot-swapped to maintain sub-50ms performance.

## Section 2: Block-Based Editor and JSONB Schema-First Storage

VibeCMS treats content not as a monolithic blob of HTML, but as a structured collection of discrete data units. This section defines the technical implementation of the Block-Based Editor and the underlying PostgreSQL JSONB storage strategy that enables sub-50ms TTFB while maintaining "AI-native" structural integrity.

### 2.1 The JSONB Storage Architecture

To eliminate the performance overhead of EAV (Entity-Attribute-Value) patterns or complex table joins, VibeCMS utilizes a single-column `blocks_data` approach using the PostgreSQL `JSONB` type within the `content_nodes` table.

#### 2.1.1 Schema Definition
The `blocks_data` column stores an array of block objects. Each object must conform to a strict internal schema to ensure the Jet/Templ rendering engine and the AI-integration layer can parse it deterministically.

**Storage Format Example:**
```json
[
  {
    "id": "uui-789-vbc",
    "type": "hero_section",
    "status": "published",
    "fields": {
      "title": "High Performance CMS",
      "subtitle": "Sub-50ms TTFB built in Go",
      "cta_label": "Get Started",
      "background_image": {
        "id": 402,
        "url": "/assets/media/hero.webp",
        "alt": "Abstract Go Gopher"
      }
    },
    "settings": {
      "padding_top": "large",
      "theme_variant": "dark"
    }
  },
  {
    "id": "uui-123-xyz",
    "type": "content_grid",
    "fields": {
      "items": [
        { "title": "Fast", "body": "Compiled Go binary." },
        { "title": "Secure", "body": "RBAC and Sandboxed scripting." }
      ]
    }
  }
]
```

#### 2.1.2 Database Constraints and Indexing
To optimize lookup and filtering within the JSONB blob, VibeCMS implements:
- **GIN (Generalized Inverted Index):** Applied to `blocks_data` to allow high-speed searches for specific block types or field values across thousands of nodes.
- **JSON Schema Validation:** A pre-insert check handled by the Go backend (using `jsonschema` validation) to ensure no malformed JSON enters the `blocks_data` column, preventing runtime errors in the Jet templates.

### 2.2 Vibe-Fields: The Atomic Data Definitions

The editor is powered by "Vibe-Fields," which are the building blocks of every block. These fields are defined in `block-definition.json` files within the theme directory.

| Field Type | JSON Structure | Admin UI Component |
| :--- | :--- | :--- |
| **Text** | `{"type": "string"}` | Single line input or Rich Text (Alpine.js/Tiptap) |
| **Repeater** | `{"type": "array", "items": {...}}` | Dynamic list with Add/Remove/Sort buttons |
| **Relationship**| `{"type": "int", "reference": "node_id"}`| Searchable dropdown/Select2 equivalent |
| **Media** | `{"type": "object", "properties": {"id": "int"}}`| Custom Media Gallery modal with WebP preview |

### 2.3 The Admin Block Editor (HTMX + Alpine.js Logic)

The editor interface is built using a server-rendered "Form-to-JSON" architecture. It avoids the weight of a heavy SPA (Single Page Application) by utilizing HTMX for state fragments and Alpine.js for client-side interactivity.

#### 2.3.1 The "Vibe-Loop" Editor Workflow
1.  **Block Selection:** When a user clicks "Add Block," HTMX performs a `GET` request to `/admin/api/blocks/template/{block_type}`.
2.  **Fragment Injection:** The server returns a partial HTML fragment containing the fields defined in the block's JSON schema.
3.  **State Management:** Alpine.js manages the local state of the block (e.g., reordering items in a Repeater field using `x-for` and basic drag-sort logic).
4.  **Serialization:** Upon clicking "Save," the entire form is serialized into a nested JSON object and sent via a `PATCH` request to the Go backend.

#### 2.3.2 Sub-100ms Interaction Latency
To meet the performance target of sub-100ms UI interaction latency:
- **Partial Updates:** HTMX only swaps the specific block being edited, not the entire page tree.
- **Client-Side Validation:** Alpine.js performs immediate type-checking (e.g., ensuring a "Media" field has an ID before submission).
- **Optimistic UI:** CSS transitions handle the visual movement of blocks during reordering while the background POST request syncs the new array index to Postgres.

### 2.4 AI-Native Schema Integration

VibeCMS is "AI-native" because the block system is fundamentally designed for LLM consumption. 

- **Token Efficiency:** Because the data is stored as clean JSON, the system can strip metadata and send only the `fields` object to OpenAI/Anthropic. This reduces token usage compared to sending raw HTML strings.
- **Structured Suggestions:** The Admin UI includes a "Suggest Content" button per block. When triggered, the Go backend sends the block's JSON schema + current context to the AI provider. The AI returns a JSON object that maps directly back into the Vibe-Fields.
- **Example AI Prompt Logic:**
  > "Using the following JSON Schema for a 'Product_Feature' block, generate three features for a high-performance CMS. Output strictly valid JSON matching the schema."

### 2.5 Logic Hooks and Data Transformation

Before the JSONB data is saved or rendered, it passes through the "Zero-Rebuild" hook system (referenced in Section 4). 

- **`on_block_save`**: A Tengo script can intercept the JSON data. For example, if a block contains a "Price" field, a Tengo script can fetch the current exchange rate and inject a `price_usd` calculated field into the JSONB before it hits the database.
- **Data Versioning:** Every change to the `blocks_data` JSONB triggers a row versioning mechanism in the `content_node_versions` table, allowing point-in-time recovery of the entire block structure.

### 2.6 Constraints and Edge Cases

- **Max Block Depth:** To ensure high-speed recursive rendering in Jet, the CMS enforces a maximum depth of 3 levels for "Repeater" fields within blocks.
- **Invalid Schema Handling:** If a block definition is deleted from the theme but existing nodes still contain that block type in their `blocks_data`, the VibeCMS renderer will gracefully skip the block and log a "Missing Template" warning in the Agency Monitoring API rather than crashing the page.
- **Large Document Performance:** For content nodes exceeding 1MB of JSON data (rare for standard pages), the system triggers a background "optimization notice" suggesting the use of Relationships instead of massive Repeaters to maintain the sub-50ms TTFB target.

## Section 3: Jet/Templ Rendering Engine and Block-to-HTML Pipeline

VibeCMS employs a dual-engine rendering strategy designed to reconcile high-performance static-like speeds with the flexibility of a dynamic, block-based CMS. The pipeline transforms structured `JSONB` data from the `content_nodes` table into semantic HTML using a combination of **Jet** (for high-speed, designer-friendly content orchestration) and **Templ** (for type-safe, logic-heavy UI components).

### 3.1 The Hybrid Rendering Strategy

To achieve a sub-50ms TTFB while maintaining an extensible "Zero-Rebuild" architecture, VibeCMS delineates rendering responsibilities:

1.  **Jet (The Layout & Block Orchestrator):** 
    *   **Role:** Acts as the primary view engine for themes. It handles the "Vibe Loop" (iterating through content blocks).
    *   **Rationale:** Jet is one of the fastest templating engines for Go, offering syntax similar to Twig or Jinja2. This allows agency designers to modify `.jet` files without Go knowledge.
    *   **Execution:** Jet templates are compiled to Go bytecode at runtime (once) and cached in memory, ensuring near-native execution speeds.

2.  **Templ (The Logic-Heavy Component Layer):**
    *   **Role:** Used for complex, stateful components that require high reliability (e.g., recursive comment trees, complex filter facets, or performance-critical internal UI).
    *   **Rationale:** Templ provides Go-native type safety. While Jet is optimized for flexibility and "Zero-Rebuild," Templ is used for the core system components that are baked into the binary.

### 3.2 The "Vibe Loop" Rendering Pipeline

The rendering lifecycle follows a strict sequence to ensure data integrity and performance:

#### Step 1: Context Hydration
When a request hits the router, the system fetches the `ContentNode` and populates a unit of execution called the `RenderContext`.
*   **Global Context:** Site settings, active language, SEO metadata, and navigation menus.
*   **Node Context:** The specific data for the current route (Title, slug, custom fields).
*   **Block Context:** The raw `blocks_data` (JSONB array) retrieved from the database.

#### Step 2: The Block-to-HTML Transformation
The engine iterates through the `blocks_data` array. For each block:
1.  **Identifier Resolution:** The system looks for a corresponding template file at `/themes/{active_theme}/blocks/{block_type}.jet`.
2.  **Tengo Hook (Optional):** If a `before_block_render.tgo` script exists for that block type, the Tengo VM executes it, allowing for dynamic data injection (e.g., fetching a "Latest Posts" list for a "Recent News" block) without recompiling the Go binary.
3.  **Fragment Rendering:** The Jet engine renders the block fragment using the block's specific JSON data.

#### Step 3: Layout Injection
Block fragments are buffered and injected into the primary layout file located at `/themes/{active_theme}/layouts/main.jet`. This layout handles the `<head>`, global headers, and footers.

### 3.3 Technical Specifications for Block Templates

Each block in VibeCMS is a self-contained unit. A typical block template (`/blocks/hero.jet`) adheres to the following structure:

```jet
{{/* Hero Block Template */}}
<section class="hero-wrapper" id="block-{{ .id }}">
    <div class="container">
        <h1>{{ .data.headline }}</h1>
        {{ if .data.subheadline }}
            <p>{{ .data.subheadline }}</p>
        {{ end }}
        
        {{ if .data.primary_cta }}
            <a href="{{ .data.primary_cta.url }}" class="btn">
                {{ .data.primary_cta.label }}
            </a>
        {{ end }}
    </div>
</section>
```

#### Data Constraints:
*   **`.id`**: A unique UUID assigned to the block instance.
*   **`.data`**: The raw JSON object from the `blocks_data` column.
*   **`.settings`**: Block-level configuration (e.g., background color, padding ratios).

### 3.4 Performance Optimization & Caching Logic

To hit the **sub-50ms TTFB** target, the pipeline implements several low-level optimizations:

1.  **Zero-Allocation Buffering:** VibeCMS utilizes a `sync.Pool` of `bytes.Buffer` objects to stream the rendering output. This minimizes Garbage Collector (GC) pressure during high-traffic periods.
2.  **Jet Set Caching:** All `.jet` templates are parsed once and stored in a global `Set` (Jet's internal cache framework). In production mode, filesystem lookups are bypassed entirely.
3.  **In-Memory Fragment Caching:** For blocks that require heavy computation (e.g., complex relationship lookups), the rendering engine supports a `cache_duration` parameter. The resulting HTML fragment is stored in a thread-safe LRU (Least Recently Used) cache keyed by the block's content hash.

### 3.5 Error Handling & Fallbacks

The rendering pipeline is designed to be resilient. If a block fails or a template is missing, the system follows a "Soft-Fail" protocol:
*   **Missing Template:** If `{block_type}.jet` is not found, the system logs a warning and renders an invisible HTML comment: `<!-- Block Error: Template 'hero' not found -->`. This prevents the entire page from crashing.
*   **JSON Mismatch:** If the block data does not match the expected schema in the template, Jet's `{{ if .data.field }}` checks prevent "nil pointer" style errors common in other engines.
*   **Panic Recovery:** A middleware recovers from any engine-level panics during rendering, serving a graceful `500.jet` error page or a "Maintenance Mode" fallback if the layout engine itself is corrupted.

### 3.6 Block Data Schema Example (JSONB)

The following structure represents how the rendering engine consumes the `blocks_data` column for a multi-block page:

```json
[
  {
    "id": "u-4592-x",
    "type": "hero_v1",
    "data": {
      "headline": "Empowering Future Agencies",
      "primary_cta": { "label": "Get Started", "url": "/contact" }
    }
  },
  {
    "id": "u-4593-y",
    "type": "repeater_grid",
    "data": {
      "items": [
        { "title": "Fast", "desc": "Sub-50ms TTFB" },
        { "title": "Safe", "desc": "Go-based architecture" }
      ]
    }
  }
]
```
The pipeline automatically maps `type: "hero_v1"` to `blocks/hero_v1.jet`, passing the corresponding `data` object as the local scope.

## Section 4: Zero-Rebuild Extension System via Tengo Scripting Hooks

To achieve the project goal of avoiding Go recompilation for custom business logic, VibeCMS implements a "Zero-Rebuild" extension system powered by the **Tengo** scripting engine. This architecture allows developers to inject high-performance, sandboxed logic into the request lifecycle and event pipeline without restarting the service or maintaining complex CI/CD rebuild triggers.

### 4.1. The Choice of Tengo as the Scripting Engine
Tengo was selected over Lua or WASM for three primary reasons:
1.  **Go-Native Performance:** Being written in Go, Tengo shares the same memory model and integrates seamlessly with Fiber/Echo without the CGO overhead required by Lua or the startup latency of WASM modules.
2.  **Syntax Familiarity:** Tengo's syntax is nearly identical to Go, reducing the cognitive load for developers already working within the VibeCMS ecosystem.
3.  **Strict Sandboxing:** Tengo provides a controlled environment where the host can explicitly define which modules (e.g., `json`, `math`, `times`) and custom objects are accessible, ensuring that third-party scripts cannot compromise the host OS or filesystem.

### 4.2. Hook Architecture and Execution Lifecycle
The system operates on an event-driven "Hook" model. When the core CMS binary reaches a specific lifecycle point, it checks for the existence of corresponding `.tgo` files within the theme's `/scripts` directory.

#### 4.2.1. Trigger Points (Hooks)
| Hook Name | File Convention | Execution Point | Primary Use Case |
| :--- | :--- | :--- | :--- |
| `before_page_render` | `hooks/before_render.tgo` | After node data fetch, before Jet/Templ execution. | Dynamic data injection, A/B testing logic, paywall checks. |
| `on_form_submit` | `hooks/form_{id}.tgo` | Upon POST request to a specific form ID. | CRM integration, custom validation, automated Slack alerts. |
| `on_cron_task` | `cron/{task_name}.tgo` | Triggered by the Section 13 Task Scheduler. | External API syncing, periodic cleanup, report generation. |

#### 4.2.2. Stateless Runtime Context
Every script execution initializes a fresh, stateless Tengo VM. To maintain the **sub-50ms TTFB** target, VM initialization and script compilation are cached in-memory using a synchronized map (`map[string]*tengo.Compiled`), with cache invalidation triggered only when the `.tgo` file's `mtime` changes.

### 4.3. Data Interoperability (The `VibeContext` Object)
Scripts interact with the CMS through a pre-populated `VibeContext` object injected into the VM's global scope. This object acts as a bridge between the Go backend and the Tengo script.

**Exposed Objects in Tengo:**
*   `request`: Read-only access to Headers, Query Params, Cookies, and Remote IP.
*   `node`: Read/Write access to the current `content_nodes` data (blocks, metadata).
*   `user`: Current SEO/RBAC context (if authenticated).
*   `out`: A mutable map for passing data back to the Jet/Templ rendering engine.
*   `services`: Limited access to CMS utilities like `mail_sender` and `http_client`.

### 4.4. Technical Implementation: The Script Runner
The following pseudo-logic represents the internal Go handler for executing a script hook:

```go
// Internal Script Execution Logic
func ExecuteHook(hookType string, ctx *fiber.Ctx, nodeData *models.ContentNode) (map[string]interface{}, error) {
    scriptPath := fmt.Sprintf("./themes/%s/scripts/%s.tgo", activeTheme, hookType)
    
    // 1. Check if script exists
    if !fileExists(scriptPath) {
        return nil, nil // No extension logic defined
    }

    // 2. Initialize Tengo Script with Sandbox constraints
    s := tengo.NewScript(readScriptFile(scriptPath))
    s.SetImports(stdlib.BuiltinModules()) // Allow json, math, text, times

    // 3. Define VibeContext
    vibeCtx := map[string]interface{}{
        "node_id": nodeData.ID,
        "blocks":  nodeData.BlocksData,
        "query":   ctx.Queries(),
        "ip":      ctx.IP(),
    }
    s.Add("context", vibeCtx)
    s.Add("extra_data", make(map[string]interface{})) // Data to pass to Jet templates

    // 4. Run with Timeout (5ms limit to maintain TTFB)
    compiled, err := s.Compile()
    if err != nil {
        return nil, err
    }
    
    if err := compiled.RunContext(ctx.Context()); err != nil {
        return nil, err
    }

    // 5. Retrieve output
    return compiled.Get("extra_data").Map(), nil
}
```

### 4.5. Practical Example: Custom API Integration
A common use case is fetching external data (e.g., cryptocurrency prices or weather) and injecting it into a page without modifying the Go binary.

**File:** `/themes/default/scripts/hooks/before_render.tgo`
```tengo
// Tengo Scripting logic
text := import("text")
fmt := import("fmt")

// Logic: If the page layout is "dashboard", fetch prices
if context.node_layout == "dashboard" {
    // In a real scenario, use the injected http_client
    res := vibe_http.get("https://api.exchange.com/v1/ticker?symbol=BTC")
    
    if res.status_code == 200 {
        price_data := json_decode(res.body)
        // Inject data into the rendering context
        extra_data["btc_price"] = price_data.last_price
        extra_data["render_status"] = "live_data"
    } else {
        extra_data["render_status"] = "cached_data"
    }
}
```
In this example, the variable `btc_price` becomes instantly available in the **Section 3 Jet Templates** as `{{ btc_price }}`, enabling dynamic content with zero downtime.

### 4.6. Security and Constraints
To prevent the "recursive scripting" or "resource exhaustion" common in interpreted systems, the following boundaries are enforced:

1.  **Memory Limit:** Each Tengo VM is capped at 8MB of RAM. Any script attempting to allocate beyond this is terminated immediately.
2.  **Execution Timeout:** Scripts are terminated by the Go context if they exceed **10ms** of execution time. This ensures that extensions cannot break the **sub-50ms TTFB** performance target.
3.  **File System Isolation:** The `os` and `io` Tengo modules are strictly forbidden. Scripts cannot read or write to the host disk. All "persistence" must happen via the `content_nodes` (JSONB) or external API calls via the `vibe_http` helper.
4.  **License Enforcement:** In accordance with the project's business logic, if a valid **Ed25519 license signature** is not detected, the Tengo runner remains dormant, and all hooks are bypassed.

### 4.7. Error Handling and Debugging
Since scripts are executed in a production environment, VibeCMS provides a specialized log for extensions.
*   **Runtime Errors:** Panics or compilation errors in Tengo scripts are caught, logged to the `system_logs` table (with the filename and line number), and the core request continues silently (fail-soft) to ensure the site remains accessible.
*   **Debug Mode:** When the CMS is in `DEV` mode, Tengo `print()` statements are piped directly to the Go standard output and the Admin UI's "Extension Log" htmx-stream for real-time debugging.

## Section 5: High-Performance Routing and Sub-50ms TTFB Optimization

To achieve the project’s mission-critical goal of a sub-50ms Time to First Byte (TTFB), VibeCMS implements a multi-layered routing and execution strategy. This architecture minimizes database round-trips, leverages Go’s concurrency model, and utilizes high-efficiency memory patterns to ensure that the journey from an incoming HTTP request to the final HTML byte stream is as direct as possible.

### 5.1 The High-Velocity Request Pipeline
VibeCMS utilizes an optimized middleware stack built on the Fiber/Echo framework (selection finalized at implementation). The pipeline is designed to avoid "middleware bloat," common in traditional CMS platforms.

1.  **Early-Exit Path Matching:** The router uses a Radix tree-based lookup. Static assets (`/assets/*`) and system endpoints (`/health`, `/stats`) are prioritized to exit the pipeline before any database or content logic is initialized.
2.  **Context Inflation:** A specialized `VibeContext` is pooled using `sync.Pool` to reduce Garbage Collection (GC) pressure. This context carries the PostgreSQL connection, the active theme configuration, and the identified `ContentNode` throughout the request lifecycle.
3.  **Soft-Fail License Check:** Verification of the Ed25519 cryptographic signature happens in a non-blocking background routine or is cached in-memory. If the license check fails, the router continues to serve content, only flagging the `Tengo` and `AI` modules as "Disabled" within the context.

### 5.2 In-Memory Routing Table (Node Cache)
Relying on a `SELECT * FROM content_nodes WHERE slug = ...` query for every request introduces unacceptable latency. VibeCMS implements a **Global Routing Map**:

*   **Structure:** A thread-safe `map[string]NodeMetadata` stored in-memory. 
*   **Metadata:** This map stores the `NodeID`, `Language`, `NodeType`, and `TemplateLayout` for every published slug.
*   **Synchronization:** The map is hydrated on boot. When a `ContentNode` is saved or published in the Admin UI, an internal event triggers a partial update to this map.
*   **Performance Impact:** This allows the router to determine if a page exists and which layout it requires in **O(1)** or **O(log n)** time without hitting PostgreSQL, reducing the initial "routing" phase to <1ms.

### 5.3 Advanced Query Optimization for JSONB
Once a route is matched, the system retrieves the `blocks_data` (JSONB). To stay under the 50ms threshold, the Data Access Layer (DAL) follows these constraints:

*   **Precomputed Joins:** Related media or "Relationship" fields are handled via GORM's `Preload` with optimized SQL joins. We avoid N+1 queries by fetching all block-related dependencies in a single, multi-statement transaction.
*   **JSONB Partial Fetching:** If a page contains a massive JSONB structure but only specific blocks are needed for a partial HTMX update, the system uses PostgreSQL's `jsonb_extract_path` to pull only the required fragments, reducing data transfer between the DB and the Go binary.

### 5.4 The "Vibe-Loop" Rendering Engine
The transition from JSON data to HTML is the most CPU-intensive part of the request. VibeCMS optimizes the Jet/Templ engine execution through:

| Optimization Technique | Description |
| :--- | :--- |
| **Template Pre-compilation** | All `.jet` and `.templ` files in the `/themes` directory are compiled into bytecode on startup. Disk I/O is zero during the request. |
| **Buffer Pooling** | Instead of allocating new strings for HTML output, VibeCMS uses a `byte.Buffer` pool. The rendered blocks are written directly to the response writer stream. |
| **Hierarchical Rendering** | The `blocks_data` array is processed linearly. For each entry, the engine maps `entry.type` to `themes/{active}/blocks/{type}.jet`. Global variables (Site Name, SEO Settings) are injected once into the base layout to avoid redundant lookups. |

### 5.5 Tengo Scripting Hook Performance
The "Zero-Rebuild" extension system via Tengo presents a potential bottleneck. To maintain sub-50ms TTFB while running `before_page_render` hooks:

*   **Script Pre-compilation:** `.tgo` scripts are compiled to Tengo bytecode and cached in a `map[string]*tengo.Compiled`.
*   **Execution Timeouts:** Every script execution is wrapped in a `context.WithTimeout`. If a script takes longer than 10ms, it is forcefully terminated, and the CMS proceeds with the standard render to protect the TTFB.
*   **Zero-Allocation Interop:** Data passed between Go and Tengo uses a custom-built wrapper that avoids deep-copying large JSON structures unless the script explicitly modifies them.

### 5.6 Real-time SEO and Sitemap Caching
Generating a `sitemap.xml` for 1,000+ nodes can normally take 200-500ms. VibeCMS eliminates this:

*   **In-Memory XML Buffer:** The `sitemap.xml` and `robots.txt` are maintained as pre-rendered byte slices in memory.
*   **Dirty-Bit Invalidation:** Whenever a node is added or a "NoIndex" toggle is flipped, a background goroutine re-generates the XML slice. 
*   **Result:** Requests to `/sitemap.xml` have a TTFB of **<5ms**, as they essentially serve a static byte array from RAM.

### 5.7 Summary of Latency Budget (Target: 50ms)
The system is built to adhere to the following strict internal latency budget per request:

1.  **Routing & Middleware:** 1-2ms (In-memory lookup).
2.  **Tengo `before_page_render` Hooks:** 0-10ms (Optional).
3.  **DB Fetch (Postgres JSONB):** 5-15ms (Indexed slug lookup).
4.  **Template Execution (Jet Engine):** 5-15ms (Pre-compiled templates).
5.  **IO / Network Write:** 2-5ms.
6.  **Total Target:** **13ms - 47ms**.

### 5.8 Edge Cases and Failure Modes
*   **Cold Starts:** On the first request after a deployment, several caches may be empty. VibeCMS uses an `init()` sequence to pre-warm the Node Map and Template Cache.
*   **Database Latency Spikes:** If PostgreSQL latency exceeds 40ms, the system is configured with a circuit breaker. It will attempt to serve a cached version of the page (if a local LRU cache is enabled) or return a 503 to prevent cascading failure in the agency's monitoring dashboard.
*   **Memory Pressure:** On low-RAM instances, the `sync.Pool` for buffers and contexts will naturally release memory back to the GC, slightly increasing TTFB in exchange for system stability.

## Section 6: Server-Rendered Admin UI (HTMX & Alpine.js Logic)

VibeCMS utilizes a "Low-JS" architectural pattern for its administrative interface, prioritizing rapid interaction latency (sub-100ms) and deep integration with the Go backend. Unlike heavy Single Page Applications (SPAs) that require substantial client-side parsing and state synchronization, the VibeCMS Admin UI leverages **HTMX** for fragmented DOM updates and **Alpine.js** for localized reactive states.

### 6.1 Architectural Philosophy: Fragments over Bundles
The Admin UI is composed of server-rendered **Jet** or **Templ** fragments. When a user interacts with the dashboard—such as saving a content node or reordering a block—HTMX intercepts the event, performs an AJAX request to the Fiber/Echo backend, and swaps only the affected HTML fragment.

*   **State Location:** Primary application state resides in the PostgreSQL database and the Go struct layer.
*   **UI Synchronization:** UI state is transient and managed by Alpine.js for immediate feedback (e.g., toggling a sidebar, local form validation).
*   **Performance Constraint:** By avoiding a heavy JS framework, the Admin UI remains functional even on low-powered devices and ensures that the "Agency Manager" can orchestrate hundreds of sites without browser memory bloat.

### 6.2 The Block-Based Form Generator
The core of the Admin UI is the dynamic form generator that maps `blocks_data` (JSONB) to editable interface elements.

#### 6.2.1 Schema-Driven Fragment Rendering
Each block type (e.g., `hero_section`, `cta_banner`) has a corresponding Jet template in the `/admin/fragments/blocks` directory. When the editor loads a Content Node:
1.  The backend iterates through the `blocks_data` JSON array.
2.  For each block, HTMX triggers a request for the specific edit-fragment: `GET /admin/api/blocks/render-form/{type}?id={index}`.
3.  The backend returns a HTML partial containing the `Vibe-Fields` (Text, Repeater, Relationship, Media) prepopulated with data.

#### 6.2.2 HTMX Reordering Logic
VibeCMS implements a "Move Up/Down" mechanism for block management to avoid the overhead of complex drag-and-drop libraries:
```html
<div class="block-wrapper" id="block-{{.Index}}">
    <!-- Block Header -->
    <div class="controls">
        <button hx-post="/admin/content/reorder" 
                hx-vals='{"direction": "up", "index": {{.Index}}, "node_id": {{.NodeID}}}'
                hx-target="#block-container" 
                hx-swap="innerHTML">
            ↑
        </button>
        <button hx-delete="/admin/content/blocks/{{.Index}}"
                hx-confirm="Are you sure?"
                hx-target="closest .block-wrapper" 
                hx-swap="outerHTML">
            Delete
        </button>
    </div>
    <!-- Field Inputs -->
    <input type="text" name="blocks[{{.Index}}][title]" value="{{.Title}}">
</div>
```

### 6.3 Alpine.js Client-Side Logic
Alpine.js is strictly reserved for "UI-only" state that does not require database persistence or complex backend calculation.

#### 6.3.1 Local State Management
*   **Modals & Slide-overs:** Managing the visibility of the Media Manager or AI Suggestion drawer.
*   **Repeater Fields:** Adding temporary rows in a `Repeater` field before a server-side "Save."
*   **Character Counters:** Real-time feedback for SEO Meta Titles and Descriptions.

#### 6.3.2 Example: Relationship Field Search
Relationship fields use Alpine.js to handle the search input and HTMX to fetch the results:
```html
<div x-data="{ open: false, search: '' }">
    <input type="text" 
           x-model="search"
           placeholder="Search nodes..."
           hx-get="/admin/api/search/nodes"
           hx-trigger="keyup changed delay:300ms"
           hx-target="#search-results"
           @focus="open = true">
    
    <div id="search-results" x-show="open" @click.away="open = false">
        <!-- HTMX injects <a> tags here -->
    </div>
</div>
```

### 6.4 Real-time Feedback & Validation
VibeCMS utilizes HTMX's `hx-post` with `hx-trigger="change"` for "save-on-change" or inline validation.

*   **Slug Generation:** As an editor types a Title, Alpine.js calculates a slug candidate. HTMX then hits `/admin/api/validate-slug` to check PostgreSQL for uniqueness, updating a status icon next to the field in real-time (<100ms).
*   **Auto-Save:** Optional per-node setting that triggers a background `PATCH` request to the `content_nodes` table using the `hx-trigger="every 30s"` directive, updating the `updated_at` timestamp without a page reload.

### 6.5 Media Manager Integration
The Media Manager is implemented as a global HTMX-powered modal.
1.  **Selection:** When a `Media` field is clicked, Alpine.js opens the modal.
2.  **Streaming Uploads:** HTMX manages the multipart form submission to the Go backend.
3.  **Optimization Feedback:** Upon upload, the backend converts the image to **WebP**. HTMX polls the server (`hx-get="/admin/media/status/{id}"`) until the optimization is complete, then swaps the "Processing" spinner for the image thumbnail.

### 6.6 Implementation Constraints & Edge Cases
*   **Concurrency:** To prevent "Lost Updates" during HTMX fragment swaps, the Admin UI implements a `version_hash` check. If the hidden version input in a form does not match the database version, HTMX returns a `409 Conflict` and Alpine.js triggers a "Refresh Required" notification.
*   **Error Handling:** All HTMX requests use the `hx-target-4xx` and `hx-target-5xx` extensions to route server errors to a global toast notification system (managed by Alpine.js).
*   **Authentication State:** If the Static Bearer Token or RBAC session expires, the backend sends a `HX-Redirect` header, forcing the browser to redirect to the login page immediately, preventing "silent" save failures.

### 6.7 UI Interaction Targets
| Action | Tech | Target Latency | Backend Hook |
| :--- | :--- | :--- | :--- |
| Block Reorder | HTMX | <50ms | `POST /admin/reorder` |
| Local Field Validation | Alpine | <10ms | N/A |
| Content Save | HTMX | <150ms | `PUT /admin/nodes/{id}` |
| Media Search | HTMX | <80ms | `GET /admin/media?q=...` |
| AI SEO Suggest | HTMX | 1s - 3s (Streamed) | `POST /admin/ai/suggest` |

## Section 7: AI-Native Integration Layer for Content and SEO Assistance

The AI-Native Integration Layer is a core performance and utility driver within VibeCMS. Rather than treating AI as a bolt-on text generator, VibeCMS treats the LLM as a structured data processor that understands the `JSONB` schema of the `content_nodes` table. This layer is provider-agnostic, supporting OpenAI, Anthropic, and custom internal agency endpoints via a unified bridge.

### 7.1 Provider-Agnostic Abstraction Bridge
The system utilizes a Go interface for LLM communications, ensuring that switching between GPT-4o, Claude 3.5 Sonnet, or a local Llama-3 instance requires only a configuration change, not a code rebuild.

**Technical Configuration:**
API credentials and model preferences are stored in the system configuration (stored in `site_settings` and encrypted at rest).
*   **Supported Drivers:** OpenAI (Chat Completions), Anthropic (Messages API), Generic OpenAI-Compatible (for Ollama or LocalAI).
*   **Fail-Safe Mechanism:** If the license check (Ed25519) fails or the API key is invalid, the UI gracefully disables AI buttons and logs a `Soft-Fail` warning to the Agency Monitoring API.

### 7.2 Structured Data Analysis (schema-first)
Unlike traditional CMSs that send a "blob" of HTML to an AI, VibeCMS sends the raw `blocks_data` JSON. This allows the AI to understand the semantic structure of the page (e.g., distinguishing a `Hero` block from a `Testimonial` block).

**The "Context Assembly" Workflow:**
1.  **Block Extraction:** The system flattens the `blocks_data` JSONB into a clean, text-heavy representation for the LLM.
2.  **Schema Injection:** The systemic prompt includes the JSON schema of each block type to ensure the AI understands the weight of specific fields (e.g., "The 'H1' field in the 'Hero' block is high-priority for SEO").
3.  **Token Optimization:** To maintain the "efficiency" ethos, the system trims redundant block metadata (IDs, styling flags) before transmission to reduce API costs and latency.

### 7.3 Real-time SEO Suggestion Engine
The SEO engine provides three layers of assistance, accessible directly within the Admin UI via HTMX-triggered overlays.

#### 7.3.1 Automated Metadata Generation
When the "Magic Wand" icon is clicked in the SEO sidebar, the CMS performs the following:
*   **Title/Description Synthesis:** Generates high-CTR Meta Titles (<60 chars) and Meta Descriptions (<160 chars) based on the aggregate content of the node.
*   **Keyword Extraction:** Identifies primary and secondary keywords for internal tracking and OpenGraph tag optimization.
*   **Auto-Slug Generation:** Recommends SEO-friendly URL slugs if the current one is flagged as non-performant (e.g., changing `/services/commercial-window-cleaning-v1` to `/window-cleaning-services`).

#### 7.3.2 Schema.org Markup Generation
The system generates valid JSON-LD scripts by analyzing the content blocks.
*   **FAQ Schema:** If the page contains a "Repeater" field of Question/Answer blocks, the AI automatically generates the `FAQPage` Schema.
*   **Article/Organization Schema:** Automatically maps node data to `Article`, `BreadcrumbList`, and `LocalBusiness` schemas.
*   **Validation:** All AI-generated schema is passed through a local Go-based JSON validator before being saved to the `seo_meta` JSONB column.

### 7.4 Content Intelligence & Block Drafting
The AI layer assists editors during the content creation phase using "Drafting Hooks."

*   **Block Expansion:** An editor provides a 2-sentence summary in a "Text" field; the AI expands it based on a pre-defined "Brand Voice" stored in the site settings.
*   **Content Summarization:** Generates an 'Excerpt' or 'Summary' block for listing pages (Archive nodes).
*   **Alt-Text Suggestion:** When a "Media" field is populated, the system can trigger an AI Vision call (if supported by the provider) to generate descriptive Alt-Text for the image, improving accessibility and image SEO.

### 7.5 Technical Implementation Details

#### Go-Level Schema Mapping
The CMS uses a dedicated `AIService` struct to handle the pipeline:

```go
type AIRequest struct {
    Action      string          `json:"action"` // "seo_suggest", "content_expand", "schema_gen"
    NodeContext ContentNode     `json:"node_context"`
    ActiveBlocks []BlockData    `json:"active_blocks"`
    UserPrompt  string          `json:"user_prompt,omitempty"`
}

type AISuggestionResponse struct {
    MetaTitle       string            `json:"meta_title"`
    MetaDescription string            `json:"meta_description"`
    Keywords        []string          `json:"keywords"`
    JsonLDSchema    map[string]interface{} `json:"json_ld_schema"`
    RecommendedSlug string            `json:"recommended_slug"`
}
```

#### HTMX Integration in Admin UI
The AI features are implemented as Alpine.js components that communicate with the Go backend using HTMX.

```html
<!-- Example Suggestion Trigger -->
<div x-data="{ loading: false }">
    <button 
        hx-post="/admin/api/ai/suggest-seo"
        hx-vals='js:{blocks: getBlockData()}'
        hx-target="#seo-fields-container"
        hx-swap="innerHTML"
        @click="loading = true"
        :class="loading ? 'opacity-50 cursor-not-allowed' : ''"
        class="vibe-btn-secondary"
    >
        <span x-show="!loading">✨ Suggest SEO</span>
        <span x-show="loading" class="animate-spin">🌀</span>
    </button>
</div>
```

### 7.6 External SEO API Integration (Ahrefs/Semrush)
For high-tier agency licenses, VibeCMS supports direct integration with external SEO tools.
*   **Keyword Difficulty Check:** Fetches real-time difficulty scores for targeted keywords within the node editor.
*   **Backlink Context:** Displays the number of internal/external links pointing to the current node (fetched via the Agency Monitoring API or direct Ahrefs integration).

### 7.7 Data Privacy & Prompt Governance
*   **Data Residency:** No content data is stored on VibeCMS servers; it is passed directly between the client instance and the AI provider.
*   **Prompt Customization:** Agencies can override the "System Prompt" via a Tengo script (`/scripts/ai_prompt_modifier.tgo`). This allows agencies to force specific SEO frameworks (e.g., "Always use the PAS - Problem, Agitation, Solution framework for descriptions").
*   **Rate Limiting:** To prevent runaway costs, the CMS implements a per-user hourly cap on AI requests, configurable in the environment variables.

## Section 8: Global SEO Suite: Real-time Sitemaps, Schema.org, and Robots.txt

VibeCMS is engineered to dominate search engine results pages (SERPs) by automating the technical overhead of SEO while providing granular surgical control for experts. This suite addresses the "Performance-Obsessed Agency" requirement by ensuring that all SEO assets are generated with sub-5ms overhead, leveraging in-memory caching and JSONB-native schema extraction.

### 8.1 Real-time XML Sitemaps (`sitemap.xml`)

Unlike traditional CMS platforms that rely on cron-based XML generation—which results in stale search indexes—VibeCMS utilizes a reactive, in-memory caching strategy for `sitemap.xml`.

*   **Architecture:**
    *   **In-Memory Router Cache:** The sitemap is served via a dedicated Fiber/Echo route. On the first request after a boot or a content change, the system performs a targeted `SELECT slug, updated_at, seo_settings FROM content_nodes WHERE status = 'published' AND noindex = false` query.
    *   **Invalidation Logic:** The cache is invalidated specifically when a `content_node` is saved, deleted, or changed from `draft` to `published`. This ensures the sitemap is always accurate to the second without redundant DB hits.
    *   **Multilingual Integration:** The sitemap automatically includes `<xhtml:link rel="alternate" ... />` tags for localized versions of nodes, mapping hreflang attributes based on the `Section 9: Native Multilingual Support` configuration.
*   **Index Fragments:** For sites exceeding 50,000 nodes, VibeCMS automatically shifts to a Sitemap Index structure (`sitemap_index.xml`), segmenting by content type (e.g., `sitemap_pages.xml`, `sitemap_properties.xml`).
*   **Performance Constraint:** The XML generation logic must complete in <10ms for 5,000 nodes to maintain the global **sub-50ms TTFB** target.

### 8.2 Dynamic Schema.org Generation

VibeCMS treats Schema.org markups as high-order entities derived directly from the `blocks_data` JSONB structure.

*   **Automated Extraction:**
    *   The system parses the `blocks_data` of a node to identify specific block types. For example, if a `Review` block or an `FAQ` block exists, the system automatically injects the corresponding `application/ld+json` snippet into the `<head>`.
    *   **Standard Schemas:** Every node, by default, outputs `WebPage` and `BreadcrumbList` schema.
    *   **Entity Mapping:** Custom Post Types (e.g., "Properties") can be mapped to specific Schema types (e.g., `https://schema.org/RealEstateListing`) via the UI-based configuration, ensuring the JSON-LD contains specific fields like `price`, `address`, and `numberOfRooms`.
*   **AI-Assisted Enrichment:**
    *   The **AI-Native Integration Layer (Section 7)** provides a "Generate Metadata" hook. When triggered, the AI analyzes the raw block content and suggests refined Schema fields (e.g., identifying the `author` and `datePublished` for an `Article` schema even if the user didn't explicitly set them).
*   **Manual Overrides:**
    *   Power users can utilize a "Header Injection" field per node to provide custom JSON-LD. This field is stored in the `seo_settings` JSONB column and takes precedence over auto-generated schema.

### 8.3 Robots.txt Manager

The `robots.txt` file is managed through a dedicated interface in the Server-Rendered Admin UI, backed by a persistent configuration entry in the database.

*   **Default Configuration:**
    ```text
    User-agent: *
    Allow: /
    Disallow: /admin/
    Disallow: /api/
    Sitemap: https://{domain}/sitemap.xml
    ```
*   **Live Injection:** The `Sitemap:` directive is dynamically populated with the current environment's primary domain to prevent issues in staging/dev environments.
*   **Tengo Scripting Hook:** A specific hook, `on_robots_request.tgo`, allows developers to programmatically alter the `robots.txt` output based on the user-agent or IP, useful for blocking specific aggressive crawlers in real-time without modifying the core binary.

### 8.4 Granular SEO Metadata Control

Each `content_node` includes a standardized `seo_settings` JSONB object, managed via an Alpine.js-powered sidebar in the editor.

| Field | Storage Key | Logic / Fallback |
| :--- | :--- | :--- |
| **Meta Title** | `seo_title` | Fallback to Node Title |
| **Meta Description** | `seo_desc` | Fallback to first 160 chars of "Text" blocks |
| **OpenGraph Image** | `og_image` | Fallback to "Featured Image" field |
| **Canonical URL** | `canonical` | Defaults to self-link; manually overridable |
| **Index Control** | `noindex` | Boolean; if true, adds `meta name="robots" content="noindex"` |
| **Follow Control** | `nofollow` | Boolean; affects all links in the node context |

### 8.5 External SEO API Integration (Ahrefs/Semrush)

VibeCMS provides native integration points for major SEO toolsets through the Agency Monitoring API.

*   **Real-time Insights:** While editing a node, the Admin UI can fetch live keyword difficulty and search volume from Ahrefs or Semrush (provided the agency adds an API key).
*   **Health Instrumentation:** The `/stats` endpoint (Section 12) includes an "SEO Health" key, reporting on:
    *   Missing meta descriptions across the site.
    *   Broken internal links (detected during the "Vibe Loop" render pipeline).
    *   Nodes with `noindex` enabled, preventing accidental "de-indexing" of critical pages after a migration.

### 8.6 Code Implementation Constraint (Go/Fiber)

The SEO suite must follow this performance-first routing pattern:

```go
// Example of the Real-time Sitemap Route
func (s *SEOService) HandleSitemap(c *fiber.Ctx) error {
    // 1. Check in-memory cache
    if s.cache.SitemapValid {
        c.Set("Content-Type", "application/xml")
        return c.Send(s.cache.SitemapData)
    }

    // 2. Build sitemap from DB (optimized JSONB selection)
    var nodes []ContentNode
    db.Select("slug, updated_at, seo_settings").Where("status = ? AND deleted_at IS NULL", "published").Find(&nodes)
    
    // 3. Generate XML byte buffer
    xmlData := s.generateXML(nodes)
    
    // 4. Update memory and respond
    s.cache.UpdateSitemap(xmlData)
    c.Set("Content-Type", "application/xml")
    return c.Send(xmlData)
}
```

This ensures that even if a site contains tens of thousands of nodes, the impact on the global **50ms TTFB** target is minimized by shifting the processing cost to the content-saving action rather than the search engine's request.

## Section 9: Native Multilingual Support and Content Localization Routing

VibeCMS treats internationalization (i18n) and localization (l10n) as first-class architectural concerns. Unlike traditional systems that rely on external plugins or duplicated table structures, VibeCMS implements a unified "Single-Node, Multi-Value" approach. This allows for sub-50ms TTFB even when resolving complex language hierarchies, while providing a seamless JSON-schema driven translation experience for AI and human editors.

### 9.1 Localization Data Model

At the core of the localized architecture is the `content_nodes` table. To maintain performance and relational integrity, VibeCMS avoids creating separate rows for every language. Instead, it utilizes a localized JSONB structure within the `blocks_data` and metadata columns.

#### 9.1.1 The Translation Schema
Each localized field within a block or node property follows a standardized keyed object structure where the key is the ISO 639-1 language code (e.g., `en`, `de`, `sk`).

```jsonc
// Example blocks_data structure for a 'Hero' block
{
  "block_id": "hero_01",
  "type": "hero",
  "data": {
    "title": {
      "en": "Welcome to VibeCMS",
      "sk": "Vitajte vo VibeCMS",
      "de": "Willkommen bei VibeCMS"
    },
    "image_id": "media_445", // Non-localized (shared across all languages)
    "cta_text": {
      "en": "Get Started",
      "sk": "Začať teraz"
    }
  }
}
```

#### 9.1.2 Field-Level Localization Controls
Developers define localization behavior within the block's JSON schema configuration:
*   **Translatable:** The field creates a keyed object for multiple languages.
*   **Synchronized:** The field maintains a single value used by all languages (common for IDs, hex colors, or media references).
*   **Inherited:** If a translation is missing, the system can fallback to a "Master" language defined in the site configuration.

### 9.2 Content Routing and URL Resolution

VibeCMS supports two primary routing strategies for multilingual content, configurable at the instance level. The routing engine is optimized in Go using a trie-based matcher to ensure lookups stay within the 50ms TTFB budget.

#### 9.2.1 Path-Based Routing (`/en/path`)
The default strategy where the language code serves as the first segment of the URL path.
*   **Root Handling:** A request to `/` automatically triggers a 302 redirect to the default language (e.g., `/en/`) based on browser `Accept-Language` headers or the configured system default.
*   **Slug Uniqueness:** Slugs are stored in a `node_routes` table that maps `(language_code, slug)` pairs to a `node_id`. This allows `/en/about-us` and `/sk/o-nas` to point to the same internal Content Node ID.

#### 9.2.2 Domain/Subdomain-Based Routing
For agencies requiring distinct regional branding (e.g., `vibecms.com` vs `vibecms.sk`).
*   **Mapping:** The configuration file maps specific HOST headers to language codes.
*   **Logic:**
    ```go
    // Internal Routing Logic Sketch
    lang := config.GetLanguageByHost(r.Host) // e.g., "vibecms.sk" -> "sk"
    node := DB.Where("slug = ? AND language = ?", path, lang).First(&node)
    ```

### 9.3 Middleware and Context Injection

The VibeCMS routing pipeline injects a `LocaleContext` into the request flow before the Jet/Templ rendering engine or Tengo scripts are executed.

#### 9.3.1 The LocaleContext Object
This object is available to all templates and scripts:
*   `CurrentLanguage`: ISO code of the resolved language.
*   `IsRTL`: Boolean indicating Right-to-Left (derived from language metadata).
*   `Translations`: A map of UI-level keys (e.g., "submit_form") fetched from the local i18n JSON files in the theme.
*   `AlternateLinks`: A list of URLs for the same node in other languages (used for `hreflang` tags).

#### 9.3.2 Static Translation Files
While page content lives in PostgreSQL, static UI strings (buttons, labels, footers) are stored in the theme directory under `/i18n/{lang}.json`. This ensures that "system" text is version-controlled with the theme code.

### 9.4 Admin UI: Side-by-Side Translation Interface

The Admin UI, powered by HTMX and Alpine.js, provides an "AI-Native" translation experience.

1.  **Split-View Editing:** Editors can open a "Comparison Mode" where the Master language is visible in a read-only pane while the target language is being edited.
2.  **State Management:** Alpine.js tracks which language is currently "Active" in the editor tab, dynamically updating the form field names (e.g., `blocks_data[0][title][sk]`).
3.  **Visual Indicators:** Fields that are out-of-sync with the Master language are flagged with an "Update Needed" badge, particularly useful after the Master content has been modified.

### 9.5 AI-Automated Localization Flow

VibeCMS leverages its AI-Native layer to automate the translation of structured block data.

*   **Contextual Translation:** Unlike generic translators, the VibeCMS AI layer sends the entire block JSON to the LLM. This provides the AI with context (e.g., it knows that "Home" is a navigation link, not a building).
*   **Schema Preservation:** The AI is instructed via system prompt to return only the translated values within the specific JSON structure, preventing breakage of the `blocks_data` JSONB blob.
*   **Batch Operations:** Agencies can trigger a "Translate Whole Page" action, which iterates through all blocks, detects missing translations, and populates them via the configured AI provider (OpenAI/Anthropic).

### 9.6 SEO and Hreflang Management

To comply with Section 8 (Global SEO Suite), the localization engine automatically manages technical SEO requirements:

1.  **Hreflang Injection:** The `before_page_render` hook automatically injects `<link rel="alternate" hreflang="..." href="..." />` tags into the `<head>` for every available translation of the current node.
2.  **Sitemap Integration:** The real-time sitemap generator includes `<xhtml:link>` entries for each localized URL, ensuring search engines index all regional versions correctly.
3.  **Canonical Logic:** By default, the current localized URL is treated as canonical. However, editors can manually override this to point to a "Master" language version to avoid duplicate content penalties if translations are only partial.

### 9.7 Edge Cases and Constraints

*   **Fallback Strategy:** If a node exists in `en` but not in `fr`, and a user requests the `fr` URL, VibeCMS can be configured to:
    *   Return a 404 (Default).
    *   Show the `en` content with a `fr` UI wrapper and a "Translation missing" banner.
    *   Transparently redirect to the `en` version.
*   **Performance Impact:** To maintain the <50ms TTFB target, the language-to-host and language-to-path lookups are cached in an LRU (Least Recently Used) in-memory cache. Database hits for route resolution are indexed using a composite B-Tree index on `(slug, language_code)`.
*   **Binary Deployment:** Since each site is an independent binary, language configurations (supported codes, currency formats, date-time locales) are defined in the instance `config.yaml` or `config.json` and loaded into memory on startup.

## Section 10: Media Manager Architecture with WebP/S3 Optimization

The VibeCMS Media Manager is a high-performance asset pipeline designed to solve the two primary bottlenecks of modern content delivery: slow image loading and infrastructure scaling. By prioritizing WebP-first delivery and a driver-based storage abstraction, the system ensures that media assets contribute to the sub-50ms TTFB goal rather than hindering it.

### 10.1 Storage Engine Abstraction (`StorageDriver` Interface)

To support the "independent binary per deployment" strategy while catering to diverse agency infrastructure, VibeCMS implements a unified `StorageDriver` interface. This allows the CMS to switch between local storage for small sites and S3-compatible cloud storage for high-traffic deployments without changing the core application logic.

#### Supported Providers:
*   **Local Disk:** Standard filesystem storage with defined directory structures.
*   **S3-Compatible:** Tested against AWS S3, DigitalOcean Spaces, and Cloudflare R2.
*   **Custom:** Integration via the Go plugin system (advanced) or manual driver registration in the source.

#### The Internal Interface Definition:
```go
type StorageDriver interface {
    Upload(ctx context.Context, file []byte, path string, contentType string) (string, error)
    Delete(ctx context.Context, path string) error
    GetURL(path string) string // Returns public URL or signed URL if private
    GetMetadata(path string) (FileMeta, error)
    Exists(path string) (bool, error)
}
```

### 10.2 The Image Optimization Pipeline (WebP Strategy)

Upon upload, VibeCMS does not merely store the raw file. It initiates a non-blocking asynchronous pipeline to generate optimized derivatives.

1.  **Original Preservation:** The original file (e.g., a 12MB PNG) is stored in a `hidden/originals` directory. This acts as the "Source of Truth" for future re-processing if global quality settings change.
2.  **WebP Encoding:** Using the `bimg` (libvips wrapper) or `imaging` library, the system generates a WebP version of every image.
    *   **Default Quality:** 80 (configurable via `config.yaml`).
    *   **Lossless Toggle:** Enabled automatically for icons/logos.
3.  **Responsive Breakpoints:** The system generates a standard set of widths to support the `srcset` attribute in Jet/Templ components:
    *   `thumbnail`: 150px (Square crop)
    *   `medium`: 800px (Proportional)
    *   `large`: 1920px (Proportional)
4.  **Metadata Extraction:** Features like EXIF stripping (for privacy) and dominant color extraction (for "blur-up" loading states) are performed during this phase.

### 10.3 Database Schema: `media_assets`

Media assets are tracked in PostgreSQL to allow for rapid querying, relationship mapping in `content_nodes`, and AI-assisted alt-text indexing.

| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key. |
| `filename` | String | Original filename for SEO purposes. |
| `slug` | String | URL-friendly unique identifier. |
| `provider` | String | Enum: `local`, `s3`, `r2`, etc. |
| `file_path` | String | Relative path in storage. |
| `webp_path` | String | Path to the optimized WebP derivative. |
| `mime_type` | String | Original MIME type. |
| `byte_size` | BigInt | File size in bytes. |
| `dimensions` | JSONB | `{ "width": 1024, "height": 768 }`. |
| `alt_text` | Text | SEO description (manually set or AI-suggested). |
| `metadata` | JSONB | Artifacts like dominant color, focal point (X/Y percentages). |
| `created_at` | Timestamp | Upload date. |

### 10.4 Admin UI Integration (HTMX + Alpine.js)

The Media Manager UI is built as a reusable Alpine.js component that interacts with the Go backend via HTMX fragments.

#### Functional Requirements:
*   **Instant Search:** Alpine.js filters the local view while HTMX performs server-side debounced searching against the `media_assets` table.
*   **Chunked Uploads:** Large files are uploaded in chunks to prevent timeout issues on restricted PHP/Nginx proxies.
*   **Focal Point Selector:** A visual UI allows users to click on an image to set the `focal_point` (stored in JSONB), ensuring that CSS `object-position` focuses on the subject during responsive cropping.
*   **Bulk Actions:** HTMX-powered bulk deletion and "Regenerate All WebP" triggers for agency-wide maintenance.

### 10.5 AI-Native Integration: "Vision" Optimization

VibeCMS leverages its AI-agnostic layer to enhance the media management workflow:
*   **Auto-Alt Text:** When an image is uploaded, the system can optionally send the thumbnail to a Vision-capable LLM (e.g., GPT-4o or Claude 3.5 Sonnet) to generate descriptive, SEO-optimized alt text.
*   **Smart Tagging:** Automated generation of categorization tags based on image content.
*   **Content-Aware Cropping:** Suggesting focal points based on human faces or high-contrast subjects detected by AI.

### 10.6 Delivery and Performance Constraints

To maintain the **sub-50ms TTFB** target, VibeCMS implements the following delivery logic:

1.  **Lazy Asset Resolution:** The `Media` field type in `content_nodes` stores only the UUID. The Jet template engine resolves these UUIDs into full asset objects via a cached lookup table, avoiding repeated N+1 queries.
2.  **CDN Integration:** If a cloud provider like Cloudflare R2 is used, the `GetURL` method automatically prefixes the asset path with the configured CDN CNAME.
3.  **Cache Headers:** The Go binary serves local assets with aggressive `Cache-Control: public, max-age=31536000, immutable` headers.
4.  **Security:** Assets are physically separated from the application root. Temporary upload paths are scrubbed every hour via the internal Cron system (Section 13).

### 10.7 Edge Case Handling

*   **SVG Support:** SVGs bypass the WebP pipeline. They are sanitized using an allow-list of tags and attributes to prevent XSS before storage.
*   **Animated Gifs:** Converted to WebP animation or kept as GIF based on the `allow_animated_webp` config flag (due to varying browser support for animated WebP).
*   **Missing Derivatives:** If a requested WebP derivative is missing due to a storage glitch, the system falls back to the original source in real-time, logs a "Repair Required" event in the Health API (Section 12), and schedules a regeneration task.

## Section 11: Communications Engine: SMTP/Resend Integration and Logging

The Communications Engine in VibeCMS is a decoupled internal service responsible for the reliable delivery, tracking, and auditing of all outbound electronic correspondence. To maintain the project's sub-50ms TTFB target, the engine operates on an asynchronous "fire-and-forget" model for the main request thread, utilizing internal Go channels and worker pools to interface with external mail providers.

### 11.1 Provider Architecture
VibeCMS supports two primary mail drivers as defined in the business logic constants. The system uses a driver-based interface to ensure that switching between providers does not require changes to the business logic or Tengo scripts.

*   **SMTP Driver:** A standard implementation supporting PLAIN, LOGIN, and CRAM-MD5 authentication over TLS/SSL. This is intended for legacy enterprise environments or specialized relay services.
*   **Resend.com Driver:** A high-performance REST-based implementation utilizing the Resend Go SDK. This is the recommended provider for agencies requiring high deliverability and native support for tags and modern API-based tracking.

#### Configuration Schema
Mail settings are stored in the system configuration table and mapped to the following Go struct:

```go
type MailConfig struct {
    Provider      string `json:"provider"` // "smtp" or "resend"
    FromEmail     string `json:"from_email"`
    FromName      string `json:"from_name"`
    ReplyTo       string `json:"reply_to"`
    SMTP struct {
        Host     string `json:"host"`
        Port     int    `json:"port"`
        Username string `json:"username"`
        Password string `json:"password"` // Encrypted at rest
        UseTLS   bool   `json:"use_tls"`
    } `json:"smtp"`
    Resend struct {
        APIKey string `json:"api_key"`
    } `json:"resend"`
}
```

### 11.2 The Mail Log: Auditing and Debugging
Every attempt to send an email—whether triggered by a built-in function (e.g., password reset) or a Tengo script hook (e.g., `on_form_submit`)—is recorded in the `mail_logs` PostgreSQL table. This table is critical for agency monitoring, allowing developers to diagnose delivery failures without accessing third-party dashboards.

**Table Schema: `mail_logs`**
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID (PK) | Unique identifier for the transaction. |
| `recipient` | VARCHAR(255) | Primary recipient email address. |
| `subject` | TEXT | Subject line of the email. |
| `provider` | VARCHAR(20) | "smtp" or "resend". |
| `status` | VARCHAR(20) | `queued`, `sent`, `failed`, or `bounced`. |
| `payload` | JSONB | Full JSON representation of the email body (optional, for debugging). |
| `response_code` | INT | HTTP status from Resend or SMTP response code. |
| `error_message` | TEXT | Detailed error stack track or provider error string. |
| `metadata` | JSONB | Tags, categories, or Tengo context IDs. |
| `created_at` | TIMESTAMPTZ | Creation timestamp. |
| `sent_at` | TIMESTAMPTZ | Actual timestamp of provider acceptance. |

### 11.3 Integration with Tengo Scripting
A key requirement of the "Zero-Rebuild Extension System" is the ability to send mail from sandboxed scripts. The Communications Engine exposes a `mail` module to the Tengo VM.

**Example Tengo Script Hook (`on_form_submit.tgo`):**
```tengo
fmt := import("fmt")
mail := import("mail")

// Logic to process a custom contact form
form_data := ctx.form_data()

res := mail.send({
    to: "sales@agency-client.com",
    subject: "New Lead: " + form_data.subject,
    body: fmt.sprintf("Name: %s\nMessage: %s", form_data.name, form_data.message),
    tags: {
        "source": "contact_form",
        "priority": "high"
    }
})

if is_error(res) {
    log.error("Failed to queue email: " + res.message)
}
```

### 11.4 Execution Pipeline and Worker Pool
To prevent external API latency (Resend) or socket timeouts (SMTP) from blocking the HTTP response, VibeCMS implements a buffered worker queue.

1.  **Request:** A function calls `CommunicationsEngine.Send()`.
2.  **Persistence:** The engine immediately writes a record to `mail_logs` with the status `queued`.
3.  **Dispatch:** The mail task is sent to an internal Go channel `chan MailTask`.
4.  **Worker Processing:** A pool of persistent background workers (default: 3) picks up the task.
5.  **Provider Execution:** The worker invokes the configured provider (SMTP/Resend).
6.  **Update:** Upon completion, the worker updates the `mail_logs` record with the `sent_at` timestamp and the provider's response metadata.

### 11.5 Resiliency and Edge Cases
*   **Rate Limiting:** If the Resend API returns a `429 Too Many Requests`, the worker implements an exponential backoff strategy (up to 3 retries) before marking the log as `failed`.
*   **Connection Timeouts:** SMTP connections are wrapped in a 10-second timeout context. If the connection fails, the engine attempts to re-establish the socket once before failing.
*   **Large Attachments:** By default, the engine limits total attachment size to 10MB to prevent PostgreSQL `JSONB` bloat and memory exhaustion.
*   **Environment Parity:** In `development` mode, the engine can be configured to "Log-Only," where emails are recorded in the database and printed to the console but never dispatched to the external provider.

### 11.6 Admin UI (HTMX & Alpine.js Logic)
The Admin panel includes a "Communications" tab providing:
*   **Real-time Log Viewer:** A searchable table (rendered via HTMX with periodic polling) showing recent outbound mail and their statuses.
*   **Provider Testing:** A "Send Test Email" utility that attempts to send a templated message using the current configuration, displaying raw SMTP/API logs in a modal for immediate troubleshooting.
*   **Log Retention Policy:** Settings to automatically purge `mail_logs` older than *N* days (handled by Section 13: Task Scheduling) to maintain database performance.

## Section 12: Agency Monitoring API and System Health Instrumentation

To facilitate the management of hundreds of independent VibeCMS deployments, each instance provides a standardized, high-performance monitoring layer. This system is designed to allow external agency dashboards to aggregate "single-pane-of-glass" views of infrastructure health, license status, and operational metrics without requiring manual login to individual admin panels.

### 12.1 Authentication and Security Model
Monitoring endpoints are strictly decoupled from standard session-based RBAC used in the Admin UI. Access is governed by the `auth_mechanism` defined in the global constants.

*   **Static Bearer Token:** Each VibeCMS instance generates a unique `MONITORING_API_KEY` during initial setup (stored in the `.env` or encrypted internal settings table).
*   **Transport Security:** External requests are only processed over TLS. The API ignores non-HTTPS traffic in production environments.
*   **Rate Limiting:** To prevent DDoS or brute-force discovery of the monitoring key, the Fiber/Echo middleware enforces a strict bucket-leak rate limit (default: 30 requests per minute per IP).
*   **Header Requirement:**
    `Authorization: Bearer <agency_static_token>`
    `X-Vibe-Instance-ID: <unique_uuid_generated_at_install>`

### 12.2 Endpoint Specification

#### 12.2.1 GET `/health` (Liveness & Readiness)
A lightweight endpoint used by load balancers and uptime monitors (e.g., UptimeRobot, BetterStack).
*   **Response Time Target:** <5ms.
*   **Logic:** Performs a "shallow" check of the Go process and a `ping` to the PostgreSQL instance.
*   **Success (200 OK):** `{"status": "up", "timestamp": "2023-10-27T10:00:00Z"}`
*   **Failure (503 Service Unavailable):** Returns a JSON object indicating which component failed (DB, Disk, or Memory pressure).

#### 12.2.2 GET `/stats` (Detailed Telemetry)
The primary data source for agency dashboards. This returns a comprehensive JSON payload covering four telemetry domains.

**A. System Resources:**
*   **CPU/RAM Usage:** Current utilization of the Go binary.
*   **Disk Usage:** Percentage of used space on the local `assets/` directory vs. S3 bucket remaining quota (if detectable).
*   **Uptime:** Total duration since the last binary restart.

**B. Database & Content Health:**
*   **Node Count:** Total entries in `content_nodes`.
*   **JSONB Bloatedness:** Estimated size of the `blocks_data` column.
*   **Migration Status:** Current schema version vs. binary expectation.

**C. Application Metrics:**
*   **Average TTFB:** A rolling average of the last 1000 requests (retrieved from an in-memory ring buffer).
*   **Mail Queue Status:** Number of pending vs. failed emails in the `mail_logs` from the last 24 hours.
*   **Task/Cron Health:** Timestamp of the last successful run of background tasks (e.g., S3 backups, Sitemap generation).

**D. License & Updates:**
*   **License State:** `Valid`, `Expired`, or `Invalid-Signature`.
*   **Version Info:** Current binary semantic version (e.g., `v1.2.4`) and the "Latest Available" version (cached from the VibeCMS update server every 24 hours).

### 12.3 System Health Instrumentation (Internal)

VibeCMS utilizes a "Proactive Instrumentation" strategy to ensure that agencies are notified of issues before a site goes offline.

#### 12.3.1 In-Memory Ring Buffers for Latency Tracking
To maintain the sub-50ms TTFB target without adding database overhead for every request, the backend implements a lock-free ring buffer (using `sync/atomic`) to track:
1.  **Request Duration:** Segregated by `Route Type` (Home, Page, Post, API).
2.  **Tengo Script Execution Time:** Duration of `.tgo` hook execution to identify poorly optimized user scripts.
3.  **DB Query Latency:** Specifically for GORM operations on the `content_nodes` table.

#### 12.3.2 Log Aggregation Requirements
System logs are structured as JSON for easy ingestion by external collectors (e.g., Vector, Logstash).
*   **Error Leveling:** Critical errors (DB Down, License Signature Mismatch) trigger an internal "Critical Flag" that remains active in the `/stats` endpoint until a manual clear or a successful recovery loop.
*   **Audit Trail:** The API logs identifying metadata for every access to the `/stats` endpoint to detect unauthorized monitoring attempts.

### 12.4 Agency Dashboard Integration Mapping
Expected JSON Schema for the Agency Monitoring API:

```json
{
  "instance_meta": {
    "site_name": "Client XYZ",
    "vibe_version": "1.4.2",
    "environment": "production",
    "license_status": "active"
  },
  "health_metrics": {
    "database_connected": true,
    "last_backup_at": "2023-10-26T02:00:00Z",
    "storage_provider": "Cloudflare R2",
    "storage_usage_percent": 42.5
  },
  "performance": {
    "p95_ttfb_ms": 38.2,
    "active_goroutines": 142,
    "memory_usage_mb": 64.8
  },
  "comms_health": {
    "mail_provider": "Resend.com",
    "failed_mails_24h": 0,
    "pending_tasks": 0
  },
  "security": {
    "failed_login_attempts_24h": 3,
    "tengo_sandbox_violations": 0
  }
}
```

### 12.5 Edge Case Handling & Constraints
1.  **Stalled Database Connections:** If PostgreSQL is unresponsive, the `/stats` endpoint must still function by serving cached data from memory, explicitly marking database metrics as `stale` or `unavailable`.
2.  **License Soft-Fail State:** When the Ed25519 signature fails, the `/stats` payload will include a `warning_code: "LICENSE_INVALID"`. This allows the Agency Manager to identify billing issues without the client site going down.
3.  **Maximum Site Limit:** As per system boundaries, the agency API is optimized to handle queries from a centralized dashboard managing up to 1000 independent VibeCMS instances.
4.  **Network Isolation:** If the instance is in a private network, the monitoring API supports an "Outbound Push" mode (configurable in `config.yaml`), where the CMS POSTS its health data to a central Agency URL on a Cron schedule, rather than waiting for a GET request.

## Section 13: Task Scheduling and Internal Cron Management System

### 13.1 Overview and Philosophy
VibeCMS requires a robust, high-precision internal scheduling system to handle background operations without relying on the host operating system's `crontab`. In alignment with the "single binary" deployment strategy, the Scheduling System is embedded directly within the Go runtime. It is designed to manage three distinct types of workloads:
1.  **System Tasks:** Hard-coded maintenance routines (e.g., license checks, log rotation).
2.  **User-Defined Tengo Tasks:** Extensible business logic defined in `.tgo` scripts.
3.  **Agency Operations:** High-overhead tasks such as automated S3 backups and image re-optimization.

To maintain the **sub-50ms TTFB** target, the scheduler operates on a dedicated priority-queue-based runner that ensures heavy background tasks do not contend with the primary request-handling goroutines for CPU cycles.

### 13.2 Architecture and Engine
The scheduler is built on a high-concurrency wrapper around a specialized cron-expression parser.

#### 13.2.1 The Runner Core
*   **Concurrency Control:** A global worker pool limits the number of simultaneous tasks (default: `runtime.NumCPU()`).
*   **Persistence:** Task definitions are stored in the PostgreSQL database in the `system_tasks` table. This ensures that even if the binary restarts, the schedule persists.
*   **Execution Strategy:** 
    *   **Native Go Tasks:** Executed via registered function pointers.
    *   **Tengo Script Tasks:** Executed within a sandboxed, stateless VM instance. 
    *   **Shell Hooks:** Minimal support for external binary execution (restricted to Admin-only configuration).

#### 13.2.2 Database Schema: `system_tasks`
```sql
CREATE TABLE system_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    task_type VARCHAR(50) NOT NULL, -- 'native', 'tengo', 'backup'
    cron_expression VARCHAR(100) NOT NULL, -- Standard Crontab syntax
    metadata JSONB DEFAULT '{}', -- Arguments, script paths, or S3 targets
    last_run_at TIMESTAMP WITH TIME ZONE,
    next_run_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    timeout_seconds INT DEFAULT 3600,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_logs (
    id BIGSERIAL PRIMARY KEY,
    task_id UUID REFERENCES system_tasks(id) ON DELETE CASCADE,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20), -- 'success', 'failed', 'timeout'
    output TEXT, -- Stdout or Error message
    execution_duration_ms INT
);
```

### 13.3 Tengo Task Integration (Zero-Rebuild Cron)
As part of the "Zero-Rebuild" philosophy, agencies can drop scripts into `/themes/{active}/scripts/cron/` to register new scheduled jobs without a server restart.

#### 13.3.1 Registration Workflow
1.  The CMS watches the `/scripts/cron/` directory using `fsnotify`.
2.  Any `.tgo` file containing a `@cron` header comment is automatically parsed and synced to the `system_tasks` table.
    *   *Example Header:* `// @cron: 0 0 * * *` (Run daily at midnight).
3.  On execution, the script is provided a restricted `Context` including:
    *   `db`: GORM-bound access to non-sensitive tables.
    *   `mail`: Access to the Section 11 Communications Engine.
    *   `media`: Access to the Media Manager for file cleanup.

#### 13.3.2 Example Tengo Task: `fetch_exchange_rates.tgo`
```tgo
// @cron: 0 */6 * * *
// description: Updates currency rates in the content_nodes every 6 hours

fmt := import("fmt")
http := import("http")
json := import("json")

// 1. Fetch data from external API
resp := http.get("https://api.exchangerate.host/latest?base=USD")
if resp.status_code == 200 {
    data := json.decode(resp.body)
    rates := data.rates
    
    // 2. Log logic via CMS internal logger
    log.info(fmt.sprintf("Updated EUR rate to: %v", rates.EUR))
    
    // 3. Update a "Settings" content node via internal DB bridge
    db.exec("UPDATE content_nodes SET blocks_data = jsonb_set(blocks_data, '{rates}', ?) WHERE slug = 'global-settings'", json.encode(rates))
}
```

### 13.4 Critical Maintenance Tasks
VibeCMS ships with pre-configured internal tasks that cannot be disabled through the standard UI by non-Admins:

| Task Name | Frequency | Description |
| :--- | :--- | :--- |
| `license_verify` | Daily (Randomized) | Performs Ed25519 signature verification against the local license file. |
| `sitemap_rebuild` | Hourly | Re-scans `content_nodes` to refresh the in-memory cached `sitemap.xml`. |
| `mail_log_cleanup` | Weekly | Rotates the `mail_logs` table, purging entries older than 90 days. |
| `media_derivative_sync` | Daily | Checks for missing WebP versions of original assets and regenerates them. |
| `agency_health_push` | Every 15 mins | Pushes health metrics (CPU, DB status, Disk) to the Agency Monitoring API. |

### 13.5 Automated Backup Strategy
The Scheduler manages the full lifecycle of system backups as defined in the global requirements.

*   **Database Archival:** Performs a `pg_dump` stream directly to the configured storage provider (S3/R2/Local).
*   **Asset Zipping:** Compresses the `/uploads` directory into encrypted ZIP archives.
*   **Retention Logic:** Implements a "Grandfather-Father-Son" (GFS) rotation policy:
    *   Keep last 7 daily backups.
    *   Keep last 4 weekly backups.
    *   Keep last 12 monthly backups.
*   **Failure Notifications:** If a backup fails, the system immediately triggers the `Communications Engine` to send an alert via SMTP/Resend to the `Agency-Manager` role.

### 13.6 Monitoring and Admin UI
The Admin Panel interface (built with HTMX) provides a real-time dashboard for cron management:

1.  **Task List:** Displays status (Idle, Running, Failed), last run time, and duration.
2.  **Manual Trigger:** A "Run Now" button that bypasses the cron schedule for immediate execution (standard for testing Tengo scripts).
3.  **Visual Log Viewer:** Tailwind-styled log output from the `task_logs` table, filtered by task ID, enabling rapid debugging of Tengo script errors.
4.  **Health Instrumentation:** Integration with `/stats` endpoint (Section 12) to report hung tasks (tasks exceeding their `timeout_seconds`).

### 13.7 Error Handling and Edge Cases
*   **Overlapping Executions:** By default, the system prevents a task from starting if a previous instance of the same task is still running (`Singleton` pattern).
*   **Panic Recovery:** Each task execution is wrapped in a Go `recover()` block. If a task panics, the error is captured, logged to `task_logs`, and the worker pool slot is released.
*   **Clock Skew/Drift:** The scheduler uses `time.Ticker` synchronized with the DB server time. In the event of a significant system clock jump, the scheduler recalculates all `next_run_at` values to avoid "storming."
*   **Soft-Fail License State:** If the license is invalid, the scheduler continues to run native system tasks but suspends all `task_type: 'tengo'` jobs.

## Section 14: Security Model: RBAC, License Verification, and API Security

VibeCMS adopts a multi-layered security posture designed to protect site integrity, administrative access, and intellectual property without compromising the sub-50ms TTFB target. The security model is divided into three primary pillars: internal Role-Based Access Control (RBAC) for the Admin UI, a cryptographic license enforcement layer, and a hardened API for agency monitoring.

### 14.1 Role-Based Access Control (RBAC)

VibeCMS implements a flat but strict RBAC system. Authorization is checked at the middleware level for all routes under the `/admin` and `/api` clusters. 

#### 14.1.1 Defined Roles and Permissions
The system recognizes three hard-coded roles defined in the `system_boundaries`. Permissions are additive:

| Role | Description | Key Permissions |
| :--- | :--- | :--- |
| **Admin** | Full system owner. | Manage Users, Alter System Config, Edit Disk/S3 Settings, Manual DB Migrations, Full Content Access. |
| **Editor** | Content-focused user. | Create/Edit/Delete Content Nodes, Media Manager access, SEO tool access, View Mail Logs. (Cannot access System Settings or Tengo Scripts). |
| **Agency-Manager** | Monitoring/Maintenance. | View Health Stats, View System Logs, Trigger Backups, Access License Info (Read-only access to content). |

#### 14.1.2 Session Management
*   **Mechanism:** Secure, HTTP-only, SameSite=Lax cookies.
*   **Storage:** Sessions are stored in PostgreSQL to allow for instant revocation across the agency portfolio.
*   **CSRF Protection:** Every state-changing request (POST/PATCH/DELETE) triggered via HTMX or standard forms must include a masked CSRF token validated by the Fiber/Echo middleware.

### 14.2 License Verification System

VibeCMS uses an offline-first, cryptographic verification model to ensure validity while preventing site downtime due to network failures or "calling home" latency.

#### 14.2.1 Ed25519 Cryptographic Signatures
Each license key is a Base64-encoded string containing a JSON payload signed with a VibeCMS private key. The CMS binary contains the corresponding **Ed25519 Public Key**.

*   **License Payload Structure:**
    ```json
    {
      "domain": "example.com",
      "issued_at": "2023-10-01T12:00:00Z",
      "expires_at": "2024-10-01T12:00:00Z",
      "tier": "Pro",
      "signature": "..."
    }
    ```
*   **Verification Workflow:**
    1.  On startup and every 24 hours (via Internal Cron), the CMS validates the local `license.key` file against the current system domain and the embedded Public Key.
    2.  If the signature is valid and the domain matches, the `SystemContext.Licensed` flag is set to `true`.

#### 14.2.2 Soft-Fail Enforcement Policy
To uphold the agency-first commitment to uptime, license failure **never** takes the website offline.
*   **Licensed State (Active):** All features enabled, including AI-assisted SEO and Tengo Scripting Hooks.
*   **Unlicensed/Expired State (Soft-fail):**
    *   **Public Site:** Maintains 100% rendering functionality (Jet/Templ engine remains active).
    *   **Tengo Engine:** All `.tgo` hooks are bypassed. Custom logic is disabled.
    *   **AI Integration:** The OpenAI/Anthropic proxy layers return a `402 Payment Required` equivalent internal error.
    *   **Admin UI:** A persistent banner appears in the Admin UI; "Update/License" prompts are shown in health logs.

### 14.3 API Security & Agency Monitoring

The Centralized Monitoring API (`/api/v1/monitor/...`) is the primary interface for agencies managing up to 1,000 sites. 

#### 14.3.1 Static Bearer Token Authentication
Security for the monitoring endpoints relies on a high-entropy **Static Bearer Token** generated during the initial site setup.
*   **Header:** `Authorization: Bearer <VIBE_AGENCY_TOKEN>`
*   **Storage:** The token hash is stored in the database. The raw token is only visible once upon generation or reset via the Admin UI.
*   **Rate Limiting:** The monitoring API implements a strict rate limiter (default: 60 requests per minute per IP) to prevent brute-force discovery of the health endpoints.

#### 14.3.2 Sensitive Data Masking
The Agency Monitoring API filters output to prevent credential leakage.
*   **Exposed:** Uptime, DB latency, Disk usage, WebP conversion queue status, pending VibeCMS core updates, and the last 10 mail log errors.
*   **Masked/Excluded:** SMTP passwords, S3 Secret Keys, AI API Keys, and User session hashes are never returned via the API.

### 14.4 Tengo Scripting Sandbox (Internal Security)

Since Tengo allows for "Zero-Rebuild" logic, the execution environment is strictly sandboxed to prevent a compromised script from escalating privileges to the OS level.

*   **Import Restrictions:** The Tengo VM instance is initialized *without* the `os`, `io/ioutil`, or `http` standard libraries. 
*   **Access Control:** Scripts only receive a scoped `Context` map containing the current request's JSON data and a limited `Mailer` tool. 
*   **Timeouts:** Every script execution is wrapped in a `context.WithTimeout` (default: 200ms). If a script enters an infinite loop or performs heavy computation, the VM is killed by the Go scheduler to protect the sub-50ms TTFB target.

### 14.5 Database and Migration Security

*   **Automated Migrations:** Database migrations (Section 15) check for a non-null `schema_version` before execution. Admin users must have the `Admin` role to trigger manual "Force Migration" actions.
*   **JSONB Injection Prevention:** While JSONB provides flexibility, VibeCMS uses GORM's parameterized queries for all `blocks_data` updates. Data entering the `blocks_data` column is validated against the block's JSON Schema before the `UPDATE` statement is issued.

## Section 15: DevOps, Backup Strategy, and Automated DB Migrations

This section defines the operational standards and automated workflows required to maintain VibeCMS instances. Given the "one deployment per website" architecture, these strategies are optimized for horizontal scalability across hundreds of independent Go binaries, prioritizing reliability, data integrity, and low-touch maintenance for agencies.

### 15.1 Automated Database Migrations
VibeCMS employs a "Schema-Version Tracking" strategy built into the application lifecycle. This ensures that as agencies update their binaries, the underlying PostgreSQL schema remains synchronized without manual intervention.

#### 15.1.1 Migration Lifecycle
1.  **Bootstrapping:** Upon binary execution, the system checks for the existence of a `system_schema_versions` table. If absent, it initializes the core tables (e.g., `content_nodes`, `users`).
2.  **Version Comparison:** The binary contains an embedded list of migration scripts (using `embed.FS`). It compares the latest version ID in the database against the hardcoded `TargetVersion` in the Go source.
3.  **Execution (Idempotent):** If `CurrentVersion < TargetVersion`, the system enters "Maintenance Mode" (returning a 503 for non-admin traffic) and executes pending migrations within a single global transaction.
4.  **Verification:** After execution, a final health check validates the JSONB constraints on the `blocks_data` column and key indices.
5.  **Failure State:** If a migration fails, the transaction is rolled back. The application logs a critical error to `stderr` and the Agency Monitoring API, then terminates. This prevents "partial upgrades" which could lead to data corruption in the block-based storage.

#### 15.1.2 Implementation Constraints (GORM + Migrate)
*   **No Auto-Migrate in Production:** While `db.AutoMigrate()` is used in development for rapid prototyping, production migrations must be explicitly defined SQL or Go scripts to handle complex JSONB transformations.
*   **JSONB Schema Evolution:** When a block type's schema changes, migrations must include a "Data Migration" step that iterates through `content_nodes.blocks_data` to ensure existing JSON objects conform to the new structure.

### 15.2 Backup Strategy: Three-Tier Redundancy
VibeCMS includes a native backup engine capable of zipping the entire site state (Database + Media Assets + Configuration) and offloading it to remote storage.

#### 15.2.1 Backup Components
*   **Database Manifest:** A compressed `pg_dump` of the PostgreSQL instance.
*   **Asset Manifest:** A recursive archive of the `/assets` directory, including WebP derivatives and original source files.
*   **System Manifest:** A snapshot of active `.tgo` extension scripts and the `config.yaml` file (excluding sensitive environment variables which should be injected via ENV).

#### 15.2.2 Automated Scheduling (Internal Cron)
Backups are governed by the Internal Cron Management System (Section 13). Agencies can configure retention policies via the Admin UI:
*   **Daily Snapshots:** Retained for 7 days.
*   **Weekly Archives:** Retained for 4 weeks.
*   **Monthly Long-term:** Retained for 3 months.

#### 15.2.3 Remote Offloading (S3-Compatible)
To ensure disaster recovery, VibeCMS supports the `Storage Driver` interface for backups.
*   **Supported Providers:** AWS S3, DigitalOcean Spaces, Cloudflare R2, and Local Disk (for tiered NAS setups).
*   **Encryption:** Backups are AES-256 encrypted using a `BACKUP_ENCRYPTION_KEY` defined in the environment variables before being transmitted over TLS.

### 15.3 Deployment & Update Policy
VibeCMS explicitly follows a "Notification-only" update policy to respect agency CI/CD pipelines.

#### 15.3.1 Update Workflow
1.  **Pulse Check:** The CMS periodically queries the VibeCMS Update Server (using the Ed25519-signed license key for authentication).
2.  **Notification:** If a newer version is available, an alert appears in the Admin UI and is reported via the `/stats` endpoint for the Agency Monitoring API.
3.  **Deployment:** The agency is responsible for replacing the binary. Recommended patterns include:
    *   **Docker:** Updating the image tag in a `docker-compose.yml` or K8s manifest.
    *   **Systemd:** Using a `fetch-and-restart` script that pulls the latest signed binary from the agency’s private repository.

#### 15.3.2 Zero-Downtime Considerations
While the Go binary is standalone, using a reverse proxy (Caddy/Nginx) is required for zero-downtime deployments. During a binary swap, the proxy should hold requests or serve a static "Updating" page until the new VibeCMS process responds to the `/health` check.

### 15.4 Agency Monitoring API & DevOps Instrumentation
For agencies managing 100+ instances, VibeCMS exposes a standardized monitoring layer.

#### 15.4.1 Monitoring Endpoints
*   **`GET /health`**: Returns `200 OK` only if the PostgreSQL connection is active, the S3 bucket is reachable, and the local disk has >10% free space.
*   **`GET /stats`**: Returns a JSON payload containing:
    ```json
    {
      "version": "1.4.2",
      "uptime_seconds": 86400,
      "db_size_bytes": 524288000,
      "last_backup_status": "Success",
      "last_backup_timestamp": "2023-10-27T10:00:00Z",
      "license_status": "Valid",
      "pending_migrations": 0,
       "resource_usage": {
          "memory_mb": 42,
          "cpu_percent": 0.5
       }
    }
    ```

#### 15.4.2 Security
All DevOps endpoints are protected by a `MONITORING_API_TOKEN` (Static Bearer Token). Access should ideally be restricted via IP whitelisting at the reverse proxy level.

### 15.5 Disaster Recovery (DR) Routine
In the event of total server failure, the recovery process is designed for speed:
1.  **Provision:** Spin up a new Go environment with the VibeCMS binary and a clean PostgreSQL instance.
2.  **Environment Setup:** Inject the original `LICENSE_KEY` and `BACKUP_ENCRYPTION_KEY`.
3.  **Restore Command:** Execute `./vibecms --restore --source=s3://bucket/backup-date.zip`.
4.  **Auto-Reconstitution:** The system downloads the archive, restores the SQL dump, unpacks assets, and validates the Tengo scripts. The site returns to live status without manual configuration.

