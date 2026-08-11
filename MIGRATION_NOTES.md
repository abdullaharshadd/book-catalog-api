package internal

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func getDBURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("DB_URL"); v != "" {
		return v
	}
	return "postgres://app:app@db:5432/app?sslmode=disable"
}

// InitDB opens a connection to the database, retries until it is ready,
// and then ensures the books table exists (CREATE TABLE IF NOT EXISTS).
func InitDB() (*sql.DB, error) {
	dsn := getDBURL()

	var db *sql.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			break
		}
		log.Printf("Database not ready (attempt %d/10): %v — retrying in 2s", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("schema migration failed: %w", err)
	}

	DB = db
	return db, nil
}

// migrate creates the books table if it does not already exist.
// Column definitions are derived from the Book model (internal/model.go).
func migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS books (
    id             SERIAL PRIMARY KEY,
    title          VARCHAR(255) NOT NULL,
    author         VARCHAR(255) NOT NULL,
    isbn           VARCHAR(13)  UNIQUE,
    published_year INTEGER,
    description    TEXT,
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
`
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create books table: %w", err)
	}
	log.Println("Schema migration complete: books table is ready")
	return nil
}