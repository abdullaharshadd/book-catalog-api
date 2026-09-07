// internal/model.go
package internal

import (
	"errors"
	"fmt"
	"migrated-app/internal/config"
	"migrated-app/pkg/database"
)

// MIGRATION_NOTE: The UniqueConstraint from SQLAlchemy is handled by the database schema directly.
// The __repr__ and __str__ methods are not directly translatable to Go.
// Instead, we provide a String method for similar functionality.

// Book represents a book in the catalog.
type Book struct {
	ID            int    `gorm:"primaryKey;autoIncrement;column:id"`
	Title         string `gorm:"not null;type:varchar(255);column:title"`
	Author        string `gorm:"not null;type:varchar(255);column:author"`
	PublishedYear int    `gorm:"not null;column:published_year"`
	Summary       string `gorm:"column:summary"`
}

// TableName returns the table name for the Book model.
func (b *Book) TableName() string {
	return "books"
}

// String provides a string representation of the Book struct.
func (b *Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// CreateBooksTable creates the books table if it does not exist.
func CreateBooksTable(db *database.DB) error {
	const createTableQuery = `
	CREATE TABLE IF NOT EXISTS books (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		author VARCHAR(255) NOT NULL,
		published_year INTEGER NOT NULL,
		summary TEXT,
		UNIQUE (title, author)
	);
	`
	result := db.Exec(createTableQuery)
	if result.Error != nil {
		return fmt.Errorf("failed to create books table: %w", result.Error)
	}
	return nil
}

// InitModel initializes the model by creating necessary tables.

// NewBook creates a new book instance.
func NewBook(title, author string, publishedYear int, summary string) (*Book, error) {
	if title == "" || author == "" || publishedYear <= 0 {
		return nil, errors.New("title, author, and publishedYear are required")
	}
	return &Book{
		Title:         title,
		Author:        author,
		PublishedYear: publishedYear,
		Summary:       summary,
	}, nil
}

// RequiresManualReview: Ensure the database connection is properly initialized in the main application.
// RequiresManualReview: Ensure that the database schema is correctly applied and managed.
```

```go
// pkg/database/db.go
package database

import (
	"database/sql"
	"fmt"
	"migrated-app/internal/config"
	"os"
	"time"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB represents the database connection.
type DB struct {
	gorm.DB
}

// NewDB creates a new database connection.
func NewDB(cfg *config.Config) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set maximum open connections
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)

	// Set maximum idle connections
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	// Set the maximum lifetime for a connection
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &DB{db}, nil
}