package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                   string
	DatabaseURL            string
	RedisURL               string
	KafkaBrokers           []string
	JWTSecret              string
	InternalServiceSecret  string
	NotificationServiceURL string // For offline push notifications
	AuthServiceURL         string // For fetching user profiles
	WSGatewayURL           string // For checking user presence

	// Cloudinary
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
}

func Load() *Config {
	// Attempt to load from .env file, but don't fail if it doesn't exist (prod might use real env vars)
	_ = godotenv.Load()

	return &Config{
		Port:                   getEnv("PORT", "6004"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgresql://user:pass@localhost:5432/db"),
		RedisURL:               getEnv("REDIS_URL", "redis://localhost:6379"),
		KafkaBrokers:           strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		JWTSecret:              getEnv("JWT_ACCESS_SECRET", "secret"),
		InternalServiceSecret:  getEnv("INTERNAL_SERVICE_SECRET", "internal_secret"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:6003"),
		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", "http://localhost:6001"),
		WSGatewayURL:           getEnv("WS_GATEWAY_URL", "http://localhost:6005"),

		CloudinaryCloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:    getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret: getEnv("CLOUDINARY_API_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
