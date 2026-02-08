package postgres

import (
	"context"
	"fmt"

	"github.com/OmarrGhorab/courses-attendance-service/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database wraps the GORM DB connection
type Database struct {
	DB *gorm.DB
}

// NewDatabase creates a new database connection
func NewDatabase(cfg config.DatabaseConfig) (*Database, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return &Database{DB: db}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// WithContext returns a DB instance with context
func (d *Database) WithContext(ctx context.Context) *gorm.DB {
	return d.DB.WithContext(ctx)
}

// TransactionManager handles database transactions
type TransactionManager struct {
	db *Database
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *Database) *TransactionManager {
	return &TransactionManager{db: db}
}

// WithTransaction executes a function within a transaction
func (tm *TransactionManager) WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return tm.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}
