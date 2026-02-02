package config

import (
	"os"
)

type Config struct {
	Port           string
	RedisAddr      string
	KafkaBrokers   []string
	ChatServiceUrl string
	JwtSecret      string
}

func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8001"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers:   []string{getEnv("KAFKA_BROKER", "localhost:9092")},
		ChatServiceUrl: getEnv("CHAT_SERVICE_URL", "http://localhost:8000"),
		JwtSecret:      getEnv("JWT_SECRET", "secret"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
