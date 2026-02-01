package database

import (
	"fmt"
	"log"
	"time"

	"github.com/graduation/chat-service/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewPostgresConnection creates a new PostgreSQL connection using GORM
func NewPostgresConnection(databaseURL string) (*gorm.DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	// Configure GORM logger
	gormLogger := logger.New(
		log.Default(),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Open database connection
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger:                 gormLogger,
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB and configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL database")
	return db, nil
}

// AutoMigrate runs GORM auto-migration to create/update database tables
func AutoMigrate(db *gorm.DB) error {
	log.Println("Running database auto-migration...")

	err := db.AutoMigrate(
		&models.Conversation{},
		&models.ConversationMember{},
		&models.Message{},
		&models.PinnedMessage{},
		&models.DeviceToken{},
	)
	if err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	log.Println("Database auto-migration completed successfully")
	return nil
}
