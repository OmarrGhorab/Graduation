package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/graduation/chat-service/internal/config"
	"github.com/graduation/chat-service/internal/database"
	"github.com/graduation/chat-service/internal/handlers"
	"github.com/graduation/chat-service/internal/kafka"
	"github.com/graduation/chat-service/internal/middleware"
	"github.com/graduation/chat-service/internal/observability"
	"github.com/graduation/chat-service/internal/repository"
	"github.com/graduation/chat-service/internal/router"
	"github.com/graduation/chat-service/internal/service"
)

func main() {
	// 1. Config
	cfg := config.Load()

	// 2. Database
	db := database.Connect(cfg)

	// 3. Kafka Producer
	producer := kafka.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	// 4. Layers
	repo := repository.NewRepository(db)
	mediaSvc := service.NewMediaService(cfg)
	userSvc := service.NewUserService(cfg.AuthServiceURL, cfg.CoursesServiceURL, cfg.InternalServiceSecret)
	presenceSvc := service.NewPresenceService(cfg.WSGatewayURL)
	notificationSvc := service.NewNotificationService(cfg.NotificationServiceURL, cfg.InternalServiceSecret)
	svc := service.NewService(repo, producer, mediaSvc, userSvc, presenceSvc, notificationSvc)
	h := handlers.NewHandler(svc)
	auth := middleware.NewAuthMiddleware(cfg)

	// 5. App
	app := fiber.New()

	// Initialize Observability
	obs := observability.Init(app)
	defer obs.Shutdown()

	app.Use(logger.New())
	app.Use(cors.New())

	// 6. Routes
	router.SetupRoutes(app, h, auth)

	// 7. Start
	log.Printf("Chat Service starting on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
