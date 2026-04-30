// Package mcp exposes Squilla as a Model Context Protocol server. AI clients
// authenticated with a bearer token can CRUD every CMS entity, render blocks
// and layouts, manage themes and extensions, and query the underlying CoreAPI.
package mcp

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/mark3labs/mcp-go/server"
	"gorm.io/gorm"

	"squilla/internal/cms"
	"squilla/internal/coreapi"
	"squilla/internal/rendering"
	"squilla/internal/uploads"
)

// Deps holds the dependencies MCP tools need. Assembled once at boot.
type Deps struct {
	DB               *gorm.DB
	CoreAPI          coreapi.CoreAPI
	TokenSvc         *TokenService
	ContentSvc       *cms.ContentService
	ExtensionLoader  *cms.ExtensionLoader
	ExtensionHandler *cms.ExtensionHandler
	ThemeLoader      *cms.ThemeLoader
	ThemeMgmtSvc     *cms.ThemeMgmtService
	TemplateRenderer *rendering.TemplateRenderer
	BlockTypeSvc     *cms.BlockTypeService
	LayoutSvc        *cms.LayoutService
	PublicHandler    *cms.PublicHandler
	PluginManager    *cms.PluginManager
	UploadStore      *uploads.Store
	// UploadBaseURL is the absolute base used when building upload_url values
	// returned from upload_init. Falls back to env SQUILLA_PUBLIC_URL when empty.
	UploadBaseURL string
}

// Server is the MCP adapter. One instance per process; mounted on Fiber at /mcp.
type Server struct {
	deps    Deps
	mcp     *server.MCPServer
	http    *server.StreamableHTTPServer
	limiter *perTokenLimiter
	auditor *auditor
	logger  *log.Logger
	// allowRawSQL gates core.data.exec behind an env flag in addition to scope=full.
	allowRawSQL bool
	// toolCatalog is a flat index of every registered tool, populated by
	// addTool. Surfaced by the core.guide meta-tool so AI clients can see
	// every available verb in one call without trawling schema metadata.
	toolCatalog []toolCatalogEntry
}

// toolCatalogEntry is the shape returned by Server.registeredTools.
type toolCatalogEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Class       string `json:"class"`
}

// New constructs the MCP server and registers every tool and resource.
func New(deps Deps) *Server {
	s := &Server{
		deps:        deps,
		limiter:     newPerTokenLimiter(envIntDefault("SQUILLA_MCP_RPM", 600), envIntDefault("SQUILLA_MCP_BURST", 60)),
		auditor:     newAuditor(deps.DB),
		logger:      log.New(os.Stderr, "[mcp] ", log.LstdFlags),
		allowRawSQL: strings.EqualFold(os.Getenv("SQUILLA_MCP_ALLOW_RAW_SQL"), "true"),
	}

	s.mcp = server.NewMCPServer(
		"squilla",
		"0.1.0",
		server.WithInstructions(instructionText),
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	// Streamable HTTP is the newer MCP transport — bidirectional HTTP without
	// long-lived SSE connections. Compatible with modern Claude clients.
	s.http = server.NewStreamableHTTPServer(
		s.mcp,
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			// Validate the bearer token directly from the HTTP request. The
			// Fiber→http adaptor does not forward Fiber's UserContext, so we
			// re-parse the header here — a cheap DB lookup since the token
			// is hashed and indexed.
			if raw := extractBearer(r.Header.Get("Authorization")); raw != "" {
				if tok, err := s.deps.TokenSvc.Validate(raw); err == nil {
					ctx = context.WithValue(ctx, ctxKeyToken, tok)
				}
			}
			ctx = withServer(ctx, s)
			return ctx
		}),
	)

	s.registerCoreTools()
	s.registerSystemTools()
	s.registerDeployTools()
	s.registerUploadTools()
	s.registerRenderTools()
	s.registerGuideTools()
	s.registerThemeChecklistTool()
	s.registerResources()

	return s
}

// Mount wires /mcp on the given Fiber app. Bearer auth runs first; every tool
// call passes through authMiddleware before reaching the MCP handler.
// SetExtensionHandler wires the ExtensionHandler after construction. Needed
// because in main.go the handler is created after the MCP server (it depends
// on PluginManager, which depends on infrastructure set up later).
func (s *Server) SetExtensionHandler(h *cms.ExtensionHandler) {
	s.deps.ExtensionHandler = h
}

// SetPluginManager wires the PluginManager after construction so MCP tools
// can proxy through to extension gRPC plugins (e.g. core.media.upload routes
// through the media-manager extension's /upload handler so MCP and admin UI
// share one upload code path with optimisation, validation, and WebP).
func (s *Server) SetPluginManager(pm *cms.PluginManager) {
	s.deps.PluginManager = pm
}

// SetUploadStore wires the presigned-upload store after construction. The
// upload route and store live alongside MCP but are constructed from main.go
// because they need the same DB handle and a background ctx.
func (s *Server) SetUploadStore(store *uploads.Store) {
	s.deps.UploadStore = store
}

func (s *Server) Mount(app *fiber.App) {
	h := adaptor.HTTPHandler(s.http)
	app.All("/mcp", s.authMiddleware(), h)
	app.All("/mcp/*", s.authMiddleware(), h)
}

// registerCoreTools is defined in tools_*.go files; each domain lives in its
// own file for navigability. The split is purely organisational.
func (s *Server) registerCoreTools() {
	s.registerNodeTools()
	s.registerNodeTypeTools()
	s.registerTaxonomyTools()
	s.registerMenuTools()
	s.registerSettingsTools()
	s.registerMediaTools()
	s.registerUserTools()
	s.registerDataTools()
	s.registerFilesTools()
	s.registerHTTPTools()
	s.registerEventTools()
	s.registerFilterTools()
	s.registerFieldTypeTools()
	s.registerEmailTools()
}

const instructionText = `Squilla — an AI-native CMS exposed via MCP.

# Naming
All tools are namespaced core.<domain>.<verb>. The verb reveals intent:
  get/list/query = read            create/update/delete = write
  activate/deactivate = lifecycle  render.* = preview without publishing

# When lost, start here
Call core.guide first. It returns a goal→tool decision tree and the current
CMS state (active theme, counts, recent nodes). That one call replaces ~10
discovery calls and primes you with what exists before you mutate anything.

# Golden-path recipes (follow these — do not reinvent)

1. PUBLISH A NEW PAGE WITH AN IMAGE
   core.media.upload       → { id, url, slug }
   core.node.create        → { node_type, title, featured_image: <media obj>,
                               blocks_data: [...], status: "published" }
   core.render.node_preview(id) → verify before telling the user "done"

2. ADD A CUSTOM CONTENT BLOCK TO A THEME
   core.block_types.list                          → confirm slug is free
   core.block_types.create { slug, field_schema,  → body is html/template
                             html_template,         source that reads .fields
                             test_data }            and .node
   core.render.block { block_type, fields }       → smoke-test
   core.node.update { id, blocks_data: [...] }    → wire into a page

3. SWITCH THE SITE'S LOOK
   core.theme.list        → find id
   core.theme.activate(id)  (NO restart required for themes)

3b. DEPLOY A THEME OR EXTENSION FROM AN OUT-OF-REPO PACKAGE
   (build the directory locally → zip → base64 → MCP)
   core.theme.deploy { body_base64, activate?:true }
   core.extension.deploy { body_base64, activate?:true }
   Both unpack into themes/<slug>/ or extensions/<slug>/ via an atomic
   directory swap, register the row, optionally hot-activate. 50 MB cap.
   gRPC plugin binaries must already be compiled for the host's OS/arch.

4. ADD/EDIT A CUSTOM NODE TYPE ("post type")
   core.nodetype.list / .get to inspect existing schemas first
   core.nodetype.create { slug, label, label_plural, url_prefixes,
                          field_schema }
   core.taxonomy.create if the type needs tagging/categories
   Then core.node.create with the new node_type slug.

5. INSPECT WHAT BROKE ON THE LIVE SITE
   core.theme.active           → which theme is serving pages
   core.layout.list            → what layouts resolved
   core.render.node_preview    → reproduce the failing page
   core.block_types.get        → inspect the template source

# Key data shapes (do NOT flatten these)
  image field  = { url, alt, width?, height? }           object, not string
  link field   = { label, url, target? }                 object, not string
  repeater     = [ { sub_field: ... }, ... ]             array of objects
  term field   = { slug, name, taxonomy? }               object, not string
  blocks_data  = [ { type: "<slug>", fields: {...} }, ... ]

# Lifecycle / side-effects flags in responses
  restart_required:true  → the call flipped a flag but a process restart is
                           needed before plugin binaries fully load.
                           (Themes never set this; some extensions do.)

# Pagination contract
Any tool with "list" or "query" in its name accepts {limit, offset}. Default
limit=25, max=200. The response includes {total} so you know when to stop.

# Preview vs publish
core.render.* never fires events, never increments view counts, never writes.
Always preview before calling update/create when a user is watching output.

# Raw SQL
core.data.exec is gated behind scope=full AND a server env flag. Prefer the
typed tools (core.node.query, core.data.query) which enforce tenancy/ACL.`
