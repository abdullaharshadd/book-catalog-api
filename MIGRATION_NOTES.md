# Migration Notes

**Overall confidence:** 0%  
**Recommendation:** REVIEW RECOMMENDED

---

## What was migrated

- `app/__init__.py` → `src/app/__init.ts` (84% confidence) ⚠️ needs review
- `app/database.py` → `src/app/database.ts` (78% confidence) ⚠️ needs review
- `app/models.py` → `src/app/models.ts` (79% confidence) ⚠️ needs review
- `app/schemas.py` → `src/app/schemas.ts` (78% confidence) ⚠️ needs review
- `app/main.py` → `src/app/main.ts` (60% confidence) ⚠️ needs review
- `tests/__init__.py` → `src/tests/__init.ts` (84% confidence) ⚠️ needs review
- `conftest.py` → `src/conftest.ts` (78% confidence) ⚠️ needs review
- `tests/test_api.py` → `src/tests/test_api.ts` (89% confidence)
- `tests/test_models.py` → `src/tests/test_models.ts` (78% confidence) ⚠️ needs review
- `tests/test_schemas.py` → `src/tests/test_schemas.ts` (84% confidence) ⚠️ needs review

## Components that could not be automatically migrated

These components require manual implementation. The migrated code contains
`MIGRATION_NOTE` comments at the relevant locations.

### `client_fixture` in `conftest.py`
**Reason:** Relies on FastAPI/Starlette-specific TestClient and app.dependency_overrides mechanism, which have no direct 1:1 equivalent in Django or other frameworks.
**Suggestion:** Manually rewrite using the target framework's test client (e.g., Django's pytest-django + APIClient, or the target stack's native test harness) and its own dependency/override or mocking approach.

### `db_session_fixture` in `conftest.py`
**Reason:** SQLAlchemy-specific engine/sessionmaker and Base.metadata table lifecycle; if migrating to a different ORM (e.g., Django ORM) the session concept and table creation differ fundamentally.
**Suggestion:** Rewrite using the target ORM's test database fixtures (e.g., Django's @pytest.mark.django_db and transactional test DB handling) instead of manual metadata create/drop.

### `Error message string assertions` in `tests/test_schemas.py`
**Reason:** Assertions rely on exact Pydantic v1 default validation messages which will not transfer to another validation framework or even Pydantic v2.
**Suggestion:** Update assertions to match the target framework's error message format after migrating the schemas; keep the intent (max length, non-empty, year range) rather than the literal strings.

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
Confidence: 78%

### `app/models.py`
Confidence: 79%

### `app/schemas.py`
Confidence: 78%

### `app/main.py`
Confidence: 60%
Issues:
  - [warning] The Expert correctly diagnosed the gap and proposed a sound fix (parseIntParam returning 422), but this is a proposal — the response does not confirm the fix was actually applied to all three routes in the delivered code. The 422 body shape may also differ from FastAPI's structured validation-error body, though the reviewer's own contract only requires the {detail: ...} wire shape.

### `tests/__init__.py`
Confidence: 84%

### `conftest.py`
Confidence: 78%

### `tests/test_models.py`
Confidence: 78%

### `tests/test_schemas.py`
Confidence: 84%
