package internal

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
	"net/http"
	"os"
	"testing"
)

// TestSuite represents the test suite for the Book Catalog API.
// It contains unit tests for models and schemas, plus integration tests for API endpoints.
func TestSuite(t *testing.T) {
	// Initialize API and database for testing
	api := NewBookCatalogAPI()
	db, err := NewDatabase()
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Unit tests for models and schemas
	testModels(t, db)
	testSchemas(t, db)

	// Integration tests for API endpoints
	testAPIEndpoints(t, api)
}

// testModels runs the unit tests for the models.
func testModels(t *testing.T, db *sqlx.DB) {
	// Placeholder for model tests
	t.Log("Running model tests...")
}

// testSchemas runs the unit tests for the schemas.
func testSchemas(t *testing.T, db *sqlx.DB) {
	// Placeholder for schema tests
	t.Log("Running schema tests...")
}

// testAPIEndpoints runs the integration tests for the API endpoints.
func testAPIEndpoints(t *testing.T, api BookCatalogAPI) {
	// Initialize the HTTP client for testing
	client := http.Client{}
	
	// Ensure the database is in a clean state before starting tests
	if err := InitDB(api.GetDB()); err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := DropAllTables(api.GetDB()); err != nil {
			log.Error().Err(err).Msg("Failed to drop tables after tests")
		}
	}()
	
	// Integration tests go here
	t.Log("Running API endpoint tests...")
}

// DropAllTables drops all tables in the database.
func DropAllTables(db *sqlx.DB) error {
	// Manually drop all tables since we are not using an ORM framework
	queries := []string{
		`DROP TABLE IF EXISTS books;`,
	}
	
	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return fmt.Errorf("failed to drop table: %w", err)
		}
	}
	
	return nil
}
