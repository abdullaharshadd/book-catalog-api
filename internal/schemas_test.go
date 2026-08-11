```go
package internal

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// helpers

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

var currentYear = time.Now().Year()

// ──────────────────────────────────────────────────────────────────────────────
// BookCreate.Validate
// ──────────────────────────────────────────────────────────────────────────────

func TestBookCreate_Validate(t *testing.T) {
	longStr255 := strings.Repeat("a", 256)  // 256 chars → too long
	longStr2001 := strings.Repeat("b", 2001) // 2001 chars → too long

	tests := []struct {
		name        string
		input       BookCreate
		wantErr     error
		wantErrFmt  string // for fmt.Errorf errors (partial match)
		wantTitle   string
		wantAuthor  string
		wantSummary *string
	}{
		// ── Happy paths ───────────────────────────────────────────────────────
		{
			name: "valid full input",
			input: BookCreate{
				Title:         "Clean Code",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
				Summary:       strPtr("A great book about writing clean code."),
			},
			wantErr:     nil,
			wantTitle:   "Clean Code",
			wantAuthor:  "Robert C. Martin",
			wantSummary: strPtr("A great book about writing clean code."),
		},
		{
			name: "title trimmed",
			input: BookCreate{
				Title:         "  Clean Code  ",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
			},
			wantErr:    nil,
			wantTitle:  "Clean Code",
			wantAuthor: "Robert C. Martin",
		},
		{
			name: "author trimmed",
			input: BookCreate{
				Title:         "Clean Code",
				Author:        "  Robert C. Martin  ",
				PublishedYear: 2008,
			},
			wantErr:    nil,
			wantTitle:  "Clean Code",
			wantAuthor: "Robert C. Martin",
		},
		{
			name: "summary trimmed",
			input: BookCreate{
				Title:         "Clean Code",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
				Summary:       strPtr("  great book  "),
			},
			wantErr:     nil,
			wantSummary: strPtr("great book"),
		},
		{
			name: "summary omitted -> nil",
			input: BookCreate{
				Title:         "Clean Code",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
				Summary:       nil,
			},
			wantErr:     nil,
			wantSummary: nil,
		},
		{
			name: "summary whitespace-only normalized to nil",
			input: BookCreate{
				Title:         "Clean Code",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
				Summary:       strPtr("   "),
			},
			wantErr:     nil,
			wantSummary: nil,
		},
		{
			name: "published_year exactly 1000",
			input: BookCreate{
				Title:         "Old Book",
				Author:        "Ancient Author",
				PublishedYear: 1000,
			},
			wantErr: nil,
		},
		{
			name: "published_year exactly current year",
			input: BookCreate{
				Title:         "New Book",
				Author:        "Modern Author",
				PublishedYear: currentYear,
			},
			wantErr: nil,
		},

		// ── Title errors ──────────────────────────────────────────────────────
		{
			name: "empty title",
			input: BookCreate{
				Title:         "",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
			},
			wantErr: ErrTitleEmpty,
		},
		{
			name: "whitespace-only title",
			input: BookCreate{
				Title:         "   ",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
			},
			wantErr: ErrTitleEmpty,
		},
		{
			name: "title too long (256 chars)",
			input: BookCreate{
				Title:         longStr255,
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
			},
			wantErr: ErrTitleTooLong,
		},

		// ── Author errors ─────────────────────────────────────────────────────
		{
			name: "empty author",
			input: BookCreate{
				Title:         "Clean Code",
				Author:        "",
				PublishedYear: 2008,
			},
			wantErr: ErrAuthorEmpty,
		},
		{
			name: "whitespace-only author",
			input: BookCreate{
				Title:         "Clean Code",
				Author:        "   ",
				PublishedYear: 2008,
			},
			wantErr: ErrAuthorEmpty,
		},
		{
			name: "author too long (256 chars)",
			input: BookCreate{
				Title:         "Clean Code",
				Author:        longStr255,
				PublishedYear: 2008,
			},
			wantErr: ErrAuthorTooLong,
		},

		// ── PublishedYear errors ──────────────────────────────────────────────
		{
			name: "published_year less than 1000",
			input: BookCreate{
				Title:         "Clean Code",
				Author:        "Robert C. Martin",
				PublishedYear: 999,
			},
			wantErr: ErrPublishedYearTooEarly,
		},
		{
			name: "published_year in the future",
			input: BookCreate{
				Title:         "Future Book",
				Author:        "Future Author",
				PublishedYear: currentYear + 1,
			},
			wantErrFmt: fmt.Sprintf("Published year cannot be in the future (current year: %d)", currentYear),
		},

		// ── Summary errors ────────────────────────────────────────────────────
		{
			name: "summary too long (2001 chars)",
			input: BookCreate{
				Title:         "Clean Code",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
				Summary:       &longStr2001,
			},
			wantErr: ErrSummaryTooLong,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()

			if tc.wantErr != nil {
				assert.True(t, errors.Is(err, tc.wantErr),
					"expected %v, got %v", tc.wantErr, err)
				return
			}
			if tc.wantErrFmt != "" {
				assert.EqualError(t, err, tc.wantErrFmt)
				return
			}

			assert.NoError(t, err)

			if tc.wantTitle != "" {
				assert.Equal(t, tc.wantTitle, tc.input.Title)
			}
			if tc.wantAuthor != "" {
				assert.Equal(t, tc.wantAuthor, tc.input.Author)
			}
			if tc.wantSummary != nil {
				assert.NotNil(t, tc.input.Summary)
				assert.Equal(t, *tc.wantSummary, *tc.input.Summary)
			} else {
				// Only check nil summary for cases where we expect it explicitly
				// (i.e., the scenario name mentions summary)
				if strings.Contains(tc.name, "summary") {
					assert.Nil(t, tc.input.Summary)
				}
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// BookUpdate.Validate
// ──────────────────────────────────────────────────────────────────────────────

func TestBookUpdate_Validate(t *testing.T) {
	longStr255 := strings.Repeat("a", 256)
	longStr2001 := strings.Repeat("b", 2001)

	tests := []struct {
		name           string
		input          BookUpdate
		wantErr        error
		wantErrFmt     string
		wantTitle      *string  // expected trimmed title (nil means "not checked")
		wantAuthor     *string  // expected trimmed author
		wantSummary    *string  // expected trimmed summary
		expectSummNil  bool     // true when we expect summary to become nil
		wantYear       *int
	}{
		// ── Happy paths ───────────────────────────────────────────────────────
		{
			name:  "no fields provided",
			input: BookUpdate{},
		},
		{
			name: "null title stays null",
			input: BookUpdate{
				Author: strPtr("Robert C. Martin"),
			},
		},
		{
			name: "null author stays null",
			input: BookUpdate{
				Title: strPtr("Clean Code"),
			},
		},
		{
			name: "null published_year stays null",
			input: BookUpdate{
				Title: strPtr("Clean Code"),
			},
		},
		{
			name: "null summary stays null",
			input: BookUpdate{
				Title: strPtr("Clean Code"),
			},
			expectSummNil: false, // nil from the start, not changed
		},
		{
			name: "valid title trimmed",
			input: BookUpdate{
				Title: strPtr("  Clean Code  "),
			},
			wantTitle: strPtr("Clean Code"),
		},
		{
			name: "valid author trimmed",
			input: BookUpdate{
				Author: strPtr("  Robert C. Martin  "),
			},
			wantAuthor: strPtr("Robert C. Martin"),
		},
		{
			name: "valid published_year within range",
			input: BookUpdate{
				PublishedYear: intPtr(2008),
			},
			wantYear: intPtr(2008),
		},
		{
			name: "valid summary trimmed",
			input: BookUpdate{
				Summary: strPtr("  great book  "),
			},
			wantSummary: strPtr("great book"),
		},
		{
			name: "whitespace-only summary normalized to nil",
			input: BookUpdate{
				Summary: strPtr("   "),
			},
			expectSummNil: true,
		},

		// ── Title errors ──────────────────────────────────────────────────────
		{
			name: "whitespace-only title",
			input: BookUpdate{
				Title: strPtr("   "),
			},
			wantErr: ErrTitleEmpty,
		},
		{
			name: "empty title",
			input: BookUpdate{
				Title: strPtr(""),
			},
			wantErr: ErrTitleEmpty,
		},
		{
			name: "title too long",
			input: BookUpdate{
				Title: &longStr255,
			},
			wantErr: ErrTitleTooLong,
		},

		// ── Author errors ─────────────────────────────────────────────────────
		{
			name: "whitespace-only author",
			input: BookUpdate{
				Author: strPtr("   "),
			},
			wantErr: ErrAuthorEmpty,
		},
		{
			name: "empty author",
			input: BookUpdate{
				Author: strPtr(""),
			},
			wantErr: ErrAuthorEmpty,
		},
		{
			name: "author too long",
			input: BookUpdate{
				Author: &longStr255,
			},
			wantErr: ErrAuthorTooLong,
		},

		// ── PublishedYear errors ──────────────────────────────────────────────
		{
			name: "published_year less than 1000",
			input: BookUpdate{
				PublishedYear: intPtr(999),
			},
			wantErr: ErrPublishedYearTooEarly,
		},
		{
			name: "published_year in the future",
			input: BookUpdate{
				PublishedYear: intPtr(currentYear + 1),
			},
			wantErrFmt: fmt.Sprintf("Published year cannot be in the future (current year: %d)", currentYear),
		},

		// ── Summary errors ────────────────────────────────────────────────────
		{
			name: "summary too long",
			input: BookUpdate{
				Summary: &longStr2001,
			},
			wantErr: ErrSummaryTooLong,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()

			if tc.wantErr != nil {
				assert.True(t, errors.Is(err, tc.wantErr),
					"expected %v, got %v", tc.wantErr, err)
				return
			}
			if tc.wantErrFmt != "" {
				assert.EqualError(t, err, tc.wantErrFmt)
				return
			}

			assert.NoError(t, err)

			if tc.wantTitle != nil {
				assert.NotNil(t, tc.input.Title)
				assert.Equal(t, *tc.wantTitle, *tc.input.Title)
			}
			if tc.wantAuthor != nil {
				assert.NotNil(t, tc.input.Author)
				assert.Equal(t, *tc.wantAuthor, *tc.input.Author)
			}
			if tc.wantYear != nil {
				assert.NotNil(t, tc.input.PublishedYear)
				assert.Equal(t, *tc.wantYear, *tc.input.PublishedYear)
			}
			if tc.wantSummary != nil {
				assert.NotNil(t, tc.input.Summary)
				assert.Equal(t, *tc.wantSummary, *tc.input.Summary)
			}
			if tc.expectSummNil {
				assert.Nil(t, tc.input.Summary)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// NewBookResponse
// ──────────────────────────────────────────────────────────────────────────────

func TestNewBookResponse(t *testing.T) {
	tests := []struct {
		name string
		book Book
		want BookResponse
	}{
		{
			name: "full book with summary",
			book: Book{
				ID:            1,
				Title:         "Clean Code",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
				Summary:       strPtr("A book about clean code."),
			},
			want: BookResponse{
				ID:            1,
				Title:         "Clean Code",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
				Summary:       strPtr("A book about clean code."),
			},
		},
		{
			name: "book without summary",
			book: Book{
				ID:            2,
				Title:         "The Pragmatic Programmer",
				Author:        "David Thomas",
				PublishedYear: 1999,
				Summary:       nil,
			},