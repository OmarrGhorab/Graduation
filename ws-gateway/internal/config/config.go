package config

import (
	"os"
)

type Config struct {
	Port           string
	RedisURL       string
	KafkaBrokers   []string
	ChatServiceUrl string
	JwtSecret      string
}

func Load() *Config {
	// Support both KAFKA_BROKERS (comma-separated) and KAFKA_BROKER (single)
	kafkaBrokers := getEnv("KAFKA_BROKERS", "")
	if kafkaBrokers == "" {
		kafkaBrokers = getEnv("KAFKA_BROKER", "localhost:9092")
	}
	
	return &Config{
		Port:           getEnv("PORT", "8001"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		KafkaBrokers:   []string{kafkaBrokers},
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
