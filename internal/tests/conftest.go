// Package tests contains the unit tests for models and schemas, and integration tests for API endpoints of the Book Catalog API.
package tests

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"migrated-app/internal/database"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
	"net/http"
)

// TestDB returns a new test database connection.
func TestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Connect to an in-memory PostgreSQL database for testing.
	db, err := sql.Open("postgres", "user=postgres dbname=test sslmode=disable")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create the books table.
	createQuery := `
	CREATE TABLE IF NOT EXISTS "books" (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		published_year INTEGER NOT NULL,
		summary TEXT
	);
	`
	_, err = db.Exec(createQuery)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Return a cleanup function to drop the table after tests.
	return db, func() {
		dropQuery := `DROP TABLE IF EXISTS "books";`
		_, err := db.Exec(dropQuery)
		if err != nil {
			t.Logf("failed to drop table: %v", err)
		}
		db.Close()
	}
}

// TestClient returns a new TestClient with the dependency injection setup.
func TestClient(t *testing.T) (*http.Client, func()) {
	t.Helper()

	// Initialize the test database.
	db, cleanup := TestDB(t)
	t.Cleanup(cleanup)

	// Override the GetDB function to return the test database.
	database.GetDB = func(ctx context.Context) (*sqlx.DB, error) {
		return sqlx.NewDb(db, "postgres"), nil
	}

	// Create a new TestClient for the app.
	client := &http.Client{}
	return client, func() {
		database.GetDB = database.GetSyncDB // Restore the original GetDB function.
	}
}
