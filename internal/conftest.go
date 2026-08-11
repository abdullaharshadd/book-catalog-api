package internal

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// MIGRATION_NOTE: The Python source (app/tests/conftest.py) was a pytest
// configuration module that declared two fixtures:
//
//   - db_session: created a SQLAlchemy engine + session against an in-memory
//     SQLite database, ran Base.metadata.create_all() before the test and
//     Base.metadata.drop_all() afterwards.
//   - client: a Starlette TestClient that overrode FastAPI's get_db dependency
//     with the test session, via app.dependency_overrides.
//
// None of these constructs have a direct Go equivalent, so this file provides
// idiomatic Go test helpers instead:
//
//   - pytest fixtures            -> plain helper functions returning a cleanup
//                                   closure (the standard t.Cleanup pattern).
//   - SQLAlchemy engine/session  -> *sqlx.DB (matching the production type used
//                                   by NewDB / BookHandler).
//   - in-memory SQLite           -> a real PostgreSQL database. SQLite's
//                                   in-memory mode does not exist for Postgres;
//                                   tests must point at a throwaway Postgres
//                                   instance (e.g. a Docker/testcontainers DB).
//                                   The DSN is read from the TEST_DATABASE_URL
//                                   environment variable so CI can inject it.
//   - Base.metadata.create_all   -> InitDB(ctx, db) (defined in database.go),
//                                   which creates the schema.
//   - Base.metadata.drop_all     -> an explicit DROP TABLE in the returned
//                                   cleanup func.
//   - FastAPI dependency_overrides -> not needed. In Go the *sqlx.DB is passed
//                                   explicitly into NewBookHandler, so the test
//                                   simply wires the test DB into the handler.
//   - Starlette TestClient       -> net/http/httptest.Server wrapping the same
//                                   chi router the production code builds.
//
// MIGRATION_NOTE: Because these are exported test helpers (used by the test
// files in this package), they live in a normal .go file rather than a _test.go
// file so they can be shared. They take *testing.T and register cleanup via
// t.Cleanup, which is the idiomatic replacement for a fixture's teardown.

// testDBEnvVar is the environment variable holding the PostgreSQL DSN used by
// the integration test suite.
const testDBEnvVar = "TEST_DATABASE_URL"

// NewTestDB opens a connection to the test PostgreSQL database, ensures the
// schema exists, and registers cleanup that drops the books table and closes
// the pool when the test completes.
//
// MIGRATION_NOTE: This replaces the pytest db_session fixture. The DSN comes
// from the TEST_DATABASE_URL environment variable; the test is skipped if it is
// not set, so unit-only runs (with no database available) do not fail.
func NewTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dsn := getTestDSN(t)
	if dsn == "" {
		t.Skipf("%s not set; skipping database-backed test", testDBEnvVar)
	}

	ctx := context.Background()

	db, err := NewDB(ctx, dsn)
	if err != nil {
		t.Fatalf("NewTestDB: open database: %v", err)
	}

	if err := InitDB(ctx, db); err != nil {
		_ = db.Close()
		t.Fatalf("NewTestDB: init schema: %v", err)
	}

	t.Cleanup(func() {
		// MIGRATION_NOTE: mirrors Base.metadata.drop_all(). We drop the
		// concrete table used by the suite rather than reflecting over a
		// metadata registry, which Go does not have.
		if _, err := db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+BooksTable); err != nil {
			t.Errorf("NewTestDB cleanup: drop table: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Errorf("NewTestDB cleanup: close: %v", err)
		}
	})

	return db
}

// NewTestRouter builds the chi router with all production routes registered
// against the supplied database. It mirrors the route table constructed in
// main.go so that integration tests exercise the exact same handlers and paths
// as production.
//
// MIGRATION_NOTE: This replaces FastAPI's app object plus dependency_overrides.
// The *sqlx.DB is injected directly through NewBookHandler, so there is no
// global override map to set or clear.
func NewTestRouter(db *sqlx.DB) *chi.Mux {
	h := NewBookHandler(db)

	r := chi.NewRouter()
	r.Get("/", Root)
	r.Get("/health", HealthCheck)
	r.Get("/books", h.ListBooks)
	r.Post("/books", h.CreateBook)
	r.Get("/books/{id}", h.GetBook)
	r.Put("/books/{id}", h.UpdateBook)
	r.Delete("/books/{id}", h.DeleteBook)

	return r
}

// NewTestServer starts an httptest.Server serving the full application router
// backed by the given database and registers cleanup to shut it down.
//
// MIGRATION_NOTE: This replaces Starlette's TestClient context manager. The
// returned *httptest.Server exposes URL for building requests with the standard
// net/http client, which is the idiomatic Go equivalent of the yielded client
// fixture.
func NewTestServer(t *testing.T, db *sqlx.DB) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(NewTestRouter(db))
	t.Cleanup(srv.Close)
	return srv
}
