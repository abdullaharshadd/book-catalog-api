```go
package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internal "bookcatalog/internal"
)

// ptrStr returns a pointer to a string literal.
func ptrStr(s string) *string { return &s }

// ptrInt returns a pointer to an int literal.
func ptrInt(i int) *int { return &i }

// ---------------------------------------------------------------------------
// BookCreate tests
// ---------------------------------------------------------------------------

func TestBookCreate_ValidAllFields(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		author        string
		publishedYear int
		summary       *string
		wantTitle     string
		wantAuthor    string
		wantSummary   *string
	}{
		{
			name:          "all valid fields with summary",
			title:         "Valid Book",
			author:        "Valid Author",
			publishedYear: 2023,
			summary:       ptrStr("A valid book summary"),
			wantTitle:     "Valid Book",
			wantAuthor:    "Valid Author",
			wantSummary:   ptrStr("A valid book summary"),
		},
		{
			name:          "valid fields without summary",
			title:         "Book Without Summary",
			author:        "Author",
			publishedYear: 2023,
			summary:       nil,
			wantTitle:     "Book Without Summary",
			wantAuthor:    "Author",
			wantSummary:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         tt.title,
				Author:        tt.author,
				PublishedYear: tt.publishedYear,
				Summary:       tt.summary,
			}

			err := bc.Validate()
			require.NoError(t, err)

			assert.Equal(t, tt.wantTitle, bc.Title)
			assert.Equal(t, tt.wantAuthor, bc.Author)
			assert.Equal(t, tt.publishedYear, bc.PublishedYear)

			if tt.wantSummary == nil {
				assert.Nil(t, bc.Summary)
			} else {
				require.NotNil(t, bc.Summary)
				assert.Equal(t, *tt.wantSummary, *bc.Summary)
			}
		})
	}
}

func TestBookCreate_WhitespaceStripping(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		author        string
		publishedYear int
		summary       *string
		wantTitle     string
		wantAuthor    string
		wantSummary   *string
	}{
		{
			name:          "strips whitespace from title, author, and summary",
			title:         "  Whitespace Book  ",
			author:        "  Whitespace Author  ",
			publishedYear: 2023,
			summary:       ptrStr("  Whitespace summary  "),
			wantTitle:     "Whitespace Book",
			wantAuthor:    "Whitespace Author",
			wantSummary:   ptrStr("Whitespace summary"),
		},
		{
			name:          "strips leading whitespace only",
			title:         "   Leading Book",
			author:        "   Leading Author",
			publishedYear: 2023,
			summary:       ptrStr("   Leading summary"),
			wantTitle:     "Leading Book",
			wantAuthor:    "Leading Author",
			wantSummary:   ptrStr("Leading summary"),
		},
		{
			name:          "strips trailing whitespace only",
			title:         "Trailing Book   ",
			author:        "Trailing Author   ",
			publishedYear: 2023,
			summary:       ptrStr("Trailing summary   "),
			wantTitle:     "Trailing Book",
			wantAuthor:    "Trailing Author",
			wantSummary:   ptrStr("Trailing summary"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         tt.title,
				Author:        tt.author,
				PublishedYear: tt.publishedYear,
				Summary:       tt.summary,
			}

			err := bc.Validate()
			require.NoError(t, err)

			assert.Equal(t, tt.wantTitle, bc.Title)
			assert.Equal(t, tt.wantAuthor, bc.Author)

			if tt.wantSummary == nil {
				assert.Nil(t, bc.Summary)
			} else {
				require.NotNil(t, bc.Summary)
				assert.Equal(t, *tt.wantSummary, *bc.Summary)
			}
		})
	}
}

func TestBookCreate_WhitespaceOnlySummaryBecomesNil(t *testing.T) {
	tests := []struct {
		name    string
		summary *string
	}{
		{
			name:    "whitespace-only summary becomes nil",
			summary: ptrStr("   "),
		},
		{
			name:    "single space summary becomes nil",
			summary: ptrStr(" "),
		},
		{
			name:    "tabs and spaces become nil",
			summary: ptrStr("\t  \t"),
		},
		{
			name:    "empty string summary becomes nil",
			summary: ptrStr(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         "Book",
				Author:        "Author",
				PublishedYear: 2023,
				Summary:       tt.summary,
			}

			err := bc.Validate()
			require.NoError(t, err)
			assert.Nil(t, bc.Summary, "whitespace-only summary should be coerced to nil")
		})
	}
}

func TestBookCreate_EmptyTitleValidation(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		wantErrMsg  string
	}{
		{
			name:       "empty string title",
			title:      "",
			wantErrMsg: "Title cannot be empty",
		},
		{
			name:       "whitespace-only title",
			title:      "   ",
			wantErrMsg: "Title cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         tt.title,
				Author:        "Author",
				PublishedYear: 2023,
			}

			err := bc.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func TestBookCreate_EmptyAuthorValidation(t *testing.T) {
	tests := []struct {
		name       string
		author     string
		wantErrMsg string
	}{
		{
			name:       "empty string author",
			author:     "",
			wantErrMsg: "Author cannot be empty",
		},
		{
			name:       "whitespace-only author",
			author:     "   ",
			wantErrMsg: "Author cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         "Title",
				Author:        tt.author,
				PublishedYear: 2023,
			}

			err := bc.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func TestBookCreate_PublishedYearValidation(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name          string
		publishedYear int
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name:          "year 999 is too early",
			publishedYear: 999,
			wantErr:       true,
			wantErrMsg:    "Published year must be after year 1000",
		},
		{
			name:          "year 0 is too early",
			publishedYear: 0,
			wantErr:       true,
			wantErrMsg:    "Published year must be after year 1000",
		},
		{
			name:          "negative year is too early",
			publishedYear: -500,
			wantErr:       true,
			wantErrMsg:    "Published year must be after year 1000",
		},
		{
			name:          "future year is invalid",
			publishedYear: currentYear + 1,
			wantErr:       true,
			wantErrMsg:    "cannot be in the future",
		},
		{
			name:          "far future year is invalid",
			publishedYear: currentYear + 100,
			wantErr:       true,
			wantErrMsg:    "cannot be in the future",
		},
		{
			name:          "year exactly 1000 is valid",
			publishedYear: 1000,
			wantErr:       false,
		},
		{
			name:          "current year is valid",
			publishedYear: currentYear,
			wantErr:       false,
		},
		{
			name:          "year 1500 is valid",
			publishedYear: 1500,
			wantErr:       false,
		},
		{
			name:          "year 2000 is valid",
			publishedYear: 2000,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         "Title",
				Author:        "Author",
				PublishedYear: tt.publishedYear,
			}

			err := bc.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.publishedYear, bc.PublishedYear)
			}
		})
	}
}

func TestBookCreate_TitleLengthValidation(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{
			name:    "title exactly 255 chars is valid",
			title:   strings.Repeat("A", 255),
			wantErr: false,
		},
		{
			name:    "title exactly 256 chars is invalid",
			title:   strings.Repeat("A", 256),
			wantErr: true,
		},
		{
			name:    "title of 300 chars is invalid",
			title:   strings.Repeat("A", 300),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         tt.title,
				Author:        "Author",
				PublishedYear: 2023,
			}

			err := bc.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "255")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBookCreate_AuthorLengthValidation(t *testing.T) {
	tests := []struct {
		name    string
		author  string
		wantErr bool
	}{
		{
			name:    "author exactly 255 chars is valid",
			author:  strings.Repeat("B", 255),
			wantErr: false,
		},
		{
			name:    "author exactly 256 chars is invalid",
			author:  strings.Repeat("B", 256),
			wantErr: true,
		},
		{
			name:    "author of 500 chars is invalid",
			author:  strings.Repeat("B", 500),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         "Title",
				Author:        tt.author,
				PublishedYear: 2023,
			}

			err := bc.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "255")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBookCreate_SummaryLengthValidation(t *testing.T) {
	tests := []struct {
		name    string
		summary *string
		wantErr bool
	}{
		{
			name:    "summary exactly 2000 chars is valid",
			summary: ptrStr(strings.Repeat("C", 2000)),
			wantErr: false,
		},
		{
			name:    "summary exactly 2001 chars is invalid",
			summary: ptrStr(strings.Repeat("C", 2001)),
			wantErr: true,
		},
		{
			name:    "summary of 3000 chars is invalid",
			summary: ptrStr(strings.Repeat("C", 3000)),
			wantErr: true,
		},
		{
			name:    "nil summary is valid",
			summary: nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         "Title",
				Author:        "Author",
				PublishedYear: 2023,
				Summary:       tt.summary,
			}

			err := bc.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "2000")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BookUpdate tests
// ---------------------------------------------------------------------------

func TestBookUpdate_ValidPartialUpdate(t *testing.T) {
	tests := []struct {
		name              string
		title             *string
		author            *string
		publishedYear     *int
		summary           *string
		wantTitle         *string
		wantAuthor        *string
		wantPublishedYear *int
		wantSummary       *string
	}{
		{
			name:              "subset: title and published_year only",
			title:             ptrStr("Updated Title"),
			author:            nil,
			publishedYear:     ptrInt(2024),
			summary:           nil,
			wantTitle:         ptrStr("Updated Title"),
			wantAuthor:        nil,
			wantPublishedYear: ptrInt(2024),
			wantSummary:       nil,
		},
		{
			name:              "subset: author only",
			title:             nil,
			author:            ptrStr("New Author"),
			publishedYear:     nil,
			summary:           nil,
			wantTitle:         nil,
			wantAuthor:        ptrStr("New Author"),
			wantPublishedYear: nil,
			wantSummary:       nil,
		},
		{
			name:              "subset: summary only",
			title:             nil,
			author:            nil,
			publishedYear: