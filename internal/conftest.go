package internal

// This file replaces the pytest fixtures from app/tests/conftest.go
// (conftest.py). In pytest those fixtures build an isolated test database,
// hand each test a per-test DB session, create/drop the schema around each
// test, and yield a Starlette TestClient with the app's get_db dependency
// overridden to use that session.
//
// Go's testing model has no fixture-injection framework, so the equivalent is
// a set of plain test-helper constructors that each *_test.go file calls
// explicitly (usually via t.Helper()/t.Cleanup()). This file provides those
// helpers so the integration tests in this package share one setup path.
//
// MIGRATION_NOTE: The source used SQLite (sqlite:///./test.db). The target
// dialect for this project is PostgreSQL, so the test harness connects to a
// PostgreSQL instance identified by the TEST_DATABASE_URL environment
// variable. This avoids a per-file SQLite translation that would diverge from
// the production dialect. If TEST_DATABASE_URL is unset the helper skips the
// test (t.Skip) rather than failing, mirroring how the source silently used a
// local throwaway database.
//
// MIGRATION_NOTE: FastAPI's app.dependency_overrides[get_db] override has no
// Go analogue — dependencies are passed explicitly. NewTestServer wires the
// same *DB directly into the Handlers, which is the Go equivalent of
// overriding get_db to return the test session.
//
// MIGRATION_NOTE: The source creates the schema in the fixture via
// Base.metadata.create_all. Here the schema is created with real CREATE TABLE
// IF NOT EXISTS DDL derived from the Book model (internal/model.go), and
// dropped in cleanup, mirroring the create_all/drop_all lifecycle.

import (
	"context"
	"database/sql"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// createBooksTableDDL creates the schema for the Book model. The columns match
// the source SQLAlchemy Book model's fields (id, title, author, published_year,
// created_at). PostgreSQL identity column replaces SQLite's autoincrement.
const createBooksTableDDL = `
CREATE TABLE IF NOT EXISTS books (
	id             SERIAL PRIMARY KEY,
	title          TEXT NOT NULL,
	author         TEXT NOT NULL,
	published_year INTEGER,
	created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
)`

// dropBooksTableDDL removes the schema created by createBooksTableDDL. It
// reproduces the fixture teardown (Base.metadata.drop_all).
const dropBooksTableDDL = `DROP TABLE IF EXISTS books`

// NewTestDB opens a connection to the test PostgreSQL database, creates the
// schema, and registers a cleanup that drops the schema and closes the
// connection when the test finishes.
//
// It replaces the db_session pytest fixture. If TEST_DATABASE_URL is not set
// the test is skipped, since there is no database to talk to.
func NewTestDB(t *testing.T) *DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	ctx := context.Background()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("ping test database: %v", err)
	}

	if err := createSchema(ctx, sqlDB); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("create schema: %v", err)
	}

	t.Cleanup(func() {
		if _, err := sqlDB.ExecContext(context.Background(), dropBooksTableDDL); err != nil {
			t.Errorf("drop schema: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	return &DB{DB: sqlDB}
}

// createSchema runs the CREATE TABLE DDL for the test database. It is the Go
// equivalent of Base.metadata.create_all(bind=engine).
func createSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, createBooksTableDDL); err != nil {
		return fmt.Errorf("create books table: %w", err)
	}
	return nil
}

// NewTestServer starts an httptest.Server backed by the given test database and
// registers a cleanup to shut it down. It replaces the client pytest fixture,
// which returned a Starlette TestClient with get_db overridden.
//
// The returned *httptest.Server exposes URL for issuing requests against the
// same routes registered by Router in internal/main.go.
func NewTestServer(t *testing.T, db *DB) *httptest.Server {
	t.Helper()

	handlers := NewHandlers(db)
	srv := httptest.NewServer(Router(handlers))
	t.Cleanup(srv.Close)
	return srv
}
