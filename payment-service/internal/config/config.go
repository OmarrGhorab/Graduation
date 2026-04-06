package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	Auth     AuthConfig
	Paymob   PaymobConfig
	Courses  CoursesConfig
	Email    EmailConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers []string
}

type AuthConfig struct {
	ServiceURL     string
	InternalSecret string
}

type PaymobConfig struct {
	APIKey              string
	CardIntegrationID   string
	WalletIntegrationID string
	IframeID            string
	HMACSecret          string
}

type CoursesConfig struct {
	ServiceURL string
}

type EmailConfig struct {
	ResendAPIKey string
	FromEmail    string
	FromName     string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8090"),
			ReadTimeout:  time.Duration(getEnvInt("SERVER_READ_TIMEOUT_SECONDS", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("SERVER_WRITE_TIMEOUT_SECONDS", 30)) * time.Second,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "graduation"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		},
		Auth: AuthConfig{
			ServiceURL:     getEnv("AUTH_SERVICE_URL", "http://localhost:6001"),
			InternalSecret: getEnv("INTERNAL_SERVICE_SECRET", ""),
		},
		Paymob: PaymobConfig{
			APIKey:              getEnv("PAYMOB_API_KEY", ""),
			CardIntegrationID:   getEnv("PAYMOB_CARD_INTEGRATION_ID", ""),
			WalletIntegrationID: getEnv("PAYMOB_WALLET_INTEGRATION_ID", ""),
			IframeID:            getEnv("PAYMOB_IFRAME_ID", ""),
			HMACSecret:          getEnv("PAYMOB_HMAC_SECRET", ""),
		},
		Courses: CoursesConfig{
			ServiceURL: getEnv("COURSES_SERVICE_URL", "http://localhost:8085"),
		},
		Email: EmailConfig{
			ResendAPIKey: getEnv("RESEND_API_KEY", ""),
			FromEmail:    getEnv("EMAIL_FROM", "onboarding@resend.dev"),
			FromName:     getEnv("EMAIL_FROM_NAME", "Payment Service"),
		},
	}

	if cfg.Auth.InternalSecret == "" {
		return nil, fmt.Errorf("INTERNAL_SERVICE_SECRET is required")
	}
	if cfg.Paymob.APIKey == "" {
		return nil, fmt.Errorf("PAYMOB_API_KEY is required")
	}

	return cfg, nil
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
