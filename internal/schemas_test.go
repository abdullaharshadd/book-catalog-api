```go
package internal

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpers

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func currentTestYear() int { return time.Now().Year() }

// hasFieldError returns true when ve contains an error for the given field with
// the given (sub)message.
func hasFieldError(ve *ValidationError, field, msgSubstr string) bool {
	for _, fe := range ve.Errors {
		if fe.Field == field && strings.Contains(fe.Message, msgSubstr) {
			return true
		}
	}
	return false
}

// asValidationError casts err to *ValidationError or fails the test.
func asValidationError(t *testing.T, err error) *ValidationError {
	t.Helper()
	require.Error(t, err)
	ve, ok := err.(*ValidationError)
	require.True(t, ok, "expected *ValidationError, got %T", err)
	return ve
}

// ---------------------------------------------------------------------------
// FieldError / ValidationError
// ---------------------------------------------------------------------------

func TestFieldError_Error(t *testing.T) {
	fe := &FieldError{Field: "title", Message: "Title cannot be empty", Type: "value_error"}
	assert.Equal(t, "title: Title cannot be empty", fe.Error())
}

func TestValidationError_Error_empty(t *testing.T) {
	ve := &ValidationError{}
	assert.Equal(t, "validation failed", ve.Error())
}

func TestValidationError_Error_single(t *testing.T) {
	ve := &ValidationError{Errors: []FieldError{
		{Field: "title", Message: "Title cannot be empty", Type: "value_error"},
	}}
	assert.Contains(t, ve.Error(), "title: Title cannot be empty")
}

func TestValidationError_Error_multiple(t *testing.T) {
	ve := &ValidationError{Errors: []FieldError{
		{Field: "title", Message: "Title cannot be empty", Type: "value_error"},
		{Field: "author", Message: "Author cannot be empty", Type: "value_error"},
	}}
	s := ve.Error()
	assert.Contains(t, s, "title: Title cannot be empty")
	assert.Contains(t, s, "author: Author cannot be empty")
}

func TestFieldError_Type_is_value_error(t *testing.T) {
	fe := newFieldError("title", "some msg")
	assert.Equal(t, "value_error", fe.Type)
}

// ---------------------------------------------------------------------------
// BookCreate.Validate
// ---------------------------------------------------------------------------

func TestBookCreate_Validate(t *testing.T) {
	year := currentTestYear()

	type tc struct {
		name string
		in   BookCreate
		// wantErr == false → expect nil error, and check out fields
		wantErr      bool
		wantTitle    string
		wantAuthor   string
		wantYear     int
		wantSummary  *string
		errField     string
		errMsgSubstr string
	}

	tests := []tc{
		// ── happy path ──────────────────────────────────────────────────────
		{
			name:        "valid minimal – no summary",
			in:          BookCreate{Title: "Go Programming", Author: "Alan Donovan", PublishedYear: 2016},
			wantErr:     false,
			wantTitle:   "Go Programming",
			wantAuthor:  "Alan Donovan",
			wantYear:    2016,
			wantSummary: nil,
		},
		{
			name:        "valid with summary",
			in:          BookCreate{Title: "Clean Code", Author: "Robert Martin", PublishedYear: 2008, Summary: strPtr("A great book")},
			wantErr:     false,
			wantTitle:   "Clean Code",
			wantAuthor:  "Robert Martin",
			wantYear:    2008,
			wantSummary: strPtr("A great book"),
		},
		{
			name:        "published_year equals current year",
			in:          BookCreate{Title: "Title", Author: "Author", PublishedYear: year},
			wantErr:     false,
			wantYear:    year,
			wantTitle:   "Title",
			wantAuthor:  "Author",
			wantSummary: nil,
		},
		{
			name:        "published_year equals 1000",
			in:          BookCreate{Title: "Title", Author: "Author", PublishedYear: 1000},
			wantErr:     false,
			wantYear:    1000,
			wantTitle:   "Title",
			wantAuthor:  "Author",
			wantSummary: nil,
		},
		// ── title whitespace trimming ────────────────────────────────────────
		{
			name:       "title trimmed",
			in:         BookCreate{Title: "  Go  ", Author: "Author", PublishedYear: 2000},
			wantErr:    false,
			wantTitle:  "Go",
			wantAuthor: "Author",
			wantYear:   2000,
		},
		// ── author whitespace trimming ────────────────────────────────────────
		{
			name:       "author trimmed",
			in:         BookCreate{Title: "Title", Author: "  Jane  ", PublishedYear: 2000},
			wantErr:    false,
			wantTitle:  "Title",
			wantAuthor: "Jane",
			wantYear:   2000,
		},
		// ── title errors ─────────────────────────────────────────────────────
		{
			name:         "title empty string",
			in:           BookCreate{Title: "", Author: "Author", PublishedYear: 2000},
			wantErr:      true,
			errField:     "title",
			errMsgSubstr: "Title cannot be empty",
		},
		{
			name:         "title whitespace only",
			in:           BookCreate{Title: "   ", Author: "Author", PublishedYear: 2000},
			wantErr:      true,
			errField:     "title",
			errMsgSubstr: "Title cannot be empty",
		},
		{
			name:         "title too long (256 chars)",
			in:           BookCreate{Title: strings.Repeat("a", 256), Author: "Author", PublishedYear: 2000},
			wantErr:      true,
			errField:     "title",
			errMsgSubstr: "ensure this value has at most 255 characters",
		},
		// ── author errors ────────────────────────────────────────────────────
		{
			name:         "author empty string",
			in:           BookCreate{Title: "Title", Author: "", PublishedYear: 2000},
			wantErr:      true,
			errField:     "author",
			errMsgSubstr: "Author cannot be empty",
		},
		{
			name:         "author whitespace only",
			in:           BookCreate{Title: "Title", Author: "   ", PublishedYear: 2000},
			wantErr:      true,
			errField:     "author",
			errMsgSubstr: "Author cannot be empty",
		},
		{
			name:         "author too long (256 chars)",
			in:           BookCreate{Title: "Title", Author: strings.Repeat("b", 256), PublishedYear: 2000},
			wantErr:      true,
			errField:     "author",
			errMsgSubstr: "ensure this value has at most 255 characters",
		},
		// ── published_year errors ────────────────────────────────────────────
		{
			name:         "published_year less than 1000",
			in:           BookCreate{Title: "Title", Author: "Author", PublishedYear: 999},
			wantErr:      true,
			errField:     "published_year",
			errMsgSubstr: "Published year must be after year 1000",
		},
		{
			name:         "published_year is 0",
			in:           BookCreate{Title: "Title", Author: "Author", PublishedYear: 0},
			wantErr:      true,
			errField:     "published_year",
			errMsgSubstr: "Published year must be after year 1000",
		},
		{
			name:         "published_year in the future",
			in:           BookCreate{Title: "Title", Author: "Author", PublishedYear: year + 1},
			wantErr:      true,
			errField:     "published_year",
			errMsgSubstr: fmt.Sprintf("Published year cannot be in the future (current year: %d)", year),
		},
		// ── summary handling ─────────────────────────────────────────────────
		{
			name:        "summary nil stays nil",
			in:          BookCreate{Title: "Title", Author: "Author", PublishedYear: 2000, Summary: nil},
			wantErr:     false,
			wantTitle:   "Title",
			wantAuthor:  "Author",
			wantYear:    2000,
			wantSummary: nil,
		},
		{
			name:        "summary empty string becomes nil",
			in:          BookCreate{Title: "Title", Author: "Author", PublishedYear: 2000, Summary: strPtr("")},
			wantErr:     false,
			wantSummary: nil,
		},
		{
			name:        "summary whitespace only becomes nil",
			in:          BookCreate{Title: "Title", Author: "Author", PublishedYear: 2000, Summary: strPtr("   ")},
			wantErr:     false,
			wantSummary: nil,
		},
		{
			name:        "summary trimmed",
			in:          BookCreate{Title: "Title", Author: "Author", PublishedYear: 2000, Summary: strPtr("  nice book  ")},
			wantErr:     false,
			wantSummary: strPtr("nice book"),
		},
		{
			name:         "summary too long (2001 chars after trim)",
			in:           BookCreate{Title: "Title", Author: "Author", PublishedYear: 2000, Summary: strPtr(strings.Repeat("x", 2001))},
			wantErr:      true,
			errField:     "summary",
			errMsgSubstr: "ensure this value has at most 2000 characters",
		},
		// ── multiple simultaneous errors ─────────────────────────────────────
		{
			name:    "both title and author empty",
			in:      BookCreate{Title: "", Author: "", PublishedYear: 2000},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			bc := tt.in // copy so mutations don't bleed between cases
			err := bc.Validate()

			if tt.wantErr {
				ve := asValidationError(t, err)
				if tt.errField != "" {
					assert.True(t, hasFieldError(ve, tt.errField, tt.errMsgSubstr),
						"expected error field=%q msg=%q in %+v", tt.errField, tt.errMsgSubstr, ve.Errors)
					// check every error has Type == "value_error"
					for _, fe := range ve.Errors {
						assert.Equal(t, "value_error", fe.Type)
					}
				}
				return
			}

			require.NoError(t, err)
			if tt.wantTitle != "" {
				assert.Equal(t, tt.wantTitle, bc.Title)
			}
			if tt.wantAuthor != "" {
				assert.Equal(t, tt.wantAuthor, bc.Author)
			}
			if tt.wantYear != 0 {
				assert.Equal(t, tt.wantYear, bc.PublishedYear)
			}
			if tt.wantSummary == nil {
				assert.Nil(t, bc.Summary)
			} else {
				require.NotNil(t, bc.Summary)
				assert.Equal(t, *tt.wantSummary, *bc.Summary)
			}
		})
	}
}

// Verify that title trimming mutates the receiver so the caller sees it.
func TestBookCreate_Validate_TitleTrimMutatesReceiver(t *testing.T) {
	bc := BookCreate{Title: "  Effective Go  ", Author: "The Go Team", PublishedYear: 2009}
	require.NoError(t, bc.Validate())
	assert.Equal(t, "Effective Go", bc.Title)
}

// Verify author trimming mutates the receiver.
func TestBookCreate_Validate_AuthorTrimMutatesReceiver(t *testing.T) {
	bc := BookCreate{Title: "Title", Author: "\tGo Gopher\t", PublishedYear: 2009}
	require.NoError(t, bc.Validate())
	assert.Equal(t, "Go Gopher", bc.Author)
}

// Verify summary trimming mutates the receiver.
func TestBookCreate_Validate_SummaryTrimMutatesReceiver(t *testing.T) {
	bc := BookCreate{Title: "Title", Author: "Author", PublishedYear: 2009, Summary: strPtr("  A summary  ")}
	require.NoError(t, bc.Validate())
	require.NotNil(t, bc.Summary)
	assert.Equal(t, "A summary", *bc.Summary)
}

// Verify that a title of exactly 255 chars is accepted.
func TestBookCreate_Validate_TitleExactly255(t *testing.T) {
	bc := BookCreate{Title: strings.Repeat("a", 255), Author: "Author", PublishedYear: 2000}
	assert.NoError(t, bc.Validate())
}

// Verify that an author of exactly 255 chars is accepted.
func TestBookCreate_Validate_AuthorExactly255(t *testing.T) {
	bc := BookCreate{Title: "Title", Author: strings.Repeat("b", 255), PublishedYear: 2000}
	assert.NoError(t, bc.Validate())
}

// Verify that a summary of exactly 2000 chars is accepted.
func TestBookCreate_Validate_SummaryExactly2000(t *testing.T) {
	bc := BookCreate{Title: "Title", Author: "Author", PublishedYear: 2000, Summary: strPtr(strings.Repeat("c", 2000))}
	assert.NoError(t, bc.Validate())
}

// Verify that multiple errors are all reported together.
func TestBookCreate_Validate_MultipleErrors(t *testing.T) {
	year := currentTestYear()
	bc := BookCreate{
		Title:        "",
		Author:       "",
		PublishedYear: year + 5,
	}
	err := bc.Validate()
	ve := asValidationError(t, err)
	assert.True(t, hasFieldError(ve, "title", "Title cannot be empty"))
	assert.True(t, hasFieldError(ve, "author", "Author cannot be empty"))
	assert.True(t, hasFieldError(ve, "published_year", "Published year cannot be in the future"))
	assert.GreaterOrEqual(t, len(ve.Errors), 3)
}

// ---------------------------------------------------------------------------
// BookUpdate.Validate
// ---------------------------------------------------------------------------

func TestBookUpdate_Validate(t *testing.T) {
	year := currentTestYear()

	type tc struct {
		name         string
		in           BookUpdate
		wantErr      bool
		errField     string
		errMsgSubstr string
		// checks applied when wantErr==false
		checkTitle   *string