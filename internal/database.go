package internal

import (
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const createSchema = `
CREATE TABLE IF NOT EXISTS books (
	id             SERIAL PRIMARY KEY,
	title          VARCHAR(255) NOT NULL,
	author         VARCHAR(255) NOT NULL,
	published_year INTEGER NOT NULL,
	summary        TEXT,
	CONSTRAINT uq_books_title_author UNIQUE (title, author)
);
CREATE INDEX IF NOT EXISTS ix_books_title ON books (title);
CREATE INDEX IF NOT EXISTS ix_books_author ON books (author);
CREATE INDEX IF NOT EXISTS ix_books_published_year ON books (published_year);
`

func ConnectDB() *sqlx.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DB_URL")
	}
	if dsn == "" {
		dsn = "postgres://app:app@migrator-sandbox-db:5432/app?sslmode=disable"
	}

	var db *sqlx.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = sqlx.Connect("postgres", dsn)
		if err == nil {
			break
		}
		log.Printf("Database connection attempt %d failed: %v; retrying in 2s...", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if _, err := db.Exec(createSchema); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	log.Println("Database connected and schema initialized")
	return db
}