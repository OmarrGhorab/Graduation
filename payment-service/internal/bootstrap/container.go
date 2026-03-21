package bootstrap

import (
	"fmt"
	"log"

	paymentApp "github.com/OmarrGhorab/payment-service/internal/application/payment"
	"github.com/OmarrGhorab/payment-service/internal/config"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/cache/redis"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/coursesclient"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/messaging/kafka"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/paymob"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/persistence/postgres"
	paymentDomain "github.com/OmarrGhorab/payment-service/internal/domain/payment"
	"github.com/OmarrGhorab/payment-service/internal/interfaces/http"
	"github.com/gofiber/fiber/v2"
)

type Container struct {
	Config *config.Config
	App    *fiber.App
	DB     *postgres.Database

	// Repositories
	PaymentRepo *postgres.PaymentRepository

	// Infrastructure
	Redis         *redis.Client
	AuthClient    *authclient.Client
	CoursesClient *coursesclient.Client
	PaymobClient  *paymob.Client
	KafkaProducer *kafka.Producer

	// Services
	PaymentService *paymentApp.Service
}

func New() (*Container, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	app := http.NewApp()

	db, err := postgres.NewDatabase(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	container := &Container{
		Config: cfg,
		App:    app,
		DB:     db,
		Redis:  redisClient,
	}

	// Auto-migrate tables
	if err := db.DB.AutoMigrate(&paymentDomain.PaymentOrder{}, &paymentDomain.PaymentTransaction{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}



	container.initInfrastructure()
	container.initRepositories()
	container.initServices()
	container.registerRoutes()

	return container, nil
}

func (c *Container) initInfrastructure() {
	c.AuthClient = authclient.NewClient(c.Config.Auth.ServiceURL, c.Config.Auth.InternalSecret)
	c.CoursesClient = coursesclient.NewClient(c.Config.Courses.ServiceURL, c.Config.Auth.InternalSecret)
	c.PaymobClient = paymob.NewClient(
		c.Config.Paymob.APIKey,
		c.Config.Paymob.CardIntegrationID,
		c.Config.Paymob.WalletIntegrationID,
		c.Config.Paymob.IframeID,
		c.Config.Paymob.HMACSecret,
	)
	c.KafkaProducer = kafka.NewProducer(c.Config.Kafka.Brokers)
}

func (c *Container) initRepositories() {
	c.PaymentRepo = postgres.NewPaymentRepository(c.DB)
}

func (c *Container) initServices() {
	c.PaymentService = paymentApp.NewService(
		c.PaymentRepo,
		c.PaymobClient,
		c.CoursesClient,
		c.Redis,
		c.KafkaProducer,
	)
}

func (c *Container) registerRoutes() {
	healthHandler := http.NewHealthHandler()
	healthHandler.RegisterRoutes(c.App)

	apiV1 := c.App.Group("/api/v1")

	paymentHandler := http.NewPaymentHandler(c.PaymentService, c.AuthClient)
	paymentHandler.RegisterRoutes(apiV1)

	webhookHandler := http.NewWebhookHandler(c.PaymentService)
	webhookHandler.RegisterRoutes(apiV1)
}

func (c *Container) Start() error {
	addr := fmt.Sprintf(":%s", c.Config.Server.Port)
	log.Printf("Starting Payment Service on %s", addr)
	return c.App.Listen(addr)
}

func (c *Container) Shutdown() error {
	log.Println("Shutting down Payment Service...")

	if c.Redis != nil {
		c.Redis.Close()
	}
	if c.DB != nil {
		c.DB.Close()
	}
	if c.KafkaProducer != nil {
		c.KafkaProducer.Close()
	}

	return c.App.Shutdown()
}
