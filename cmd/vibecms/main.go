package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"vibecms/internal/api"
	"vibecms/internal/auth"
	"vibecms/internal/cms"
	"vibecms/internal/config"
	"vibecms/internal/coreapi"
	"vibecms/internal/db"
	"vibecms/internal/email"
	"vibecms/internal/events"
	"vibecms/internal/rbac"
	"vibecms/internal/rendering"
	"vibecms/internal/scripting"
	pb "vibecms/pkg/plugin/coreapipb"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()
	log.Printf("VibeCMS starting | env=%s port=%s", cfg.AppEnv, cfg.Port)

	database, err := db.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	log.Println("database connected successfully")

	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}
	log.Println("database migrations applied")

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			log.Println("migrations complete, exiting")
			return
		case "seed":
			if err := db.Seed(database); err != nil {
				log.Fatalf("database seed failed: %v", err)
			}
			log.Println("database seeded, exiting")
			return
		}
	}

	// Event bus.
	eventBus := events.New()

	// Fiber app.
	app := fiber.New(fiber.Config{
		AppName:               "VibeCMS",
		DisableStartupMessage: false,
	})

	app.Use(fiberlogger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins(cfg.AppEnv),
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))

	// Services.
	sessionSvc := auth.NewSessionService(database, cfg.SessionExpiryHours)
	contentSvc := cms.NewContentService(database, eventBus)
	nodeTypeSvc := cms.NewNodeTypeService(database)
	langSvc := cms.NewLanguageService(database)
	themeAssets := cms.NewThemeAssetRegistry()
	blockTypeSvc := cms.NewBlockTypeService(database, eventBus, themeAssets)
	templateSvc := cms.NewTemplateService(database)
	layoutSvc := cms.NewLayoutService(database, eventBus, themeAssets)
	layoutBlockSvc := cms.NewLayoutBlockService(database, eventBus, themeAssets)
	menuSvc := cms.NewMenuService(database, eventBus)
	isDev := cfg.AppEnv == "development"
	renderer := rendering.NewTemplateRenderer("ui/templates", isDev)

	// Email services.
	emailRuleSvc := email.NewRuleService(database)
	emailLogSvc := email.NewLogService(database)
	emailDispatcher := email.NewDispatcher(database, emailRuleSvc, emailLogSvc)
	eventBus.SubscribeAll(emailDispatcher.HandleEvent)

	// Handlers.
	authHandler := auth.NewAuthHandler(database, sessionSvc)
	userHandler := auth.NewUserHandler(database, eventBus)
	nodeHandler := cms.NewNodeHandler(contentSvc, database, eventBus)
	nodeTypeHandler := cms.NewNodeTypeHandler(nodeTypeSvc)
	langHandler := cms.NewLanguageHandler(langSvc)
	blockTypeHandler := cms.NewBlockTypeHandler(blockTypeSvc)
	templateHandler := cms.NewTemplateHandler(templateSvc)
	layoutHandler := cms.NewLayoutHandler(layoutSvc)
	layoutBlockHandler := cms.NewLayoutBlockHandler(layoutBlockSvc)
	menuHandler := cms.NewMenuHandler(menuSvc)
	healthHandler := api.NewHealthHandler(database)
	roleHandler := rbac.NewRoleHandler(database)
	settingsHandler := cms.NewSettingsHandler(database, eventBus)
	pageAuthHandler := auth.NewPageAuthHandler(database, sessionSvc, eventBus)

	// Theme loading.
	themeLoader := cms.NewThemeLoader(database, themeAssets, eventBus)
	themePath := os.Getenv("THEME_PATH")
	if themePath == "" {
		themePath = "themes/default"
	}
	themeLoader.LoadTheme(themePath)

	// Theme management.
	themeMgmtSvc := cms.NewThemeMgmtService(database, themeLoader, "themes")
	themeHandler := cms.NewThemeHandler(database, themeMgmtSvc)

	// CoreAPI — unified API facade for extensions.
	coreAPI := coreapi.NewCoreImpl(database, eventBus, contentSvc, menuSvc, nil, nodeTypeSvc, emailDispatcher, app)

	// Theme scripting engine.
	scriptEngine := scripting.NewScriptEngine(eventBus, coreAPI)
	if err := scriptEngine.LoadThemeScripts(themePath); err != nil {
		log.Printf("WARN: theme script loading failed: %v", err)
	}
	// Extension loading.
	extLoader := cms.NewExtensionLoader(database, "extensions")
	extLoader.ScanAndRegister()
	activeExts, _ := extLoader.GetActive()
	for _, ext := range activeExts {
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
	publicHandler := cms.NewPublicHandler(database, renderer, sessionSvc, layoutSvc, layoutBlockSvc, menuSvc, renderCtx, eventBus)

	// --- Public HTML pages ---
	pageAuthHandler.RegisterRoutes(app)

	// --- API Auth routes ---
	authHandler.RegisterRoutes(app)

	// Health check.
	app.Get("/api/v1/health", healthHandler.HealthCheck)
	app.Get("/api/v1/stats", api.BearerTokenRequired(cfg.MonitorBearerToken), healthHandler.Stats)

	// --- Admin API routes (session auth required) ---
	adminAPI := app.Group("/admin/api", auth.AuthRequired(sessionSvc))
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
	settingsHandler.RegisterRoutes(adminAPI)
	cacheHandler := cms.NewCacheHandler(publicHandler, eventBus)
	cacheHandler.RegisterRoutes(adminAPI)
	themeHandler.RegisterRoutes(adminAPI)

	// Plugin manager for gRPC extension plugins.
	hostRegistrar := cms.HostServerRegistrar(func(slug string, capabilities map[string]bool) func(s *grpc.Server) {
		caller := coreapi.CallerInfo{Slug: slug, Type: "grpc", Capabilities: capabilities}
		hostServer := coreapi.NewGRPCHostServer(coreAPI, caller)
		return func(s *grpc.Server) {
			pb.RegisterVibeCMSHostServer(s, hostServer)
		}
	})
	pluginManager := cms.NewPluginManager(eventBus, hostRegistrar)
	defer pluginManager.StopAll()

	// Start plugins for already-active extensions.
	for _, ext := range activeExts {
		var manifest cms.ExtensionManifest
		_ = json.Unmarshal(ext.Manifest, &manifest)
		caps := manifest.CapabilityMap()
		if err := pluginManager.StartPlugins(ext.Path, ext.Slug, json.RawMessage(ext.Manifest), caps); err != nil {
			log.Printf("WARN: extension %s plugin start failed: %v", ext.Slug, err)
		}
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
	extHandler.RegisterRoutes(adminAPI)

	// Theme deploy webhook (public, authenticated by secret).
	themeHandler.RegisterWebhook(app)

	// --- Media files ---
	app.Static("/media", "./storage/media")

	// --- Theme static assets ---
	app.Static("/theme/assets", filepath.Join(themePath, "assets"))

	// --- Admin SPA ---
	// Hashed assets: cache forever
	app.Static("/admin/assets", "./admin-ui/dist/assets", fiber.Static{
		MaxAge: 31536000, // 1 year — filenames are hashed by Vite
	})
	// Shims, previews, extension UIs: no cache — unhashed filenames
	noCache := func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		return c.Next()
	}
	app.Use("/admin/shims", noCache)
	app.Static("/admin/shims", "./admin-ui/dist/shims")
	app.Use("/admin/previews", noCache)
	app.Static("/admin/previews", "./admin-ui/dist/previews")
	app.Get("/admin/*", func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache")
		return c.SendFile("./admin-ui/dist/index.html")
	})

	// --- Theme script API routes ---
	scriptEngine.MountHTTPRoutes(app)

	// --- Public content routes (must be last) ---
	publicHandler.RegisterRoutes(app)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("server failed to start: %v", err)
		}
	}()

	log.Printf("VibeCMS ready | http://localhost:%s | env=%s", cfg.Port, cfg.AppEnv)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down gracefully...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	log.Println("VibeCMS stopped")
}

func corsOrigins(env string) string {
	if env == "development" {
		return "http://localhost:3000,http://localhost:8080"
	}
	if origins := os.Getenv("CORS_ORIGINS"); origins != "" {
		return origins
	}
	return "http://localhost:8099"
}
