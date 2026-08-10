```go
package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Minimal schema types mirroring internal/schemas.go
// ---------------------------------------------------------------------------
// These are defined locally so the test file is self-contained.  In a real
// repository you would import the package under test; replace the type
// definitions and Validate methods below with the real imports.

// BookCreate mirrors the Python Pydantic BookCreate model.
type BookCreate struct {
	Title         string
	Author        string
	PublishedYear int
	Summary       *string // nil means "not provided / whitespace-only"
}

// Validate applies the same rules the Python validators enforced.
func (b *BookCreate) Validate() error {
	// --- title ---
	b.Title = strings.TrimSpace(b.Title)
	if b.Title == "" {
		return newValidationError(MsgTitleEmpty)
	}
	if len(b.Title) > 255 {
		return newValidationError("title: " + MsgFieldTooLong255)
	}

	// --- author ---
	b.Author = strings.TrimSpace(b.Author)
	if b.Author == "" {
		return newValidationError(MsgAuthorEmpty)
	}
	if len(b.Author) > 255 {
		return newValidationError("author: " + MsgFieldTooLong255)
	}

	// --- published_year ---
	if b.PublishedYear <= 1000 {
		return newValidationError(MsgYearTooEarly)
	}
	if b.PublishedYear > time.Now().Year() {
		return newValidationError("published_year " + MsgYearFuture)
	}

	// --- summary ---
	if b.Summary != nil {
		stripped := strings.TrimSpace(*b.Summary)
		if stripped == "" {
			b.Summary = nil
		} else {
			if len(stripped) > 2000 {
				return newValidationError("summary: " + MsgSummaryTooLong)
			}
			b.Summary = &stripped
		}
	}

	return nil
}

// BookUpdate mirrors the Python Pydantic BookUpdate model (all optional).
type BookUpdate struct {
	Title         *string
	Author        *string
	PublishedYear *int
	Summary       *string
}

// Validate validates only the fields that are non-nil.
func (u *BookUpdate) Validate() error {
	if u.Title != nil {
		trimmed := strings.TrimSpace(*u.Title)
		if trimmed == "" {
			return newValidationError(MsgTitleEmpty)
		}
		if len(trimmed) > 255 {
			return newValidationError("title: " + MsgFieldTooLong255)
		}
		u.Title = &trimmed
	}

	if u.Author != nil {
		trimmed := strings.TrimSpace(*u.Author)
		if trimmed == "" {
			return newValidationError(MsgAuthorEmpty)
		}
		if len(trimmed) > 255 {
			return newValidationError("author: " + MsgFieldTooLong255)
		}
		u.Author = &trimmed
	}

	if u.PublishedYear != nil {
		if *u.PublishedYear <= 1000 {
			return newValidationError(MsgYearTooEarly)
		}
		if *u.PublishedYear > time.Now().Year() {
			return newValidationError("published_year " + MsgYearFuture)
		}
	}

	if u.Summary != nil {
		stripped := strings.TrimSpace(*u.Summary)
		if stripped == "" {
			u.Summary = nil
		} else {
			if len(stripped) > 2000 {
				return newValidationError("summary: " + MsgSummaryTooLong)
			}
			u.Summary = &stripped
		}
	}

	return nil
}

// BookResponse mirrors the Python Pydantic BookResponse model.
type BookResponse struct {
	ID            int
	Title         string
	Author        string
	PublishedYear int
	Summary       *string
}

// Validate checks that the required id field is present (non-zero).
func (r *BookResponse) Validate() error {
	if r.ID == 0 {
		return newValidationError("id: field required")
	}
	return nil
}

// validationError is a simple error type used by all Validate methods.
type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

func newValidationError(msg string) error { return &validationError{msg: msg} }

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// ---------------------------------------------------------------------------
// Tests: BookCreate
// ---------------------------------------------------------------------------

func TestBookCreate_ValidCases(t *testing.T) {
	cy := CurrentYear()

	tests := []struct {
		name            string
		input           BookCreate
		wantTitle       string
		wantAuthor      string
		wantYear        int
		wantSummaryNil  bool
		wantSummaryText string
	}{
		{
			name:            "all valid fields",
			input:           BookCreate{Title: "Dune", Author: "Frank Herbert", PublishedYear: 1965, Summary: strPtr("A sci-fi epic")},
			wantTitle:       "Dune",
			wantAuthor:      "Frank Herbert",
			wantYear:        1965,
			wantSummaryText: "A sci-fi epic",
		},
		{
			name:           "summary omitted (nil)",
			input:          BookCreate{Title: "Dune", Author: "Frank Herbert", PublishedYear: 1965, Summary: nil},
			wantTitle:      "Dune",
			wantAuthor:     "Frank Herbert",
			wantYear:       1965,
			wantSummaryNil: true,
		},
		{
			name:            "whitespace stripped from title, author, summary",
			input:           BookCreate{Title: "  Dune  ", Author: "  Frank Herbert  ", PublishedYear: 1965, Summary: strPtr("  epic  ")},
			wantTitle:       "Dune",
			wantAuthor:      "Frank Herbert",
			wantYear:        1965,
			wantSummaryText: "epic",
		},
		{
			name:           "whitespace-only summary becomes nil",
			input:          BookCreate{Title: "Dune", Author: "Frank Herbert", PublishedYear: 1965, Summary: strPtr("   ")},
			wantTitle:      "Dune",
			wantAuthor:     "Frank Herbert",
			wantYear:       1965,
			wantSummaryNil: true,
		},
		{
			name:           "published_year exactly 1000 (boundary)",
			input:          BookCreate{Title: "Old Book", Author: "Author One", PublishedYear: 1000, Summary: nil},
			wantTitle:      "Old Book",
			wantAuthor:     "Author One",
			wantYear:       1000,
			wantSummaryNil: true,
		},
		{
			name:           "published_year equals current year (boundary)",
			input:          BookCreate{Title: "New Book", Author: "Author Two", PublishedYear: cy, Summary: nil},
			wantTitle:      "New Book",
			wantAuthor:     "Author Two",
			wantYear:       cy,
			wantSummaryNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.input // copy so mutations don't bleed
			err := b.Validate()
			require.NoError(t, err)

			assert.Equal(t, tc.wantTitle, b.Title)
			assert.Equal(t, tc.wantAuthor, b.Author)
			assert.Equal(t, tc.wantYear, b.PublishedYear)

			if tc.wantSummaryNil {
				assert.Nil(t, b.Summary)
			} else {
				require.NotNil(t, b.Summary)
				assert.Equal(t, tc.wantSummaryText, *b.Summary)
			}
		})
	}
}

func TestBookCreate_InvalidCases(t *testing.T) {
	cy := CurrentYear()

	tests := []struct {
		name        string
		input       BookCreate
		wantErrFrag string
	}{
		{
			name:        "empty title",
			input:       BookCreate{Title: "", Author: "Frank Herbert", PublishedYear: 1965},
			wantErrFrag: MsgTitleEmpty,
		},
		{
			name:        "whitespace-only title",
			input:       BookCreate{Title: "   ", Author: "Frank Herbert", PublishedYear: 1965},
			wantErrFrag: MsgTitleEmpty,
		},
		{
			name:        "empty author",
			input:       BookCreate{Title: "Dune", Author: "", PublishedYear: 1965},
			wantErrFrag: MsgAuthorEmpty,
		},
		{
			name:        "whitespace-only author",
			input:       BookCreate{Title: "Dune", Author: "   ", PublishedYear: 1965},
			wantErrFrag: MsgAuthorEmpty,
		},
		{
			name:        "published_year 999 (less than 1000)",
			input:       BookCreate{Title: "Dune", Author: "Frank Herbert", PublishedYear: 999},
			wantErrFrag: MsgYearTooEarly,
		},
		{
			name:        "published_year exactly 1001 is OK but 0 is not",
			input:       BookCreate{Title: "Dune", Author: "Frank Herbert", PublishedYear: 0},
			wantErrFrag: MsgYearTooEarly,
		},
		{
			name:        "published_year in the future",
			input:       BookCreate{Title: "Dune", Author: "Frank Herbert", PublishedYear: cy + 1},
			wantErrFrag: MsgYearFuture,
		},
		{
			name:        "title exceeds 255 characters",
			input:       BookCreate{Title: strings.Repeat("a", 256), Author: "Frank Herbert", PublishedYear: 1965},
			wantErrFrag: MsgFieldTooLong255,
		},
		{
			name:        "author exceeds 255 characters",
			input:       BookCreate{Title: "Dune", Author: strings.Repeat("b", 256), PublishedYear: 1965},
			wantErrFrag: MsgFieldTooLong255,
		},
		{
			name:        "summary exceeds 2000 characters",
			input:       BookCreate{Title: "Dune", Author: "Frank Herbert", PublishedYear: 1965, Summary: strPtr(strings.Repeat("x", 2001))},
			wantErrFrag: MsgSummaryTooLong,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.input
			err := b.Validate()
			assert.True(t, ErrorContains(err, tc.wantErrFrag),
				"expected error containing %q, got: %v", tc.wantErrFrag, err)
		})
	}
}

// TestBookCreate_MissingRequiredFields verifies that zero-value fields trigger
// the appropriate validation messages (mirrors "field required" Python cases).
func TestBookCreate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		input       BookCreate
		wantErrFrag string
	}{
		{
			// title zero value -> empty string -> MsgTitleEmpty
			name:        "missing title (zero value)",
			input:       BookCreate{Author: "Frank Herbert", PublishedYear: 1965},
			wantErrFrag: MsgTitleEmpty,
		},
		{
			// author zero value -> empty string -> MsgAuthorEmpty
			name:        "missing author (zero value)",
			input:       BookCreate{Title: "Dune", PublishedYear: 1965},
			wantErrFrag: MsgAuthorEmpty,
		},
		{
			// published_year zero value -> 0 -> MsgYearTooEarly
			name:        "missing published_year (zero value)",
			input:       BookCreate{Title: "Dune", Author: "Frank Herbert"},
			wantErrFrag: MsgYearTooEarly,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.input
			err := b.Validate()
			assert.True(t, ErrorContains(err, tc.wantErrFrag),
				"expected error containing %q, got: %v", tc.wantErrFrag, err)
		})
	}
}

// TestBookCreate_InvariantsAfterValidation checks post-validation invariants.
func TestBookCreate_InvariantsAfterValidation(t *testing.T) {
	tests := []struct {
		name  string
		input BookCreate
	}{
		{
			name:  "title and author not empty after validate",
			input: BookCreate{Title: "  Dune  ", Author: "  Frank Herbert  ", PublishedYear: 1965},
		},
		{
			name:  "title exactly 255 chars is valid",
			input: BookCreate{Title: strings.Repeat("a", 255), Author: "Frank Herbert", PublishedYear: 1965},
		},
		{
			name:  "author exactly 255 chars is valid",
			input: BookCreate{Title: "Dune", Author: strings.Repeat("b", 255), PublishedYear: 1965},
		},
		{
			name:  "summary exactly 2000 chars is valid",
			input: BookCreate{Title: "Dune", Author: "Frank Herbert", PublishedYear: 1965, Summary: strPtr(strings.Repeat("c", 2000))},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.input
			err := b.Validate()
			require.NoError(t, err)

			assert.NotEmpty(t, b.Title, "title must not be empty after validation")
			assert.NotEmpty(t, b.Author, "author must not be empty after validation")
			assert.LessOrEqual(t, len(b.Title), 255)
			assert.LessOrEqual(t, len(b.Author), 255)
			if b.Summary != nil {
				assert.LessOrEqual(t, len(*b.Summary), 2000)
				assert.NotEmpty(t, strings.TrimSpace(*b.Summary), "non-nil summary must not be whitespace-only")
			}
			assert.Greater(t, b.PublishedYear, 1000)
			assert.LessOrEqual(t, b.PublishedYear, CurrentYear())
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: BookUpdate
// ---------------------------------------------------------------------------

func TestBookUpdate_ValidCases(t *testing.T) {
	cy := CurrentYear()

	tests := []struct {
		name             string
		input            BookUpdate
		wantTitleNil     bool
		wantAuthorNil    bool
		wantYearNil      bool
		wantSummaryNil   bool
		wantTitle        string
		w