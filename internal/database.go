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

// OpenDB opens and verifies a PostgreSQL connection using the DATABASE_URL
// environment variable, falling back to the canonical connection string for
// the sibling "db" container.
func OpenDB() (*DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://app:app@db:5432/app?sslmode=disable"
	}

	sqlxDB, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlxDB.SetMaxOpenConns(25)
	sqlxDB.SetMaxIdleConns(5)
	sqlxDB.SetConnMaxLifetime(5 * time.Minute)

	// Retry ping to allow the sibling container to finish starting.
	const maxAttempts = 10
	for i := 1; i <= maxAttempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = sqlxDB.PingContext(ctx)
		cancel()
		if err == nil {
			break
		}
		if i == maxAttempts {
			_ = sqlxDB.Close()
			return nil, fmt.Errorf("ping database: %w", err)
		}
		time.Sleep(time.Duration(i) * time.Second)
	}

	return &DB{sqlxDB}, nil
}

// InitSchema creates the books table if it does not already exist.
func (db *DB) InitSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS books (
    id             BIGSERIAL PRIMARY KEY,
    title          TEXT        NOT NULL,
    author         TEXT        NOT NULL,
    published_year INT,
    summary        TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (title, author)
);`
	_, err := db.ExecContext(ctx, ddl)
	return err
}