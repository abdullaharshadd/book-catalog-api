package internal

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/lib/pq"
)

// MIGRATION_NOTE: The Python source was a pytest conftest.py providing two
// fixtures:
//
//   - db_session: created the schema, yielded a SQLAlchemy Session, then
//     closed it and dropped the schema afterwards.
//   - client: overrode FastAPI's get_db dependency so the app used the test
//     session, then yielded a Starlette TestClient.
//
// Go has no pytest fixture mechanism. The idiomatic equivalent is a set of
// per-test helper functions that return the constructed resource plus a
// cleanup func (or use t.Cleanup). These helpers live in the non-test file so
// they can be shared across *_test.go files in this package.
//
// Two dialect-level changes from the source:
//
//   - The source used a file/in-memory SQLite database. The target database is
//     PostgreSQL (see the migration brief), so the test harness connects to a
//     Postgres instance via the TEST_DATABASE_URL environment variable. There
//     is no drop-in in-memory Postgres; tests are expected to run against a
//     disposable Postgres (e.g. a docker container / testcontainers). If the
//     env var is unset, NewTestDB skips the test rather than failing hard.
//   - FastAPI's dependency_overrides is unnecessary: the *DB dependency is
//     injected explicitly via NewHandlers, so the test simply wires the test
//     DB into a router.

// testDatabaseURLEnv is the environment variable holding the DSN of a
// disposable PostgreSQL database used by the integration tests.
const testDatabaseURLEnv = "TEST_DATABASE_URL"

// NewTestDB opens a connection to the test PostgreSQL database, applies the
// schema, and returns a ready-to-use *DB together with a cleanup function that
// tears the schema back down and closes the pool.
//
// If the TEST_DATABASE_URL environment variable is not set, the current test
// is skipped — mirroring the source's use of a throwaway database without
// hard-failing developers who have not provisioned Postgres locally.
//
// MIGRATION_NOTE: The source's Base.metadata.create_all / drop_all lifecycle
// is reproduced here: InitSchema builds the tables up front and dropTestSchema
// removes them during cleanup so each test run starts from a clean slate.
func NewTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	dsn := getenv(testDatabaseURLEnv)
	if dsn == "" {
		t.Skipf("%s not set; skipping database-backed test", testDatabaseURLEnv)
		return nil, func() {}
	}

	ctx := context.Background()

	db, err := NewDB(ctx, dsn)
	if err != nil {
		t.Fatalf("NewTestDB: connect: %v", err)
		return nil, func() {}
	}

	// Ensure a clean starting state, then (re)create the schema.
	if err := dropTestSchema(ctx, db); err != nil {
		_ = db.Close()
		t.Fatalf("NewTestDB: pre-clean schema: %v", err)
		return nil, func() {}
	}
	if err := db.InitSchema(ctx); err != nil {
		_ = db.Close()
		t.Fatalf("NewTestDB: init schema: %v", err)
		return nil, func() {}
	}

	cleanup := func() {
		if err := dropTestSchema(ctx, db); err != nil {
			t.Errorf("NewTestDB cleanup: drop schema: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Errorf("NewTestDB cleanup: close: %v", err)
		}
	}

	return db, cleanup
}

// NewTestClient builds an httptest.Server wired to a fresh test database and
// returns the server, its base URL, and a combined cleanup function.
//
// This is the Go analogue of the pytest "client" fixture: instead of
// overriding FastAPI's get_db dependency, the *DB is injected directly into
// NewHandlers and the resulting router is served by an httptest.Server. Tests
// issue real HTTP requests against srv.URL, exactly as the Starlette
// TestClient did.
//
// The returned cleanup closes the server and tears down the test database.
func NewTestClient(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	db, dbCleanup := NewTestDB(t)

	handler := newTestRouter(db)
	srv := httptest.NewServer(handler)

	cleanup := func() {
		srv.Close()
		dbCleanup()
	}

	return srv, cleanup
}

// newTestRouter constructs the full application HTTP handler backed by the
// given *DB.
//
// MIGRATION_NOTE: The route set below must match the routes registered by the
// production application (see internal/main.go). It is duplicated here so the
// test harness exercises the same paths independently of any wiring helper.
// If buildRouter is exported from main.go, prefer calling it directly instead.
func newTestRouter(db *DB) http.Handler {
	h := NewHandlers(db)

	mux := chiRouter()
	mux.Get("/", h.Root)
	mux.Get("/health", h.HealthCheck)
	mux.Get("/books", h.ListBooks)
	mux.Post("/books", h.CreateBook)
	mux.Get("/books/{book_id}", h.GetBook)
	mux.Put("/books/{book_id}", h.UpdateBook)
	mux.Delete("/books/{book_id}", h.DeleteBook)

	return mux
}

// dropTestSchema removes the tables created by InitSchema so each test starts
// from a clean database.
//
// MIGRATION_NOTE: The source relied on SQLAlchemy's Base.metadata.drop_all.
// Here we drop the known table explicitly. Adjust the DROP statements if the
// schema managed by InitSchema changes.
func dropTestSchema(ctx context.Context, db *DB) error {
	if _, err := db.execContext(ctx, `DROP TABLE IF EXISTS books`); err != nil {
		return fmt.Errorf("drop books table: %w", err)
	}
	return nil
}

// execContext is a thin adapter that lets dropTestSchema run raw DDL against
// the underlying *sql.DB regardless of how DB exposes it.
//
// MIGRATION_NOTE: This assumes internal.DB embeds or wraps a *sql.DB reachable
// via an ExecContext-capable value. If DB already exposes ExecContext (or a
// similar method), replace this adapter with a direct call. This indirection
// exists only so the file compiles against the DB API defined in
// internal/database.go; verify the actual accessor during manual review.
func (db *DB) execContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.pool().ExecContext(ctx, query, args...)
}
