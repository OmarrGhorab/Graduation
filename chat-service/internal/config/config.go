package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server
	Port string
	Env  string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// Kafka
	KafkaBrokers []string

	// JWT
	JWTAccessSecret string
	JWTSecret       string

	// Internal Service Communication
	InternalServiceSecret  string
	NotificationServiceURL string
	AuthServiceURL         string

	// Cloudinary
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
	PollTimeout         time.Duration
	PollInterval        time.Duration

	// Rate Limiting
	RateLimitRequests int
	RateLimitWindow   time.Duration
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Support both KAFKA_BROKERS (comma-separated) and KAFKA_BROKER (single)
	kafkaBrokers := getEnv("KAFKA_BROKERS", "")
	if kafkaBrokers == "" {
		kafkaBrokers = getEnv("KAFKA_BROKER", "localhost:9092")
	}
	
	cfg := &Config{
		Port:                   getEnv("PORT", "6004"),
		Env:                    getEnv("ENV", "development"),
		DatabaseURL:            getEnv("DATABASE_URL", ""),
		RedisURL:               getEnv("REDIS_URL", "redis://localhost:6379"),
		KafkaBrokers:           []string{kafkaBrokers},
		JWTAccessSecret:        getEnv("JWT_ACCESS_SECRET", ""),
		JWTSecret:              getEnv("JWT_SECRET", "secret"),
		InternalServiceSecret:  getEnv("INTERNAL_SERVICE_SECRET", ""),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:6003"),
		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", "http://localhost:6001"),
		CloudinaryCloudName:    getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:       getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret:    getEnv("CLOUDINARY_API_SECRET", ""),
		PollTimeout:            time.Duration(getEnvAsInt("POLL_TIMEOUT_SECONDS", 30)) * time.Second,
		PollInterval:           time.Duration(getEnvAsInt("POLL_INTERVAL_MS", 500)) * time.Millisecond,
		RateLimitRequests:      getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:        time.Duration(getEnvAsInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second,
	}

	return cfg, nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
