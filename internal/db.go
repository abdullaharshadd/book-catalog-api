package internal

import (
	"os"
	"database/sql"
)

func getDBURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("DB_URL"); v != "" {
		return v
	}
	return "postgres://app:app@db:5432/app?sslmode=disable"
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