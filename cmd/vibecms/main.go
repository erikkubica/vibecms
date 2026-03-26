package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"vibecms/internal/api"
	"vibecms/internal/auth"
	"vibecms/internal/cms"
	"vibecms/internal/config"
	"vibecms/internal/db"
	"vibecms/internal/rendering"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Load configuration from environment.
	cfg := config.Load()

	log.Printf("VibeCMS starting | env=%s port=%s", cfg.AppEnv, cfg.Port)

	// Connect to PostgreSQL (fatal on failure per architecture convention).
	database, err := db.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	log.Println("database connected successfully")

	// Run migrations.
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}
	log.Println("database migrations applied")

	// Handle CLI sub-commands: "migrate" and "seed" exit after their task.
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

	// Initialize Fiber.
	app := fiber.New(fiber.Config{
		AppName:               "VibeCMS",
		DisableStartupMessage: false,
	})

	// Global middleware.
	app.Use(fiberlogger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins(cfg.AppEnv),
		AllowMethods:     "GET,POST,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))

	// Create services.
	sessionSvc := auth.NewSessionService(database, cfg.SessionExpiryHours)
	contentSvc := cms.NewContentService(database)
	nodeTypeSvc := cms.NewNodeTypeService(database)
	langSvc := cms.NewLanguageService(database)
	blockTypeSvc := cms.NewBlockTypeService(database)
	templateSvc := cms.NewTemplateService(database)
	isDev := cfg.AppEnv == "development"
	renderer := rendering.NewTemplateRenderer("ui/templates", isDev)

	// Create handlers.
	authHandler := auth.NewAuthHandler(database, sessionSvc)
	userHandler := auth.NewUserHandler(database)
	nodeHandler := cms.NewNodeHandler(contentSvc, database)
	nodeTypeHandler := cms.NewNodeTypeHandler(nodeTypeSvc)
	langHandler := cms.NewLanguageHandler(langSvc)
	blockTypeHandler := cms.NewBlockTypeHandler(blockTypeSvc)
	templateHandler := cms.NewTemplateHandler(templateSvc)
	healthHandler := api.NewHealthHandler(database)
	publicHandler := cms.NewPublicHandler(database, renderer, sessionSvc)
	pageAuthHandler := auth.NewPageAuthHandler(database, sessionSvc, renderer)

	// --- Public HTML pages ---
	pageAuthHandler.RegisterRoutes(app)

	// --- API Auth routes (login is public, logout/me require auth) ---
	authHandler.RegisterRoutes(app)

	// Health check (public).
	app.Get("/api/v1/health", healthHandler.HealthCheck)

	// --- Monitoring routes (bearer token) ---
	app.Get("/api/v1/stats", api.BearerTokenRequired(cfg.MonitorBearerToken), healthHandler.Stats)

	// --- Admin API routes (session auth required) ---
	adminAPI := app.Group("/admin/api", auth.AuthRequired(sessionSvc))
	userHandler.RegisterRoutes(adminAPI)
	nodeHandler.RegisterRoutes(adminAPI)
	nodeTypeHandler.RegisterRoutes(adminAPI)
	langHandler.RegisterRoutes(adminAPI)
	blockTypeHandler.RegisterRoutes(adminAPI)
	templateHandler.RegisterRoutes(adminAPI)

	// --- Admin SPA (serve built React app) ---
	app.Static("/admin/assets", "./admin-ui/dist/assets")
	app.Get("/admin/*", func(c *fiber.Ctx) error {
		return c.SendFile("./admin-ui/dist/index.html")
	})

	// --- Public content routes (must be last - catches /:slug) ---
	publicHandler.RegisterRoutes(app)

	// Start server in a goroutine for graceful shutdown.
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("server failed to start: %v", err)
		}
	}()

	log.Printf("VibeCMS ready | http://localhost:%s | env=%s", cfg.Port, cfg.AppEnv)

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down gracefully...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	log.Println("VibeCMS stopped")
}

// corsOrigins returns allowed CORS origins based on the application environment.
func corsOrigins(env string) string {
	if env == "development" {
		return "http://localhost:3000,http://localhost:8080"
	}
	return ""
}
