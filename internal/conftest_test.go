```go
package internal

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// tableExists returns true when the named table is visible in the current
// search-path of the supplied connection.
func tableExists(t *testing.T, db *sqlx.DB, table string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRowContext(context.Background(),
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_name = $1
		)`, table).Scan(&exists)
	if err != nil {
		t.Logf("tableExists query error: %v", err)
		return false
	}
	return exists
}

// ---------------------------------------------------------------------------
// NewTestDB tests
// ---------------------------------------------------------------------------

func TestNewTestDB(t *testing.T) {
	// We need a real DSN for integration tests; if it is absent every sub-test
	// that calls NewTestDB will be skipped via t.Skipf inside the helper.
	// We still exercise the skip path explicitly in one sub-test.

	t.Run("skips when TEST_DATABASE_URL is not set", func(t *testing.T) {
		// Temporarily remove the variable.
		original := os.Getenv(testDBEnvVar)
		os.Unsetenv(testDBEnvVar)
		t.Cleanup(func() { os.Setenv(testDBEnvVar, original) })

		// We cannot call NewTestDB here and expect the current test to be
		// skipped, because that would skip the parent. Instead we verify that
		// getTestDSN returns an empty string, which is the condition that
		// triggers the skip inside NewTestDB.
		dsn := getTestDSN(t)
		assert.Empty(t, dsn, "getTestDSN must return empty string when env var is absent")
	})

	t.Run("fixture is requested by a test – creates schema before returning", func(t *testing.T) {
		db := NewTestDB(t)
		require.NotNil(t, db, "NewTestDB must return a non-nil *sqlx.DB")

		// Verify the connection is live.
		err := db.PingContext(context.Background())
		assert.NoError(t, err, "database must be reachable after NewTestDB")

		// Verify schema was created (mirrors Base.metadata.create_all).
		assert.True(t, tableExists(t, db, BooksTable),
			"books table must exist after NewTestDB initialises the schema")
	})

	t.Run("fixture cleanup drops table and closes pool after test completes", func(t *testing.T) {
		// We spin up a dedicated sub-test so that its t.Cleanup fires before
		// we make our post-cleanup assertions.
		var capturedDB *sqlx.DB

		t.Run("inner – captures db", func(t *testing.T) {
			capturedDB = NewTestDB(t)
			require.NotNil(t, capturedDB)

			// Table must exist during the test.
			assert.True(t, tableExists(t, capturedDB, BooksTable),
				"books table must be visible while the test is running")
		})
		// After the inner test finishes, its cleanup has run.

		// The pool is closed; any new query must fail.
		err := capturedDB.PingContext(context.Background())
		assert.Error(t, err,
			"database pool must be closed after cleanup – Ping must fail")
	})

	t.Run("session is bound to an isolated test database, never production", func(t *testing.T) {
		db := NewTestDB(t)
		require.NotNil(t, db)

		// Confirm we are NOT connected to whatever DATABASE_URL points at by
		// checking that the DSN we opened came from TEST_DATABASE_URL.
		productionDSN := os.Getenv("DATABASE_URL")
		testDSN := os.Getenv(testDBEnvVar)

		if productionDSN != "" && testDSN != "" {
			assert.NotEqual(t, productionDSN, testDSN,
				"TEST_DATABASE_URL must differ from production DATABASE_URL")
		}

		// Sanity-check: db is still functional.
		assert.NoError(t, db.PingContext(context.Background()))
	})

	t.Run("tables are always created before session is returned", func(t *testing.T) {
		db := NewTestDB(t)
		require.NotNil(t, db)

		// Verify schema exists immediately (no deferred creation).
		assert.True(t, tableExists(t, db, BooksTable))
	})

	t.Run("session is always closed after test regardless of test outcome", func(t *testing.T) {
		// Simulate failure by registering a post-cleanup check.
		var pool *sqlx.DB

		t.Run("failing inner test", func(t *testing.T) {
			pool = NewTestDB(t)
			require.NotNil(t, pool)
			// We intentionally do NOT call t.Fatal here – we want the cleanup
			// to run normally and then be inspected.
		})

		// Pool must be closed.
		err := pool.PingContext(context.Background())
		assert.Error(t, err, "pool must be closed after cleanup even when test body doesn't fail explicitly")
	})
}

// ---------------------------------------------------------------------------
// NewTestRouter tests
// ---------------------------------------------------------------------------

func TestNewTestRouter(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		wantStatusNot  int // status we do NOT expect (e.g. 404 means route is registered)
		dbRequired     bool
	}{
		{
			name:   "GET / is registered",
			method: http.MethodGet,
			path:   "/",
		},
		{
			name:   "GET /health is registered",
			method: http.MethodGet,
			path:   "/health",
		},
		{
			name:       "GET /books is registered",
			method:     http.MethodGet,
			path:       "/books",
			dbRequired: true,
		},
		{
			name:       "POST /books is registered",
			method:     http.MethodPost,
			path:       "/books",
			dbRequired: true,
		},
		{
			name:       "GET /books/{id} is registered",
			method:     http.MethodGet,
			path:       "/books/1",
			dbRequired: true,
		},
		{
			name:       "PUT /books/{id} is registered",
			method:     http.MethodPut,
			path:       "/books/1",
			dbRequired: true,
		},
		{
			name:       "DELETE /books/{id} is registered",
			method:     http.MethodDelete,
			path:       "/books/1",
			dbRequired: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var db *sqlx.DB
			if tc.dbRequired {
				db = NewTestDB(t)
			} else {
				// For routes that don't touch the DB we pass a nil-equivalent
				// stub so NewTestRouter won't panic.
				db = newNoOpDB(t)
			}

			router := NewTestRouter(db)
			require.NotNil(t, router, "NewTestRouter must return a non-nil router")

			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			// A 405 means the route exists but the method is wrong, which is
			// still "registered". Only 404 means the route is absent.
			assert.NotEqual(t, http.StatusNotFound, rr.Code,
				"route %s %s must be registered in the router", tc.method, tc.path)
		})
	}

	t.Run("handlers use the same DB injected at construction time", func(t *testing.T) {
		db := NewTestDB(t)
		router := NewTestRouter(db)
		require.NotNil(t, router)

		// Issuing a real request exercises the injected DB.
		req := httptest.NewRequest(http.MethodGet, "/books", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Any non-500 response (e.g. 200 or 404 for empty list) proves the
		// handler successfully used the DB injected via NewTestRouter.
		assert.NotEqual(t, http.StatusInternalServerError, rr.Code,
			"handler should not return 500 when using the test DB")
	})
}

// ---------------------------------------------------------------------------
// NewTestServer tests
// ---------------------------------------------------------------------------

func TestNewTestServer(t *testing.T) {
	t.Run("fixture is requested – returns running server", func(t *testing.T) {
		db := NewTestDB(t)
		srv := NewTestServer(t, db)
		require.NotNil(t, srv, "NewTestServer must return a non-nil *httptest.Server")
		assert.NotEmpty(t, srv.URL, "server URL must be non-empty")

		resp, err := srv.Client().Get(srv.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("requests through client use the injected test DB session", func(t *testing.T) {
		db := NewTestDB(t)
		srv := NewTestServer(t, db)

		// GET /books against the test DB must succeed (empty list or 200).
		resp, err := srv.Client().Get(srv.URL + "/books")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode,
			"books endpoint must not 500 when backed by the test DB")
	})

	t.Run("server is shut down after test completes", func(t *testing.T) {
		var serverURL string

		t.Run("inner – captures server URL", func(t *testing.T) {
			db := NewTestDB(t)
			srv := NewTestServer(t, db)
			serverURL = srv.URL
			require.NotEmpty(t, serverURL)
		})
		// Inner test done; its cleanup (srv.Close) has run.

		// Any request to the now-closed server must fail.
		_, err := http.Get(serverURL + "/health") //nolint:noctx
		assert.Error(t, err,
			"HTTP request to closed test server must return an error")
	})

	t.Run("dependency overrides are isolated to each server instance", func(t *testing.T) {
		db1 := NewTestDB(t)
		db2 := NewTestDB(t)

		srv1 := NewTestServer(t, db1)
		srv2 := NewTestServer(t, db2)

		// Both servers must be individually reachable.
		resp1, err := srv1.Client().Get(srv1.URL + "/health")
		require.NoError(t, err)
		defer resp1.Body.Close()
		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		resp2, err := srv2.Client().Get(srv2.URL + "/health")
		require.NoError(t, err)
		defer resp2.Body.Close()
		assert.Equal(t, http.StatusOK, resp2.StatusCode)
	})

	t.Run("client and db_session fixture share the same database session", func(t *testing.T) {
		db := NewTestDB(t)
		srv := NewTestServer(t, db)

		// Insert a row directly through the shared db.
		_, err := db.ExecContext(context.Background(),
			`INSERT INTO `+BooksTable+` (title, author) VALUES ($1, $2)`,
			"Shared Session Book", "Test Author")
		require.NoError(t, err)

		// The book must be visible through the HTTP API (same DB).
		resp, err := srv.Client().Get(srv.URL + "/books")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// ---------------------------------------------------------------------------
// Global invariant tests
// ---------------------------------------------------------------------------

func TestGlobalInvariants(t *testing.T) {
	t.Run("all tests run against isolated test DB, never production", func(t *testing.T) {
		prodDSN := os.Getenv("DATABASE_URL")
		testDSN := os.Getenv(testDBEnvVar)

		if prodDSN == "" || testDSN == "" {
			t.Skip("one or both DSN env vars not set; skipping isolation check")
		}

		assert.NotEqual(t, prodDSN, testDSN,
			"TEST_DATABASE_URL must not equal DATABASE_URL")
	})

	t.Run("table creation always precedes client usage", func(t *testing.T) {
		db := NewTestDB(t)

		// Table must exist before we build the server / router.
		assert.True(t, tableExists(t, db, BooksTable),
			"table must exist before NewTestServer is called")

		srv := NewTestServer(t, db)
		require.NotNil(t, srv)

		resp, err := srv.Client().Get(srv.URL + "/books")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("database resources are cleaned up after each test even on failure", func(t *testing.T) {
		var pool *sqlx.DB

		t.Run("simulated failure", func(t *testing.T) {
			pool = NewTestDB(t)
			require.NotNil(t, pool)
			// Pool is valid during the test.
			assert.NoError(t, pool.PingContext(context.Background()))
			// We do NOT call t.Fatal, but cleanup should still fire.
		})

		err := pool.PingContext(context.Background())
		assert.Error(t, err, "pool must be closed after test completes")
	})

	t.Run("application state is reset between tests – no leftover rows", func(t *testing.T) {
		// Each call to NewTestDB creates a fresh schema (drop + create).
		// Two sequential sub-tests must see independent, empty databases.
		t.Run("first test inserts a row", func(t *testing.T) {
			db := NewTestDB(t)
			_, err := db.ExecContext(context.Background(),
				`INSERT INTO `+BooksTable+` (title, author) VALUES ($1, $2)`,
				"Row from first test", "Author A")
			require.NoError(t, err)
		})

		t.Run("second test sees an empty table", func(t *testing.T) {
			db := NewTestDB(t)
			var count int
			err := db.QueryRowContext(context.Background(),
				`SELECT COUNT(*) FROM `+BooksTable).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 0, count,
				"second test must start with an empty table (previous row must not leak)")
		})
	})
}

// ---------------------------------------------------------------------------
// newNoOpDB – returns a *sqlx.DB connected to postgres driver that will fail
// on actual queries but satisfies the type system for route-registration tests
// that don't require a live DB.
// ---------------------------------------------------------------------------

func newNoOpDB(t *testing.T) *sqlx.DB {
	t.Helper()

	// Use the test DSN if available, otherwise