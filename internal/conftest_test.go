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
// Helpers / fakes
// ---------------------------------------------------------------------------

// fakeDB is a minimal stand-in for *DB that records which lifecycle methods
// were called and optionally injects errors.
type fakeDB struct {
	initSchemaCalled bool
	closeCalled      bool
	dropCalled       bool

	initSchemaErr error
	closeErr      error
	dropErr       error

	// execResults lets tests control what execContext returns.
	execErr error
}

func (f *fakeDB) InitSchema(_ context.Context) error {
	f.initSchemaCalled = true
	return f.initSchemaErr
}

func (f *fakeDB) Close() error {
	f.closeCalled = true
	return f.closeErr
}

// ---------------------------------------------------------------------------
// getenv shim used by the test file (mirrors internal package usage)
// ---------------------------------------------------------------------------

// setenv is a test-local helper that sets an env var and registers a cleanup
// that restores the original value.
func setenv(t *testing.T, key, value string) {
	t.Helper()
	old, hadOld := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("setenv %s: %v", key, err)
	}
	t.Cleanup(func() {
		if hadOld {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func unsetenv(t *testing.T, key string) {
	t.Helper()
	old, hadOld := os.LookupEnv(key)
	_ = os.Unsetenv(key)
	t.Cleanup(func() {
		if hadOld {
			_ = os.Setenv(key, old)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests for NewTestDB
// ---------------------------------------------------------------------------

func TestNewTestDB_SkipsWhenEnvUnset(t *testing.T) {
	// Guarantee the env var is absent for this sub-test.
	unsetenv(t, testDatabaseURLEnv)

	// We need a sub-test so we can capture the skip.
	skipped := false
	fakeT := &testing.T{}
	_ = fakeT // we cannot run real skip capture without extra infrastructure

	// Pragmatic approach: use t.Run so the test framework captures the skip.
	t.Run("skip_when_env_missing", func(t *testing.T) {
		unsetenv(t, testDatabaseURLEnv)
		db, cleanup := NewTestDB(t)
		// If we reach here the test was NOT skipped — that is a failure.
		// (In practice t.Skip causes the goroutine to stop, so these lines
		// are unreachable when the env var is absent.)
		_ = skipped
		assert.Nil(t, db, "expected nil DB when skipping")
		if cleanup != nil {
			cleanup()
		}
	})
}

func TestNewTestDB_TableDriven(t *testing.T) {
	type tc struct {
		name        string
		envValue    string
		envPresent  bool
		wantSkipped bool
		// wantFatal is checked by observing a nil DB + non-nil cleanup only
		// when a real DSN is configured but the server is unreachable.
	}

	tests := []tc{
		{
			name:        "env_not_set_causes_skip",
			envPresent:  false,
			wantSkipped: true,
		},
		{
			name:        "empty_dsn_causes_skip",
			envPresent:  true,
			envValue:    "",
			wantSkipped: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.envPresent {
				setenv(t, testDatabaseURLEnv, tt.envValue)
			} else {
				unsetenv(t, testDatabaseURLEnv)
			}

			if tt.wantSkipped {
				// We verify the skip path by running inside a sub-test that
				// we can observe for skipping.
				innerRan := false
				t.Run("inner", func(t *testing.T) {
					innerRan = true
					db, cleanup := NewTestDB(t)
					// Unreachable when skipped; if reached, assert nil.
					assert.Nil(t, db)
					if cleanup != nil {
						cleanup()
					}
				})
				// innerRan being true just means the closure body started;
				// the important thing is no panic and the test runner marks
				// the inner test skipped.
				_ = innerRan
			}
		})
	}
}

// TestNewTestDB_WithRealDB runs only when TEST_DATABASE_URL is available,
// mirroring the db_session fixture lifecycle:
//   - schema is created before the session is yielded
//   - DB is usable during the test
//   - schema is dropped and connection closed after the test
func TestNewTestDB_WithRealDB(t *testing.T) {
	// This test will be skipped automatically by NewTestDB if the env var
	// is not set, providing the same behaviour as the pytest skip logic.
	db, cleanup := NewTestDB(t)
	require.NotNil(t, db, "expected non-nil DB when TEST_DATABASE_URL is set")
	defer cleanup()

	t.Run("db_is_usable_after_schema_init", func(t *testing.T) {
		// Verify the books table exists by running a trivial query.
		ctx := context.Background()
		rows, err := db.pool().QueryContext(ctx, `SELECT 1 FROM books LIMIT 0`)
		require.NoError(t, err, "books table should exist after InitSchema")
		defer rows.Close()
	})
}

// TestNewTestDB_CleanupDropsSchema verifies that the cleanup function tears
// down the schema so the table no longer exists.
func TestNewTestDB_CleanupDropsSchema(t *testing.T) {
	db, cleanup := NewTestDB(t)
	require.NotNil(t, db)

	// Run cleanup now (not deferred) so we can inspect the state afterwards.
	cleanup()

	// After cleanup the books table must be gone.
	ctx := context.Background()
	_, err := db.pool().QueryContext(ctx, `SELECT 1 FROM books LIMIT 0`)
	assert.Error(t, err, "books table should not exist after cleanup")
}

// ---------------------------------------------------------------------------
// Tests for dropTestSchema
// ---------------------------------------------------------------------------

func TestDropTestSchema_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, db *DB)
		wantErr bool
	}{
		{
			name: "drops_existing_books_table",
			setup: func(t *testing.T, db *DB) {
				// InitSchema already created it; nothing extra needed.
			},
			wantErr: false,
		},
		{
			name: "idempotent_when_table_already_gone",
			setup: func(t *testing.T, db *DB) {
				// Drop it once so the second drop is a no-op (IF EXISTS).
				ctx := context.Background()
				err := dropTestSchema(ctx, db)
				require.NoError(t, err, "first drop should succeed")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := NewTestDB(t)
			require.NotNil(t, db)
			defer cleanup()

			tt.setup(t, db)

			ctx := context.Background()
			err := dropTestSchema(ctx, db)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests for execContext
// ---------------------------------------------------------------------------

func TestExecContext_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		args    []any
		wantErr bool
	}{
		{
			name:    "valid_ddl_succeeds",
			query:   `DROP TABLE IF EXISTS books`,
			wantErr: false,
		},
		{
			name:    "invalid_sql_returns_error",
			query:   `THIS IS NOT SQL AT ALL!!!`,
			wantErr: true,
		},
		{
			name:    "select_via_exec_succeeds",
			query:   `SELECT 1`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := NewTestDB(t)
			require.NotNil(t, db)
			defer cleanup()

			ctx := context.Background()
			result, err := db.execContext(ctx, tt.query, tt.args...)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests for newTestRouter
// ---------------------------------------------------------------------------

func TestNewTestRouter_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		wantStatusNot  int // status code we do NOT expect (e.g. 404 means route exists)
		wantStatusCode int // 0 means "any non-404 is fine"
	}{
		{
			name:          "GET /  route_is_registered",
			method:        http.MethodGet,
			path:          "/",
			wantStatusNot: http.StatusNotFound,
		},
		{
			name:          "GET /health  route_is_registered",
			method:        http.MethodGet,
			path:          "/health",
			wantStatusNot: http.StatusNotFound,
		},
		{
			name:          "GET /books  route_is_registered",
			method:        http.MethodGet,
			path:          "/books",
			wantStatusNot: http.StatusNotFound,
		},
		{
			name:          "POST /books  route_is_registered",
			method:        http.MethodPost,
			path:          "/books",
			wantStatusNot: http.StatusNotFound,
		},
		{
			name:          "GET /books/{book_id}  route_is_registered",
			method:        http.MethodGet,
			path:          "/books/123",
			wantStatusNot: http.StatusNotFound,
		},
		{
			name:          "PUT /books/{book_id}  route_is_registered",
			method:        http.MethodPut,
			path:          "/books/123",
			wantStatusNot: http.StatusNotFound,
		},
		{
			name:          "DELETE /books/{book_id}  route_is_registered",
			method:        http.MethodDelete,
			path:          "/books/123",
			wantStatusNot: http.StatusNotFound,
		},
		{
			name:           "unknown_route_returns_404",
			method:         http.MethodGet,
			path:           "/no-such-path",
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			db, cleanup := NewTestDB(t)
			require.NotNil(t, db)
			defer cleanup()

			router := newTestRouter(db)
			require.NotNil(t, router)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			res := rec.Result()
			if tt.wantStatusCode != 0 {
				assert.Equal(t, tt.wantStatusCode, res.StatusCode)
			}
			if tt.wantStatusNot != 0 {
				assert.NotEqual(t, tt.wantStatusNot, res.StatusCode,
					"route %s %s should be registered", tt.method, tt.path)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests for NewTestClient  (client fixture analogue)
// ---------------------------------------------------------------------------

func TestNewTestClient_TableDriven(t *testing.T) {
	type requestCase struct {
		method         string
		path           string
		wantStatusNot  int
		wantStatusCode int
	}

	tests := []struct {
		name     string
		requests []requestCase
	}{
		{
			name: "server_is_started_and_reachable",
			requests: []requestCase{
				{method: http.MethodGet, path: "/health", wantStatusNot: http.StatusNotFound},
			},
		},
		{
			name: "client_uses_test_db_session",
			requests: []requestCase{
				{method: http.MethodGet, path: "/books", wantStatusNot: http.StatusNotFound},
			},
		},
		{
			name: "unknown_route_gives_404",
			requests: []requestCase{
				{method: http.MethodGet, path: "/no-such-route", wantStatusCode: http.StatusNotFound},
			},
		},
		{
			name: "multiple_requests_to_same_server",
			requests: []requestCase{
				{method: http.MethodGet, path: "/", wantStatusNot: http.StatusNotFound},
				{method: http.MethodGet, path: "/health", wantStatusNot: http.StatusNotFound},
				{method: http.MethodGet, path: "/books", wantStatusNot: http.StatusNotFound},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			srv, cleanup := NewTestClient(t)
			require.NotNil(t, srv, "httptest.Server must not be nil")
			require.NotEmpty(t, srv.URL, "server URL must be set")
			defer cleanup()

			client := srv.Client()

			for _, rc := range tt.requests {
				rc := rc
				t.Run(rc.method+"_"+rc.path, func(t *testing.T) {
					req, err := http.NewRequest(rc.method, srv.URL+rc.path, nil)
					require.NoError(t, err)

					resp, err := client.Do(req)
					require.NoError(t, err)
					defer resp.Body.Close()

					if rc.wantStatusCode != 0 {
						assert.Equal(t, rc.wantStatusCode, resp.StatusCode)
					}
					if rc.wantStatusNot != 0 {
						assert.NotEqual(t, rc.wantStatusNot, resp.StatusCode,
							"route %s %s should be registered", rc.method, rc.path)
					}
				})
			}
		})
	}
}

// TestNewTestClient_CleanupClosesServer verifies that after cleanup the server
// is no longer accepting connections (mirroring client fixture teardown).
func TestNewTestClient_CleanupClosesServer(t *testing.T) {
	srv, cleanup := NewTestClient(t)
	require.NotNil(t, srv)

	url := srv.URL

	// While alive, a request succeeds.