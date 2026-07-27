```go
package tests

import (
	"strings"
	"testing"
	"time"

	"app/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helperStr returns a pointer to the given string, for optional fields.
func helperStr(s string) *string { return &s }

// helperInt returns a pointer to the given int, for optional fields.
func helperInt(i int) *int { return &i }

// ---------------------------------------------------------------------------
// BookCreate tests
// ---------------------------------------------------------------------------

func TestBookCreateValid(t *testing.T) {
	bc := internal.BookCreate{
		Title:         "Valid Book",
		Author:        "Valid Author",
		PublishedYear: 2023,
		Summary:       helperStr("A valid book summary"),
	}

	require.NoError(t, bc.Validate())
	assert.Equal(t, "Valid Book", bc.Title)
	assert.Equal(t, "Valid Author", bc.Author)
	assert.Equal(t, 2023, bc.PublishedYear)
	require.NotNil(t, bc.Summary)
	assert.Equal(t, "A valid book summary", *bc.Summary)
}

func TestBookCreateWithoutSummary(t *testing.T) {
	bc := internal.BookCreate{
		Title:         "Book Without Summary",
		Author:        "Author",
		PublishedYear: 2023,
	}

	require.NoError(t, bc.Validate())
	assert.Equal(t, "Book Without Summary", bc.Title)
	assert.Equal(t, "Author", bc.Author)
	assert.Equal(t, 2023, bc.PublishedYear)
	assert.Nil(t, bc.Summary)
}

func TestBookCreateStripsWhitespace(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		author        string
		summary       *string
		wantTitle     string
		wantAuthor    string
		wantSummary   *string
	}{
		{
			name:        "strips title and author",
			title:       "  Whitespace Book  ",
			author:      "  Whitespace Author  ",
			summary:     helperStr("  Whitespace summary  "),
			wantTitle:   "Whitespace Book",
			wantAuthor:  "Whitespace Author",
			wantSummary: helperStr("Whitespace summary"),
		},
		{
			name:        "strips only title",
			title:       "  Only Title  ",
			author:      "Clean Author",
			wantTitle:   "Only Title",
			wantAuthor:  "Clean Author",
			wantSummary: nil,
		},
		{
			name:        "strips only author",
			title:       "Clean Title",
			author:      "  Only Author  ",
			wantTitle:   "Clean Title",
			wantAuthor:  "Only Author",
			wantSummary: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         tt.title,
				Author:        tt.author,
				PublishedYear: 2023,
				Summary:       tt.summary,
			}

			require.NoError(t, bc.Validate())
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

func TestBookCreateEmptySummaryBecomesNil(t *testing.T) {
	tests := []struct {
		name    string
		summary *string
	}{
		{name: "whitespace only", summary: helperStr("   ")},
		{name: "empty string", summary: helperStr("")},
		{name: "tabs and spaces", summary: helperStr("\t  \t")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := internal.BookCreate{
				Title:         "Book",
				Author:        "Author",
				PublishedYear: 2023,
				Summary:       tt.summary,
			}

			require.NoError(t, bc.Validate())
			assert.Nil(t, bc.Summary, "expected whitespace-only summary to be normalised to nil")
		})
	}
}

func TestBookCreateEmptyTitleValidation(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{name: "empty string", title: ""},
		{name: "whitespace only", title: "   "},
		{name: "single space", title: " "},
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
			assert.Contains(t, err.Error(), "Title cannot be empty")
		})
	}
}

func TestBookCreateEmptyAuthorValidation(t *testing.T) {
	tests := []struct {
		name   string
		author string
	}{
		{name: "empty string", author: ""},
		{name: "whitespace only", author: "   "},
		{name: "single space", author: " "},
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
			assert.Contains(t, err.Error(), "Author cannot be empty")
		})
	}
}

func TestBookCreatePublishedYearValidation(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name          string
		publishedYear int
		wantErr       bool
		wantContains  string
	}{
		{
			name:          "too early - year 999",
			publishedYear: 999,
			wantErr:       true,
			wantContains:  "Published year must be after year 1000",
		},
		{
			name:          "too early - year 0",
			publishedYear: 0,
			wantErr:       true,
			wantContains:  "Published year must be after year 1000",
		},
		{
			name:          "too early - negative year",
			publishedYear: -500,
			wantErr:       true,
			wantContains:  "Published year must be after year 1000",
		},
		{
			name:          "future year - current+1",
			publishedYear: currentYear + 1,
			wantErr:       true,
			wantContains:  "cannot be in the future",
		},
		{
			name:          "future year - current+10",
			publishedYear: currentYear + 10,
			wantErr:       true,
			wantContains:  "cannot be in the future",
		},
		{
			name:          "min valid - year 1000",
			publishedYear: 1000,
			wantErr:       false,
		},
		{
			name:          "current year - valid max boundary",
			publishedYear: currentYear,
			wantErr:       false,
		},
		{
			name:          "mid range valid",
			publishedYear: 1500,
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
				if tt.wantContains != "" {
					assert.Contains(t, err.Error(), tt.wantContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.publishedYear, bc.PublishedYear)
			}
		})
	}
}

func TestBookCreateLengthValidation(t *testing.T) {
	tests := []struct {
		name         string
		book         internal.BookCreate
		wantContains string
	}{
		{
			name:         "title exactly 255 characters - valid",
			book:         internal.BookCreate{Title: strings.Repeat("A", 255), Author: "Author", PublishedYear: 2023},
			wantContains: "",
		},
		{
			name:         "title 256 characters - invalid",
			book:         internal.BookCreate{Title: strings.Repeat("A", 256), Author: "Author", PublishedYear: 2023},
			wantContains: "at most 255 characters",
		},
		{
			name:         "title 300 characters - invalid",
			book:         internal.BookCreate{Title: strings.Repeat("A", 300), Author: "Author", PublishedYear: 2023},
			wantContains: "at most 255 characters",
		},
		{
			name:         "author exactly 255 characters - valid",
			book:         internal.BookCreate{Title: "Title", Author: strings.Repeat("B", 255), PublishedYear: 2023},
			wantContains: "",
		},
		{
			name:         "author 256 characters - invalid",
			book:         internal.BookCreate{Title: "Title", Author: strings.Repeat("B", 256), PublishedYear: 2023},
			wantContains: "at most 255 characters",
		},
		{
			name:         "author 300 characters - invalid",
			book:         internal.BookCreate{Title: "Title", Author: strings.Repeat("B", 300), PublishedYear: 2023},
			wantContains: "at most 255 characters",
		},
		{
			name:         "summary exactly 2000 characters - valid",
			book:         internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: 2023, Summary: helperStr(strings.Repeat("C", 2000))},
			wantContains: "",
		},
		{
			name:         "summary 2001 characters - invalid",
			book:         internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: 2023, Summary: helperStr(strings.Repeat("C", 2001))},
			wantContains: "at most 2000 characters",
		},
		{
			name:         "summary 3000 characters - invalid",
			book:         internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: 2023, Summary: helperStr(strings.Repeat("C", 3000))},
			wantContains: "at most 2000 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.book.Validate()

			if tt.wantContains == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantContains)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BookUpdate tests
// ---------------------------------------------------------------------------

func TestBookUpdateValidPartial(t *testing.T) {
	tests := []struct {
		name         string
		update       internal.BookUpdate
		wantTitle    *string
		wantAuthor   *string
		wantYear     *int
		wantSummary  *string
	}{
		{
			name: "title and year only",
			update: internal.BookUpdate{
				Title:         helperStr("Updated Title"),
				PublishedYear: helperInt(2024),
			},
			wantTitle:  helperStr("Updated Title"),
			wantYear:   helperInt(2024),
		},
		{
			name: "author only",
			update: internal.BookUpdate{
				Author: helperStr("New Author"),
			},
			wantAuthor: helperStr("New Author"),
		},
		{
			name: "summary only",
			update: internal.BookUpdate{
				Summary: helperStr("New summary text"),
			},
			wantSummary: helperStr("New summary text"),
		},
		{
			name: "all fields set",
			update: internal.BookUpdate{
				Title:         helperStr("All Fields"),
				Author:        helperStr("All Author"),
				PublishedYear: helperInt(2020),
				Summary:       helperStr("All summary"),
			},
			wantTitle:   helperStr("All Fields"),
			wantAuthor:  helperStr("All Author"),
			wantYear:    helperInt(2020),
			wantSummary: helperStr("All summary"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.update.Validate()
			require.NoError(t, err)

			if tt.wantTitle == nil {
				assert.Nil(t, tt.update.Title)
			} else {
				require.NotNil(t, tt.update.Title)
				assert.Equal(t, *tt.wantTitle, *tt.update.Title)
			}

			if tt.wantAuthor == nil {
				assert.Nil(t, tt.update.Author)
			} else {
				require.NotNil(t, tt.update.Author)
				assert.Equal(t, *tt.wantAuthor, *tt.update.Author)
			}

			if tt.wantYear == nil {
				assert.Nil(t, tt.update.PublishedYear)
			} else {
				require.NotNil(t, tt.update.PublishedYear)
				assert.Equal(t, *tt.wantYear, *tt.update.PublishedYear)
			}

			if tt.wantSummary == nil {
				assert.Nil(t, tt.update.Summary)
			} else {
				require.NotNil(t, tt.update.Summary)
				assert.Equal(t, *tt.wantSummary, *tt.update.Summary)
			}
		})
	}
}

func TestBookUpdateEmpty(t *testing.T) {
	bu := internal.BookUpdate{}

	err := bu.Validate()
	require.NoError(t, err)

	assert.Nil(t, bu.Title)
	assert.Nil(t, bu.Author)
	assert.Nil(t, bu.PublishedYear)
	assert.Nil(t, bu.Summary)
}

func TestBookUpdateValidationSameAsCreate(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name         string
		update       internal.BookUpdate
		wantContains string
	}{
		{
			name:         "empty title string",
			update:       internal.BookUpdate{Title: helperStr("")},
			wantContains: "Title cannot be empty",
		},
		{
			name:         "whitespace-only title",
			update:       internal.BookUpdate{Title: helperStr("   ")},
			wantContains: "Title cannot be empty",
		},
		{
			name:         "empty author string",