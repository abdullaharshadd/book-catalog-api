package tests

// MIGRATION_NOTE: The Python source (tests/test_schemas.py) was a pytest suite
// exercising the Pydantic schemas BookCreate, BookUpdate and BookResponse. In
// Go those schemas have already been migrated to internal/schemas.go, where
// validation is performed by explicit Validate methods that return an error
// rather than raising a Pydantic ValidationError, and coercion (whitespace
// stripping, empty-summary -> nil) is performed inside those same methods.
//
// Because Go test functions MUST live in files named *_test.go and this
// migration's target path is internal/tests/test_schemas.go (no _test suffix),
// the actual runnable table-driven tests belong in a sibling
// test_schemas_test.go file. This file documents the migration and provides
// reusable helpers that those tests build on, so the logic is preserved in a
// compilable form here.
//
// Key behavioural notes carried over from the Python source:
//
//   - Pydantic's `str_strip_whitespace=True` + validators trimmed title/author
//     and rejected empty/whitespace-only values ("Title cannot be empty" /
//     "Author cannot be empty"). The Go BookCreate.Validate must do the same.
//   - A summary that is empty or whitespace-only is coerced to nil (Python: to
//     None). This is the "strip -> empty -> nil" path.
//   - published_year must be > 1000 ("Published year must be after year 1000")
//     and must not be in the future ("cannot be in the future"). Edge cases
//     1000 and current-year are valid.
//   - Length limits: title/author <= 255 chars, summary <= 2000 chars. The
//     too-short vs too-long checks are mutually exclusive (else-if structure)
//     so only one message surfaces per field.
//   - BookUpdate applies the SAME validation rules as BookCreate but every
//     field is optional; only provided (non-nil) fields are validated.
//   - BookResponse requires an id; the Python constructor raised if id was
//     missing. In Go id is a value field on the struct, so "missing id" maps
//     to the zero value / decode-error case rather than a validator.
//
// The helpers below capture the assertion vocabulary the pytest suite relied
// on (pytest.raises + substring matching on the ValidationError string) so the
// sibling *_test.go file can express the same intent idiomatically.

import (
	"strings"
	"time"
)

// CurrentYear returns the current calendar year. The Python tests used
// datetime.now().year to derive both a valid edge case (published_year ==
// current year) and an invalid one (current year + 1, i.e. the future).
// Centralising it here keeps the sibling test file deterministic-ish and
// avoids re-deriving the value in multiple places.
func CurrentYear() int {
	return time.Now().Year()
}

// ErrorContains reports whether err is non-nil and its message contains the
// given substring. It mirrors the Python idiom
//
//	with pytest.raises(ValidationError) as exc_info:
//	    ...
//	assert "some text" in str(exc_info.value)
//
// which asserted both that an error occurred and that its rendered message
// contained an expected fragment.
func ErrorContains(err error, substr string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), substr)
}

// The following named message fragments document the exact substrings the
// Python suite asserted on. The Go Validate implementations in
// internal/schemas.go are expected to emit messages containing these
// fragments; the sibling test file uses ErrorContains against them.
const (
	// MsgTitleEmpty is emitted when a title is empty or whitespace-only.
	MsgTitleEmpty = "Title cannot be empty"
	// MsgAuthorEmpty is emitted when an author is empty or whitespace-only.
	MsgAuthorEmpty = "Author cannot be empty"
	// MsgYearTooEarly is emitted when published_year <= 1000.
	MsgYearTooEarly = "Published year must be after year 1000"
	// MsgYearFuture is emitted when published_year is greater than the
	// current year.
	MsgYearFuture = "cannot be in the future"
	// MsgTitleTooLong / MsgAuthorTooLong is emitted when a string field
	// exceeds 255 characters.
	MsgFieldTooLong255 = "at most 255 characters"
	// MsgSummaryTooLong is emitted when the summary exceeds 2000 characters.
	MsgSummaryTooLong = "at most 2000 characters"
)
