package bootstrap

import (
	"fmt"
	"log"

	absenceApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/absence"
	attendanceApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/attendance"
	calendarApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/calendar"
	courseApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/course"
	lessonApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/lesson"
	progressApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/progress"
	"github.com/OmarrGhorab/courses-attendance-service/internal/config"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/cache/redis"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/messaging/kafka"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/notificationevents"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/qr"
	"github.com/OmarrGhorab/courses-attendance-service/internal/interfaces/http"
	"github.com/gofiber/fiber/v2"
)

// Container holds all dependencies for the application.
type Container struct {
	Config *config.Config
	App    *fiber.App
	DB     *postgres.Database

	// Repositories
	CourseRepo            *postgres.CourseRepository
	SubjectRepo           *postgres.SubjectRepository
	EnrollmentRepo        *postgres.EnrollmentRepository
	AssistantRepo         *postgres.CourseAssistantRepository
	LessonRepo            *postgres.LessonRepository
	AttendanceSessionRepo *postgres.AttendanceSessionRepository
	AttendanceQRTokenRepo *postgres.AttendanceQRTokenRepository
	AttendanceRecordRepo  *postgres.AttendanceRecordRepository
	AbsenceRequestRepo    *postgres.AbsenceRequestRepository
	ProgressSnapshotRepo  *postgres.ProgressSnapshotRepository

	// Infrastructure
	Redis           *redis.Client
	AuthClient      *authclient.Client
	QRGenerator     *qr.Generator
	KafkaProducer   *kafka.Producer
	EventDispatcher *notificationevents.EventDispatcher
	Clock           clock.Clock

	// Services
	CourseService     *courseApp.Service
	LessonService     *lessonApp.Service
	AttendanceService *attendanceApp.Service
	AbsenceService    *absenceApp.Service
	ProgressService   *progressApp.Service
	CalendarService   *calendarApp.Service
}

// New creates a new dependency injection container.
func New() (*Container, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	clk := clock.New()
	app := http.NewApp()

	// Initialize database
	db, err := postgres.NewDatabase(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize Redis
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	container := &Container{
		Config: cfg,
		Clock:  clk,
		App:    app,
		DB:     db,
		Redis:  redisClient,
	}

	// Initialize infrastructure
	container.initInfrastructure()

	// Initialize repositories
	container.initRepositories()

	// Initialize services
	container.initServices()

	// Register routes
	container.registerRoutes()

	return container, nil
}

func (c *Container) initInfrastructure() {
	c.AuthClient = authclient.NewClient(c.Config.Auth.ServiceURL, c.Config.Auth.InternalSecret)
	c.QRGenerator = qr.NewGenerator(c.Config.QR.SigningSecret, c.Clock)

	// Messaging
	c.KafkaProducer = kafka.NewProducer(c.Config.Kafka.Brokers)
	c.EventDispatcher = notificationevents.NewEventDispatcher(c.KafkaProducer, c.Clock)
}

func (c *Container) initRepositories() {
	c.CourseRepo = postgres.NewCourseRepository(c.DB)
	c.SubjectRepo = postgres.NewSubjectRepository(c.DB)
	c.EnrollmentRepo = postgres.NewEnrollmentRepository(c.DB)
	c.AssistantRepo = postgres.NewCourseAssistantRepository(c.DB)
	c.LessonRepo = postgres.NewLessonRepository(c.DB)
	c.AttendanceSessionRepo = postgres.NewAttendanceSessionRepository(c.DB)
	c.AttendanceQRTokenRepo = postgres.NewAttendanceQRTokenRepository(c.DB)
	c.AttendanceRecordRepo = postgres.NewAttendanceRecordRepository(c.DB)
	c.AbsenceRequestRepo = postgres.NewAbsenceRequestRepository(c.DB)
	c.ProgressSnapshotRepo = postgres.NewProgressSnapshotRepository(c.DB)
}

func (c *Container) initServices() {
	c.CourseService = courseApp.NewService(
		c.CourseRepo,
		c.SubjectRepo,
		c.EnrollmentRepo,
		c.AssistantRepo,
		c.EventDispatcher,
		c.Clock,
	)

	c.ProgressService = progressApp.NewService(
		c.ProgressSnapshotRepo,
		c.AttendanceRecordRepo,
		c.CourseRepo,
		c.LessonRepo,
		c.EventDispatcher,
		c.Clock,
	)

	c.LessonService = lessonApp.NewService(
		c.LessonRepo,
		c.CourseRepo,
		c.ProgressService,
		c.EventDispatcher,
		c.Clock,
	)

	c.AttendanceService = attendanceApp.NewService(
		c.LessonRepo,
		c.CourseRepo,
		c.EnrollmentRepo,
		c.AttendanceSessionRepo,
		c.AttendanceQRTokenRepo,
		c.AttendanceRecordRepo,
		c.Redis,
		c.AuthClient,
		c.QRGenerator,
		c.ProgressService,
		c.EventDispatcher,
		c.Clock,
		attendanceApp.ServiceConfig{
			RotationSeconds: c.Config.QR.RotationIntervalSeconds,
			ExpirySeconds:   c.Config.QR.ExpirySeconds,
		},
	)

	c.AbsenceService = absenceApp.NewService(
		c.AbsenceRequestRepo,
		c.AttendanceRecordRepo,
		c.LessonRepo,
		c.AuthClient,
		c.EventDispatcher,
		c.Clock,
	)

	c.CalendarService = calendarApp.NewService(
		c.LessonRepo,
		c.CourseRepo,
		c.EnrollmentRepo,
	)
}

func (c *Container) registerRoutes() {
	// Health endpoints
	healthHandler := http.NewHealthHandler()
	healthHandler.RegisterRoutes(c.App)

	// API v1 group
	apiV1 := c.App.Group("/api/v1")

	// Course routes
	courseHandler := http.NewCourseHandler(c.CourseService)
	courseHandler.RegisterRoutes(apiV1)

	// Lesson routes
	lessonHandler := http.NewLessonHandler(c.LessonService)
	lessonHandler.RegisterRoutes(apiV1)

	// Attendance routes
	attendanceHandler := http.NewAttendanceHandler(c.AttendanceService)
	attendanceHandler.RegisterRoutes(apiV1)

	// Absence routes
	absenceHandler := http.NewAbsenceHandler(c.AbsenceService)
	absenceHandler.RegisterRoutes(apiV1)

	// Progress routes
	progressHandler := http.NewProgressHandler(c.ProgressService)
	progressHandler.RegisterRoutes(apiV1)

	// Calendar routes
	calendarHandler := http.NewCalendarHandler(c.CalendarService)
	calendarHandler.RegisterRoutes(apiV1)
}

// Start starts the HTTP server.
func (c *Container) Start() error {
	addr := fmt.Sprintf(":%s", c.Config.Server.Port)
	log.Printf("Starting server on %s", addr)
	return c.App.Listen(addr)
}

// Shutdown gracefully shuts down the container.
func (c *Container) Shutdown() error {
	log.Println("Shutting down server...")

	// Close Redis connection
	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		}
	}

	// Close database connection
	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}

	return c.App.Shutdown()
}
