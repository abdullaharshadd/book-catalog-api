# Migration Notes

**Overall confidence:** 0%  
**Recommendation:** REVIEW RECOMMENDED

---

## What was migrated

- `app/__init__.py` → `internal/doc.go` (85% confidence)
- `app/database.py` → `internal/database.go` (64% confidence) ⚠️ needs review
- `app/models.py` → `internal/model.go` (90% confidence)
- `app/schemas.py` → `internal/schemas.go` (70% confidence) ⚠️ needs review
- `app/main.py` → `internal/main.go` (60% confidence) ⚠️ needs review
- `tests/__init__.py` → `internal/tests/__init__.go` (97% confidence)
- `conftest.py` → `internal/conftest.go` (50% confidence) ⚠️ needs review
- `tests/test_api.py` → `internal/tests/test_api.go` (0% confidence) ⚠️ needs review
- `tests/test_models.py` → `internal/tests/test_models.go` (78% confidence) ⚠️ needs review
- `tests/test_schemas.py` → `internal/tests/test_schemas.go` (80% confidence) ⚠️ needs review

## Components that could not be automatically migrated

These components require manual implementation. The migrated code contains
`MIGRATION_NOTE` comments at the relevant locations.

### `async_engine / async_session / get_db` in `app/database.py`
**Reason:** Django's ORM does not support async sessions in the same SQLAlchemy sense; there is no direct equivalent to an async SQLAlchemy AsyncSession dependency generator.
**Suggestion:** For Django, drop these and use Django's synchronous ORM (or sync_to_async wrappers for async views). To retain async SQLAlchemy behavior, keep SQLAlchemy rather than migrating to the Django ORM.

### `init_db (Base.metadata.create_all)` in `app/database.py`
**Reason:** Django uses a migration system rather than programmatic table creation from metadata.
**Suggestion:** Replace with Django migrations; run manage.py makemigrations and migrate instead of a runtime init function.

### `client_fixture / app.dependency_overrides` in `conftest.py`
**Reason:** FastAPI/Starlette dependency injection and override mechanism has no direct equivalent in Django, whose ORM uses a global connection rather than per-request injected sessions.
**Suggestion:** Manually rewrite using pytest-django fixtures: use the 'db' fixture (or @pytest.mark.django_db) for DB isolation and rest_framework.test.APIClient (or django.test.Client) for the test client. Remove dependency override logic entirely.

### `db_session_fixture (SQLAlchemy session)` in `conftest.py`
**Reason:** SQLAlchemy sessionmaker, engine, and Base.metadata table management are ORM-specific and do not translate directly to Django's ORM and migration-based schema handling.
**Suggestion:** Replace with pytest-django's transactional test database handling; Django creates/destroys the test DB automatically and wraps each test in a transaction, so explicit session creation and table create/drop are unnecessary.

### `test_db fixture (SQLAlchemy engine/session setup)` in `tests/test_api.py`
**Reason:** Tightly coupled to SQLAlchemy engine, StaticPool, sessionmaker, and FastAPI's dependency_overrides which have no direct Django equivalent.
**Suggestion:** Rewrite using Django's test database fixtures (pytest-django's @pytest.mark.django_db or Django TestCase) rather than SQLAlchemy in-memory SQLite.

### `client fixture (FastAPI TestClient)` in `tests/test_api.py`
**Reason:** FastAPI TestClient is not usable against a Django app.
**Suggestion:** Replace with DRF's APIClient or Django's test Client; adjust JSON assertion patterns accordingly.

### `HTTP status code assertions (201/204/422)` in `tests/test_api.py`
**Reason:** DRF default conventions differ (e.g., 204 vs 200 for delete, 400 vs 422 for validation errors).
**Suggestion:** Manually reconcile expected status codes to match DRF defaults or configure custom exception handlers to preserve FastAPI-style codes.

### `db_session fixture` in `tests/test_models.py`
**Reason:** SQLAlchemy engine/sessionmaker setup is framework-specific and does not translate directly to Django's test database machinery
**Suggestion:** Replace with Django's pytest-django 'db' fixture or override with a TestCase; remove manual engine/session creation and use Django's ORM manager methods.

### `Book(...) + db_session.add/commit/refresh calls` in `tests/test_models.py`
**Reason:** SQLAlchemy session lifecycle differs fundamentally from Django ORM's active-record style
**Suggestion:** Manually rewrite to Book.objects.create(**fields) and assert on the returned instance; use assertRaises(IntegrityError) for the unique constraint test.

### `Pydantic ValidationError message assertions` in `tests/test_schemas.py`
**Reason:** Assertions depend on exact Pydantic v1 error message text which does not exist verbatim in Django/DRF or Pydantic v2.
**Suggestion:** Manually rewrite assertions to match target framework's validation error format (e.g. DRF serializer.errors dict keys/messages or Pydantic v2 error strings).

## Observer agent findings

The Observer agent monitored the migration and identified these patterns:

- **After 3 modules:** Could not parse observer output
- **After 6 modules:** Could not parse observer output
- **After 9 modules:** Could not parse observer output

## Files requiring manual review

These files were migrated but scored below the confidence threshold.
Review them carefully before merging.

### `app/database.py`
Confidence: 64%
Issues:
  - [warning] The Target Expert concedes the concern and proposes a correct DSNFromEnv()/DefaultDSN helper, but this is still only proposed in the response — the reviewed file itself does not yet contain the env read. Until the code is actually amended, the behavior remains absent from this file.
  - [warning] Conceded as a real omission. The proposed fix is only a sqlEchoEnabled() flag and a description of a logging wrapper — no actual logging is emitted in the code yet. The observable 'log all SQL' behavior is still gone until a real logging middleware (e.g. sqldblogger/pgx tracer) is wired in.

### `app/schemas.py`
Confidence: 70%
Issues:
  - [info] Length checks for title/author are applied to the TRIMMED value in Go (strings.TrimSpace then len), whereas the Python source applies the max-255 length check to the UNTRIMMED input value (validates `len(v)` before returning `v.strip()`). This is an explicitly stated invariant in the spec.
  - [warning] Go uses byte-length (len on string) rather than character-count for the 255/2000 limits. For multibyte (non-ASCII) input this differs from Python's len() which counts Unicode code points. Real-world titles with accented/CJK characters could be rejected earlier than in Python.

### `app/main.py`
Confidence: 60%
Issues:
  - [warning] The equivalence of update_book depends entirely on ApplyTo in schemas.go implementing true presence tracking (distinguishing absent vs zero-value fields). This is not verifiable from main.go, and the Target Expert correctly concedes this is the real boundary. If ApplyTo naively copies struct fields, omitting a field like summary would blank it, diverging from Python's exclude_unset behavior.
  - [info] The 400-vs-500 mapping is correct only if the migrated DDL declares UNIQUE (title, author). This is a database.go/DDL concern not verifiable from main.go. If absent, duplicate inserts silently succeed. Additionally, the substring fallback matching 'unique constraint' is broad and could mis-map non-target constraint violations to 400.

### `conftest.py`
Confidence: 50%
Issues:
  - [warning] The migrated file is a comment-only placeholder that promises helpers 'below' which do not exist, so it delivers no working scaffolding. The Target Expert acknowledges this defect but has not actually applied the fix — the corrected code is only shown as a proposed patch in the response, not integrated into the artifact.

### `tests/test_api.py`
Confidence: 0%
Issues:
  - [critical] The 'complete corrected file' is again truncated mid-statement inside TestUpdateBook (ends at 'Title:'). The full test file still cannot be verified to compile or contain all required tests.
  - [critical] TestUpdateBook is left incomplete and any tests that follow it (delete, validation, etc.) are still absent from the delivered response.
  - [warning] These helpers are referenced but their definitions are not confirmed present anywhere in the truncated artifact, so undefined-symbol errors cannot be ruled out.

### `tests/test_models.py`
Confidence: 78%

### `tests/test_schemas.py`
Confidence: 80%
