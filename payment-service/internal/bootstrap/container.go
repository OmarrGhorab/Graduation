package bootstrap

import (
	"context"
	"fmt"
	"log"
	"time"

	cartApp "github.com/OmarrGhorab/payment-service/internal/application/cart"
	"github.com/OmarrGhorab/payment-service/internal/application/jobs"
	paymentApp "github.com/OmarrGhorab/payment-service/internal/application/payment"
	subscriptionApp "github.com/OmarrGhorab/payment-service/internal/application/subscription"
	"github.com/OmarrGhorab/payment-service/internal/config"
	cartDomain "github.com/OmarrGhorab/payment-service/internal/domain/cart"
	paymentDomain "github.com/OmarrGhorab/payment-service/internal/domain/payment"
	paymentMethodDomain "github.com/OmarrGhorab/payment-service/internal/domain/paymentmethod"
	subscriptionDomain "github.com/OmarrGhorab/payment-service/internal/domain/subscription"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/cache/redis"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/coursesclient"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/messaging/kafka"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/notification"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/paymob"
	"github.com/OmarrGhorab/payment-service/internal/infrastructure/persistence/postgres"
	"github.com/OmarrGhorab/payment-service/internal/interfaces/http"
	"github.com/gofiber/fiber/v2"
)

type Container struct {
	Config *config.Config
	App    *fiber.App
	DB     *postgres.Database

	// Repositories
	PaymentRepo       *postgres.PaymentRepository
	CartRepo          *postgres.CartRepository
	SubscriptionRepo  *postgres.SubscriptionRepository
	PaymentMethodRepo *postgres.PaymentMethodRepository

	// Infrastructure
	Redis         *redis.Client
	AuthClient    *authclient.Client
	CoursesClient *coursesclient.Client
	PaymobClient  *paymob.Client
	KafkaProducer *kafka.Producer
	EmailService  *notification.EmailService

	// Services
	PaymentService      *paymentApp.Service
	CartService         *cartApp.Service
	SubscriptionService *subscriptionApp.Service

	// Jobs
	BillingJob *jobs.SubscriptionBillingJob

	// Context for background jobs
	jobCtx    context.Context
	jobCancel context.CancelFunc
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

	jobCtx, jobCancel := context.WithCancel(context.Background())

	container := &Container{
		Config:    cfg,
		App:       app,
		DB:        db,
		Redis:     redisClient,
		jobCtx:    jobCtx,
		jobCancel: jobCancel,
	}

	// Auto-migrate tables
	if err := db.DB.AutoMigrate(
		&paymentDomain.PaymentOrder{},
		&paymentDomain.PaymentTransaction{},
		&paymentDomain.PaymentOrderItem{},
		&cartDomain.Cart{},
		&cartDomain.CartItem{},
		&subscriptionDomain.Subscription{},
		&paymentMethodDomain.PaymentMethod{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	container.initInfrastructure()
	container.initRepositories()
	container.initServices()
	container.initJobs()
	container.registerRoutes()
	container.startBackgroundJobs()

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
	c.EmailService = notification.NewEmailService(
		c.Config.Email.ResendAPIKey,
		c.Config.Email.FromEmail,
		c.Config.Email.FromName,
	)
}

func (c *Container) initRepositories() {
	c.PaymentRepo = postgres.NewPaymentRepository(c.DB)
	c.CartRepo = postgres.NewCartRepository(c.DB.DB)
	c.SubscriptionRepo = postgres.NewSubscriptionRepository(c.DB.DB)
	c.PaymentMethodRepo = postgres.NewPaymentMethodRepository(c.DB.DB)
}

func (c *Container) initServices() {
	c.PaymentService = paymentApp.NewService(
		c.PaymentRepo,
		c.CartRepo,
		c.SubscriptionRepo,
		c.PaymentMethodRepo,
		c.PaymobClient,
		c.CoursesClient,
		c.Redis,
		c.KafkaProducer,
	)

	c.CartService = cartApp.NewService(
		c.CartRepo,
		c.CoursesClient,
	)

	c.SubscriptionService = subscriptionApp.NewService(
		c.SubscriptionRepo,
		c.CoursesClient,
	)
}

func (c *Container) initJobs() {
	c.BillingJob = jobs.NewSubscriptionBillingJob(
		c.SubscriptionService,
		c.PaymentService,
		c.SubscriptionRepo,
		c.PaymentMethodRepo,
		c.EmailService,
	)
}

func (c *Container) registerRoutes() {
	healthHandler := http.NewHealthHandler()
	healthHandler.RegisterRoutes(c.App)

	apiV1 := c.App.Group("/api/v1")

	paymentHandler := http.NewPaymentHandler(c.PaymentService, c.AuthClient)
	paymentHandler.RegisterRoutes(apiV1)

	cartHandler := http.NewCartHandler(c.CartService, c.PaymentService, c.AuthClient)
	cartHandler.RegisterRoutes(apiV1)

	subscriptionHandler := http.NewSubscriptionHandler(c.SubscriptionService, c.AuthClient)
	subscriptionHandler.RegisterRoutes(apiV1)

	webhookHandler := http.NewWebhookHandler(c.PaymentService)
	webhookHandler.RegisterRoutes(apiV1)
}

func (c *Container) startBackgroundJobs() {
	// Start subscription billing job (runs daily at 2 AM)
	// For testing, you can change this to a shorter interval like 1*time.Hour
	go c.BillingJob.StartScheduler(c.jobCtx, 24*time.Hour)
	log.Println("Subscription billing job scheduler started")
}

func (c *Container) Start() error {
	addr := fmt.Sprintf(":%s", c.Config.Server.Port)
	log.Printf("Starting Payment Service on %s", addr)
	return c.App.Listen(addr)
}

func (c *Container) Shutdown() error {
	log.Println("Shutting down Payment Service...")

	// Cancel background jobs
	if c.jobCancel != nil {
		c.jobCancel()
	}

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
