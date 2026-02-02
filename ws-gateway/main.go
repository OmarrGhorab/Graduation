package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/graduation/ws-gateway/internal/config"
	"github.com/graduation/ws-gateway/internal/handlers"
	"github.com/graduation/ws-gateway/internal/kafka"
	wsCore "github.com/graduation/ws-gateway/internal/websocket"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis at %s: %v", cfg.RedisAddr, err)
	} else {
		log.Printf("Connected to Redis at %s", cfg.RedisAddr)
	}

	// Initialize WebSocket Manager
	handlers.Manager = wsCore.NewManager(redisClient)
	go handlers.Manager.Run()

	// Initialize Kafka Consumer
	// Note: consumer logic uses "internal/events" which defines the topics.
	// We use the internal/kafka package for the consumer implementation.
	consumer := kafka.NewConsumer(cfg.KafkaBrokers, "ws-gateway-group", redisClient)
	go consumer.Start(context.Background())

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName: "WS Gateway Service",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Health Check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("WS Gateway is healthy")
	})

	// WebSocket Routes
	// fiber-websocket middleware handles upgrade
	app.Get("/ws", handlers.WebSocketHandler, websocket.New(handlers.WebSocketConnection))

	// Start server
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Panicf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down WS Gateway...")
	if err := app.Shutdown(); err != nil {
		log.Printf("Error shutting down: %v", err)
	}
}
