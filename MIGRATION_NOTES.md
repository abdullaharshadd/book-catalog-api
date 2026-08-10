# Migration Notes

**Overall confidence:** 0%  
**Recommendation:** REVIEW RECOMMENDED

---

## What was migrated

- `app/__init__.py` → `internal/__init__.go` (83% confidence) ⚠️ needs review
- `app/database.py` → `internal/database.go` (58% confidence) ⚠️ needs review
- `app/models.py` → `internal/model.go` (85% confidence)
- `app/schemas.py` → `internal/schemas.go` (78% confidence) ⚠️ needs review
- `app/main.py` → `internal/main.go` (66% confidence) ⚠️ needs review
- `tests/__init__.py` → `internal/tests/__init__.go` (85% confidence)
- `conftest.py` → `internal/conftest.go` (64% confidence) ⚠️ needs review
- `tests/test_api.py` → `internal/tests/test_api.go` (58% confidence) ⚠️ needs review
- `tests/test_models.py` → `internal/tests/test_models.go` (43% confidence) ⚠️ needs review
- `tests/test_schemas.py` → `internal/tests/test_schemas.go` (58% confidence) ⚠️ needs review

## Components that could not be automatically migrated

These components require manual implementation. The migrated code contains
`MIGRATION_NOTE` comments at the relevant locations.

### `get_db / get_sync_db` in `app/database.py`
**Reason:** The yield-based dependency-injection session generators are a FastAPI/SQLAlchemy idiom with no Django counterpart; Django manages the request-scoped connection lifecycle internally.
**Suggestion:** If target is Django, remove these and rely on Django's ORM connection handling; if target keeps FastAPI/SQLAlchemy, migrate as-is.

### `init_db (Base.metadata.create_all)` in `app/database.py`
**Reason:** Direct schema creation from SQLAlchemy metadata does not map to Django's migration system.
**Suggestion:** In Django, replace with model definitions plus makemigrations/migrate; do not call create_all manually.

### `client_fixture (app.dependency_overrides)` in `conftest.py`
**Reason:** FastAPI's dependency_overrides mechanism has no direct Django counterpart; Django does not use dependency injection for DB sessions.
**Suggestion:** If migrating to Django, replace with pytest-django's client/db fixtures and Django's test database machinery; the override logic should be dropped, not translated.

### `db_session_fixture (SQLAlchemy engine/session)` in `conftest.py`
**Reason:** Uses SQLAlchemy engine/sessionmaker and manual metadata create_all/drop_all, which do not map to Django's ORM or test DB lifecycle.
**Suggestion:** Manual rewrite: use Django's ORM with pytest-django's @pytest.mark.django_db and settings.DATABASES for the test DB, letting the test runner manage table creation/teardown.

### `test_db fixture (app.dependency_overrides / SQLAlchemy engine setup)` in `tests/test_api.py`
**Reason:** The dependency override mechanism and SQLAlchemy StaticPool in-memory setup are FastAPI/SQLAlchemy-specific and have no direct equivalent in other frameworks.
**Suggestion:** Manually rewrite using the target framework's test database configuration and dependency/bean override (e.g. @DataJpaTest or test profile in Spring, jest setup with a test DB in Node).

### `client fixture (fastapi.testclient.TestClient)` in `tests/test_api.py`
**Reason:** TestClient is a FastAPI/Starlette-specific in-process HTTP test client.
**Suggestion:** Replace with the target stack's equivalent HTTP test client (MockMvc/TestRestTemplate for Spring, supertest for Express, etc.).

### `db_session fixture` in `tests/test_models.py`
**Reason:** Directly instantiates SQLAlchemy engine/sessionmaker which has no direct Django equivalent; Django manages test DB setup differently.
**Suggestion:** Replace with pytest-django's db fixture or Django TestCase, and remove manual engine/session creation since Django handles transaction rollback per test.

### `SQLAlchemy session operations (add/commit/refresh)` in `tests/test_models.py`
**Reason:** SQLAlchemy Unit-of-Work API differs from Django's active-record style ORM.
**Suggestion:** Rewrite as Book.objects.create(...) or book.save(); replace refresh() with book.refresh_from_db() only if needed.

### `Validation error message string assertions` in `tests/test_schemas.py`
**Reason:** Assertions rely on exact Pydantic v1 error message text which will not match another validation library or Pydantic v2
**Suggestion:** Manually rewrite each assertion to match the target validation library's error output, or assert on error field/type rather than message text

## Observer agent findings

The Observer agent monitored the migration and identified these patterns:

- **After 3 modules:** Could not parse observer output
- **After 6 modules:** Could not parse observer output
- **After 9 modules:** Could not parse observer output

## Files requiring manual review

These files were migrated but scored below the confidence threshold.
Review them carefully before merging.

### `app/__init__.py`
Confidence: 83%

### `app/database.py`
Confidence: 58%
Issues:
  - [critical] The current InitSchema still ships fabricated DDL (invented books table/columns) that was never derived from app/models.py. The Target Expert concedes this but the code has not actually been changed — the proposed fixes (obtain models.py, fail-loud stub, or migration tool) are described, not applied.

### `app/schemas.py`
Confidence: 78%

### `app/main.py`
Confidence: 66%
Issues:
  - [warning] The migration returns 400 for both JSON-decode failures and Validate() failures, whereas FastAPI returns 422 for both. The Target Expert conceded this is a real divergence but has not applied the fix in code — it remains unresolved pending the described change.
  - [info] Even after fixing the status code, FastAPI emits a structured list body {"detail": [{"loc":..., "msg":..., "type":...}]} while writeError emits {"detail": "<string>"}. This is a genuine body-shape divergence, though it only matters for clients that parse the validation error payload.

### `conftest.py`
Confidence: 64%
Issues:
  - [warning] The helper file assumes methods like NewDB, InitSchema, Close, and either pool()/execContext or ExecContext exist on the target DB type. The Expert concedes the execContext/pool() adapter is fragile and should be removed, but the correct replacement accessor is still unverified against internal/database.go.
  - [warning] As-shipped, dropTestSchema only drops the 'books' table, which is asymmetric with the source's Base.metadata.drop_all and leaks any other created tables across runs. The Expert proposes correct fixes (DropSchema mirror or DROP SCHEMA public CASCADE) but has not committed the change in-code.

### `tests/test_api.py`
Confidence: 58%
Issues:
  - [critical] The Target Expert concedes the entire gap is real and provides only a partial, truncated fix (TestReadRoot, TestHealthCheck, TestCreateBook, and an incomplete TestCreateBookWithoutSummary). The full ~18 source test functions covering update, delete, validation, and duplicate behavior are not shown as completed, and the fix references undefined helpers (app.NewTestDB, app.NewHandlers, newRouter) that are not verified to exist.

### `tests/test_models.py`
Confidence: 43%
Issues:
  - [critical] The original migrated file (tests/test_models.py) still contains zero test functions and zero assertions. The Target Expert concedes this and proposes a NEW sibling file (test_models_test.go), but that file was not actually delivered as part of the migration under review — it is only shown as a proposed fix and is even truncated mid-statement.
  - [critical] Because the runnable tests are only proposed (and incomplete/truncated), none of the source's validated behaviors are actually exercised in the delivered migration.

### `tests/test_schemas.py`
Confidence: 58%
Issues:
  - [critical] The Target Expert concedes the delivered migration contains no executable tests, and the 'fix' provided is truncated mid-function (TestEmptyTitleValidation cuts off) and was never actually integrated into the delivered file. Coverage remains unpreserved as delivered.
