// internal/database.go
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
)

var (
	db          *gorm.DB
	asyncDb     *sqlx.DB
	err         error
	migrationUp = "migrations/0001_init.up.sql"
)

func InitializeDatabase() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=books sslmode=disable"
	}

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	asyncDb, err = sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect async database: %v", err)
	}

	CreateSchema(context.Background())
}

func CreateSchema(ctx context.Context) error {
	// Using go:embed to run SQL migrations
	var migrationBytes []byte
	migrationBytes, err = embed.ReadFile(migrationUp)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	_, err = asyncDb.ExecContext(ctx, string(migrationBytes))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

func GetDB() *gorm.DB {
	return db
}

func GetAsyncDB() *sqlx.DB {
	return asyncDb
}

func NewTransaction(ctx context.Context) (*sqlx.Tx, error) {
	tx := asyncDb.MustBegin()
	return tx, nil
}

// MIGRATION_NOTE: The original Python code uses a generator for database sessions.
// In Go, we handle transactions manually and ensure the session is closed after use.
func GetSession(ctx context.Context) (*sqlx.Tx, error) {
	tx, err := NewTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return tx, nil
}

func CloseSession(tx *sqlx.Tx) {
	if tx != nil {
		tx.Rollback()
	}
}

func CommitSession(tx *sqlx.Tx) error {
	return tx.Commit()
}

// Example usage of the database within a function
func ListBooks(ctx context.Context) ([]model.Book, error) {
	var books []model.Book
	if err := db.Find(&books).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch books: %w", err)
	}

	return books, nil
}

func GetBook(ctx context.Context, id uint) (*model.Book, error) {
	var book model.Book
	if err := db.First(&book, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("book not found: %w", err)
		}
		return nil, fmt.Errorf("failed to fetch book: %w", err)
	}

	return &book, nil
}

func CreateBook(ctx context.Context, bookCreate *schemas.BookCreate) (*model.Book, error) {
	if err := bookCreate.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	book := model.NewBook(bookCreate.Title, bookCreate.Author, bookCreate.PublishedYear, bookCreate.Summary)
	result := db.Create(&book)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create book: %w", result.Error)
	}

	return &book, nil
}

func UpdateBook(ctx context.Context, id uint, bookUpdate *schemas.BookUpdate) (*model.Book, error) {
	if err := bookUpdate.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	var book model.Book
	if err := db.First(&book, id).Error; err != nil {
		return nil, fmt.Errorf("failed to find book: %w", err)
	}

	book.Title = bookUpdate.Title
	book.Author = bookUpdate.Author
	book.PublishedYear = bookUpdate.PublishedYear
	book.Summary = bookUpdate.Summary

	result := db.Save(&book)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update book: %w", result.Error)
	}

	return &book, nil
}

func DeleteBook(ctx context.Context, id uint) error {
	var book model.Book
	if err := db.First(&book, id).Error; err != nil {
		return fmt.Errorf("failed to find book: %w", err)
	}

	result := db.Delete(&book)
	if result.Error != nil {
		return fmt.Errorf("failed to delete book: %w", result.Error)
	}

	return nil
}
```

```sql
-- migrations/0001_init.up.sql
CREATE TABLE IF NOT EXISTS "books" (
    "id" SERIAL PRIMARY KEY,
    "title" VARCHAR(255) NOT NULL,
    "author" VARCHAR(255) NOT NULL,
    "published_year" INTEGER NOT NULL,
    "summary" TEXT
);
```

```go
// internal/migrations/migrations.go
package migrations

import (
	"embed"
	"fmt"
	"log"
	"migrated-app/internal/database"
	"os"
	"path/filepath"
)

//go:embed *.sql
var migrations embed.FS

func RunMigrations(ctx context.Context) error {
	migrationFiles, err := migrations.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	for _, f := range migrationFiles {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".up.sql" {
			data, err := migrations.ReadFile(f.Name())
			if err != nil {
				return fmt.Errorf("failed to read migration file %s: %w", f.Name(), err)
			}

			if _, err := database.GetAsyncDB().ExecContext(ctx, string(data)); err != nil {
				return fmt.Errorf("failed to execute migration file %s: %w", f.Name(), err)
			}
		}
	}

	log.Println("Database migrations applied successfully.")
	return nil
}