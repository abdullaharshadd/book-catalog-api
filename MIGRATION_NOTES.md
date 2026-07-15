# Migration Notes

**Overall confidence:** 0%  
**Recommendation:** REVIEW RECOMMENDED

---

## What was migrated

- `app/__init__.py` → `internal/catalog.go` (10% confidence) ⚠️ needs review
- `app/database.py` → `internal/database.py.go` (10% confidence) ⚠️ needs review
- `app/main.py` → `internal/main.py.go` (69% confidence) ⚠️ needs review
- `app/models.py` → `internal/models.py.go` (10% confidence) ⚠️ needs review
- `app/schemas.py` → `internal/schemas.py.go` (75% confidence) ⚠️ needs review
- `tests/__init__.py` → `internal/tests/__init__.py.go` (85% confidence)
- `conftest.py` → `internal/conftest.py.go` (10% confidence) ⚠️ needs review
- `tests/test_api.py` → `internal/tests/test_api.py.go` (43% confidence) ⚠️ needs review
- `tests/test_models.py` → `internal/tests/test_models.py.go` (10% confidence) ⚠️ needs review
- `tests/test_schemas.py` → `internal/tests/test_schemas.py.go` (64% confidence) ⚠️ needs review

## Components that could not be automatically migrated

These components require manual implementation. The migrated code contains
`MIGRATION_NOTE` comments at the relevant locations.

### `client_fixture (app.dependency_overrides / get_db override)` in `conftest.py`
**Reason:** FastAPI/Starlette dependency-injection override mechanism has no direct Django equivalent; Django manages test DB and request handling differently.
**Suggestion:** Manually rewrite using pytest-django fixtures (db, client) or DRF's APIClient; remove the dependency_overrides logic entirely.

### `db_session_fixture (SQLAlchemy engine/session + create_all/drop_all)` in `conftest.py`
**Reason:** Uses SQLAlchemy session and manual schema create/drop, which is replaced by Django's ORM and automatic test database lifecycle.
**Suggestion:** Replace with pytest-django's built-in database fixtures (@pytest.mark.django_db or the db fixture); let Django's test runner handle table creation/teardown via migrations.

### `test_db fixture (app.dependency_overrides / get_sync_db)` in `tests/test_api.py`
**Reason:** FastAPI's dependency_overrides mechanism is framework-specific and has no direct equivalent in Django or other frameworks; it relies on FastAPI's dependency injection system.
**Suggestion:** If keeping FastAPI, migrate directly. If moving to Django, rewrite using Django's TestCase/APITestCase with a test database configured in settings and Django's ORM-based fixtures; the dependency override is replaced by Django's automatic test DB creation.

### `TestClient (fastapi.testclient)` in `tests/test_api.py`
**Reason:** FastAPI's TestClient (built on Starlette/httpx) is tied to the ASGI FastAPI app instance.
**Suggestion:** For a Django target, replace with Django REST Framework's APIClient or Django's test Client. For staying on FastAPI, keep as-is.

### `db_session fixture` in `tests/test_models.py`
**Reason:** Uses SQLAlchemy-specific engine and sessionmaker APIs that have no direct 1:1 Django equivalent.
**Suggestion:** Replace with pytest-django's @pytest.mark.django_db decorator and Django's test database, or keep SQLAlchemy if the target stack is FastAPI/SQLAlchemy rather than Django.

### `Session-based persistence calls (add/commit/refresh)` in `tests/test_models.py`
**Reason:** SQLAlchemy session pattern differs fundamentally from Django's active-record ORM.
**Suggestion:** Manual rewrite: replace db_session.add(book); db_session.commit(); db_session.refresh(book) with Book.objects.create(...) or book.save() in Django.

### `ValidationError message string assertions` in `tests/test_schemas.py`
**Reason:** Pydantic-specific error message text (e.g. 'ensure this value has at most 255 characters') will not match DRF's error output, so assertions relying on exact strings will break.
**Suggestion:** Manually rewrite assertions to match DRF ValidationError message formats after converting schemas to serializers.

### `BookCreate/BookUpdate/BookResponse (via app.schemas)` in `tests/test_schemas.py`
**Reason:** These are Pydantic models, not Django ORM/DRF constructs; the tests depend on Pydantic's constructor-based validation API which differs from DRF's serializer(data=...).is_valid() flow.
**Suggestion:** Either keep Pydantic (recommended if the app is actually FastAPI) or manually rewrite as DRF serializers and adapt the test invocation pattern accordingly.

## Observer agent findings

The Observer agent monitored the migration and identified these patterns:

- **After 3 modules:** Could not parse observer output
- **After 6 modules:** Could not parse observer output
- **After 9 modules:** Could not parse observer output

## Files requiring manual review

These files were migrated but scored below the confidence threshold.
Review them carefully before merging.

### `app/__init__.py`
Confidence: 10%

### `app/database.py`
Confidence: 10%

### `app/main.py`
Confidence: 69%
Issues:
  - [warning] Schema validation (Pydantic BookCreate requiring title/author/published_year) is not enforced. Go json.Decode succeeds even with missing required fields, so a payload without title/author would produce a 500 (or a book with empty fields) rather than the 422 validation error the original returns.
  - [info] A non-numeric book_id path value returns 404 in the migration. In FastAPI a non-int path param would return 422. This is a minor edge-case difference; for real numeric IDs the behavior is correct.

### `app/models.py`
Confidence: 10%

### `app/schemas.py`
Confidence: 75%
Issues:
  - [info] The Python title/author length check uses len() on the string before trimming for the 255 limit? Actually Python checks len(v) then returns v.strip(). But since str_strip_whitespace=True strips before the validator runs, the original effectively checks trimmed length too. The Go code trims then checks length — behaviorally equivalent for normal inputs. Not a real defect.
  - [warning] Go uses len() which counts bytes, not runes/characters. For multibyte UTF-8 titles/summaries, the length limit would trigger earlier than Python's character-based len(). Edge case with non-ASCII input.

### `conftest.py`
Confidence: 10%

### `tests/test_api.py`
Confidence: 43%
Issues:
  - [warning] The migrated file is truncated mid-way through TestUpdateBookValidationError (ends at 'status, body := doJS'). The delete tests (TestDeleteBook, TestDeleteBookNotFound), duplicate-creation test (TestCreateDuplicateBook), same-title-different-authors test, and full CRUD workflow test are not present in the shown code.

### `tests/test_models.py`
Confidence: 10%

### `tests/test_schemas.py`
Confidence: 64%
Issues:
  - [info] The 'id field missing' test scenario is replaced with t.Skip(). The original test verified that omitting id raises a ValidationError. The Go migration reconstructs BookResponse from a Book model via NewBookResponse, where ID is always present, so the required-id validation behavior is not actually tested. This is an acceptable paradigm difference (in Go a struct field always has a zero value and id-required is enforced by construction), but the behavior is no longer verified.
