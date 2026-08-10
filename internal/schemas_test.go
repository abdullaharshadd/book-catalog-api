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

var currentYear = time.Now().Year()

// ---------------------------------------------------------------------------
// BookCreate.Validate()
// ---------------------------------------------------------------------------

func TestBookCreate_Validate(t *testing.T) {
	longString := strings.Repeat("a", 256)          // 256 chars > 255
	longSummary := strings.Repeat("b", 2001)         // 2001 chars > 2000
	exactMaxTitle := strings.Repeat("c", 255)        // 255 chars == max
	exactMaxSummary := strings.Repeat("d", 2000)     // 2000 chars == max

	tests := []struct {
		name        string
		input       BookCreate
		wantErr     bool
		errContains string
		// post-validate expected values (only checked when wantErr == false)
		wantTitle   string
		wantAuthor  string
		wantSummary *string // nil means expect nil
	}{
		// ── happy paths ────────────────────────────────────────────────────
		{
			name: "valid all fields set",
			input: BookCreate{
				Title:        "The Pragmatic Programmer",
				Author:       "Andrew Hunt",
				PublishedYear: 1999,
				Summary:      strPtr("A great book."),
			},
			wantTitle:   "The Pragmatic Programmer",
			wantAuthor:  "Andrew Hunt",
			wantSummary: strPtr("A great book."),
		},
		{
			name: "title and author with surrounding whitespace are stripped",
			input: BookCreate{
				Title:        "  Clean Code  ",
				Author:       "  Robert Martin  ",
				PublishedYear: 2008,
				Summary:      nil,
			},
			wantTitle:   "Clean Code",
			wantAuthor:  "Robert Martin",
			wantSummary: nil,
		},
		{
			name: "summary with surrounding whitespace is stripped",
			input: BookCreate{
				Title:        "SICP",
				Author:       "Abelson",
				PublishedYear: 1996,
				Summary:      strPtr("  Great intro.  "),
			},
			wantTitle:   "SICP",
			wantAuthor:  "Abelson",
			wantSummary: strPtr("Great intro."),
		},
		{
			name: "summary omitted (nil) stays nil",
			input: BookCreate{
				Title:        "DDD",
				Author:       "Evans",
				PublishedYear: 2003,
				Summary:      nil,
			},
			wantSummary: nil,
		},
		{
			name: "summary empty string normalized to nil",
			input: BookCreate{
				Title:        "DDD",
				Author:       "Evans",
				PublishedYear: 2003,
				Summary:      strPtr(""),
			},
			wantSummary: nil,
		},
		{
			name: "summary whitespace-only normalized to nil",
			input: BookCreate{
				Title:        "DDD",
				Author:       "Evans",
				PublishedYear: 2003,
				Summary:      strPtr("   \t  "),
			},
			wantSummary: nil,
		},
		{
			name: "published_year equal to 1000 is accepted",
			input: BookCreate{
				Title:        "Ancient Text",
				Author:       "Unknown",
				PublishedYear: 1000,
			},
			wantTitle:  "Ancient Text",
			wantAuthor: "Unknown",
		},
		{
			name: "published_year equal to current year is accepted",
			input: BookCreate{
				Title:        "Modern Book",
				Author:       "Someone",
				PublishedYear: currentYear,
			},
			wantTitle:  "Modern Book",
			wantAuthor: "Someone",
		},
		{
			name: "title exactly at max length (255) is accepted",
			input: BookCreate{
				Title:        exactMaxTitle,
				Author:       "Author",
				PublishedYear: 2000,
			},
			wantTitle:  exactMaxTitle,
			wantAuthor: "Author",
		},
		{
			name: "summary exactly at max length (2000) is accepted",
			input: BookCreate{
				Title:        "T",
				Author:       "A",
				PublishedYear: 2000,
				Summary:      strPtr(exactMaxSummary),
			},
			wantSummary: strPtr(exactMaxSummary),
		},

		// ── title errors ───────────────────────────────────────────────────
		{
			name: "title empty string returns error",
			input: BookCreate{
				Title:        "",
				Author:       "Author",
				PublishedYear: 2000,
			},
			wantErr:     true,
			errContains: "Title cannot be empty",
		},
		{
			name: "title whitespace-only returns error",
			input: BookCreate{
				Title:        "   ",
				Author:       "Author",
				PublishedYear: 2000,
			},
			wantErr:     true,
			errContains: "Title cannot be empty",
		},
		{
			name: "title longer than 255 characters returns error",
			input: BookCreate{
				Title:        longString,
				Author:       "Author",
				PublishedYear: 2000,
			},
			wantErr:     true,
			errContains: "ensure this value has at most 255 characters",
		},

		// ── author errors ──────────────────────────────────────────────────
		{
			name: "author empty string returns error",
			input: BookCreate{
				Title:        "Title",
				Author:       "",
				PublishedYear: 2000,
			},
			wantErr:     true,
			errContains: "Author cannot be empty",
		},
		{
			name: "author whitespace-only returns error",
			input: BookCreate{
				Title:        "Title",
				Author:       "   ",
				PublishedYear: 2000,
			},
			wantErr:     true,
			errContains: "Author cannot be empty",
		},
		{
			name: "author longer than 255 characters returns error",
			input: BookCreate{
				Title:        "Title",
				Author:       longString,
				PublishedYear: 2000,
			},
			wantErr:     true,
			errContains: "ensure this value has at most 255 characters",
		},

		// ── published_year errors ──────────────────────────────────────────
		{
			name: "published_year less than 1000 returns error",
			input: BookCreate{
				Title:        "Title",
				Author:       "Author",
				PublishedYear: 999,
			},
			wantErr:     true,
			errContains: "Published year must be after year 1000",
		},
		{
			name: "published_year zero (missing / default) returns error",
			input: BookCreate{
				Title:        "Title",
				Author:       "Author",
				PublishedYear: 0,
			},
			wantErr:     true,
			errContains: "Published year must be after year 1000",
		},
		{
			name: "published_year greater than current year returns error",
			input: BookCreate{
				Title:        "Title",
				Author:       "Author",
				PublishedYear: currentYear + 1,
			},
			wantErr:     true,
			errContains: fmt.Sprintf("Published year cannot be in the future (current year: %d)", currentYear),
		},

		// ── summary errors ─────────────────────────────────────────────────
		{
			name: "summary longer than 2000 characters returns error",
			input: BookCreate{
				Title:        "Title",
				Author:       "Author",
				PublishedYear: 2000,
				Summary:      strPtr(longSummary),
			},
			wantErr:     true,
			errContains: "ensure this value has at most 2000 characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.input // copy so mutations are scoped
			err := b.Validate()

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}

			require.NoError(t, err)

			if tc.wantTitle != "" {
				assert.Equal(t, tc.wantTitle, b.Title)
			}
			if tc.wantAuthor != "" {
				assert.Equal(t, tc.wantAuthor, b.Author)
			}
			if tc.wantSummary == nil {
				assert.Nil(t, b.Summary)
			} else {
				require.NotNil(t, b.Summary)
				assert.Equal(t, *tc.wantSummary, *b.Summary)
			}
		})
	}
}

// Verify that Validate mutates the struct in-place (stripping whitespace).
func TestBookCreate_Validate_MutatesInPlace(t *testing.T) {
	b := BookCreate{
		Title:        "  My Title  ",
		Author:       "  My Author  ",
		PublishedYear: 2000,
		Summary:      strPtr("  My Summary  "),
	}
	require.NoError(t, b.Validate())
	assert.Equal(t, "My Title", b.Title)
	assert.Equal(t, "My Author", b.Author)
	require.NotNil(t, b.Summary)
	assert.Equal(t, "My Summary", *b.Summary)
}

// ---------------------------------------------------------------------------
// BookUpdate.Validate()
// ---------------------------------------------------------------------------

func TestBookUpdate_Validate(t *testing.T) {
	longString := strings.Repeat("a", 256)
	longSummary := strings.Repeat("b", 2001)

	tests := []struct {
		name          string
		input         BookUpdate
		wantErr       bool
		errContains   string
		wantTitle     *string
		wantAuthor    *string
		wantYear      *int
		wantSummary   *string // nil = expect nil pointer; use strPtr("") for empty non-nil
		checkSummaryNil bool  // when true, assert summary IS nil
	}{
		// ── happy paths ────────────────────────────────────────────────────
		{
			name:  "all fields nil – valid empty update",
			input: BookUpdate{},
		},
		{
			name: "title nil – validation skipped",
			input: BookUpdate{Title: nil},
		},
		{
			name: "author nil – validation skipped",
			input: BookUpdate{Author: nil},
		},
		{
			name: "published_year nil – validation skipped",
			input: BookUpdate{PublishedYear: nil},
		},
		{
			name:            "summary nil – stays nil",
			input:           BookUpdate{Summary: nil},
			checkSummaryNil: true,
		},
		{
			name: "valid title with whitespace is stripped",
			input: BookUpdate{
				Title: strPtr("  Refactoring  "),
			},
			wantTitle: strPtr("Refactoring"),
		},
		{
			name: "valid author with whitespace is stripped",
			input: BookUpdate{
				Author: strPtr("  Fowler  "),
			},
			wantAuthor: strPtr("Fowler"),
		},
		{
			name: "valid published_year at boundary 1000",
			input: BookUpdate{
				PublishedYear: intPtr(1000),
			},
			wantYear: intPtr(1000),
		},
		{
			name: "valid published_year at current year",
			input: BookUpdate{
				PublishedYear: intPtr(currentYear),
			},
			wantYear: intPtr(currentYear),
		},
		{
			name: "summary empty string coerced to nil pointer (empty string becomes empty ptr)",
			input: BookUpdate{
				Summary: strPtr(""),
			},
			// The implementation sets an empty non-nil string when blank
			// (MIGRATION_NOTE in BookUpdate.Validate): pointer is set to &""
			wantSummary: strPtr(""),
		},
		{
			name: "summary whitespace-only coerced to empty string pointer",
			input: BookUpdate{
				Summary: strPtr("   "),
			},
			wantSummary: strPtr(""),
		},
		{
			name: "summary with valid content is stripped",
			input: BookUpdate{
				Summary: strPtr("  Great read.  "),
			},
			wantSummary: strPtr("Great read."),
		},

		// ── title errors ───────────────────────────────────────────────────
		{
			name: "title empty string returns error",
			input: BookUpdate{
				Title: strPtr(""),
			},
			wantErr:     true,
			errContains: "Title cannot be empty",
		},
		{
			name: "title whitespace-only returns error",
			input: BookUpdate{
				Title: strPtr("   "),
			},
			wantErr:     true,
			errContains: "Title cannot be empty",
		},
		{
			name: "title longer than 255 returns error",
			input: BookUpdate{
				Title: strPtr(longString),
			},
			wantErr:     true,
			errContains: "ensure this value has at most 255 characters",
		},

		// ── author errors ──────────────────────────────────────────────────
		{
			name: "author empty string returns error",
			input: BookUpdate{
				Author: strPtr(""),
			},
			wantErr:     true,
			errContains: "Author cannot be empty",
		},
		{
			name: "author whitespace-only returns error",
			input: BookUpdate{
				Author: strPtr("   "),
			},
			wantErr:     true,
			errContains: "Author cannot be empty",
		},
		{
			name: "author longer than 255 returns error",
			input: BookUpdate{
				Author: strPtr(longString),
			},
			wantErr:     true,
			errContains: "ensure this value has at most 255 characters",
		},

		// ── published_year errors ──────────────────────────────────────────
		{
			name: "published_year less than 1000 returns error",
			input: BookUpdate{
				PublishedYear: intPtr(999),
			},
			wantErr:     true,
			errContains: "Published year must be after year 1000",
		},
		{
			name: "published_year greater than current year returns error",
			input: BookUpdate{
				PublishedYear: intPtr(currentYear + 1),
			},
			wantErr:     true,
			errContains: fmt.Sprintf("Published year cannot be in the future (current year: %d)", currentYear),
		},

		// ── summary errors ─────────────────────────────────────────────────
		{
			name: "summary longer than 2000 characters returns error",
			input: BookUpdate{
				Summary: strPtr(longSummary),
			},
			wantErr:     true,
			errContains: "ensure this value has at most 2000 characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.input