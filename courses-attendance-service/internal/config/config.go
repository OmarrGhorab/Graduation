package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Kafka     KafkaConfig
	Auth      AuthConfig
	QR        QRConfig
	Cloudinary CloudinaryConfig
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
	ServiceURL    string
	InternalSecret string
}

type QRConfig struct {
	SigningSecret          string
	RotationIntervalSeconds int
	ExpirySeconds          int
}

type CloudinaryConfig struct {
	CloudName string
	APIKey    string
	APISecret string
	Folder    string
}

func Load() (*Config, error) {
	_ = godotenv.Load() // Load .env if it exists, ignore errors
	
	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		redisDB = 0
	}

	qrRotation, _ := strconv.Atoi(getEnv("QR_ROTATION_INTERVAL_SECONDS", "30"))
	qrExpiry, _ := strconv.Atoi(getEnv("QR_EXPIRY_SECONDS", "35"))

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8085"),
			ReadTimeout:  time.Duration(getEnvInt("SERVER_READ_TIMEOUT_SECONDS", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("SERVER_WRITE_TIMEOUT_SECONDS", 30)) * time.Second,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "courses_attendance"),
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
			ServiceURL:    getEnv("AUTH_SERVICE_URL", "http://localhost:8080"),
			InternalSecret: getEnv("INTERNAL_SERVICE_SECRET", ""),
		},
		QR: QRConfig{
			SigningSecret:          getEnv("QR_SIGNING_SECRET", ""),
			RotationIntervalSeconds: qrRotation,
			ExpirySeconds:          qrExpiry,
		},
		Cloudinary: CloudinaryConfig{
			CloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
			APIKey:    getEnv("CLOUDINARY_API_KEY", ""),
			APISecret: getEnv("CLOUDINARY_API_SECRET", ""),
			Folder:    getEnv("CLOUDINARY_FOLDER", "course-materials"),
		},
	}

	if cfg.Auth.InternalSecret == "" {
		return nil, fmt.Errorf("INTERNAL_SERVICE_SECRET is required")
	}
	if cfg.QR.SigningSecret == "" {
		return nil, fmt.Errorf("QR_SIGNING_SECRET is required")
	}
	if cfg.Cloudinary.CloudName == "" || cfg.Cloudinary.APIKey == "" || cfg.Cloudinary.APISecret == "" {
		return nil, fmt.Errorf("Cloudinary credentials are required")
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
