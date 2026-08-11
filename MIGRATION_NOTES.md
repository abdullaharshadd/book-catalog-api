# Migration Notes

**Overall confidence:** 0%  
**Recommendation:** REVIEW RECOMMENDED

---

## What was migrated

- `app/__init__.py` → `internal/__init__.go` (84% confidence) ⚠️ needs review
- `app/database.py` → `internal/database.go` (37% confidence) ⚠️ needs review
- `app/main.py` → `internal/main.go` (84% confidence) ⚠️ needs review
- `app/models.py` → `internal/model.go` (78% confidence) ⚠️ needs review
- `app/schemas.py` → `internal/schemas.go` (78% confidence) ⚠️ needs review
- `tests/__init__.py` → `internal/tests/__init__.go` (85% confidence)
- `conftest.py` → `internal/conftest.go` (56% confidence) ⚠️ needs review
- `tests/test_api.py` → `internal/tests/test_api.go` (46% confidence) ⚠️ needs review
- `tests/test_models.py` → `internal/tests/test_models.go` (46% confidence) ⚠️ needs review
- `tests/test_schemas.py` → `internal/tests/test_schemas.go` (44% confidence) ⚠️ needs review
## Files requiring manual review

These files were migrated but scored below the confidence threshold.
Review them carefully before merging.

### `app/__init__.py`
Confidence: 84%

### `app/database.py`
Confidence: 37%
Issues:
  - [critical] The schemaDDL constant remains fabricated — table name, column names, types, nullability, defaults, PK strategy, and constraints were all guessed without access to app/models.py. Repository queries against invented columns will fail at first request.
  - [critical] CREATE TABLE IF NOT EXISTS silently no-ops against a pre-existing table, masking schema drift so startup succeeds but queries fail in production. No fix applied yet — the VerifySchema function is proposed but not implemented.

### `app/main.py`
Confidence: 84%
Issues:
  - [info] FastAPI's RequestValidationError emits {"detail": [ {loc, msg, type}, ... ]} — a structured list — while the migration emits {"detail": "<string>"}. The Target Expert acknowledges this divergence but characterizes it as an accepted paradigm difference. It remains a real body-shape divergence for 422 responses, though low-impact for clients that only check the status code and the 'detail' key.

### `app/models.py`
Confidence: 78%

### `app/schemas.py`
Confidence: 78%

### `conftest.py`
Confidence: 56%
Issues:
  - [warning] The response concedes the structural difference and proposes a sound enumeration-based fix, but the outcome is conditional: it is only correct today if 'Book' is the sole model. The Target Expert explicitly could not confirm this from conftest.py and defers verification to internal/model.go and the source app/ modules. Until that check is done, whether this is a live defect or a hardening improvement is unresolved.

### `tests/test_api.py`
Confidence: 46%
Issues:
  - [critical] The actual Test functions exercising business logic are still not provided; the Target Expert conceded they belong in test_api_test.go but that file was never delivered as part of this migration.
  - [critical] The mustNewTestServer(t) helper (schema creation + DB teardown, analogue of Base.metadata.create_all + fixture cleanup) is referenced but not provided, so even the harness will not compile.

### `tests/test_models.py`
Confidence: 46%
Issues:
  - [critical] The proposed test_models_test.go was not produced; the submitted file still contains only comments with zero executable logic. Field-persistence assertions remain unimplemented.
  - [critical] No executable code asserting summary is nil when omitted; only a proposed snippet exists.
  - [critical] GoString validation not present in any submitted file; proposed snippet also missing the fmt import.

### `tests/test_schemas.py`
Confidence: 44%
Issues:
  - [critical] No runnable tests present; all assertions deferred to an unprovided sibling file. No coverage of valid construction, whitespace stripping, empty-field rejection, year bounds, or length limits.
  - [critical] No UpdateRequest type and no partial-update, empty-update, or shared-validation tests exist.
  - [critical] No BookResponse type and no id-required test exist.
