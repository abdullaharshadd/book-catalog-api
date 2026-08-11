```go
package internal

import (
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

// isPostgresAvailable does a quick ping so tests that need a real DB can be
// skipped when one is not reachable (CI without PostgreSQL, etc.).
func isPostgresAvailable(t *testing.T) bool {
	t.Helper()
	db, err := sqlx.Connect("postgres", testDatabaseURL())
	if err != nil {
		return false
	}
	_ = db.Close()
	return true
}

func skipIfNoPostgres(t *testing.T) {
	t.Helper()
	if !isPostgresAvailable(t) {
		t.Skip("PostgreSQL not available; skipping integration test")
	}
}

// ---------------------------------------------------------------------------
// testDatabaseURL
// ---------------------------------------------------------------------------

func TestTestDatabaseURL(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string
	}{
		{
			name:     "returns default when env var is not set",
			envValue: "",
			want:     defaultTestDatabaseURL,
		},
		{
			name:     "returns env var value when it is set",
			envValue: "postgres://user:pass@remotehost:5432/mydb?sslmode=require",
			want:     "postgres://user:pass@remotehost:5432/mydb?sslmode=require",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Restore the original env value after each sub-test.
			original := os.Getenv("TEST_DATABASE_URL")
			t.Cleanup(func() {
				if original == "" {
					_ = os.Unsetenv("TEST_DATABASE_URL")
				} else {
					_ = os.Setenv("TEST_DATABASE_URL", original)
				}
			})

			if tc.envValue == "" {
				_ = os.Unsetenv("TEST_DATABASE_URL")
			} else {
				_ = os.Setenv("TEST_DATABASE_URL", tc.envValue)
			}

			got := testDatabaseURL()
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// NewTestDB – db_session_fixture analog
// ---------------------------------------------------------------------------

func TestNewTestDB(t *testing.T) {
	skipIfNoPostgres(t)

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "returns a live, pingable DB connection",
			test: func(t *testing.T) {
				db := NewTestDB(t)
				require.NotNil(t, db)
				assert.NoError(t, db.Ping())
			},
		},
		{
			name: "creates the books table before returning",
			test: func(t *testing.T) {
				db := NewTestDB(t)

				// pg_tables is always available; confirms DDL ran.
				var exists bool
				err := db.QueryRow(
					`SELECT EXISTS (
						SELECT 1 FROM information_schema.tables
						WHERE table_schema = 'public' AND table_name = 'books'
					)`,
				).Scan(&exists)
				require.NoError(t, err)
				assert.True(t, exists, "books table should exist after NewTestDB")
			},
		},
		{
			name: "returns an empty books table (resetBooks was called)",
			test: func(t *testing.T) {
				db := NewTestDB(t)

				var count int
				err := db.QueryRow(`SELECT COUNT(*) FROM books`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 0, count, "books table should be empty at the start of each test")
			},
		},
		{
			name: "data written during the test is visible via the same db handle",
			test: func(t *testing.T) {
				db := NewTestDB(t)

				_, err := db.Exec(
					`INSERT INTO books (title, author, published_year)
					 VALUES ($1, $2, $3)`,
					"Test Book", "Test Author", 2024,
				)
				require.NoError(t, err)

				var count int
				err = db.QueryRow(`SELECT COUNT(*) FROM books`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "each call to NewTestDB starts with an empty table (isolation between tests)",
			test: func(t *testing.T) {
				// Insert a row in the first sub-DB.
				db1 := NewTestDB(t)
				_, err := db1.Exec(
					`INSERT INTO books (title, author, published_year) VALUES ($1,$2,$3)`,
					"Leftover Book", "Leftover Author", 2020,
				)
				require.NoError(t, err)

				// A second call must reset so the next test sees an empty table.
				db2 := NewTestDB(t)
				var count int
				err = db2.QueryRow(`SELECT COUNT(*) FROM books`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 0, count, "second NewTestDB call should reset the books table")
			},
		},
		{
			name: "cleanup closes the connection (registered via t.Cleanup)",
			test: func(t *testing.T) {
				var capturedDB *sqlx.DB

				// Run a nested sub-test so its t.Cleanup fires before we check.
				t.Run("inner", func(inner *testing.T) {
					capturedDB = NewTestDB(inner)
					require.NotNil(inner, capturedDB)
					assert.NoError(inner, capturedDB.Ping())
				})

				// After the inner test completes its cleanup, the DB should be closed.
				err := capturedDB.Ping()
				assert.Error(t, err, "db should be closed after t.Cleanup fires")
			},
		},
		{
			name: "connects to the test database, not a production database",
			test: func(t *testing.T) {
				db := NewTestDB(t)

				// The DSN in use should never point at a database whose name
				// contains "prod" or equals the default production name.
				var dbName string
				err := db.QueryRow(`SELECT current_database()`).Scan(&dbName)
				require.NoError(t, err)

				assert.NotContains(t, dbName, "prod",
					"test DB name must not contain 'prod'")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, tc.test)
	}
}

// ---------------------------------------------------------------------------
// resetBooks
// ---------------------------------------------------------------------------

func TestResetBooks(t *testing.T) {
	skipIfNoPostgres(t)

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "truncates all rows",
			test: func(t *testing.T) {
				db := NewTestDB(t)

				// Insert a couple of rows.
				for i := 0; i < 3; i++ {
					_, err := db.Exec(
						`INSERT INTO books (title, author, published_year) VALUES ($1,$2,$3)`,
						"Book", "Author", 2000+i,
					)
					require.NoError(t, err)
				}

				resetBooks(t, db)

				var count int
				err := db.QueryRow(`SELECT COUNT(*) FROM books`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "resets the identity sequence so id starts at 1 again",
			test: func(t *testing.T) {
				db := NewTestDB(t)

				_, err := db.Exec(
					`INSERT INTO books (title, author, published_year) VALUES ($1,$2,$3)`,
					"Book1", "Author1", 2000,
				)
				require.NoError(t, err)

				resetBooks(t, db)

				_, err = db.Exec(
					`INSERT INTO books (title, author, published_year) VALUES ($1,$2,$3)`,
					"Book2", "Author2", 2001,
				)
				require.NoError(t, err)

				var id int
				err = db.QueryRow(`SELECT id FROM books WHERE title = $1`, "Book2").Scan(&id)
				require.NoError(t, err)
				assert.Equal(t, 1, id, "identity sequence should restart at 1 after reset")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, tc.test)
	}
}

// ---------------------------------------------------------------------------
// NewTestServer – client_fixture analog
// ---------------------------------------------------------------------------

func TestNewTestServer(t *testing.T) {
	skipIfNoPostgres(t)

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "returns a non-nil BookServer",
			test: func(t *testing.T) {
				server, db := NewTestServer(t)
				assert.NotNil(t, server)
				assert.NotNil(t, db)
			},
		},
		{
			name: "server shares the same db handle as the returned db",
			test: func(t *testing.T) {
				server, db := NewTestServer(t)

				// The server must be wired to the same DB we got back.
				// We verify by inserting through db and reading through server's handler.
				_, err := db.Exec(
					`INSERT INTO books (title, author, published_year) VALUES ($1,$2,$3)`,
					"Shared Book", "Shared Author", 2024,
				)
				require.NoError(t, err)

				// The server's handler should be able to see the row.
				// We use httptest.NewRecorder to exercise the HTTP layer.
				req := httptest.NewRequest(http.MethodGet, "/books", nil)
				rr := httptest.NewRecorder()
				server.ServeHTTP(rr, req)

				// A 200 response confirms the handler reached the DB and
				// didn't get a "no connection" error.
				assert.Equal(t, http.StatusOK, rr.Code,
					"GET /books should return 200 when the server is wired to a live DB")
			},
		},
		{
			name: "server's HTTP handler is usable with httptest.NewServer",
			test: func(t *testing.T) {
				server, _ := NewTestServer(t)

				ts := httptest.NewServer(server)
				t.Cleanup(ts.Close)

				resp, err := http.Get(ts.URL + "/books")
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)
			},
		},
		{
			name: "server sees an empty books table at the start of each test",
			test: func(t *testing.T) {
				_, db := NewTestServer(t)

				var count int
				err := db.QueryRow(`SELECT COUNT(*) FROM books`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 0, count)
			},
		},
		{
			name: "data inserted via db is visible to subsequent requests through the server",
			test: func(t *testing.T) {
				server, db := NewTestServer(t)

				_, err := db.Exec(
					`INSERT INTO books (title, author, published_year) VALUES ($1,$2,$3)`,
					"Visible Book", "Some Author", 2022,
				)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodGet, "/books", nil)
				rr := httptest.NewRecorder()
				server.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusOK, rr.Code)
				// The response body contains the book we just inserted.
				assert.Contains(t, rr.Body.String(), "Visible Book")
			},
		},
		{
			name: "db is closed via t.Cleanup after the test finishes",
			test: func(t *testing.T) {
				var capturedDB *sqlx.DB

				t.Run("inner server test", func(inner *testing.T) {
					_, capturedDB = NewTestServer(inner)
					assert.NoError(inner, capturedDB.Ping())
				})

				// After the inner test's cleanup runs the DB must be closed.
				err := capturedDB.Ping()
				assert.Error(t, err, "db should be closed after the test's t.Cleanup fires")
			},
		},
		{
			name: "two independent NewTestServer calls get isolated tables",
			test: func(t *testing.T) {
				_, db1 := NewTestServer(t)
				_, err := db1.Exec(
					`INSERT INTO books (title, author, published_year) VALUES ($1,$2,$3)`,
					"IsolatedBook", "IsolatedAuthor", 2023,
				)
				require.NoError(t, err)

				// Second server/db must start clean.
				_, db2 := NewTestServer(t)
				var count int
				err = db2.QueryRow(`SELECT COUNT(*) FROM books`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 0, count,
					"second NewTestServer should start with an empty table")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, tc.test)
	}
}

// ---------------------------------------------------------------------------
// createBooksTableDDL – content sanity checks (no DB needed)
// ---------------------------------------------------------------------------

func TestCreateBooksTableDDL(t *testing.T) {
	tests := []struct {
		name     string
		contains string
	}{
		{"contains CREATE TABLE IF NOT EXISTS", "CREATE TABLE IF NOT EXISTS books"},
		{"defines id column", "id"},
		{"defines title column", "title"},
		{"defines author column", "author"},
		{"defines published_year column", "published_year"},
		{"defines created_at column", "created_at"},
		{"uses IDENTITY for primary key", "IDENTITY"},
		{"uses TIMESTAMPTZ for created_at", "TIMESTAMPTZ"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert.Contains(t, createBooksTableDDL, tc.contains)
		})
	}
}

// ---------------------------------------------------------------------------
// Idempotency of schema creation
// ---------------------------------------------------------------------------

func TestNewTestDB_SchemaIdempotency(t *testing.T) {
	skipIfNoPostgres(t)

	t.Run("calling NewTestDB twice does not fail (IF NOT EXISTS is idempotent)", func(t *testing.T) {
		db := NewTestDB(t)
		require.NotNil(t, db)

		// Manually re-run the DDL on the already-open connection; it must not error.
		_, err := db.Exec(createBooksTableDDL)
		assert.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// defaultTestDatabaseURL constant sanity check
// ---------------------------------------------------------------------------

func TestDefaultTestDatabaseURL(t *testing.T) {
	tests := []struct {
		name     string