// Package tests contains the unit tests for models and schemas, and integration tests for API endpoints of the Book Catalog API.
package tests

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
	"migrated-app/internal/database"
	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
)

var testDBScenarios = []struct {
	name           string
	setup          func(*testing.T) (*sql.DB, func())
	expectedOutput bool
}{
	{
		name: "test_db_connection",
		setup: func(t *testing.T) (*sql.DB, func()) {
			db, cleanup := TestDB(t)
			return db, cleanup
		},
		expectedOutput: true,
	},
}

func TestDB(t *testing.T) {
	for _, scenario := range testDBScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			db, cleanup := scenario.setup(t)
			defer cleanup()

			assert.NotNil(t, db, "database should not be nil")

			// Check if the table exists
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'books'").Scan(&count)
			assert.NoError(t, err, "querying table existence should not return an error")
			assert.Equal(t, 1, count, "table 'books' should exist in the database")
		})
	}
}

var testClientScenarios = []struct {
	name           string
	setup          func(*testing.T) (*http.Client, func())
	expectedOutput bool
}{
	{
		name: "test_client_with_db_override",
		setup: func(t *testing.T) (*http.Client, func()) {
			client, cleanup := TestClient(t)
			return client, cleanup
		},
		expectedOutput: true,
	},
}

func TestClient(t *testing.T) {
	for _, scenario := range testClientScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			client, cleanup := scenario.setup(t)
			defer cleanup()

			assert.NotNil(t, client, "HTTP client should not be nil")

			// Mock a handler to test the client
			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			server := httptest.NewServer(mockHandler)
			defer server.Close()

			resp, err := client.Get(server.URL)
			assert.NoError(t, err, "GET request should not return an error")
			assert.Equal(t, http.StatusOK, resp.StatusCode, "response status should be OK")
		})
	}
}

func TestGetDBOverride(t *testing.T) {
	scenarios := []struct {
		name           string
		setup          func(*testing.T)
		expectedOutput bool
	}{
		{
			name: "override_get_db",
			setup: func(t *testing.T) {
				_, cleanup := TestClient(t)
				defer cleanup()
			},
			expectedOutput: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			scenario.setup(t)

			ctx := context.Background()
			db, err := database.GetDB(ctx)
			assert.NoError(t, err, "getting DB should not return an error")
			assert.NotNil(t, db, "DB should not be nil")
		})
	}
}

func TestRestoreOriginalGetDB(t *testing.T) {
	_, cleanup := TestClient(t)
	defer cleanup()

	// Ensure the original GetDB is restored
	ctx := context.Background()
	db, err := database.GetDB(ctx)
	assert.NoError(t, err, "getting DB should not return an error")
	assert.NotNil(t, db, "DB should not be nil")
}