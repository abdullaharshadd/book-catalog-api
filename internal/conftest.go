package internal

// This file is the Go analog of the source project's app/tests/conftest.py,
// which defined pytest fixtures that (a) created a fresh SQLite schema via
// SQLAlchemy's Base.metadata.create_all, (b) handed each test a scoped DB
// session, and (c) built a Starlette TestClient with FastAPI's get_db
// dependency overridden to use that session.
//
// MIGRATION_NOTE: Go's testing package has no fixture/dependency-injection
// machinery like pytest. The idiomatic replacement is a small set of test
// helpers, wired once and reused, plus per-test isolation. There is no
// app.dependency_overrides equivalent: a single *sqlx.DB is captured by the
// BookServer (see internal/main.go) and shared with every handler, so tests
// simply construct the server around the same *sqlx.DB they inspect directly.
//
// MIGRATION_NOTE: The source conftest targeted SQLite ("sqlite:///./test.db").
// Per the project's target-dialect requirement, this harness targets
// PostgreSQL instead. The connection string is read from TEST_DATABASE_URL
// (falling back to a local default) so CI can point it at a throwaway
// database. Schema creation mirrors Base.metadata.create_all by running
// idempotent CREATE TABLE IF NOT EXISTS DDL derived from the Book model in
// internal/model.go (id, title, author, published_year, created_at).
//
// These helpers live in a non-test file so both the internal package tests and
// the internal/tests package can import and reuse them. Functions accept
// *testing.T rather than returning errors so failures abort the test cleanly,
// matching pytest fixture semantics where setup failure fails the test.

import (
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// defaultTestDatabaseURL is used when TEST_DATABASE_URL is not set in the
// environment. It points at a conventional local PostgreSQL instance.
const defaultTestDatabaseURL = "postgres://postgres:postgres@localhost:5432/books_test?sslmode=disable"

// createBooksTableDDL creates the books table if it does not already exist.
//
// The column list is derived directly from the source Book model:
//   - id             (primary key, auto-incrementing)
//   - title          (string)
//   - author         (string)
//   - published_year (integer)
//   - created_at     (timestamp)
//
// GENERATED ALWAYS AS IDENTITY is used for the primary key rather than
// MySQL-style AUTO_INCREMENT or SQLite's implicit rowid.
const createBooksTableDDL = `
CREATE TABLE IF NOT EXISTS books (
    id             INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title          VARCHAR(255) NOT NULL,
    author         VARCHAR(255) NOT NULL,
    published_year INTEGER      NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
)`

// testDatabaseURL returns the connection string for the test database,
// preferring the TEST_DATABASE_URL environment variable.
func testDatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}
	return defaultTestDatabaseURL
}

// NewTestDB opens a connection to the test PostgreSQL database and ensures the
// books schema exists. It is the Go analog of the db_session pytest fixture's
// setup phase (create_engine + Base.metadata.create_all).
//
// The returned *sqlx.DB is registered for cleanup via t.Cleanup, so callers do
// not need to close it manually. Any failure aborts the test immediately,
// mirroring pytest fixture failure semantics.
func NewTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Connect("postgres", testDatabaseURL())
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}

	if _, err := db.Exec(createBooksTableDDL); err != nil {
		_ = db.Close()
		t.Fatalf("create books schema: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("close test database: %v", err)
		}
	})

	// Start from a clean slate so tests never observe rows left by a prior run.
	resetBooks(t, db)

	return db
}

// resetBooks truncates the books table and restarts its identity sequence so
// each test observes deterministic primary keys.
//
// MIGRATION_NOTE: The source dropped and recreated all tables between tests
// (Base.metadata.drop_all). On PostgreSQL, TRUNCATE ... RESTART IDENTITY is
// both faster and the direct analog of "empty table + reset the auto-increment
// counter" (which under SQLite meant resetting sqlite_sequence).
func resetBooks(t *testing.T, db *sqlx.DB) {
	t.Helper()
	if _, err := db.Exec(`TRUNCATE TABLE books RESTART IDENTITY`); err != nil {
		t.Fatalf("reset books table: %v", err)
	}
}

// NewTestServer builds a BookServer wired to a freshly-prepared test database
// and returns both. It is the Go analog of the client pytest fixture, which
// built a TestClient with get_db overridden to use the test session.
//
// Because handlers and tests share the same *sqlx.DB, there is no dependency
// override to install or clear: constructing the server around the returned db
// is sufficient. Callers can hand the server's HTTP handler to
// httptest.NewServer (or exercise it via httptest.NewRecorder) and inspect the
// same db directly.
func NewTestServer(t *testing.T) (*BookServer, *sqlx.DB) {
	t.Helper()
	db := NewTestDB(t)
	server := NewBookServer(db)
	return server, db
}
