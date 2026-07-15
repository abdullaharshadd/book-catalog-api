// Package catalog contains shared test helpers for the Book Catalog API.
//
// This file was migrated from app/tests/conftest.py, a pytest configuration
// module that provided two fixtures:
//
//   - db_session: created an in-memory / file-backed SQLite database, ran the
//     schema DDL, yielded a session, then closed it and dropped the tables.
//   - client: overrode the FastAPI get_db dependency so that HTTP handlers
//     used the test session, then yielded a Starlette TestClient.
//
// Go has no pytest fixture mechanism. The idiomatic equivalent is a set of
// helper constructors that each test calls explicitly (usually via t.Helper
// and t.Cleanup for teardown). Because this file provides test infrastructure
// it lives in a _test.go-style helper; however, to keep the exported helpers
// usable from sibling test files in the same package we place them in a
// normal .go file per the required target path.
//
// MIGRATION_NOTE: The original SQLALCHEMY_DATABASE_URL was "sqlite:///./test.db".
// Tests should prefer a purely in-memory database. We default to
// "file::memory:?cache=shared" so that multiple connections in the same process
// observe the same schema (a plain ":memory:" DSN gives each connection its own
// empty database, which breaks database/sql connection pooling).
package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	// MIGRATION_NOTE: adjust these import paths to match your module path.
	// The already-migrated symbols (NewDB, InitDB, DB, Close, CreateTableStmts)
	// live in internal/database.py.go and internal/models.py.go.
	_ "github.com/mattn/go-sqlite3"
)

// testDatabaseDSN is the data source name used for the test database.
//
// MIGRATION_NOTE: the Python code used a file-backed SQLite DB ("./test.db").
// A shared in-memory database avoids leaving files on disk while still being
// visible across all pooled connections opened by database/sql.
const testDatabaseDSN = "file::memory:?cache=shared&_foreign_keys=on"

// TestDB bundles a live *DB together with a teardown function. It is the Go
// analogue of the pytest "db_session" fixture.
type TestDB struct {
	// DB is the initialised database handle (schema already applied).
	DB *DB
}

// NewTestDB opens an in-memory SQLite database, applies the schema DDL, and
// registers cleanup so the database is closed when the test finishes.
//
// It replaces the pytest "db_session" fixture:
//
//   - Base.metadata.create_all -> InitDB (runs CreateTableStmts).
//   - db.close()               -> Close, registered via t.Cleanup.
//   - Base.metadata.drop_all   -> unnecessary; the in-memory DB is discarded
//     when the last connection closes.
//
// NewTestDB fails the test immediately (via t.Fatalf) if the database cannot be
// opened or migrated, mirroring pytest's fail-fast fixture behaviour.
func NewTestDB(ctx context.Context, t *testing.T) *TestDB {
	t.Helper()

	db, err := NewDB(ctx, testDatabaseDSN)
	if err != nil {
		t.Fatalf("NewTestDB: open database: %v", err)
	}

	if err := InitDB(ctx, db); err != nil {
		// Best-effort close before failing so we do not leak a connection.
		_ = Close(db)
		t.Fatalf("NewTestDB: init schema: %v", err)
	}

	t.Cleanup(func() {
		if err := Close(db); err != nil {
			t.Errorf("NewTestDB: close database: %v", err)
		}
	})

	return &TestDB{DB: db}
}

// TestClient bundles an httptest server together with its base URL. It is the
// Go analogue of the pytest "client" fixture built on Starlette's TestClient.
type TestClient struct {
	// Server is the running in-process HTTP test server.
	Server *httptest.Server
	// DB is the database backing the handlers, so tests can assert on state.
	DB *DB
}

// BaseURL returns the root URL of the test server (e.g. "http://127.0.0.1:port").
func (c *TestClient) BaseURL() string {
	return c.Server.URL
}

// NewTestClient starts an in-process HTTP server wired to a fresh test database
// and registers cleanup for both. It replaces the pytest "client" fixture.
//
// MIGRATION_NOTE: FastAPI's app.dependency_overrides[get_db] = override_get_db
// has no direct Go equivalent. Go does not use runtime dependency injection
// containers; instead the *DB is passed explicitly into the router constructor.
// This is the idiomatic replacement for the dependency override: the test
// simply constructs the handler with the test database.
//
// The handlerFactory argument must build the application's http.Handler from a
// *DB. It is expected to register ALL application routes at their exact paths
// (e.g. GET/POST /books, GET/PUT/DELETE /books/{id}). See internal/main.py.go
// for the concrete router (typically a NewRouter(db) constructor). Passing the
// factory keeps this helper decoupled from the router package while still
// guaranteeing the real routes are exercised.
func NewTestClient(ctx context.Context, t *testing.T, handlerFactory func(db *DB) http.Handler) *TestClient {
	t.Helper()

	if handlerFactory == nil {
		t.Fatalf("NewTestClient: handlerFactory must not be nil")
	}

	testDB := NewTestDB(ctx, t)

	handler := handlerFactory(testDB.DB)
	if handler == nil {
		t.Fatalf("NewTestClient: handlerFactory returned nil handler")
	}

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return &TestClient{
		Server: srv,
		DB:     testDB.DB,
	}
}

// applySchema is a low-level helper that runs the schema DDL against a raw
// *sql.DB. It exists for tests that need direct database/sql access without the
// full *DB wrapper.
//
// MIGRATION_NOTE: this mirrors Base.metadata.create_all but uses the migrated
// CreateTableStmts (from internal/models.py.go) as the source of DDL.
func applySchema(ctx context.Context, db *sql.DB) error {
	for i, stmt := range CreateTableStmts() {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("applySchema: statement %d: %w", i, err)
		}
	}
	return nil
}
