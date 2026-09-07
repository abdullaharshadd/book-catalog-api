package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"migrated-app/internal/model"
)

var (
	db          *gorm.DB
	asyncDb     *gorm.DB
	err         error
)

// InitializeDatabase initializes the database connection and schema.
func InitializeDatabase(ctx context.Context) error {
	// Database configuration
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgresql://user:password@localhost/dbname?sslmode=disable"
	}
	asyncDatabaseURL := os.Getenv("ASYNC_DATABASE_URL")
	if asyncDatabaseURL == "" {
		asyncDatabaseURL = "postgresql://user:password@localhost/dbname?sslmode=disable"
	}

	// Create sync engine for sync operations (like creating tables)
	db, err = gorm.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Create async engine for async operations
	asyncDb, err = gorm.Open("postgres", asyncDatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to open async database connection: %w", err)
	}

	// Use sync engine to create tables
	if err := db.AutoMigrate(&model.Book{).Error; err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return nil
}

// GetDb returns the synchronous database session.
func GetDb(ctx context.Context) (*gorm.DB, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}
	return db, nil
}

// GetAsyncDb returns the asynchronous database session.
func GetAsyncDb(ctx context.Context) (*gorm.DB, error) {
	if asyncDb == nil {
		return nil, errors.New("async database not initialized")
	}
	return asyncDb, nil
}