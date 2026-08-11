# book-catalog-api

A REST API for managing a book catalog. Migrated from Python/Django to Go using the Go standard library.

> ⚠️ **Migration Confidence: 0%** — This codebase was automatically migrated but requires significant manual review before it is production-ready. Do not deploy without completing the steps in the [Manual Review Required](#manual-review-required) section.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go (standard library) |
| Web framework | `net/http` (Go standard library) |
| Database | TBD — verify in `app/database.go` after migration |
| Testing | Go `testing` package |

---

## Prerequisites

- Go 1.21 or later
- Node.js / npm (detected in setup commands — verify whether this is actually required for this project)
- A running database instance (check `app/database.go` for the expected engine and connection details)

---

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/abdullaharshadd/book-catalog-api.git
cd book-catalog-api
```

### 2. Install dependencies

The setup plan detected the following install command. Verify whether this applies to the Go project or is a leftover from the migration tooling:

```bash
npm install
```

For Go dependencies:

```bash
go mod tidy
```

### 3. Environment variables

No environment variables were detected automatically. However, you should verify the following files for any hardcoded configuration that should be extracted into environment variables:

- `app/database.go` — database connection string
- `app/main.go` — server host/port

Copy and populate a `.env` file if one is provided, or set variables directly in your shell before running.

### 4. Database setup

No database setup command was detected during migration. You will need to manually:

1. Create the database schema. Check `app/models.go` for table definitions.
2. Run any required migrations or DDL scripts manually.

```bash
# Example — replace with the actual command once you have reviewed app/database.go
# psql -U youruser -d yourdatabase -f schema.sql
```

### 5. Run the server

No run command was detected. Once you have verified `app/main.go`, the standard Go run command is:

```bash
go run ./app/main.go
```

Or build and execute:

```bash
go build -o book-catalog-api ./app/main.go
./book-catalog-api
```

---

## Running Tests

No test command was detected during migration. The standard Go test command is:

```bash
go test ./...
```

For verbose output:

```bash
go test -v ./...
```

> ⚠️ The test suite requires manual rewriting before it will run. See [Known Limitations](#known-limitations) and [Manual Review Required](#manual-review-required) for details.

---

## Environment Variables

No environment variables were automatically detected. This table will need to be populated after reviewing the migrated source files.

| Variable | Required | Default | Description |
|---|---|---|---|
| *(none detected)* | — | — | Review `app/database.go` and `app/main.go` |

---

## Architecture Overview

The migrated Go project follows the module structure below, derived from the original Django application layout:

```
book-catalog-api/
├── app/
│   ├── main.go         # Entry point, HTTP server setup, route registration
│   ├── database.go     # Database connection and initialization
│   ├── models.go       # Data model structs (migrated from Django ORM models)
│   └── schemas.go      # Request/response structs (migrated from Pydantic schemas)
├── tests/
│   ├── test_api.go     # API-level integration tests
│   ├── test_models.go  # Model unit tests
│   └── test_schemas.go # Schema/validation unit tests
├── conftest.go         # Test fixtures and setup (partially unmigrable — see below)
├── go.mod
└── go.sum
```

**Request flow:**

```
HTTP Request → net/http ServeMux (main.go) → Handler function → models.go/database.go → Response
```

---

## Migration Notes

This project was automatically migrated from **Python/Django** to **Go/standard library**.

### What changed

| Concern | Original (Django) | Migrated (Go) |
|---|---|---|
| Web framework | Django views / URL conf | `net/http` handlers and `ServeMux` |
| ORM | Django ORM (`models.Model`) | Manual struct definitions in `models.go` |
| Serialization | Django REST Framework serializers or Pydantic schemas | Go structs with `encoding/json` |
| Database sessions | Django ORM queryset / session management | Direct DB calls in `database.go` |
| Test client | `django.test.Client` or FastAPI `TestClient` | Go `net/http/httptest` |
| Test DB setup | `pytest-django` `@pytest.mark.django_db` | Requires manual implementation (see below) |
| Dependency injection | FastAPI `Depends()` + `dependency_overrides` | Not applicable in Go standard library |
| Configuration | Django `settings.py` | Environment variables or config struct in `main.go` |

> **Note:** The migration metadata references both Django and FastAPI/SQLAlchemy patterns in the source. Review `app/main.go` and `app/database.go` carefully — the original project may have been partially FastAPI-based rather than pure Django.

---

## Known Limitations

The following components could not be fully migrated and require manual implementation:

### 1. `conftest.py` → `conftest.go` — `db_session_fixture`

**Reason:** The original fixture used SQLAlchemy's `Base.metadata.create_all` / `drop_all` and `sessionmaker` to spin up and tear down a test database. Neither SQLAlchemy nor Django's test transaction rollback mechanism exists in Go.

**Required action:** Implement test database setup and teardown manually in Go. Options:
- Use a dedicated test database and run schema creation in `TestMain`.
- Use a library such as [`testcontainers-go`](https://github.com/testcontainers/testcontainers-go) for isolated DB containers per test run.
- Wrap tests in explicit transactions and roll back after each test.

```go
// Example pattern in tests/main_test.go
func TestMain(m *testing.M) {
    // setup: create schema
    // m.Run()
    // teardown: drop schema
}
```

---

### 2. `conftest.py` → `conftest.go` — `client_fixture`

**Reason:** The original fixture used FastAPI's `app.dependency_overrides` to inject a test database session into the running application. This mechanism does not exist in Go's standard library.

**Required action:** Replace with Go's `net/http/httptest` package. Since Go handlers receive dependencies via closure or a shared struct rather than injection, wire the test DB connection directly at handler construction time.

```go
// Example
ts := httptest.NewServer(NewRouter(testDB))
defer ts.Close()
```

---

### 3. `tests/test_api.py` → `tests/test_api.go` — `test_db` fixture

**Reason:** The fixture created a SQLAlchemy engine and overrode FastAPI dependencies. Neither construct exists in the target stack.

**Required action:** Rewrite all API tests using `net/http/httptest`. Create a helper that returns an `*httptest.Server` wired to a test database, and use it in place of the original `TestClient`.

---

## Manual Review Required

All 10 migrated modules have **low confidence** and must be reviewed by a developer before this codebase is used. The table below lists each file, its risk level, and what to check.

| File | Risk | What to verify |
|---|---|---|
| `app/main.go` | 🔴 High | Route registration, middleware, server startup, port configuration |
| `app/database.go` | 🔴 High | Connection string, driver used, connection pool settings, error handling |
| `app/models.go` | 🔴 High | All struct fields match the original schema; data types are correct; no fields dropped |
| `app/schemas.go` | 🔴 High | JSON tags match expected API contract; validation logic preserved |
| `tests/test_api.go` | 🔴 High | Tests will not compile or run until fixtures are rewritten (see Known Limitations) |
| `conftest.go` | 🔴 High | Entirely unmigrable — must be rewritten from scratch |
| `tests/test_models.go` | 🟡 Medium | Verify test logic matches original; DB fixture dependency |
| `tests/test_schemas.go` | 🟡 Medium | Verify validation edge cases are preserved |
| `app/__init__.py` (migrated) | 🟡 Medium | Confirm any package-level initialization was carried over |
| `tests/__init__.py` (migrated) | 🟡 Medium | Confirm no test helpers were silently dropped |

### Recommended review order

1. `app/database.go` — nothing else works without a valid DB connection.
2. `app/models.go` — confirm the schema is correct.
3. `app/main.go` — verify routes match the original API surface.
4. `app/schemas.go` — verify the JSON contract is intact.
5. Rewrite `conftest.go` test fixtures.
6. `tests/test_api.go` — rewrite using `httptest`.
7. `tests/test_models.go`, `tests/test_schemas.go` — update to use new fixtures.

---

## Contributing

Because migration confidence is 0%, please do not merge any changes to this repository until a full manual review has been completed and the test suite passes with `go test ./...`.