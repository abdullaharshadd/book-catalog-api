# book-catalog-api

A RESTful API for managing a book catalog. Supports creating, reading, updating, and deleting book records. Originally built with Python/Django, migrated to Go using the standard library.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go |
| HTTP | Go `net/http` (standard library) |
| Database | See [Migration Notes](#migration-notes) |
| Testing | Go `testing` package |

---

## Prerequisites

- Go 1.21 or later
- Node.js / npm (required for tooling — see [Getting Started](#getting-started))
- A running database instance (see [Migration Notes](#migration-notes) — database setup requires manual configuration)

---

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/abdullaharshadd/book-catalog-api.git
cd book-catalog-api
```

### 2. Install dependencies

```bash
npm install
```

> **Note:** This command was detected in the migration setup plan. If the project has been fully migrated to Go, you may also need to run `go mod download` to fetch Go module dependencies. Verify which tooling requires npm before running in a Go-only environment.

### 3. Configure environment variables

No environment variables were detected as required by the automated migration. However, because database configuration and several other components require manual rewriting (see [Known Limitations](#known-limitations)), you will likely need to add variables for database connection strings before the application runs correctly. Create a `.env` file or export variables directly:

```bash
# Example — actual variable names must be confirmed after manual review
export DATABASE_URL="..."
export PORT="8080"
```

See the [Environment Variables](#environment-variables) section for details.

### 4. Database setup

> ⚠️ **Manual step required.** The original codebase used SQLAlchemy's `Base.metadata.create_all()` for schema creation and FastAPI's dependency injection for session management. Neither maps directly to the Go standard library. You must manually implement database initialization. See [Known Limitations](#known-limitations).

No automated database setup command was detected. After resolving the database layer (see [Known Limitations](#known-limitations)), run your schema migrations or initialization script manually.

### 5. Run the application

No run command was detected by the migration tooling. To start the server with Go:

```bash
go run ./...
```

Confirm the correct entry point (`main.go` location) after completing manual review of `app/main.py` → Go equivalent.

---

## Running Tests

No test command was detected by the migration tooling. To run Go tests:

```bash
go test ./...
```

> ⚠️ **Test files require significant manual rewriting before they will pass.** See [Known Limitations](#known-limitations) and [Manual Review Required](#manual-review-required) for details on which test components could not be migrated automatically.

---

## Environment Variables

No environment variables were confirmed by the automated migration. The table below reflects what is likely needed based on the original application's structure. **Verify and expand this table after completing manual review.**

| Variable | Description | Required | Default |
|---|---|---|---|
| `DATABASE_URL` | Connection string for the database | Likely yes | — |
| `PORT` | Port the HTTP server listens on | No | `8080` |

---

## Architecture Overview

The migrated project follows a flat package structure targeting the Go standard library. The layout maps from the original Django/FastAPI modules as follows:

```
book-catalog-api/
├── main.go              # Entry point; replaces app/main.py
├── handlers/            # HTTP handler functions; replaces Django views and FastAPI route definitions
├── models/              # Data models and database interaction; replaces app/models and SQLAlchemy ORM
├── schemas/             # Request/response types; replaces app/schemas.py (Pydantic models)
├── database/            # Database connection setup; replaces app/database.py
└── tests/               # Go test files; replaces tests/test_api.py, test_models.py, test_schemas.py
```

> **Note:** The migration confidence is 0% overall, meaning the above structure is the intended target but the actual generated files require substantial manual correction before they reflect a working Go application. Treat the migrated files as scaffolding, not finished code.

---

## Migration Notes

### What changed from the original Django/FastAPI codebase

The original project was identified as Python/Django in the migration metadata but the source modules (`app/database.py`, `app/schemas.py`, `conftest.py`) are characteristic of a **FastAPI + SQLAlchemy + Pydantic** stack, not Django. This inconsistency is itself a finding that requires developer attention.

| Original (Python) | Migrated (Go) | Notes |
|---|---|---|
| FastAPI route definitions | `net/http` `HandlerFunc` | Manual wiring of routes required |
| SQLAlchemy ORM models | Go structs + database/sql or third-party driver | ORM layer is not included in the standard library |
| Pydantic schemas | Go structs with JSON tags | Validation logic must be reimplemented manually |
| `get_db` / `get_sync_db` dependency injection | No equivalent | Must be replaced with explicit DB connection passing |
| `Base.metadata.create_all()` | No equivalent | Use migrations or manual `CREATE TABLE` statements |
| FastAPI `TestClient` | `net/http/httptest` | Test client must be rewritten |
| SQLAlchemy `Session` in tests | `*sql.DB` or equivalent | Session lifecycle must be managed manually |
| Pydantic validation error messages | Go error types | Error message format will differ; assertions must be rewritten |
| `pytest` + `conftest.py` fixtures | Go `TestMain` or per-test setup | No direct equivalent; rewrite test infrastructure |

---

## Known Limitations

The following components **could not be automatically migrated** and require manual implementation before the application will build or behave correctly.

### `app/database.py` — `get_db` / `get_sync_db`

**Reason:** These are FastAPI/SQLAlchemy yield-based dependency injection session generators. Go's `net/http` has no dependency injection mechanism.

**Action required:** Implement an explicit database connection or connection pool (e.g., `database/sql` with a driver such as `lib/pq` for PostgreSQL or `mattn/go-sqlite3`). Pass the `*sql.DB` instance directly to handlers, or use a middleware pattern.

---

### `app/database.py` — `init_db` (`Base.metadata.create_all`)

**Reason:** SQLAlchemy metadata-driven schema creation has no equivalent in the Go standard library.

**Action required:** Write explicit SQL `CREATE TABLE` statements, use a migration tool (e.g., [golang-migrate](https://github.com/golang-migrate/migrate)), or integrate a lightweight ORM such as [sqlc](https://sqlc.dev/) or [GORM](https://gorm.io/).

---

### `conftest.py` — `client_fixture` (`app.dependency_overrides`)

**Reason:** FastAPI's `dependency_overrides` dict has no Django or Go counterpart.

**Action required:** In Go tests, use `net/http/httptest.NewServer` or `httptest.NewRecorder` and inject a test database connection directly into your handler setup. Drop the override logic entirely.

---

### `conftest.py` — `db_session_fixture`

**Reason:** Uses SQLAlchemy `engine`/`sessionmaker` and `create_all`/`drop_all` for test database lifecycle. Django and Go both manage this differently.

**Action required:** In Go, create a test `*sql.DB` pointed at an in-memory or throwaway test database, run your schema setup SQL in `TestMain`, and tear down after tests complete.

---

### `tests/test_api.py` — `test_db` fixture and `client` fixture

**Reason:** `StaticPool` in-memory SQLAlchemy setup and FastAPI `TestClient` are framework-specific with no automatic Go equivalent.

**Action required:** Rewrite using `net/http/httptest`. Create a test server with `httptest.NewServer(yourRouter)` and use a standard `http.Client` to make requests against it.

---

### `tests/test_models.py` — `db_session` fixture and SQLAlchemy session operations

**Reason:** SQLAlchemy Unit-of-Work (`session.add`, `session.commit`, `session.refresh`) does not map to Go's `database/sql`.

**Action required:** Rewrite model tests using direct SQL queries or your chosen Go ORM. Replace `session.add(book)` with `db.Exec("INSERT INTO books ...")` or the equivalent ORM call.

---

### `tests/test_schemas.py` — Pydantic validation error message assertions

**Reason:** Assertions check exact Pydantic v1 error message strings, which will not match any Go validation library's output.

**Action required:** Rewrite assertions to check Go error values, field names, or HTTP response status codes rather than exact message text.

---

## Manual Review Required

The following files were flagged as low-confidence by the migration tool. **Do not treat these as production-ready.** Each must be opened, understood, and corrected by a developer before use.

| File | What to verify |
|---|---|
| `app/__init__.py` → Go equivalent | Confirm package initialization; Go does not use `__init__` files |
| `app/database.py` → `database/` package | Entire file requires manual rewrite; see [Known Limitations](#known-limitations) |
| `app/schemas.py` → Go structs | Verify all fields, types, and JSON tags; reimplement validation logic |
| `app/main.py` → `main.go` | Verify route registration, middleware, and server startup are correctly wired |
| `conftest.py` → test setup | Entire fixture setup requires manual rewrite for Go test infrastructure |
| `tests/test_api.py` | Rewrite test client and DB fixture setup; verify all endpoint assertions |
| `tests/test_models.py` | Rewrite DB session setup; replace SQLAlchemy operations with Go equivalents |
| `tests/test_schemas.py` | Rewrite all validation error assertions to match Go error output format |

---

## Migration Confidence

> **Overall migration confidence: 0%**
>
> All 10 modules were processed, but no module passed automated confidence checks. The migrated files should be treated as a structural starting point only. A developer familiar with both the original FastAPI/SQLAlchemy codebase and Go must review every file before this project is buildable or testable.

---

## Contributing

After completing the manual fixes documented above, ensure:

1. `go build ./...` completes without errors
2. `go vet ./...` reports no issues
3. `go test ./...` passes with a real or in-memory test database