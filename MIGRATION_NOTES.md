# Migration Notes

**Overall confidence:** 0%  
**Recommendation:** REVIEW RECOMMENDED

---

## What was migrated

- `app/__init__.py` → `internal/__init__.go` (85% confidence)
- `app/database.py` → `internal/database.go` (4% confidence) ⚠️ needs review
- `app/main.py` → `internal/main.go` (74% confidence) ⚠️ needs review
- `app/models.py` → `internal/model.go` (82% confidence) ⚠️ needs review
- `app/schemas.py` → `internal/schemas.go` (82% confidence) ⚠️ needs review
- `tests/__init__.py` → `internal/tests/__init__.go` (97% confidence)
- `conftest.py` → `internal/conftest.go` (78% confidence) ⚠️ needs review
- `tests/test_api.py` → `internal/tests/test_api.go` (52% confidence) ⚠️ needs review
- `tests/test_models.py` → `internal/tests/test_models.go` (78% confidence) ⚠️ needs review
- `tests/test_schemas.py` → `internal/tests/test_schemas.go` (66% confidence) ⚠️ needs review
## Files requiring manual review

These files were migrated but scored below the confidence threshold.
Review them carefully before merging.

### `app/database.py`
Confidence: 4%
Issues:
  - [critical] The Target Expert concedes the fabricated schema and proposes an error-returning stub, but this is a proposal — the shipped code still contains fabricated DDL until the stub is actually applied. The correct DDL remains blocked on app/models.py.
  - [warning] The proposed DefaultDSN restores env-var handling but introduces an unresolved inconsistency: it returns a SQLite default while the driver is lib/pq (Postgres). This is a genuine open decision the Expert flags but does not resolve.
  - [warning] The Expert concedes WithTx auto-commits whereas the source is caller-managed non-committing. A Session accessor is proposed but the response is truncated and not confirmed as shipped; the behavioral change persists until the non-committing accessor is actually added and callers updated.

### `app/main.py`
Confidence: 74%
Issues:
  - [warning] The Target Expert concedes the shipped code still contains the buggy `if in.Summary != nil` branch. An explicit `{"summary": null}` request fails to clear the column to NULL, diverging from the Python `exclude_unset` semantics. The proposed OptionalString fix is correct but has not been merged into the shipped code.

### `app/models.py`
Confidence: 82%

### `app/schemas.py`
Confidence: 82%

### `conftest.py`
Confidence: 78%

### `tests/test_api.py`
Confidence: 52%
Issues:
  - [critical] The Target Expert's response is itself truncated mid-statement (ends at `buf` inside the `post` helper). The promised missing test functions (testUpdateBook completion, testUpdateBookNotFound, testDeleteBook, etc.) and helpers (numEquals, idKey, idSetsEqual, assertBooksEqual) are still not delivered. The file still does not compile.

### `tests/test_models.py`
Confidence: 78%

### `tests/test_schemas.py`
Confidence: 66%
Issues:
  - [warning] The committed test only asserts Go's zero-value semantics (br.ID != 0), which is a tautology and provides no behavioral coverage of the source's 'id is required' ValidationError invariant. The Target Expert concedes this but the proposed fix (adding BookResponse.Validate() and rewriting the test) is not yet committed.
