# book-catalog-api

A RESTful API for managing a catalog of books. Supports creating, reading, updating, and deleting book records. Originally built with Python/Django, migrated to Go using the standard library.

---

## ⚠️ Migration Warning

**Overall migration confidence: 0%**

This codebase was automatically migrated from Python/Django to Go (standard library). The migration tooling flagged all core modules as low confidence and could not fully migrate several components. **Do not run this in production without a thorough manual review.** See [Manual Review Required](#manual-review-required) and [Known Limitations](#known-limitations) below.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go (standard library) |
| HTTP | `net/http` |
| Database | To be confirmed (see migration notes) |
| Testing | Go `testing` package |
| Build tooling | Go modules (`go.mod`) |

---

## Prerequisites

- Go 1.21 or later
- Node.js / npm (detected in setup commands — verify whether this is actually required for this project)
- A running database instance (type TBD — see [Migration Notes](#migration-notes))

---

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/abdullaharshadd/book-catalog-api.git
cd book-catalog-api
```

### 2. Install dependencies

The automated migration detected the following install command. Verify it applies to your setup:

```bash
npm install
```

> **Note:** `npm install` is unusual for a Go project. This may have been detected in error or may apply to a frontend/tooling component. Check `package.json` if it exists. For Go dependencies, run:
>
> ```bash
> go mod tidy
> ```

### 3. Configure environment variables

No environment variables were detected by the migration tooling. However, given this project connects to a database and runs an HTTP server, you will almost certainly need to set some. Create a `.env` file or export variables manually. See the [Environment Variables](#environment-variables) section for a starting template.

### 4. Database setup

No database setup command was detected. You will need to manually:

- Confirm which database the Go code connects to
- Create the database schema (the original project used SQLAlchemy with `create_all` — this must be rewritten for the Go target)
- Run any applicable migrations

### 5. Run the server

No run command was detected by the migration tooling. Once the code compiles, the typical Go invocation is:

```bash
go run ./...
```

or build and run:

```bash
go build -o book-catalog-api .
./book-catalog-api
```

Verify the correct entry point in `app/main.go`.

---

## Running Tests

No test command was detected. Standard Go test invocation:

```bash
go test ./...
```

For verbose output:

```bash
go test -v ./...
```

> **Warning:** The test suite has significant unmigrated components. Tests will likely not compile or pass without manual intervention. See [Known Limitations](#known-limitations).

---

## Environment Variables

No environment variables were detected during migration. The table below is a recommended starting point based on the project type — populate it as you review the migrated code.

| Variable | Required | Description | Example |
|---|---|---|---|
| `SERVER_PORT` | Yes | Port the HTTP server listens on | `8080` |
| `DATABASE_URL` | Yes | Connection string for the database | `postgres://user:pass@localhost:5432/bookcatalog` |
| `DATABASE_NAME` | Yes | Name of the database | `bookcatalog` |

Update this table once you have confirmed what the Go application actually reads.

---

## Architecture Overview

The migrated project maps the original Python module structure to Go as follows:

```
book-catalog-api/
├── app/
│   ├── main.go         # Entry point, HTTP server setup and route registration
│   ├── database.go     # Database connection and initialization
│   ├── models.go       # Data models / structs representing DB entities
│   └── schemas.go      # Request/response types (formerly Pydantic schemas)
├── tests/
│   ├── test_api.go     # API-level integration tests
│   ├── test_models.go  # Model/persistence tests
│   └── test_schemas.go # Input validation tests
├── conftest.go         # Shared test setup (partially unmigrated — see below)
├── go.mod
└── go.sum
```

> **Note:** The Go file names and exact structure should be verified against what was actually generated. The above reflects the expected mapping from the original Python layout.

---

## Migration Notes

### What changed from the original Django codebase

| Area | Original (Python/Django) | Migrated (Go/standard library) |
|---|---|---|
| Framework | Django + FastAPI (ASGI) | Go `net/http` |
| ORM | SQLAlchemy (async sessions) | To be confirmed — manual rewrite required |
| Schema validation | Pydantic models | Go structs (likely with manual validation) |
| Dependency injection | FastAPI `Depends()` / `dependency_overrides` | No equivalent; must be handled via function parameters or interfaces |
| Test client | FastAPI `TestClient` (Starlette/httpx) | Go `net/http/httptest` |
| Test DB lifecycle | SQLAlchemy `create_all` / `drop_all` | Must be managed manually in test setup/teardown |
| Fixtures | pytest `conftest.py` | Go `TestMain` or per-test setup functions |

### Source project clarification

The original project appears to have been **FastAPI + SQLAlchemy** (not Django ORM), despite the migration being labeled `python/django → go/standard`. The unmigrable components reference FastAPI-specific patterns (`dependency_overrides`, `TestClient`, `Starlette`). Keep this in mind when reviewing the output — the generated Go code may have been produced against incorrect assumptions about the source framework.

---

## Known Limitations

The following components could not be automatically migrated and require manual rewrites before the application will function correctly.

### `conftest.py` — `client_fixture`

**Reason:** FastAPI's `app.dependency_overrides` / `get_db` override mechanism has no direct equivalent in the target stack.

**What to do:** Rewrite test client setup using Go's `httptest.NewServer` or `httptest.NewRecorder`. Pass database dependencies explicitly via constructors or interfaces rather than using an injection override.

---

### `conftest.py` — `db_session_fixture`

**Reason:** Uses SQLAlchemy engine, sessionmaker, `create_all`, and `drop_all`. These are not available in the migrated stack.

**What to do:** Implement test database setup in Go using `TestMain` in a `_test.go` file. Create and tear down tables explicitly using your chosen Go database driver (e.g., `database/sql` with raw DDL, or a migration tool like `golang-migrate`).

---

### `tests/test_api.py` — `test_db` fixture

**Reason:** Relies on FastAPI's `dependency_overrides` with `get_sync_db`, which is framework-specific.

**What to do:** Replace with a Go test helper that initialises a test database connection and passes it into the handler under test. Use `httptest.NewRecorder` to capture responses.

---

### `tests/test_api.py` — `TestClient`

**Reason:** FastAPI's `TestClient` is tied to the ASGI app instance.

**What to do:** Replace with `net/http/httptest` — use `httptest.NewServer(handler)` for integration tests or `httptest.NewRecorder()` for unit tests against individual handlers.

---

### `tests/test_models.py` — `db_session` fixture and session-based persistence

**Reason:** SQLAlchemy session patterns (`session.add()`, `session.commit()`, `session.refresh()`) have no direct Go equivalent.

**What to do:** Rewrite using your Go database layer. For example, replace `db_session.add(book); db_session.commit()` with an insert via `database/sql` or your chosen query library, then re-fetch to verify persistence.

---

### `tests/test_schemas.py` — Pydantic validation and error message assertions

**Reason:** `BookCreate`, `BookUpdate`, and `BookResponse` are Pydantic models. Go structs do not have Pydantic's constructor-based validation. Assertions on exact Pydantic error strings (e.g., `"ensure this value has at most 255 characters"`) will not match Go validation output.

**What to do:** Rewrite validation logic in Go (manually or using a library such as `go-playground/validator`). Update test assertions to match whatever error format your Go validation layer produces.

---

## Manual Review Required

The following files were flagged as low confidence by the migration tooling. A developer must read, verify, and likely rewrite significant portions before the application is usable.

| File | Concern |
|---|---|
| `app/__init__.py` → Go equivalent | Low confidence migration; verify package structure is correct |
| `app/database.go` | Database driver, connection pooling, and schema setup must be confirmed and likely rewritten |
| `app/main.go` | Route registration and server startup must be verified against the original endpoint structure |
| `app/models.go` | Struct field types, tags (JSON, DB column names), and any ORM hooks must be verified |
| `app/schemas.go` | Validation logic (originally Pydantic) must be manually implemented or delegated to a validation library |
| `conftest.go` | Shared test fixtures are not migrated; this file likely requires a full rewrite |
| `tests/test_api.go` | Dependency injection and test client patterns require rewriting (see Known Limitations) |
| `tests/test_models.go` | SQLAlchemy session patterns require rewriting (see Known Limitations) |
| `tests/test_schemas.go` | Pydantic validation test patterns require rewriting (see Known Limitations) |

---

## Recommended Next Steps

1. **Confirm the database driver.** Choose between `database/sql` with a raw driver, `sqlx`, or an ORM such as `gorm`. Update `app/database.go` accordingly.
2. **Verify the endpoint surface.** Compare the original FastAPI route definitions against what `app/main.go` registers in the Go HTTP mux.
3. **Rewrite the test infrastructure.** Start with `TestMain` for database lifecycle, then rewrite each test file (see Known Limitations).
4. **Add validation.** Decide on a validation approach for request bodies (manual or `go-playground/validator`) and update `app/schemas.go`.
5. **Clarify the `npm install` command.** If there is no frontend or JS tooling in this project, remove this step.
6. **Set up CI.** Run `go vet ./...` and `go test ./...` as baseline checks before adding any feature work.