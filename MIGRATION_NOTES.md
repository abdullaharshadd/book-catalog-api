# Migration Notes

**Overall confidence:** 0%  
**Recommendation:** REVIEW RECOMMENDED

---

## What was migrated

- `app/__init__.py` → `internal/version.go` (84% confidence) ⚠️ needs review
- `app/database.py` → `internal/database.go` (58% confidence) ⚠️ needs review
- `app/models.py` → `internal/model.go` (78% confidence) ⚠️ needs review
- `app/schemas.py` → `internal/schemas.go` (70% confidence) ⚠️ needs review
- `app/main.py` → `internal/main.go` (78% confidence) ⚠️ needs review
- `tests/__init__.py` → `internal/tests/__init__.go` (84% confidence) ⚠️ needs review
- `conftest.py` → `internal/conftest.go` (55% confidence) ⚠️ needs review
- `tests/test_api.py` → `internal/tests/test_api.go` (0% confidence) ⚠️ needs review
- `tests/test_models.py` → `internal/tests/test_models.go` (78% confidence) ⚠️ needs review
- `tests/test_schemas.py` → `internal/tests/test_schemas.go` (56% confidence) ⚠️ needs review

## Components that could not be automatically migrated

These components require manual implementation. The migrated code contains
`MIGRATION_NOTE` comments at the relevant locations.

### `db_session_fixture` in `conftest.py`
**Reason:** Uses SQLAlchemy's Base.metadata.create_all/drop_all and sessionmaker, which have no direct Django ORM equivalent; Django manages schema via migrations and test transactions.
**Suggestion:** Rewrite using pytest-django's @pytest.mark.django_db and the built-in db fixture, letting Django handle test DB setup/teardown and transaction rollback.

### `client_fixture` in `conftest.py`
**Reason:** Relies on FastAPI/Starlette's app.dependency_overrides and TestClient, which do not exist in Django.
**Suggestion:** Use Django's test Client (django.test.Client) or DRF's APIClient via the pytest-django 'client' fixture; remove dependency override logic entirely since Django does not use injected DB sessions.

### `test_db fixture (SQLAlchemy engine + dependency_overrides)` in `tests/test_api.py`
**Reason:** Uses SQLAlchemy engine/session and FastAPI's app.dependency_overrides mechanism, which have no direct equivalent in Django; the DB fixture must be re-implemented using the target framework's test DB setup.
**Suggestion:** Manual rewrite using pytest-django's db fixture or Django TestCase, and the target framework's test client. If keeping FastAPI, no migration needed.

## Observer agent findings

The Observer agent monitored the migration and identified these patterns:

- **After 3 modules:** Could not parse observer output
- **After 6 modules:** Could not parse observer output
- **After 9 modules:** Could not parse observer output

## Files requiring manual review

These files were migrated but scored below the confidence threshold.
Review them carefully before merging.

### `app/__init__.py`
Confidence: 84%

### `app/database.py`
Confidence: 58%
Issues:
  - [critical] The Target Expert concedes the concern is valid but the proposed fix (Migrate method, renamed Ping, embedded migrations/*.sql, wired startup) is presented as a plan and has not been applied to the actual code. InitDB still only pings, is still misnamed, and no migrations directory or SQL files exist in the codebase. The schema guarantee remains absent.

### `app/models.py`
Confidence: 78%

### `app/schemas.py`
Confidence: 70%
Issues:
  - [warning] The Target Expert concedes this is a real bug and describes the correct fix (utf8.RuneCountInString), but presents it as a proposed change rather than committed code. The divergence remains until the fix is actually applied to all five length checks.

### `app/main.py`
Confidence: 78%

### `tests/__init__.py`
Confidence: 84%

### `conftest.py`
Confidence: 55%
Issues:
  - [critical] NewTestDB calls getTestDSN(t), which is not defined in the file, and there is no os import. The file will not compile as migrated. The Target Expert acknowledged this but did not actually apply the fix in the submitted code.

### `tests/test_api.py`
Confidence: 0%
Issues:
  - [critical] The Target Expert conceded the delivered file is genuinely truncated mid-TestUpdateBookNotFound and will not compile. Their proposed fix itself is also truncated (ends at 'srv, cleanup := setup'), so the missing tests (test_create_duplicate_book, test_books_same_title_different_authors, test_full_crud_workflow) are still not fully delivered as compilable code.
  - [critical] Original Python asserts 400 with 'already exists'; no verified Go counterpart is present in delivered code (the fix cuts off before implementing it).
  - [warning] Original asserts both books created (201) with distinct ids; no Go counterpart delivered or verified.

### `tests/test_models.py`
Confidence: 78%

### `tests/test_schemas.py`
Confidence: 56%
Issues:
  - [warning] The 'missing published_year' subcase relies on the Go zero value (0) for PublishedYear triggering a year-range validation error. This depends on internal.Validate treating 0 as invalid; if the validator only checks 'must be after 1000' the year 0 would fail that rule, but the assertion looks for the substring 'published' which may not appear in a 'must be after year 1000' message.
  - [info] The 'missing id raises ValidationError' behavior is not migrated. Go's int zero value is valid and the struct has no construction-time validation, so the required-id invariant is not enforced. The test was replaced with a zero-value assertion that documents different semantics.
