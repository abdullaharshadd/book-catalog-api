# book-catalog-api

A REST API for managing a book catalog. Allows clients to create, read, update, and delete book records. Originally written in Python/Django; this version has been migrated to Go using the standard library.

> ⚠️ **Migration confidence: 0%** — This migration was completed at extremely low confidence. Every module requires manual review before this codebase is considered production-ready. Do not deploy without a thorough audit.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go |
| HTTP | Go standard library (`net/http`) |
| Database | TBD — see [Migration Notes](#migration-notes) |
| Testing | Go standard library (`testing`) |

---

## Prerequisites

- Go 1.21 or later
- Node.js / npm (detected in setup commands — verify whether this is actually required for this project)
- A running database instance (see [Migration Notes](#migration-notes) for database setup caveats)

---

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/abdullaharshadd/book-catalog-api.git
cd book-catalog-api
```

### 2. Install dependencies

The setup plan detected the following install command:

```bash
npm install
```

> ⚠️ **Review required.** `npm install` was detected but this is a Go project. This command may be a leftover artifact from the migration process or may be needed for an auxiliary toolchain (e.g., API doc generation, linting scripts). Verify `package.json` exists and whether this step is genuinely needed. The canonical Go dependency command is:
>
> ```bash
> go mod tidy
> ```

### 3. Environment variables

No environment variables were detected by the automated migration. However, given the 0% confidence score, it is likely that database connection strings, secret keys, or other configuration values are required but were not captured. See [Environment Variables](#environment-variables) below.

### 4. Database setup

> ⚠️ **No database setup command was detected.** The original Python codebase used SQLAlchemy async sessions and programmatic schema creation (`Base.metadata.create_all`). Neither of these patterns has a direct equivalent in the migrated Go code. You must manually configure and initialize the database. See [Known Limitations](#known-limitations) for details.

### 5. Run the server

> ⚠️ **No run command was detected.** After resolving the issues listed in this document, the typical command for a Go HTTP server would be:

```bash
go run ./cmd/server/main.go
```

or, after building:

```bash
go build -o book-catalog-api ./...
./book-catalog-api
```

Verify the actual entry point in the repository.

---

## Running Tests

> ⚠️ **No test command was detected.** The standard Go test command is:

```bash
go test ./...
```

For verbose output:

```bash
go test -v ./...
```

> Multiple test files (`tests/test_api.py`, `tests/test_models.py`, `tests/test_schemas.py`) were flagged as unmigrable or low-confidence. The test suite **will not run correctly** without significant manual rewriting. See [Manual Review Required](#manual-review-required).

---

## Environment Variables

No environment variables were captured during migration. The table below is a placeholder. You must audit the source codebase and populate this before the application can run.

| Variable | Description | Required | Default |
|---|---|---|---|
| _(none detected)_ | — | — | — |

Common variables to check for in the original source:

- Database connection URL or DSN
- Application secret key
- Debug / environment mode flag
- Server port
- Allowed hosts / CORS origins

---

## Architecture Overview

The project was migrated from a Python/Django structure to Go using the standard library. The expected Go layout after migration:

```
book-catalog-api/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point (verify this exists)
├── internal/
│   ├── handlers/            # HTTP handlers (migrated from app/main.py views/routes)
│   ├── models/              # Data models (migrated from app/models.py)
│   ├── schemas/             # Request/response types (migrated from app/schemas.py)
│   └── database/            # Database setup (migrated from app/database.py)
├── tests/                   # Go test files (migrated from tests/*.py)
├── go.mod
└── go.sum
```

> ⚠️ This layout reflects the expected output of the migration tooling. Verify that the actual directory structure matches before proceeding.

---

## Migration Notes

This project was automatically migrated from **Python / Django** (originally using FastAPI + SQLAlchemy, despite the Django label) to **Go / standard library**.

### What changed

| Area | Original (Python) | Migrated (Go) |
|---|---|---|
| Language | Python 3.x | Go 1.21+ |
| HTTP framework | FastAPI | `net/http` (standard library) |
| ORM / DB layer | SQLAlchemy (async) | TBD — not fully resolved |
| Schema validation | Pydantic v1 | Go structs + manual validation |
| Testing client | FastAPI `TestClient` | `net/http/httptest` |
| Test DB isolation | SQLAlchemy in-memory SQLite | TBD — not fully resolved |
| Dependency injection | FastAPI `Depends()` | Manual wiring |

### Key behavioral differences to verify

- **HTTP status codes.** The original API returned `201` for creation, `204` for deletion, and `422` for validation errors. Go's `net/http` has no opinion on these — confirm the migrated handlers return the same codes if API compatibility is required.
- **Validation errors.** Pydantic v1 validation error shapes are not replicated automatically. If clients depend on the error response format, this must be manually implemented.
- **Async behavior.** The original used SQLAlchemy async sessions. Go is synchronous by default at the handler level (goroutines handle concurrency differently). Verify the database access pattern in the migrated code is correct.

---

## Known Limitations

The following components could not be automatically migrated. They require manual implementation.

### `app/database.py` — Async engine / async session / `get_db`

**Reason:** Django's ORM does not support async sessions in the same SQLAlchemy sense. There is no direct equivalent to an async SQLAlchemy `AsyncSession` dependency generator in either Django or the Go standard library.

**Action required:** Implement database connectivity directly in Go. Options include `database/sql` with a driver (e.g., `lib/pq` for Postgres, `mattn/go-sqlite3` for SQLite), or a lightweight query builder. Wire the connection manually into handlers.

---

### `app/database.py` — `init_db` (`Base.metadata.create_all`)

**Reason:** The original used programmatic schema creation from SQLAlchemy metadata. Go's standard library has no equivalent.

**Action required:** Write SQL migration files manually, or integrate a migration tool such as [golang-migrate](https://github.com/golang-migrate/migrate) or [goose](https://github.com/pressly/goose). Do not rely on runtime schema creation in production.

---

### `conftest.py` — `client_fixture` / `app.dependency_overrides`

**Reason:** FastAPI's dependency injection override mechanism has no equivalent in Go or Django.

**Action required:** Rewrite test setup to use `net/http/httptest` with a test server. Inject a test database connection directly rather than through a dependency override system.

---

### `conftest.py` — `db_session_fixture`

**Reason:** SQLAlchemy sessionmaker and `Base.metadata` table management are ORM-specific and do not translate.

**Action required:** Implement test database isolation manually. Use a separate test database or wrap each test in a transaction that is rolled back after the test completes.

---

### `tests/test_api.py` — `test_db` fixture, `client` fixture

**Reason:** Tightly coupled to SQLAlchemy in-memory SQLite and FastAPI's `TestClient`. Neither exists in Go.

**Action required:** Rewrite all API tests using `net/http/httptest`. Set up an in-memory or test-specific SQLite/Postgres instance for database tests.

---

### `tests/test_api.py` — HTTP status code assertions (`201` / `204` / `422`)

**Reason:** These codes were FastAPI/Pydantic defaults. The migrated Go handlers may return different codes unless explicitly set.

**Action required:** Audit every handler's response code. Update test assertions to match actual behavior, or update handlers to match the original contract.

---

### `tests/test_models.py` — `db_session` fixture, SQLAlchemy session lifecycle

**Reason:** SQLAlchemy's `session.add()` / `session.commit()` / `session.refresh()` pattern does not exist in Go.

**Action required:** Rewrite model tests to interact with the database via the Go data access layer you implement. Use `sql.DB` or your chosen library directly.

---

### `tests/test_schemas.py` — Pydantic `ValidationError` message assertions

**Reason:** Assertions depend on exact Pydantic v1 error message strings, which do not exist in Go.

**Action required:** Rewrite validation tests to assert against the Go validation error format used in the migrated code. Update expected error messages and structures entirely.

---

## Manual Review Required

The following files were flagged as low-confidence by the migration tool. A developer must read, understand, and verify each one before relying on any behavior.

| File | Primary concern |
|---|---|
| `app/database.py` | Async DB session and schema init not migrated — see Known Limitations |
| `app/schemas.py` | Pydantic schema → Go struct conversion; validation behavior not guaranteed equivalent |
| `app/main.py` | Route registration, middleware, and request/response lifecycle may differ significantly |
| `conftest.py` | Test fixtures entirely incompatible with Go testing — full rewrite required |
| `tests/test_api.py` | Test client, DB fixture, and status code assertions all require manual rewrite |
| `tests/test_models.py` | SQLAlchemy session usage must be replaced with Go DB access patterns |
| `tests/test_schemas.py` | Validation error assertions tied to Pydantic v1 message format — rewrite required |

---

## Contributing

Before submitting changes, ensure:

1. `go vet ./...` passes with no errors
2. `go test ./...` passes (after test suite is manually repaired)
3. All items in [Manual Review Required](#manual-review-required) have been addressed and documented

---

## Original Project

Source repository: [abdullaharshadd/book-catalog-api](https://github.com/abdullaharshadd/book-catalog-api)
Original stack: Python · FastAPI · SQLAlchemy (async) · Pydantic v1