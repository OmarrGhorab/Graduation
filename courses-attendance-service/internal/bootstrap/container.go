package bootstrap

import (
	"context"
	"fmt"
	"log"
	"time"

	absenceApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/absence"
	attendanceApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/attendance"
	calendarApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/calendar"
	courseApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/course"
	lessonApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/lesson"
	progressApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/progress"
	watchtimeApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/watchtime"
	"github.com/OmarrGhorab/courses-attendance-service/internal/application/jobs"
	parentApp "github.com/OmarrGhorab/courses-attendance-service/internal/application/parent"
	"github.com/OmarrGhorab/courses-attendance-service/internal/config"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/aiclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/authclient"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/cache/redis"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/cloudinary"
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
	TeacherRatingRepo     *postgres.TeacherRatingRepository
	CourseRatingRepo      *postgres.CourseRatingRepository
	WatchTimeRepo         *postgres.WatchTimeRepository

	// Infrastructure
	Redis            *redis.Client
	AuthClient       *authclient.Client
	QRGenerator      *qr.Generator
	KafkaProducer    *kafka.Producer
	EventDispatcher  *notificationevents.EventDispatcher
	CloudinaryClient *cloudinary.Client
	AIClient         *aiclient.Client
	Clock            clock.Clock

	// Services
	CourseService     *courseApp.Service
	LessonService     *lessonApp.Service
	AttendanceService *attendanceApp.Service
	AbsenceService    *absenceApp.Service
	ProgressService   *progressApp.Service
	CalendarService   *calendarApp.Service
	WatchTimeService  *watchtimeApp.Service
	ParentService     *parentApp.Service

	// Jobs
	LessonRemindersJob *jobs.LessonRemindersJob
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

	// Cloudinary
	cloudinaryClient, err := cloudinary.NewClient(c.Config.Cloudinary)
	if err != nil {
		log.Fatalf("Failed to initialize Cloudinary: %v", err)
	}
	c.CloudinaryClient = cloudinaryClient

	// AI Service
	c.AIClient = aiclient.NewClient(c.Config.AI.RecommendationServiceURL, c.Config.Auth.InternalSecret)

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
	c.TeacherRatingRepo = postgres.NewTeacherRatingRepository(c.DB)
	c.CourseRatingRepo = postgres.NewCourseRatingRepository(c.DB)
	c.WatchTimeRepo = postgres.NewWatchTimeRepository(c.DB)
}

func (c *Container) initServices() {
	c.CourseService = courseApp.NewService(
		c.CourseRepo,
		c.SubjectRepo,
		c.EnrollmentRepo,
		c.AssistantRepo,
		c.EventDispatcher,
		c.TeacherRatingRepo,
		c.ProgressSnapshotRepo,
		c.AuthClient,
		c.AIClient,
		c.CloudinaryClient,
		c.Clock,
	)

	c.ProgressService = progressApp.NewService(
		c.ProgressSnapshotRepo,
		c.AttendanceRecordRepo,
		c.CourseRepo,
		c.LessonRepo,
		c.WatchTimeRepo,
		c.EventDispatcher,
		c.Clock,
	)

	c.LessonService = lessonApp.NewService(
		c.LessonRepo,
		c.CourseRepo,
		c.EnrollmentRepo,
		c.ProgressService,
		c.EventDispatcher,
		c.CloudinaryClient,
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
		c.AIClient,
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
		c.CourseRepo,
		c.AuthClient,
		c.EventDispatcher,
		c.Clock,
	)

	c.CalendarService = calendarApp.NewService(
		c.LessonRepo,
		c.CourseRepo,
		c.EnrollmentRepo,
	)

	c.WatchTimeService = watchtimeApp.NewService(
		c.WatchTimeRepo,
		c.LessonRepo,
		c.CourseRepo,
		c.EnrollmentRepo,
		c.AttendanceRecordRepo,
		c.Clock,
		c.AIClient,
		c.AuthClient,
		c.Config.AI.PaymentServiceURL,
		c.Config.Auth.InternalSecret,
	)

	c.ParentService = parentApp.NewService(
		c.AuthClient,
		c.ProgressService,
		c.AttendanceService,
		c.EnrollmentRepo,
		c.CourseRepo,
		c.LessonRepo,
		c.AttendanceRecordRepo,
	)

	// Initialize Jobs
	c.LessonRemindersJob = jobs.NewLessonRemindersJob(
		c.LessonRepo,
		c.EventDispatcher,
		c.Clock,
	)
}

func (c *Container) registerRoutes() {
	// Health endpoints
	healthHandler := http.NewHealthHandler()
	healthHandler.RegisterRoutes(c.App)

	// API v1 group
	apiV1 := c.App.Group("/api/v1")

	// Course routes (with combined endpoint support)
	courseHandler := http.NewCourseHandlerWithServices(
		c.CourseService,
		c.LessonService,
		c.ProgressService,
		c.AuthClient,
		c.TeacherRatingRepo,
		c.CourseRatingRepo,
		c.EnrollmentRepo,
		c.AttendanceRecordRepo,
		c.AbsenceRequestRepo,
	)
	courseHandler.RegisterRoutes(apiV1)

	// Internal routes
	internalHandler := http.NewInternalHandler(c.CourseService, c.WatchTimeService, c.AuthClient, c.Config.Auth.InternalSecret)
	internalHandler.RegisterRoutes(apiV1)


	// Teacher routes
	teacherHandler := http.NewTeacherHandler(c.TeacherRatingRepo, c.AuthClient)
	teacherHandler.RegisterRoutes(apiV1)

	// Lesson routes
	lessonHandler := http.NewLessonHandler(c.LessonService, c.AttendanceService, c.AuthClient, c.CloudinaryClient)
	lessonHandler.RegisterRoutes(apiV1)

	// Attendance routes
	attendanceHandler := http.NewAttendanceHandler(c.AttendanceService, c.AuthClient)
	attendanceHandler.RegisterRoutes(apiV1)

	// Absence routes
	absenceHandler := http.NewAbsenceHandler(c.AbsenceService, c.AuthClient)
	absenceHandler.RegisterRoutes(apiV1)

	// Progress routes
	progressHandler := http.NewProgressHandler(c.ProgressService, c.AuthClient)
	progressHandler.RegisterRoutes(apiV1)

	// Calendar routes
	calendarHandler := http.NewCalendarHandler(c.CalendarService, c.AuthClient)
	calendarHandler.RegisterRoutes(apiV1)

	// Watch time tracking routes
	watchTimeHandler := http.NewWatchTimeHandler(c.WatchTimeService, c.AuthClient)
	watchTimeHandler.RegisterRoutes(apiV1)

	// Parent routes
	parentHandler := http.NewParentHandler(c.ParentService, c.AuthClient)
	parentHandler.RegisterRoutes(apiV1)
}

// Start starts the HTTP server.
func (c *Container) Start() error {
	addr := fmt.Sprintf(":%s", c.Config.Server.Port)
	log.Printf("Starting server on %s", addr)

	// Start background jobs
	go c.LessonRemindersJob.StartScheduler(context.Background(), 1*time.Minute)

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
