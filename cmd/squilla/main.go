package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"squilla/internal/api"
	"squilla/internal/auth"
	"squilla/internal/cms"
	"squilla/internal/config"
	"squilla/internal/coreapi"
	"squilla/internal/db"
	"squilla/internal/email"
	"squilla/internal/events"
	"squilla/internal/logging"
	"squilla/internal/mcp"
	"squilla/internal/models"
	"squilla/internal/rbac"
	"squilla/internal/rendering"
	"squilla/internal/scripting"
	"squilla/internal/sdui"
	"squilla/internal/secrets"
	pb "squilla/pkg/plugin/coreapipb"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"google.golang.org/grpc"
)

func main() {
	// CLI subcommands that exit before normal startup.
	if handlePreConfigCLI() {
		return
	}

	cfg := config.Load()
	logging.Init(strings.EqualFold(cfg.AppEnv, "development"))
	logging.Default().Info("squilla_starting", "env", cfg.AppEnv, "port", cfg.Port)
	log.Printf("Squilla starting | env=%s port=%s", cfg.AppEnv, cfg.Port)

	// In production, refuse to boot with development defaults — empty
	// SESSION_SECRET, default DB password, disabled TLS, etc. Each is a
	// real foot-gun for a deploy that just shipped without configuring envs.
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}

	// At-rest encryption service. NewFromEnv reads SQUILLA_SECRET_KEY;
	// in dev it returns an inactive service that passes plaintext
	// through, so deployments without the env var keep working.
	// Production has already been gated by cfg.Validate above.
	secretsSvc, err := secrets.NewFromEnv()
	if err != nil {
		log.Fatalf("secrets init failed: %v", err)
	}
	if secretsSvc.IsActive() {
		log.Println("secrets: SQUILLA_SECRET_KEY loaded; at-rest encryption enabled")
	} else {
		log.Println("secrets: SQUILLA_SECRET_KEY unset; secret settings stored in plaintext (dev only)")
	}

	database, err := db.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	log.Println("database connected successfully")

	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}
	log.Println("database migrations applied")

	if handlePostMigrationCLI(database) {
		return
	}

	// Auto-seed on first boot when the DB has no users yet. Idempotent —
	// once seeded, subsequent boots skip this. This is what makes
	// zero-config deploys (e.g. Coolify) usable out of the box.
	if err := db.SeedIfEmpty(database); err != nil {
		log.Fatalf("first-boot seed failed: %v", err)
	}

	// Event bus.
	eventBus := events.New()

	// SDUI — Server-Driven UI engine and SSE broadcaster.
	sduiEngine := sdui.NewEngine(database, eventBus)
	sduiBroadcaster := sdui.NewBroadcaster(eventBus)

	// Fiber app.
	app := fiber.New(fiber.Config{
		AppName:               "Squilla",
		DisableStartupMessage: false,
		BodyLimit:             50 * 1024 * 1024, // 50 MB — theme/extension uploads
		ReadBufferSize:        16 * 1024,        // 16 KB — handles large cookies without 431
	})

	// Request-ID middleware MUST come first so every downstream
	// handler/middleware sees a stable correlation ID (logs, errors,
	// access entries, panic recovery). The fiberlogger access entry
	// stays as a backup; logging.RequestID emits its own structured
	// access log too.
	app.Use(logging.RequestID())
	app.Use(fiberlogger.New())
	app.Use(recover.New())
	// Strict admin/public CORS: cookie-bearing requests (admin SPA, SSE)
	// must originate from the configured allowlist. /mcp is intentionally
	// excluded — it's bearer-token gated and meant to be reachable by any
	// AI client (Claude Code, Cursor, custom integrations) regardless of
	// origin. The MCP-specific permissive CORS is registered just below.
	app.Use(cors.New(cors.Config{
		Next: func(c *fiber.Ctx) bool {
			return strings.HasPrefix(c.Path(), "/mcp")
		},
		AllowOrigins:     corsOrigins(cfg.AppEnv),
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))
	// Permissive CORS scoped to /mcp. The Authorization bearer token is
	// the actual access control — anyone holding a valid token is meant
	// to reach this endpoint from anywhere, including browser-based MCP
	// inspectors. AllowCredentials must stay false because the wildcard
	// origin and credentials are mutually exclusive per the CORS spec.
	app.Use("/mcp", cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,OPTIONS",
		AllowHeaders:     "*",
		ExposeHeaders:    "Mcp-Session-Id,Mcp-Protocol-Version",
		AllowCredentials: false,
	}))

	// Services.
	// Background lifetimes (session cleanup, retention crons, etc.) hang
	// off this context — cancelled on graceful shutdown.
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()

	sessionSvc := auth.NewSessionService(database, cfg.SessionExpiryHours)
	// Sweep expired sessions hourly. Without this the sessions table grows
	// linearly and stale token_hashes accumulate forever.
	sessionSvc.StartCleanupLoop(bgCtx, time.Hour)

	// Same retention treatment for password reset tokens — they're already
	// short-lived (1h expiry) but get marked used rather than deleted on
	// successful reset, so without a sweep they pile up over time.
	auth.NewPasswordResetService(database).StartCleanupLoop(bgCtx, time.Hour)
	contentSvc := cms.NewContentService(database, eventBus)
	nodeTypeSvc := cms.NewNodeTypeService(database, eventBus)
	langSvc := cms.NewLanguageService(database)
	themeAssets := cms.NewThemeAssetRegistry()
	if err := themeAssets.LoadBlockAssetsFromDB(database); err != nil {
		log.Printf("WARN: failed to seed block asset registry from DB: %v", err)
	}
	blockTypeSvc := cms.NewBlockTypeService(database, eventBus, themeAssets)
	templateSvc := cms.NewTemplateService(database, themeAssets)
	layoutSvc := cms.NewLayoutService(database, eventBus, themeAssets)
	layoutBlockSvc := cms.NewLayoutBlockService(database, eventBus, themeAssets)
	menuSvc := cms.NewMenuService(database, eventBus)
	isDev := cfg.AppEnv == "development"
	renderer := rendering.NewTemplateRenderer("ui/templates", isDev)

	// Allow operators (or the media-manager extension on activation)
	// to override the URL convention used by image_url / image_srcset
	// template helpers via a site setting. Empty disables the
	// transform — see internal/rendering/media_funcs.go for rationale.
	var imgPrefixSetting models.SiteSetting
	if err := database.Where("\"key\" = ?", "image_cache_url_prefix").Limit(1).Find(&imgPrefixSetting).Error; err == nil && imgPrefixSetting.Value != nil && *imgPrefixSetting.Value != "" {
		renderer.SetImageURLPrefix(*imgPrefixSetting.Value)
	}

	// Asset URI resolver — turns "theme-asset:<key>" /
	// "extension-asset:<slug>:<key>" into real URLs at template-render time
	// so {{ image_url .photo "" }} doesn't get sanitised to "#ZgotmplZ" when
	// a theme passes through a raw asset URI.
	renderer.SetAssetResolver(func(uri string) string {
		lookup := cms.LoadActiveAssetLookupExported(database)
		if row, ok := cms.ResolveAssetURI(uri, lookup); ok {
			return row
		}
		return ""
	})

	// Email services.
	emailRuleSvc := email.NewRuleService(database)
	emailLogSvc := email.NewLogService(database)
	emailDispatcher := email.NewDispatcher(database, emailRuleSvc, emailLogSvc)
	eventBus.SubscribeAll(emailDispatcher.HandleEvent)
	// Daily sweep of email_logs older than email_log_retention_days
	// (default 30). Stops the table from growing unbounded.
	emailLogSvc.StartCleanupLoop(bgCtx, 24*time.Hour)

	// Daily sweep of content_node_revisions, keeping the most recent N
	// per node (default 50). Without this, chatty editors fill the table
	// indefinitely.
	contentSvc.StartRevisionCleanupLoop(bgCtx, 24*time.Hour)

	// Handlers.
	authHandler := auth.NewAuthHandler(database, sessionSvc)
	userHandler := auth.NewUserHandler(database, eventBus)
	nodeHandler := cms.NewNodeHandler(contentSvc, database, eventBus)
	nodeTypeHandler := cms.NewNodeTypeHandler(nodeTypeSvc)
	langHandler := cms.NewLanguageHandler(langSvc)
	blockTypeHandler := cms.NewBlockTypeHandler(blockTypeSvc, database)
	blockTypeHandler.SetThemeAssets(themeAssets)
	templateHandler := cms.NewTemplateHandler(templateSvc, database)
	layoutHandler := cms.NewLayoutHandler(layoutSvc, layoutBlockSvc)
	layoutHandler.SetDB(database)
	layoutBlockHandler := cms.NewLayoutBlockHandler(layoutBlockSvc)
	menuHandler := cms.NewMenuHandler(menuSvc)
	taxonomyHandler := cms.NewTaxonomyHandler(database, eventBus)
	termHandler := cms.NewTermHandler(database)
	healthHandler := api.NewHealthHandler(database)
	roleHandler := rbac.NewRoleHandler(database, eventBus)
	settingsHandler := cms.NewSettingsHandler(database, eventBus, secretsSvc)
	pageAuthHandler := auth.NewPageAuthHandler(database, sessionSvc, eventBus)

	// SDUI handlers — boot manifest, layout trees, and SSE events.
	bootHandler := api.NewBootHandler(database, sduiEngine)

	// Theme loader construction (NOT loading yet — must wait for extensions).
	// Boot order is core → extensions → themes: we only construct the loader
	// here so routes and services below can reference it. LoadTheme is called
	// after the gRPC plugin manager has started and all extensions have
	// subscribed to lifecycle events.
	themeLoader := cms.NewThemeLoader(database, themeAssets, eventBus)
	themePath := os.Getenv("THEME_PATH")
	if themePath == "" {
		// Try to load from active theme in DB.
		var activeTheme struct{ Path string }
		if err := database.Raw("SELECT path FROM themes WHERE is_active = true LIMIT 1").Scan(&activeTheme).Error; err == nil && activeTheme.Path != "" {
			themePath = activeTheme.Path
		} else {
			themePath = "themes/default"
		}
	}

	// Theme management. The mgmt service encrypts user-supplied git
	// tokens before persisting and decrypts before passing to git
	// operations; the handler does the same for direct token edits via
	// the admin API and for the webhook secret read at request time.
	// Bundled themes ship in the image at "themes/"; user-installed ones
	// land in "data/themes/" which docker-compose mounts as a persistent
	// volume. Both are scanned at boot; data wins on slug collision.
	themeMgmtSvc := cms.NewThemeMgmtService(database, themeLoader, "themes", "data/themes", secretsSvc)
	themeMgmtSvc.ScanAndRegister()
	themeHandler := cms.NewThemeHandler(database, themeMgmtSvc, secretsSvc)

	// CoreAPI — unified API facade for extensions.
	// `coreAPI` is unguarded — used internally and by MCP (which enforces
	// access via scope×class on token classes, not capability strings).
	// `guardedAPI` wraps coreAPI with the capability guard and is what
	// extension plugins (gRPC) and theme/extension scripts (Tengo) see.
	// The guard reads CallerInfo from context and denies any non-internal
	// caller that lacks the required capability declared in extension.json.
	// MediaService backs core.media.* MCP tools and the CoreAPI media methods
	// for callers that go through CoreAPI directly (Tengo via the guarded
	// adapter, internal seed paths, MCP). The media-manager extension owns the
	// admin UI / public optimization pipeline via its own gRPC HTTP routes;
	// this service is the bare-bones path that doesn't depend on the extension
	// being active. Both write to the same `media_files` table and the same
	// `storage/media` directory served by `app.Static("/media", ...)` below.
	mediaSvc := cms.NewMediaService(database, "storage/media")
	coreAPI := coreapi.NewCoreImpl(database, eventBus, contentSvc, menuSvc, mediaSvc, nodeTypeSvc, emailDispatcher, app, secretsSvc)
	guardedAPI := coreapi.NewCapabilityGuard(coreAPI)
	themeSettingsHandler := cms.NewThemeSettingsHandler(themeLoader.SettingsRegistry, coreAPI, database, secretsSvc, eventBus)

	// Theme scripting engine (theme .tgo scripts are loaded later, after
	// extensions have subscribed and after the theme is activated).
	scriptEngine := scripting.NewScriptEngine(eventBus, guardedAPI, themeLoader.SettingsRegistry)
	// Wire script engine into theme management so runtime activation loads Tengo scripts.
	themeMgmtSvc.SetScriptLoader(scriptEngine.LoadThemeScripts, scriptEngine.UnloadThemeScripts)

	// Extension loading.
	// Same dual-dir model as themes: image-bundled in "extensions/",
	// operator-installed in "data/extensions/" (persistent volume).
	extLoader := cms.NewExtensionLoader(database, "extensions", "data/extensions")
	extLoader.ScanAndRegister()
	extLoader.LoadBlocksForActiveExtensions(themeAssets)

	// Drop-in watchers — eliminate the "drop a folder + restart" step. We
	// only watch the data dirs because image-bundled dirs don't change at
	// runtime (they're baked into the image). New theme/extension folders
	// dropped into the persistent volume trigger an immediate rescan.
	if err := cms.NewDropInWatcher("data/themes", "theme.json", themeMgmtSvc.ScanAndRegister).Start(bgCtx); err != nil {
		log.Printf("WARN: themes drop-in watcher: %v", err)
	}
	if err := cms.NewDropInWatcher("data/extensions", "extension.json", extLoader.ScanAndRegister).Start(bgCtx); err != nil {
		log.Printf("WARN: extensions drop-in watcher: %v", err)
	}
	activeExts, _ := extLoader.GetActive()
	for _, ext := range activeExts {
		// Run pending SQL migrations for this extension.
		if err := cms.RunExtensionMigrations(database, ext.Path, ext.Slug); err != nil {
			log.Printf("WARN: extension %s migrations failed: %v", ext.Slug, err)
		}

		var manifest cms.ExtensionManifest
		_ = json.Unmarshal(ext.Manifest, &manifest)
		caps := manifest.CapabilityMap()
		if err := scriptEngine.LoadExtensionScripts(ext.Path, ext.Slug, caps); err != nil {
			log.Printf("WARN: extension %s script loading failed: %v", ext.Slug, err)
		}
	}

	renderer.SetEventRunner(scriptEngine.RunEvent)
	renderer.SetFilterRunner(scriptEngine.ApplyFilter)

	renderCtx := cms.NewRenderContext(database, layoutSvc, layoutBlockSvc, menuSvc, themeAssets)
	publicHandler := cms.NewPublicHandler(database, renderer, sessionSvc, layoutSvc, layoutBlockSvc, menuSvc, renderCtx, eventBus, themeLoader.SettingsRegistry, coreAPI)

	// --- Public HTML pages ---
	pageAuthHandler.RegisterRoutes(app)

	// --- API Auth routes ---
	authHandler.RegisterRoutes(app)

	// Health check.
	app.Get("/api/v1/health", healthHandler.HealthCheck)
	app.Get("/api/v1/stats", api.BearerTokenRequired(cfg.MonitorBearerToken), healthHandler.Stats)

	// --- Public API routes (no auth required) ---
	publicAPI := app.Group("/api/v1")
	nodeHandler.RegisterPublicRoutes(publicAPI)

	// --- Admin API routes (session auth required) ---
	// AuthRequired runs first so the request has a user context; the JSON-only
	// CSRF guard runs second on every mutation method (POST/PUT/PATCH/DELETE)
	// to stop cross-origin form submissions even if a stale session leaks.
	adminAPI := app.Group("/admin/api", auth.AuthRequired(sessionSvc), auth.JSONOnlyMutations())
	userHandler.RegisterRoutes(adminAPI)
	nodeHandler.RegisterRoutes(adminAPI)
	nodeTypeHandler.RegisterRoutes(adminAPI)
	langHandler.RegisterRoutes(adminAPI)
	blockTypeHandler.RegisterRoutes(adminAPI)
	templateHandler.RegisterRoutes(adminAPI)
	layoutHandler.RegisterRoutes(adminAPI)
	layoutBlockHandler.RegisterRoutes(adminAPI)
	menuHandler.RegisterRoutes(adminAPI)
	roleHandler.RegisterRoutes(adminAPI)
	taxonomyHandler.RegisterRoutes(adminAPI)
	termHandler.RegisterRoutes(adminAPI)
	settingsHandler.RegisterRoutes(adminAPI)
	cacheHandler := cms.NewCacheHandler(publicHandler, eventBus)
	cacheHandler.RegisterRoutes(adminAPI)
	themeHandler.RegisterRoutes(adminAPI)
	themeSettingsHandler.RegisterRoutes(adminAPI)
	cms.NewFieldTypeHandler().RegisterRoutes(adminAPI)
	publicHandler.RegisterAdminPreviewRoutes(adminAPI)
	cms.NewRevisionHandler(database, contentSvc).RegisterRoutes(adminAPI)

	// SDUI endpoints — boot manifest, layout trees, and SSE event stream.
	bootHandler.RegisterRoutes(adminAPI)
	adminAPI.Get("/events", sduiBroadcaster.Handler())

	// MCP — token admin CRUD (session-authed) + /mcp public endpoint (bearer-authed).
	mcpTokenSvc := mcp.NewTokenService(database)
	mcp.NewTokenHandler(mcpTokenSvc).RegisterRoutes(adminAPI)
	mcpServer := mcp.New(mcp.Deps{
		DB:               database,
		CoreAPI:          coreAPI,
		TokenSvc:         mcpTokenSvc,
		ContentSvc:       contentSvc,
		ExtensionLoader:  extLoader,
		ThemeLoader:      themeLoader,
		ThemeMgmtSvc:     themeMgmtSvc,
		TemplateRenderer: renderer,
		BlockTypeSvc:     blockTypeSvc,
		LayoutSvc:        layoutSvc,
		PublicHandler:    publicHandler,
	})
	mcpServer.Mount(app)
	// Daily sweep of mcp_audit_log older than mcp_audit_retention_days
	// (default 90). Each MCP tool call writes a row, so this can grow fast.
	mcpServer.StartAuditCleanupLoop(bgCtx, 24*time.Hour)

	// Plugin manager for gRPC extension plugins.
	// Plugins receive the capability-guarded CoreAPI — every method call from
	// the plugin is checked against the capabilities declared in its
	// extension.json manifest.
	hostRegistrar := cms.HostServerRegistrar(func(slug string, capabilities map[string]bool, ownedTables map[string]bool) func(s *grpc.Server) {
		caller := coreapi.CallerInfo{
			Slug:         slug,
			Type:         "grpc",
			Capabilities: capabilities,
			OwnedTables:  ownedTables,
		}
		hostServer := coreapi.NewGRPCHostServer(guardedAPI, caller)
		return func(s *grpc.Server) {
			pb.RegisterSquillaHostServer(s, hostServer)
		}
	})
	pluginManager := cms.NewPluginManager(eventBus, hostRegistrar, database)
	defer pluginManager.StopAll()

	// Start plugins for already-active extensions.
	for _, ext := range activeExts {
		var manifest cms.ExtensionManifest
		_ = json.Unmarshal(ext.Manifest, &manifest)
		caps := manifest.CapabilityMap()
		owned := manifest.OwnedTablesMap()
		if err := pluginManager.StartPlugins(ext.Path, ext.Slug, json.RawMessage(ext.Manifest), caps, owned); err != nil {
			log.Printf("WARN: extension %s plugin start failed: %v", ext.Slug, err)
		}
		// Announce the extension is active so other extensions can react —
		// e.g. media-manager importing any assets shipped with it.
		cms.PublishExtensionActivated(eventBus, ext.Slug, ext.Path, json.RawMessage(ext.Manifest))
	}

	// ──────────────────────────────────────────────────────────────
	// Theme activation — runs AFTER extensions are subscribed so that
	// lifecycle events (theme.activated / theme.deactivated) can be
	// handled by extensions (e.g. media-manager importing theme assets).
	// ──────────────────────────────────────────────────────────────
	themeLoader.LoadTheme(themePath)
	if err := scriptEngine.LoadThemeScripts(themePath); err != nil {
		log.Printf("WARN: theme script loading failed: %v", err)
	}
	if err := themeLoader.PurgeInactiveThemes(); err != nil {
		log.Printf("WARN: purge inactive themes: %v", err)
	}

	// Wire email dispatcher's send function to call the provider plugin directly.
	// This bypasses the event bus for synchronous error propagation.
	emailDispatcher.SetSendFunc(func(req email.SendRequest) error {
		providerSlug := req.Settings["provider"]
		if providerSlug == "" {
			return fmt.Errorf("no email provider configured")
		}
		client := pluginManager.GetClient(providerSlug)
		if client == nil {
			return fmt.Errorf("email provider %s is not running", providerSlug)
		}
		payload, err := json.Marshal(map[string]interface{}{
			"to":       req.To,
			"subject":  req.Subject,
			"html":     req.HTML,
			"settings": req.Settings,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal email payload: %w", err)
		}
		resp, err := client.HandleEvent("email.send", payload)
		if err != nil {
			return fmt.Errorf("provider plugin error: %w", err)
		}
		if resp.Error != "" {
			return fmt.Errorf("%s", resp.Error)
		}
		return nil
	})

	// Extension HTTP proxy (forwards /admin/api/ext/:slug/* to gRPC plugins).
	extensionProxy := cms.NewExtensionProxy(pluginManager)
	extensionProxy.RegisterRoutes(adminAPI)

	// Extension admin handler.
	extHandler := cms.NewExtensionHandler(database, extLoader)
	extHandler.SetScriptLoader(scriptEngine)
	extHandler.SetPluginManager(pluginManager)
	extHandler.SetAssetRegistry(themeAssets)
	extHandler.SetEventBus(eventBus)
	extHandler.SetThemeLoader(themeLoader)
	extHandler.RegisterRoutes(adminAPI)
	mcpServer.SetExtensionHandler(extHandler)
	mcpServer.SetPluginManager(pluginManager)

	// Theme deploy webhook (public, authenticated by secret).
	themeHandler.RegisterWebhook(app)

	// --- Public extension route proxy (before static routes) ---
	publicProxy := cms.NewPublicExtensionProxy(pluginManager)
	publicProxy.RegisterPublicRoutes(app, activeExts)

	// Media files served by media-manager extension via public_routes proxy.
	// Fallback static handler for when extension is not active.
	app.Static("/media", "./storage/media")

	// --- Public block + theme assets + admin SPA static mounts ---
	registerBlockAssets(app)
	themeAssetsDir := newThemeAssetsResolver(database, eventBus, themePath)
	registerThemeAssets(app, themeAssetsDir)
	registerAdminSPA(app)

	// --- Theme script API routes ---
	scriptEngine.MountHTTPRoutes(app)

	// --- .well-known/* registry (short-circuit before public catch-all) ---
	wellKnown := cms.NewWellKnownRegistry()
	scriptEngine.MountWellKnown(wellKnown)
	wellKnown.Mount(app)

	// --- Public content routes (must be last) ---
	publicHandler.RegisterRoutes(app)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("server failed to start: %v", err)
		}
	}()

	log.Printf("Squilla ready | http://localhost:%s | env=%s", cfg.Port, cfg.AppEnv)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down gracefully...")
	// Bounded shutdown — without a timeout, in-flight SSE streams or hung
	// MCP calls would block forever. Cancel the background context first
	// so cleanup loops exit before app.Shutdown waits on connections.
	bgCancel()
	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		// Don't log.Fatalf here — that calls os.Exit(1) which skips the
		// `defer pluginManager.StopAll()` above. Returning normally lets
		// every defer run.
		log.Printf("server shutdown error: %v", err)
	}
	log.Println("Squilla stopped")
}

func corsOrigins(env string) string {
	if env == "development" {
		return "http://localhost:3000,http://localhost:8080"
	}
	if origins := os.Getenv("CORS_ORIGINS"); origins != "" {
		return normalizeCORSOrigins(origins)
	}
	return "http://localhost:8099"
}

// normalizeCORSOrigins makes the configured allowlist resilient to small
// authoring mistakes that Fiber's CORS middleware would otherwise reject
// with a startup panic: missing scheme on a bare hostname (Coolify's
// SERVICE_FQDN_* variables, copy-pasted domain values), trailing slashes
// (browsers send origins without one), and stray whitespace from
// hand-edited env files. Wildcard "*" and the literal "null" origin
// (used by sandboxed iframes / file:// pages) pass through unchanged.
func normalizeCORSOrigins(raw string) string {
	parts := strings.Split(raw, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "*" || p == "null" {
			out = append(out, p)
			continue
		}
		if !strings.Contains(p, "://") {
			p = "https://" + p
		}
		p = strings.TrimRight(p, "/")
		out = append(out, p)
	}
	return strings.Join(out, ",")
}
