package internal

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// BooksTable is the name of the database table backing the Book model.
const BooksTable = "books"

// UniqueTitleAuthorConstraint is the name of the PostgreSQL unique constraint
// covering (title, author).
const UniqueTitleAuthorConstraint = "unique_title_author"

// Book represents a single book in the catalog. It maps to the "books" table.
type Book struct {
	ID            int64          `db:"id" json:"id"`
	Title         string         `db:"title" json:"title"`
	Author        string         `db:"author" json:"author"`
	PublishedYear int            `db:"published_year" json:"published_year"`
	Summary       sql.NullString `db:"summary" json:"-"`
}

// String returns a human-readable representation of the book.
func (b Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// GoString returns a debug representation of the book.
func (b Book) GoString() string {
	return fmt.Sprintf("<Book(id=%d, title=%q, author=%q, year=%d)>",
		b.ID, b.Title, b.Author, b.PublishedYear)
}

// DB is the shared database connection pool.
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
			if err == nil {
				break
			}
		}
		log.Printf("Database not ready, retrying in 2s... (%d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("could not connect to database after retries: %w", err)
	}

	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	DB = db
	return db, nil
}

// createSchema ensures the books table and its indexes/constraints exist.
func createSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS books (
			id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			title         VARCHAR(255) NOT NULL,
			author        VARCHAR(255) NOT NULL,
			published_year INTEGER NOT NULL,
			summary        TEXT,
			CONSTRAINT unique_title_author UNIQUE (title, author)
		);
		CREATE INDEX IF NOT EXISTS ix_books_id     ON books (id);
		CREATE INDEX IF NOT EXISTS ix_books_title  ON books (title);
		CREATE INDEX IF NOT EXISTS ix_books_author ON books (author);
	`)
	return err
}