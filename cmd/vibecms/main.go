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

	// Create handlers.
	authHandler := auth.NewAuthHandler(database, sessionSvc)
	userHandler := auth.NewUserHandler(database)
	nodeHandler := cms.NewNodeHandler(contentSvc)
	healthHandler := api.NewHealthHandler(database)

	// --- Public routes ---

	// Auth routes (login is public, logout/me require auth).
	authHandler.RegisterRoutes(app)

	// Health check (public).
	app.Get("/api/v1/health", healthHandler.HealthCheck)

	// --- Monitoring routes (bearer token) ---
	app.Get("/api/v1/stats", api.BearerTokenRequired(cfg.MonitorBearerToken), healthHandler.Stats)

	// --- Admin API routes (session auth required) ---
	adminAPI := app.Group("/admin/api", auth.AuthRequired(sessionSvc))
	userHandler.RegisterRoutes(adminAPI)
	nodeHandler.RegisterRoutes(adminAPI)

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
