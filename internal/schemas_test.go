```go
package internal

import (
	"encoding/json"
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

var currentYear = time.Now().Year()

// ─────────────────────────────────────────────
// validateTitle
// ─────────────────────────────────────────────

func TestValidateTitle(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantTrimmed string
		wantErrMsg  string
	}{
		{
			name:        "valid title",
			input:       "The Go Programming Language",
			wantTrimmed: "The Go Programming Language",
		},
		{
			name:        "title with surrounding whitespace",
			input:       "  Clean Code  ",
			wantTrimmed: "Clean Code",
		},
		{
			name:       "empty title",
			input:      "",
			wantErrMsg: "title: Title cannot be empty",
		},
		{
			name:       "whitespace-only title",
			input:      "   ",
			wantErrMsg: "title: Title cannot be empty",
		},
		{
			name:        "title exactly 255 characters",
			input:       strings.Repeat("a", 255),
			wantTrimmed: strings.Repeat("a", 255),
		},
		{
			name:       "title exceeds 255 characters",
			input:      strings.Repeat("a", 256),
			wantErrMsg: "title: ensure this value has at most 255 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateTitle(tt.input)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrMsg, err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTrimmed, got)
			}
		})
	}
}

// ─────────────────────────────────────────────
// validateAuthor
// ─────────────────────────────────────────────

func TestValidateAuthor(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantTrimmed string
		wantErrMsg  string
	}{
		{
			name:        "valid author",
			input:       "Robert C. Martin",
			wantTrimmed: "Robert C. Martin",
		},
		{
			name:        "author with surrounding whitespace",
			input:       "  Brian Kernighan  ",
			wantTrimmed: "Brian Kernighan",
		},
		{
			name:       "empty author",
			input:      "",
			wantErrMsg: "author: Author cannot be empty",
		},
		{
			name:       "whitespace-only author",
			input:      "\t\t",
			wantErrMsg: "author: Author cannot be empty",
		},
		{
			name:        "author exactly 255 characters",
			input:       strings.Repeat("b", 255),
			wantTrimmed: strings.Repeat("b", 255),
		},
		{
			name:       "author exceeds 255 characters",
			input:      strings.Repeat("b", 256),
			wantErrMsg: "author: ensure this value has at most 255 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateAuthor(tt.input)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrMsg, err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTrimmed, got)
			}
		})
	}
}

// ─────────────────────────────────────────────
// validatePublishedYear
// ─────────────────────────────────────────────

func TestValidatePublishedYear(t *testing.T) {
	tests := []struct {
		name       string
		input      int
		wantYear   int
		wantErrMsg string
	}{
		{
			name:     "exactly 1000 (lower boundary)",
			input:    1000,
			wantYear: 1000,
		},
		{
			name:     "exactly current year (upper boundary)",
			input:    currentYear,
			wantYear: currentYear,
		},
		{
			name:     "valid mid-range year",
			input:    1984,
			wantYear: 1984,
		},
		{
			name:       "year less than 1000",
			input:      999,
			wantErrMsg: "published_year: Published year must be after year 1000",
		},
		{
			name:       "year 0",
			input:      0,
			wantErrMsg: "published_year: Published year must be after year 1000",
		},
		{
			name:       "negative year",
			input:      -500,
			wantErrMsg: "published_year: Published year must be after year 1000",
		},
		{
			name:       "year greater than current year",
			input:      currentYear + 1,
			wantErrMsg: fmt.Sprintf("published_year: Published year cannot be in the future (current year: %d)", currentYear),
		},
		{
			name:       "year far in the future",
			input:      3000,
			wantErrMsg: fmt.Sprintf("published_year: Published year cannot be in the future (current year: %d)", currentYear),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validatePublishedYear(tt.input)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrMsg, err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantYear, got)
			}
		})
	}
}

// ─────────────────────────────────────────────
// validateSummary
// ─────────────────────────────────────────────

func TestValidateSummary(t *testing.T) {
	tests := []struct {
		name       string
		input      *string
		wantNil    bool
		wantVal    string
		wantErrMsg string
	}{
		{
			name:    "nil summary",
			input:   nil,
			wantNil: true,
		},
		{
			name:    "empty string summary",
			input:   strPtr(""),
			wantNil: true,
		},
		{
			name:    "whitespace-only summary",
			input:   strPtr("   "),
			wantNil: true,
		},
		{
			name:    "valid summary",
			input:   strPtr("A great book"),
			wantVal: "A great book",
		},
		{
			name:    "summary with surrounding whitespace",
			input:   strPtr("  trimmed summary  "),
			wantVal: "trimmed summary",
		},
		{
			name:    "summary exactly 2000 characters",
			input:   strPtr(strings.Repeat("x", 2000)),
			wantVal: strings.Repeat("x", 2000),
		},
		{
			name:       "summary exceeds 2000 characters",
			input:      strPtr(strings.Repeat("x", 2001)),
			wantErrMsg: "summary: ensure this value has at most 2000 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateSummary(tt.input)
			if tt.wantErrMsg != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErrMsg, err.Error())
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.wantVal, *got)
			}
		})
	}
}

// ─────────────────────────────────────────────
// ValidationError
// ─────────────────────────────────────────────

func TestValidationError_Error(t *testing.T) {
	ve := &ValidationError{Field: "title", Message: "Title cannot be empty"}
	assert.Equal(t, "title: Title cannot be empty", ve.Error())
}

func TestNewValidationError(t *testing.T) {
	ve := newValidationError("author", "Author cannot be empty")
	assert.Equal(t, "author", ve.Field)
	assert.Equal(t, "Author cannot be empty", ve.Message)
}

// ─────────────────────────────────────────────
// BookCreate.Validate
// ─────────────────────────────────────────────

func TestBookCreate_Validate(t *testing.T) {
	tests := []struct {
		name        string
		input       BookCreate
		wantErrMsg  string
		wantTitle   string
		wantAuthor  string
		wantYear    int
		wantSumNil  bool
		wantSumVal  string
	}{
		{
			name: "all valid fields",
			input: BookCreate{
				Title:         "Clean Code",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
				Summary:       strPtr("A handbook of agile software craftsmanship"),
			},
			wantTitle:  "Clean Code",
			wantAuthor: "Robert C. Martin",
			wantYear:   2008,
			wantSumVal: "A handbook of agile software craftsmanship",
		},
		{
			name: "whitespace in title and author is trimmed",
			input: BookCreate{
				Title:         "  Clean Code  ",
				Author:        "  Robert C. Martin  ",
				PublishedYear: 2008,
				Summary:       nil,
			},
			wantTitle:  "Clean Code",
			wantAuthor: "Robert C. Martin",
			wantYear:   2008,
			wantSumNil: true,
		},
		{
			name: "nil summary normalised to nil",
			input: BookCreate{
				Title:         "SICP",
				Author:        "Abelson",
				PublishedYear: 1996,
				Summary:       nil,
			},
			wantTitle:  "SICP",
			wantAuthor: "Abelson",
			wantYear:   1996,
			wantSumNil: true,
		},
		{
			name: "whitespace-only summary normalised to nil",
			input: BookCreate{
				Title:         "SICP",
				Author:        "Abelson",
				PublishedYear: 1996,
				Summary:       strPtr("   "),
			},
			wantTitle:  "SICP",
			wantAuthor: "Abelson",
			wantYear:   1996,
			wantSumNil: true,
		},
		{
			name: "summary with surrounding whitespace is trimmed",
			input: BookCreate{
				Title:         "SICP",
				Author:        "Abelson",
				PublishedYear: 1996,
				Summary:       strPtr("  great book  "),
			},
			wantTitle:  "SICP",
			wantAuthor: "Abelson",
			wantYear:   1996,
			wantSumVal: "great book",
		},
		{
			name: "published_year exactly 1000",
			input: BookCreate{
				Title:         "Old Book",
				Author:        "Ancient Author",
				PublishedYear: 1000,
			},
			wantTitle:  "Old Book",
			wantAuthor: "Ancient Author",
			wantYear:   1000,
			wantSumNil: true,
		},
		{
			name: "published_year exactly current year",
			input: BookCreate{
				Title:         "New Book",
				Author:        "Modern Author",
				PublishedYear: currentYear,
			},
			wantTitle:  "New Book",
			wantAuthor: "Modern Author",
			wantYear:   currentYear,
			wantSumNil: true,
		},
		// error cases
		{
			name: "empty title",
			input: BookCreate{
				Title:         "",
				Author:        "Author",
				PublishedYear: 2000,
			},
			wantErrMsg: "title: Title cannot be empty",
		},
		{
			name: "whitespace-only title",
			input: BookCreate{
				Title:         "   ",
				Author:        "Author",
				PublishedYear: 2000,
			},
			wantErrMsg: "title: Title cannot be empty",
		},
		{
			name: "title too long",
			input: BookCreate{
				Title:         strings.Repeat("a", 256),
				Author:        "Author",
				PublishedYear: 2000,
			},
			wantErrMsg: "title: ensure this value has at most 255 characters",
		},
		{
			name: "empty author",
			input: BookCreate{
				Title:         "Title",
				Author:        "",
				PublishedYear: 2000,
			},
			wantErrMsg: "author: Author cannot be empty",
		},
		{
			name: "whitespace-only author",
			input: BookCreate{
				Title:         "Title",
				Author:        "  ",
				PublishedYear: 2000,
			},
			wantErrMsg: "author: Author cannot be empty",
		},
		{
			name: "author too long",
			input: BookCreate{
				Title:         "Title",
				Author:        strings.Repeat("b", 256),
				PublishedYear: 2000,
			},
			wantErrMsg: "author: ensure this value has at most 255 characters",
		},
		{
			name: "published_year less than 1000",
			input: BookCreate{
				Title:         "Title",
				Author:        "Author",
				PublishedYear: 999,
			},
			wantErrMsg: "published_year: Published year must be after year 1000",
		},
		{
			name: "published_year 0 (missing/zero value)",
			input: BookCreate{
				Title:         "Title",
				Author:        "Author",
				PublishedYear: 0,
			},
			wantErrMsg: "published_year: Published year must be after year 1000",
		},
		{
			name: "published_year in the future",
			input: BookCreate{
				Title:         "Title",
				Author:        "Author",
				PublishedYear: currentYear + 1,
			},
			wantErrMsg: fmt.Sprintf("published_year: Published year cannot be in the future (current year: %d)", currentYear),
		},
		{
			name: "summary too long",
			input: BookCreate{
				Title:         "Title",
				Author:        "Author",
				PublishedYear: 2000,
				Summary:       strPtr(