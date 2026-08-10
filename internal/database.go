package internal

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// DB wraps sqlx.DB to provide a named type for dependency injection.
type DB struct {
	*sqlx.DB
}

// getenv returns the value of the environment variable named by key, or
// fallback if the variable is unset or empty.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// OpenDB opens a PostgreSQL database connection using the DATABASE_URL
// environment variable, falling back to the known-good default for this
// deployment environment.
func OpenDB() (*DB, error) {
	dsn := getenv("DATABASE_URL", "postgres://app:app@db:5432/app?sslmode=disable")

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Retry ping to allow the database container to finish starting.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		if err := db.PingContext(ctx); err == nil {
			break
		} else {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("ping database: %w", err)
			case <-time.After(500 * time.Millisecond):
			}
		}
	}

	return &DB{db}, nil
}

// InitSchema creates the books table if it does not already exist.
func InitSchema(db *DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS books (
    id             BIGSERIAL PRIMARY KEY,
    title          TEXT NOT NULL,
    author         TEXT NOT NULL,
    published_year INTEGER,
    summary        TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT books_title_author_uniq UNIQUE (title, author)
);`
	_, err := db.Exec(ddl)
	return err
}