package main

import (
	"context"
	"log"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/graduation/ws-gateway/internal/config"
	"github.com/graduation/ws-gateway/internal/handlers"
	"github.com/graduation/ws-gateway/internal/kafka"
	"github.com/graduation/ws-gateway/internal/observability"
	wsCore "github.com/graduation/ws-gateway/internal/websocket"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	// 4. Fiber App
	app := fiber.New()

	// Initialize Observability
	obs := observability.Init(app)
	defer obs.Shutdown()

	// 1. Redis
	opt, _ := redis.ParseURL(cfg.RedisURL)
	redisClient := redis.NewClient(opt)

	// 2. WebSocket Manager
	manager := wsCore.NewManager(redisClient)
	go manager.Run()

	// 3. Kafka Consumer
	consumer := kafka.NewConsumer(cfg.KafkaBrokers, "ws-gateway-group", manager)
	go consumer.Start(context.Background())


	// WebSocket Route
	app.Use("/ws", handlers.WebSocketHandler(manager, cfg))
	app.Get("/ws", websocket.New(handlers.WebSocketConnection(manager)))

	// Presence Check Endpoint
	app.Post("/api/presence", handlers.CheckPresenceHandler(manager, cfg))

	// Health Check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Sentry Debug
	app.Get("/debug-sentry", func(c *fiber.Ctx) error {
		panic("Sentry Debug Test: WS Gateway is connected!")
	})

	log.Printf("WS Gateway starting on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
