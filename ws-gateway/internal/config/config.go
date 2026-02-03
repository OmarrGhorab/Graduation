package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	RedisURL       string
	KafkaBrokers   []string
	ChatServiceUrl string
	JwtSecret      string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:           getEnv("PORT", "8001"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		KafkaBrokers:   strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		ChatServiceUrl: getEnv("CHAT_SERVICE_URL", "http://localhost:6004"),
		JwtSecret:      getEnv("JWT_ACCESS_SECRET", "secret"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
