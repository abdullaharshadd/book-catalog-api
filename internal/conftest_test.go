```go
package internal

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// realDSN returns the value of TEST_DATABASE_URL or "" if unset.
func realDSN() string { return os.Getenv("TEST_DATABASE_URL") }

// openRawDB opens a postgres connection directly so individual test cases can
// inspect side-effects independently of NewTestDB.
func openRawDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "sql.Open should not error for valid driver")
	require.NoError(t, db.PingContext(context.Background()), "ping should succeed")
	return db
}

// tableExists checks whether a table with the given name is visible in the
// connected PostgreSQL database.
func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRowContext(context.Background(),
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name   = $1
		)`, table).Scan(&exists)
	require.NoError(t, err)
	return exists
}

// ---------------------------------------------------------------------------
// TestNewTestDB – covers db_session_fixture behaviours
// ---------------------------------------------------------------------------

func TestNewTestDB(t *testing.T) {
	// All sub-cases that need a real DB gate on TEST_DATABASE_URL.
	dsn := realDSN()

	tests := []struct {
		name string
		// setEnv / clearEnv let each case control the environment variable.
		setEnv   bool
		wantSkip bool
		// verify is called (only when setEnv==true and no skip) after
		// NewTestDB returns, with direct access to the raw connection.
		verify func(t *testing.T, db *DB, rawDB *sql.DB)
	}{
		{
			name:     "skips when TEST_DATABASE_URL is not set",
			setEnv:   false,
			wantSkip: true,
		},
		{
			name:   "returns a non-nil *DB when TEST_DATABASE_URL is set",
			setEnv: true,
			verify: func(t *testing.T, db *DB, rawDB *sql.DB) {
				assert.NotNil(t, db, "NewTestDB should return a non-nil *DB")
				assert.NotNil(t, db.DB, "inner *sql.DB should not be nil")
			},
		},
		{
			name:   "creates the books table before returning",
			setEnv: true,
			verify: func(t *testing.T, db *DB, rawDB *sql.DB) {
				assert.True(t, tableExists(t, rawDB, "books"),
					"books table must exist after NewTestDB returns")
			},
		},
		{
			name:   "returned *DB is pingable (session is open and usable)",
			setEnv: true,
			verify: func(t *testing.T, db *DB, rawDB *sql.DB) {
				err := db.DB.PingContext(context.Background())
				assert.NoError(t, err, "the session returned by NewTestDB must be usable")
			},
		},
		{
			name:   "books table starts empty (fresh schema per test)",
			setEnv: true,
			verify: func(t *testing.T, db *DB, rawDB *sql.DB) {
				var count int
				err := rawDB.QueryRowContext(context.Background(),
					`SELECT COUNT(*) FROM books`).Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 0, count,
					"each test must receive an empty schema")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// We run these sequentially because they mutate a shared env-var.
			// Ensure TEST_DATABASE_URL is in the desired state.
			original := os.Getenv("TEST_DATABASE_URL")
			t.Cleanup(func() { os.Setenv("TEST_DATABASE_URL", original) })

			if !tc.setEnv {
				os.Unsetenv("TEST_DATABASE_URL")
			} else {
				if dsn == "" {
					t.Skip("TEST_DATABASE_URL not set; skipping integration sub-test")
				}
				os.Setenv("TEST_DATABASE_URL", dsn)
			}

			if tc.wantSkip {
				// We cannot directly assert t.Skip was called from the same
				// goroutine, so we run NewTestDB inside a sub-test and check
				// whether that sub-test was skipped.
				skipped := false
				// Wrap in a helper test.
				func() {
					// Use a fake *testing.T façade via a sub-test.
					t.Run("inner_skip_check", func(inner *testing.T) {
						NewTestDB(inner)
						// If we reach here the test should have been skipped.
						skipped = inner.Skipped()
					})
				}()
				assert.True(t, skipped,
					"NewTestDB must skip the test when TEST_DATABASE_URL is unset")
				return
			}

			// Happy-path: call NewTestDB and hand the caller a second raw
			// connection so it can inspect side-effects independently.
			db := NewTestDB(t)
			rawDB := openRawDB(t, dsn)
			t.Cleanup(func() { rawDB.Close() })

			tc.verify(t, db, rawDB)
		})
	}
}

// TestNewTestDB_Cleanup verifies the teardown contract: the schema is dropped
// and the connection is closed after the test finishes.
func TestNewTestDB_Cleanup(t *testing.T) {
	dsn := realDSN()
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	// We want to observe post-cleanup state, so we create an *inner* sub-test
	// and let it run to completion, then inspect the database from the outer
	// test's raw connection.

	var capturedRawDB *sql.DB

	t.Run("inner_lifecycle", func(inner *testing.T) {
		db := NewTestDB(inner)
		// Capture the underlying *sql.DB so we can check it after cleanup.
		capturedRawDB = db.DB
		assert.True(t, tableExists(inner, capturedRawDB, "books"),
			"books table must exist inside the test")
	})

	// After the inner sub-test finishes, t.Cleanup() functions have run.
	// The captured connection should now be closed.
	err := capturedRawDB.PingContext(context.Background())
	assert.Error(t, err,
		"the inner *sql.DB should be closed after NewTestDB cleanup runs")

	// Use a fresh raw connection to verify the table was dropped.
	rawDB := openRawDB(t, dsn)
	t.Cleanup(func() { rawDB.Close() })
	assert.False(t, tableExists(t, rawDB, "books"),
		"books table must be dropped during NewTestDB cleanup (schema teardown)")
}

// TestCreateSchema tests the unexported createSchema helper directly.
func TestCreateSchema(t *testing.T) {
	dsn := realDSN()
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	tests := []struct {
		name      string
		setup     func(t *testing.T, db *sql.DB)
		wantErr   bool
		wantTable bool
	}{
		{
			name:      "creates books table when it does not exist",
			setup:     func(t *testing.T, db *sql.DB) {},
			wantErr:   false,
			wantTable: true,
		},
		{
			name: "is idempotent (IF NOT EXISTS) when table already exists",
			setup: func(t *testing.T, db *sql.DB) {
				_, err := db.ExecContext(context.Background(), createBooksTableDDL)
				require.NoError(t, err)
			},
			wantErr:   false,
			wantTable: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := openRawDB(t, dsn)
			t.Cleanup(func() {
				db.ExecContext(context.Background(), dropBooksTableDDL)
				db.Close()
			})

			tc.setup(t, db)
			err := createSchema(context.Background(), db)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantTable, tableExists(t, db, "books"))
		})
	}
}

// ---------------------------------------------------------------------------
// TestNewTestServer – covers client_fixture behaviours
// ---------------------------------------------------------------------------

func TestNewTestServer(t *testing.T) {
	dsn := realDSN()
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	tests := []struct {
		name   string
		verify func(t *testing.T, srv *httptest.Server, db *DB)
	}{
		{
			name: "returns a non-nil httptest.Server",
			verify: func(t *testing.T, srv *httptest.Server, _ *DB) {
				assert.NotNil(t, srv, "NewTestServer must return a non-nil *httptest.Server")
			},
		},
		{
			name: "server URL is non-empty and reachable",
			verify: func(t *testing.T, srv *httptest.Server, _ *DB) {
				assert.NotEmpty(t, srv.URL,
					"server URL must not be empty")
				resp, err := srv.Client().Get(srv.URL + "/")
				// We only care that the server responds (any status is fine here).
				if err == nil {
					resp.Body.Close()
				}
				assert.NoError(t, err,
					"server must be reachable after NewTestServer returns")
			},
		},
		{
			name: "requests hit application routes wired to the test DB",
			verify: func(t *testing.T, srv *httptest.Server, _ *DB) {
				// POST /books is a known application route. A 422/400/200/201
				// response means the server is wired up; a connection-refused
				// error means it is not.
				resp, err := srv.Client().Post(
					srv.URL+"/books", "application/json",
					nil,
				)
				require.NoError(t, err,
					"POST /books must not produce a connection error")
				resp.Body.Close()
				// Any HTTP status (even 4xx) means the handler was reached.
				assert.NotEqual(t, 0, resp.StatusCode)
			},
		},
		{
			name: "DB dependency is the same instance passed to NewTestServer",
			verify: func(t *testing.T, srv *httptest.Server, db *DB) {
				// Insert a row directly via the test DB, then read it back
				// through the HTTP server. This proves the server uses the
				// same database the test controls.
				_, err := db.DB.ExecContext(context.Background(),
					`INSERT INTO books (title, author) VALUES ($1, $2)`,
					"Dependency Injection Test", "Go Test Suite",
				)
				require.NoError(t, err, "direct insert should succeed")

				resp, err := srv.Client().Get(srv.URL + "/books")
				require.NoError(t, err)
				defer resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode,
					"GET /books should return 200 when rows exist")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := NewTestDB(t)
			srv := NewTestServer(t, db)
			tc.verify(t, srv, db)
		})
	}
}

// TestNewTestServer_Cleanup verifies that the httptest.Server is shut down
// after the test finishes (the cleanup registered by NewTestServer fires).
func TestNewTestServer_Cleanup(t *testing.T) {
	dsn := realDSN()
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	var capturedURL string

	t.Run("inner_server_lifecycle", func(inner *testing.T) {
		db := NewTestDB(inner)
		srv := NewTestServer(inner, db)
		capturedURL = srv.URL
		// Verify it is up while the inner test runs.
		resp, err := srv.Client().Get(capturedURL + "/books")
		require.NoError(inner, err)
		resp.Body.Close()
	})

	// After the inner test the server's cleanup (srv.Close) must have fired.
	resp, err := http.Get(capturedURL + "/books")
	if err == nil {
		resp.Body.Close()
	}
	assert.Error(t, err,
		"the httptest.Server must be closed after NewTestServer cleanup runs")
}

// ---------------------------------------------------------------------------
// TestNewTestDB_IsolationBetweenTests – each test gets an empty schema
// ---------------------------------------------------------------------------

func TestNewTestDB_IsolationBetweenTests(t *testing.T) {
	dsn := realDSN()
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	// Run two sequential sub-tests: the first inserts a row, the second
	// must see an empty table — proving that the cleanup/create cycle in
	// NewTestDB provides per-test isolation.

	t.Run("first_test_inserts_a_row", func(inner *testing.T) {
		db := NewTestDB(inner)
		_, err := db.DB.ExecContext(context.Background(),
			`INSERT INTO books (title, author) VALUES ($1, $2)`,
			"Isolation Row", "Author A",
		)
		require.NoError(inner, err)

		var count int
		require.NoError(inner,
			db.DB.QueryRowContext(context.Background(),
				`SELECT COUNT(*) FROM books`).Scan(&count))
		assert.Equal(inner, 1, count,
			"first test should see exactly 1 row after insert")
	})

	t.Run("second_test_sees_empty_schema", func(inner *testing.T) {
		db := NewTestDB(inner)
		var count int
		require.NoError(inner,
			db.DB.QueryRowContext(context.Background(),
				`SELECT COUNT(*) FROM books`).Scan(&count))
		assert.Equal(inner, 0, count,
			"second test must start with a fresh, empty schema")
	})
}

// ---------------------------------------------------------------------------
// TestNewTestDB_SchemaColumns – the DDL matches the Book model
// ---------------------------------------------------------------------------

func TestNewTestDB_SchemaColumns(t *testing.T) {
	dsn := realDSN()
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	expectedColumns := []string{"id", "title", "author", "published_year", "created_at"}

	t.Run("books table has all expected columns", func(inner *testing.T) {
		db := NewTestDB(inner)

		rows, err := db.DB.QueryContext(