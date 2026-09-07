package schemas_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"internal/schemas"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// ─── ValidationError ─────────────────────────────────────────────────────────

func TestValidationError_Error(t *testing.T) {
	e := &schemas.ValidationError{Field: "title", Message: "field required"}
	assert.Equal(t, `validation error on field "title": field required`, e.Error())
}

func TestValidationErrors_Error(t *testing.T) {
	ve := schemas.ValidationErrors{
		{Field: "title", Message: "field required"},
		{Field: "author", Message: "field required"},
	}
	got := ve.Error()
	assert.Contains(t, got, "title")
	assert.Contains(t, got, "author")
}

func TestValidationErrors_HasField(t *testing.T) {
	ve := schemas.ValidationErrors{
		{Field: "title", Message: "field required"},
	}
	assert.True(t, ve.HasField("title"))
	assert.False(t, ve.HasField("author"))
}

func TestValidationErrors_MessageFor(t *testing.T) {
	ve := schemas.ValidationErrors{
		{Field: "title", Message: "Title cannot be empty"},
		{Field: "author", Message: "Author cannot be empty"},
	}
	assert.Equal(t, "Title cannot be empty", ve.MessageFor("title"))
	assert.Equal(t, "Author cannot be empty", ve.MessageFor("author"))
	assert.Equal(t, "", ve.MessageFor("summary"))
}

// ─── ValidateBookCreate ───────────────────────────────────────────────────────

func TestValidateBookCreate(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name        string
		input       schemas.BookCreateInput
		wantErr     bool
		errFields   []string
		errContains map[string]string
		checkResult func(t *testing.T, r *schemas.BookCreate)
	}{
		{
			name: "valid_all_fields_no_summary",
			input: schemas.BookCreateInput{
				Title:         strPtr("The Go Programming Language"),
				Author:        strPtr("Alan Donovan"),
				PublishedYear: intPtr(2015),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				assert.Equal(t, "The Go Programming Language", r.Title)
				assert.Equal(t, "Alan Donovan", r.Author)
				assert.Equal(t, 2015, r.PublishedYear)
				assert.Nil(t, r.Summary)
			},
		},
		{
			name: "valid_with_summary",
			input: schemas.BookCreateInput{
				Title:         strPtr("Clean Code"),
				Author:        strPtr("Robert Martin"),
				PublishedYear: intPtr(2008),
				Summary:       strPtr("A handbook of agile software craftsmanship."),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				require.NotNil(t, r.Summary)
				assert.Equal(t, "A handbook of agile software craftsmanship.", *r.Summary)
			},
		},
		{
			name: "title_with_surrounding_whitespace",
			input: schemas.BookCreateInput{
				Title:         strPtr("  Go In Action  "),
				Author:        strPtr("William Kennedy"),
				PublishedYear: intPtr(2015),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				assert.Equal(t, "Go In Action", r.Title)
			},
		},
		{
			name: "author_with_surrounding_whitespace",
			input: schemas.BookCreateInput{
				Title:         strPtr("Go In Action"),
				Author:        strPtr("  William Kennedy  "),
				PublishedYear: intPtr(2015),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				assert.Equal(t, "William Kennedy", r.Author)
			},
		},
		{
			name: "title_empty_string",
			input: schemas.BookCreateInput{
				Title:         strPtr(""),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
			},
			wantErr:   true,
			errFields: []string{"title"},
			errContains: map[string]string{
				"title": "Title cannot be empty",
			},
		},
		{
			name: "title_only_whitespace",
			input: schemas.BookCreateInput{
				Title:         strPtr("   "),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
			},
			wantErr:   true,
			errFields: []string{"title"},
			errContains: map[string]string{
				"title": "Title cannot be empty",
			},
		},
		{
			name: "title_too_long",
			input: schemas.BookCreateInput{
				Title:         strPtr(strings.Repeat("a", 256)),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
			},
			wantErr:   true,
			errFields: []string{"title"},
			errContains: map[string]string{
				"title": "ensure this value has at most 255 characters",
			},
		},
		{
			name: "title_exactly_255_chars",
			input: schemas.BookCreateInput{
				Title:         strPtr(strings.Repeat("a", 255)),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				assert.Equal(t, 255, len(r.Title))
			},
		},
		{
			name: "title_missing",
			input: schemas.BookCreateInput{
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
			},
			wantErr:   true,
			errFields: []string{"title"},
			errContains: map[string]string{
				"title": "field required",
			},
		},
		{
			name: "author_empty_string",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr(""),
				PublishedYear: intPtr(2000),
			},
			wantErr:   true,
			errFields: []string{"author"},
			errContains: map[string]string{
				"author": "Author cannot be empty",
			},
		},
		{
			name: "author_only_whitespace",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr("   "),
				PublishedYear: intPtr(2000),
			},
			wantErr:   true,
			errFields: []string{"author"},
			errContains: map[string]string{
				"author": "Author cannot be empty",
			},
		},
		{
			name: "author_too_long",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr(strings.Repeat("b", 256)),
				PublishedYear: intPtr(2000),
			},
			wantErr:   true,
			errFields: []string{"author"},
			errContains: map[string]string{
				"author": "ensure this value has at most 255 characters",
			},
		},
		{
			name: "author_exactly_255_chars",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr(strings.Repeat("b", 255)),
				PublishedYear: intPtr(2000),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				assert.Equal(t, 255, len(r.Author))
			},
		},
		{
			name: "author_missing",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				PublishedYear: intPtr(2000),
			},
			wantErr:   true,
			errFields: []string{"author"},
			errContains: map[string]string{
				"author": "field required",
			},
		},
		{
			name: "published_year_less_than_1000",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(999),
			},
			wantErr:   true,
			errFields: []string{"published_year"},
			errContains: map[string]string{
				"published_year": "Published year must be after year 1000",
			},
		},
		{
			name: "published_year_exactly_1000",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(1000),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				assert.Equal(t, 1000, r.PublishedYear)
			},
		},
		{
			name: "published_year_greater_than_current",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(currentYear + 1),
			},
			wantErr:   true,
			errFields: []string{"published_year"},
			errContains: map[string]string{
				"published_year": "Published year cannot be in the future",
			},
		},
		{
			name: "published_year_equals_current_year",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(currentYear),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				assert.Equal(t, currentYear, r.PublishedYear)
			},
		},
		{
			name: "published_year_missing",
			input: schemas.BookCreateInput{
				Title:  strPtr("Title"),
				Author: strPtr("Author"),
			},
			wantErr:   true,
			errFields: []string{"published_year"},
			errContains: map[string]string{
				"published_year": "field required",
			},
		},
		{
			name: "summary_nil",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
				Summary:       nil,
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				assert.Nil(t, r.Summary)
			},
		},
		{
			name: "summary_empty_string",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
				Summary:       strPtr(""),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				assert.Nil(t, r.Summary)
			},
		},
		{
			name: "summary_only_whitespace",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
				Summary:       strPtr("   "),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				assert.Nil(t, r.Summary)
			},
		},
		{
			name: "summary_with_surrounding_whitespace",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
				Summary:       strPtr("  A great book.  "),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				require.NotNil(t, r.Summary)
				assert.Equal(t, "A great book.", *r.Summary)
			},
		},
		{
			name: "summary_too_long",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
				Summary:       strPtr(strings.Repeat("x", 2001)),
			},
			wantErr:   true,
			errFields: []string{"summary"},
			errContains: map[string]string{
				"summary": "ensure this value has at most 2000 characters",
			},
		},
		{
			name: "summary_exactly_2000_chars",
			input: schemas.BookCreateInput{
				Title:         strPtr("Title"),
				Author:        strPtr("Author"),
				PublishedYear: intPtr(2000),
				Summary:       strPtr(strings.Repeat("x", 2000)),
			},
			wantErr: false,
			checkResult: func(t *testing.T, r *schemas.BookCreate) {
				require.NotNil(t, r.Summary)
				assert.Equal(t, 2000, len(*r.Summary))
			},
		},
		{
			name: "multiple_missing_required_fields",
			input: schemas.BookCreateInput{},
			wantErr:   true,
			errFields: []string{"title", "author", "published_year"},
		},
		{
			name: "multiple_validation_errors_collected",
			input: schemas.BookCreateInput{
				Title:         strPtr(""),
				Author:        strPtr(""),
				PublishedYear: intPtr(999),
			},
			wantErr:   true,
			errFields: []string{"title", "author", "published_year"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, errs := schemas.ValidateBookCreate(tc.input)

			if tc.wantErr {
				assert.Nil(t, result)
				require.NotNil(t, errs)

				for _, field := range tc.errFields {
					assert.True(t, errs.HasField(field), "expected error for field %q", field)
				}

				for field, msgSubstring := range tc.errContains {
					msg := errs.MessageFor(field)
					assert.Contains(t, msg, msgSubstring, "field %q message mismatch", field)
				}
			} else {
				assert.Nil(t, errs)
				require.NotNil(t, result)

				if tc.checkResult != nil {
					tc.checkResult(t, result)
				}
			}
		})
	}
}

// Verify future-year error message contains the current year.
func TestValidateBookCreate_FutureYearMessage(t *testing.T) {
	currentYear := time.Now().Year()
	in := schemas.BookCreateInput{
		Title:         strPtr