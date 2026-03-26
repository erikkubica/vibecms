package main

import (
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
	"vibecms/internal/db"
	"vibecms/internal/email"
	"vibecms/internal/events"
	"vibecms/internal/rbac"
	"vibecms/internal/rendering"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
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
		AllowMethods:     "GET,POST,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))

	// Services.
	sessionSvc := auth.NewSessionService(database, cfg.SessionExpiryHours)
	contentSvc := cms.NewContentService(database, eventBus)
	nodeTypeSvc := cms.NewNodeTypeService(database)
	langSvc := cms.NewLanguageService(database)
	blockTypeSvc := cms.NewBlockTypeService(database, eventBus)
	templateSvc := cms.NewTemplateService(database)
	themeAssets := cms.NewThemeAssetRegistry()
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
	nodeHandler := cms.NewNodeHandler(contentSvc, database)
	nodeTypeHandler := cms.NewNodeTypeHandler(nodeTypeSvc)
	langHandler := cms.NewLanguageHandler(langSvc)
	blockTypeHandler := cms.NewBlockTypeHandler(blockTypeSvc)
	templateHandler := cms.NewTemplateHandler(templateSvc)
	layoutHandler := cms.NewLayoutHandler(layoutSvc)
	layoutBlockHandler := cms.NewLayoutBlockHandler(layoutBlockSvc)
	menuHandler := cms.NewMenuHandler(menuSvc)
	healthHandler := api.NewHealthHandler(database)
	roleHandler := rbac.NewRoleHandler(database)
	emailHandler := email.NewEmailHandler(database)
	pageAuthHandler := auth.NewPageAuthHandler(database, sessionSvc, eventBus)

	// Theme loading.
	themeLoader := cms.NewThemeLoader(database, themeAssets)
	themePath := os.Getenv("THEME_PATH")
	if themePath == "" {
		themePath = "themes/default"
	}
	themeLoader.LoadTheme(themePath)

	// Theme management.
	themeMgmtSvc := cms.NewThemeMgmtService(database, themeLoader, "themes")
	themeHandler := cms.NewThemeHandler(database, themeMgmtSvc)
	pageTemplateHandler := cms.NewPageTemplateHandler(themeAssets)

	renderCtx := cms.NewRenderContext(database, layoutSvc, layoutBlockSvc, menuSvc, themeAssets)
	publicHandler := cms.NewPublicHandler(database, renderer, sessionSvc, layoutSvc, layoutBlockSvc, menuSvc, renderCtx)

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
	emailHandler.RegisterRoutes(adminAPI)
	themeHandler.RegisterRoutes(adminAPI)
	pageTemplateHandler.RegisterRoutes(adminAPI)

	// Theme deploy webhook (public, authenticated by secret).
	themeHandler.RegisterWebhook(app)

	// --- Theme static assets ---
	app.Static("/theme/assets", filepath.Join(themePath, "assets"))

	// --- Admin SPA ---
	app.Static("/admin/assets", "./admin-ui/dist/assets")
	app.Get("/admin/*", func(c *fiber.Ctx) error {
		return c.SendFile("./admin-ui/dist/index.html")
	})

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
	return ""
}
