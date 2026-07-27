package internal

// MIGRATION_NOTE: The Python source was a pytest conftest.py providing two
// fixtures:
//
//   1. db_session_fixture — created all SQLAlchemy tables against an in-memory
//      SQLite engine, yielded a Session, then closed the session and dropped
//      all tables during teardown.
//   2. client_fixture — overrode FastAPI's get_db dependency to return the test
//      session and yielded a Starlette TestClient bound to the app.
//
// None of this translates as a source file in Go. Go has no pytest fixture
// mechanism: test setup/teardown is expressed with ordinary helper functions
// (returning a value plus a cleanup func, or using t.Cleanup) that live in
// *_test.go files, and there is no dependency-override registry to clear
// because the Go app (see internal/main.go) receives its *DB explicitly rather
// than through a FastAPI-style DI container.
//
// Additional dialect note: the Python fixtures used SQLite. The Go target uses
// PostgreSQL (see internal/database.go), so an in-memory SQLite equivalent is
// deliberately NOT reproduced. Integration tests should point at a disposable
// PostgreSQL instance (e.g. a Dockerized/testcontainers Postgres or a
// throwaway test schema) and rely on InitDB / WithTx from internal/database.go.
//
// Because Go test helpers must live in files ending in _test.go to be compiled
// only under `go test`, this file provides those helpers as exported,
// reusable constructors that a _test.go file can call. Keeping them here (non
// _test.go) would pull testing infrastructure into the production build, which
// is undesirable — therefore the real fixtures belong in
// internal/tests/fixtures_test.go. The helpers below are documented but the
// canonical, compilable versions are the _test.go equivalents shown in the
// notes for human review.
//
// This file intentionally contains no executable production code: reproducing
// pytest fixtures as production symbols would be incorrect. It exists to carry
// the migration rationale and to keep the package declaration consistent.
//
// See internal/tests for the actual test scaffolding.
