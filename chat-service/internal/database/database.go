package database

import (
	"log"

	"github.com/graduation/chat-service/internal/config"
	"github.com/graduation/chat-service/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg *config.Config) *gorm.DB {
	dsn := cfg.DatabaseURL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto Migrate
	err = db.AutoMigrate(
		&models.Conversation{},
		&models.ConversationMember{},
		&models.Message{},
		&models.PinnedMessage{},
		&models.DeviceToken{},
	)
	if err != nil {
		log.Printf("Warning: AutoMigrate failed: %v", err)
	}

	log.Println("Connected to PostgreSQL and ran migrations.")
	return db
}
