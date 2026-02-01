package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/graduation/chat-service/internal/config"
	"github.com/graduation/chat-service/internal/handlers"
	"github.com/graduation/chat-service/internal/middleware"
	"github.com/graduation/chat-service/internal/repositories"
	"github.com/graduation/chat-service/internal/router"
	"github.com/graduation/chat-service/internal/services"
	"github.com/graduation/chat-service/pkg/cache"
	"github.com/graduation/chat-service/pkg/database"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	db, err := database.NewPostgresConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run database migrations
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Initialize Redis connection
	redisClient, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize repositories
	repos := repositories.NewRepositories(db)

	// Initialize services
	svcs := services.NewServices(repos, redisClient, cfg)

	// Initialize handlers
	hdlrs := handlers.NewHandlers(svcs)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTAccessSecret)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "Chat Service",
		ReadTimeout:           time.Second * 60,
		WriteTimeout:          time.Second * 60,
		IdleTimeout:           time.Second * 120,
		DisableStartupMessage: false,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,x-internal-service-secret",
		AllowCredentials: false,
	}))

	// Setup routes
	router.SetupRoutes(app, hdlrs, authMiddleware)

	// Start server
	go func() {
		log.Printf("Chat service starting on port %s", cfg.Port)
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down chat service...")

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}

	// Close database connection
	sqlDB, _ := db.DB()
	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}

	log.Println("Chat service stopped")
}
