```go
package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"bookcatalog/internal"
)

// helper: returns err.Error() or "" if nil
func errString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

// ---------------------------------------------------------------------------
// BookCreate
// ---------------------------------------------------------------------------

func TestBookCreate_ValidWithSummary(t *testing.T) {
	summary := "A valid book summary"
	b := internal.BookCreate{
		Title:         "Valid Book",
		Author:        "Valid Author",
		PublishedYear: 2023,
		Summary:       &summary,
	}

	err := b.Validate()
	require.NoError(t, err)
	assert.Equal(t, "Valid Book", b.Title)
	assert.Equal(t, "Valid Author", b.Author)
	assert.Equal(t, 2023, b.PublishedYear)
	require.NotNil(t, b.Summary)
	assert.Equal(t, "A valid book summary", *b.Summary)
}

func TestBookCreate_ValidWithoutSummary(t *testing.T) {
	b := internal.BookCreate{
		Title:         "Book Without Summary",
		Author:        "Author",
		PublishedYear: 2023,
	}

	err := b.Validate()
	require.NoError(t, err)
	assert.Equal(t, "Book Without Summary", b.Title)
	assert.Equal(t, "Author", b.Author)
	assert.Equal(t, 2023, b.PublishedYear)
	assert.Nil(t, b.Summary)
}

func TestBookCreate_WhitespaceStripping(t *testing.T) {
	summary := "  Whitespace summary  "
	b := internal.BookCreate{
		Title:         "  Whitespace Book  ",
		Author:        "  Whitespace Author  ",
		PublishedYear: 2023,
		Summary:       &summary,
	}

	err := b.Validate()
	require.NoError(t, err)
	assert.Equal(t, "Whitespace Book", b.Title)
	assert.Equal(t, "Whitespace Author", b.Author)
	require.NotNil(t, b.Summary)
	assert.Equal(t, "Whitespace summary", *b.Summary)
}

func TestBookCreate_WhitespaceOnlySummaryBecomesNil(t *testing.T) {
	summary := "   "
	b := internal.BookCreate{
		Title:         "Book",
		Author:        "Author",
		PublishedYear: 2023,
		Summary:       &summary,
	}

	err := b.Validate()
	require.NoError(t, err)
	assert.Nil(t, b.Summary)
}

func TestBookCreate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		input   internal.BookCreate
		wantSub string
	}{
		{
			name:    "missing title",
			input:   internal.BookCreate{Author: "Author", PublishedYear: 2023},
			wantSub: "title",
		},
		{
			name:    "missing author",
			input:   internal.BookCreate{Title: "Title", PublishedYear: 2023},
			wantSub: "author",
		},
		{
			name:    "missing published_year",
			input:   internal.BookCreate{Title: "Title", Author: "Author"},
			wantSub: "published",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			err := input.Validate()
			require.Error(t, err, "expected a validation error")
			assert.Contains(t, strings.ToLower(err.Error()), tt.wantSub)
		})
	}
}

func TestBookCreate_EmptyTitleValidation(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := internal.BookCreate{Title: tt.title, Author: "Author", PublishedYear: 2023}
			err := b.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Title cannot be empty")
		})
	}
}

func TestBookCreate_EmptyAuthorValidation(t *testing.T) {
	tests := []struct {
		name   string
		author string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := internal.BookCreate{Title: "Title", Author: tt.author, PublishedYear: 2023}
			err := b.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Author cannot be empty")
		})
	}
}

func TestBookCreate_PublishedYearValidation(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name      string
		year      int
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "year too early (999)",
			year:      999,
			wantErr:   true,
			errSubstr: "Published year must be after year 1000",
		},
		{
			name:      "future year (current+1)",
			year:      currentYear + 1,
			wantErr:   true,
			errSubstr: "cannot be in the future",
		},
		{
			name:    "valid minimum edge (1000)",
			year:    1000,
			wantErr: false,
		},
		{
			name:    "valid current year",
			year:    currentYear,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := internal.BookCreate{Title: "Title", Author: "Author", PublishedYear: tt.year}
			err := b.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.year, b.PublishedYear)
			}
		})
	}
}

func TestBookCreate_TitleLengthValidation(t *testing.T) {
	tests := []struct {
		name      string
		titleLen  int
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "title of 256 characters",
			titleLen:  256,
			wantErr:   true,
			errSubstr: "at most 255 characters",
		},
		{
			name:     "title of exactly 255 characters",
			titleLen: 255,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := internal.BookCreate{
				Title:         strings.Repeat("A", tt.titleLen),
				Author:        "Author",
				PublishedYear: 2023,
			}
			err := b.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBookCreate_AuthorLengthValidation(t *testing.T) {
	tests := []struct {
		name      string
		authorLen int
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "author of 256 characters",
			authorLen: 256,
			wantErr:   true,
			errSubstr: "at most 255 characters",
		},
		{
			name:      "author of exactly 255 characters",
			authorLen: 255,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := internal.BookCreate{
				Title:         "Title",
				Author:        strings.Repeat("B", tt.authorLen),
				PublishedYear: 2023,
			}
			err := b.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBookCreate_SummaryLengthValidation(t *testing.T) {
	tests := []struct {
		name       string
		summaryLen int
		wantErr    bool
		errSubstr  string
	}{
		{
			name:       "summary of 2001 characters",
			summaryLen: 2001,
			wantErr:    true,
			errSubstr:  "at most 2000 characters",
		},
		{
			name:       "summary of exactly 2000 characters",
			summaryLen: 2000,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := strings.Repeat("C", tt.summaryLen)
			b := internal.BookCreate{
				Title:         "Title",
				Author:        "Author",
				PublishedYear: 2023,
				Summary:       &s,
			}
			err := b.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BookUpdate
// ---------------------------------------------------------------------------

func TestBookUpdate_PartialUpdate(t *testing.T) {
	title := "Updated Title"
	year := 2024
	u := internal.BookUpdate{
		Title:         &title,
		PublishedYear: &year,
	}

	err := u.Validate()
	require.NoError(t, err)
	require.NotNil(t, u.Title)
	assert.Equal(t, "Updated Title", *u.Title)
	assert.Nil(t, u.Author)
	require.NotNil(t, u.PublishedYear)
	assert.Equal(t, 2024, *u.PublishedYear)
	assert.Nil(t, u.Summary)
}

func TestBookUpdate_EmptyUpdate(t *testing.T) {
	u := internal.BookUpdate{}

	err := u.Validate()
	require.NoError(t, err)
	assert.Nil(t, u.Title)
	assert.Nil(t, u.Author)
	assert.Nil(t, u.PublishedYear)
	assert.Nil(t, u.Summary)
}

func TestBookUpdate_ValidationRules(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name      string
		setup     func() internal.BookUpdate
		wantErr   bool
		errSubstr string
	}{
		{
			name: "empty title fails",
			setup: func() internal.BookUpdate {
				s := ""
				return internal.BookUpdate{Title: &s}
			},
			wantErr:   true,
			errSubstr: "Title cannot be empty",
		},
		{
			name: "whitespace-only title fails",
			setup: func() internal.BookUpdate {
				s := "   "
				return internal.BookUpdate{Title: &s}
			},
			wantErr:   true,
			errSubstr: "Title cannot be empty",
		},
		{
			name: "empty author fails",
			setup: func() internal.BookUpdate {
				s := ""
				return internal.BookUpdate{Author: &s}
			},
			wantErr:   true,
			errSubstr: "Author cannot be empty",
		},
		{
			name: "year 999 fails",
			setup: func() internal.BookUpdate {
				y := 999
				return internal.BookUpdate{PublishedYear: &y}
			},
			wantErr:   true,
			errSubstr: "Published year must be after year 1000",
		},
		{
			name: "future year fails",
			setup: func() internal.BookUpdate {
				y := currentYear + 1
				return internal.BookUpdate{PublishedYear: &y}
			},
			wantErr:   true,
			errSubstr: "cannot be in the future",
		},
		{
			name: "title over 255 chars fails",
			setup: func() internal.BookUpdate {
				s := strings.Repeat("A", 256)
				return internal.BookUpdate{Title: &s}
			},
			wantErr:   true,
			errSubstr: "at most 255 characters",
		},
		{
			name: "author over 255 chars fails",
			setup: func() internal.BookUpdate {
				s := strings.Repeat("B", 256)
				return internal.BookUpdate{Author: &s}
			},
			wantErr:   true,
			errSubstr: "at most 255 characters",
		},
		{
			name: "summary over 2000 chars fails",
			setup: func() internal.BookUpdate {
				s := strings.Repeat("C", 2001)
				return internal.BookUpdate{Summary: &s}
			},
			wantErr:   true,
			errSubstr: "at most 2000 characters",
		},
		{
			name: "valid title passes",
			setup: func() internal.BookUpdate {
				s := "Valid Title"
				return internal.BookUpdate{Title: &s}
			},
			wantErr: false,
		},
		{
			name: "valid year 1000 passes",
			setup: func() internal.BookUpdate {
				y := 1000
				return internal.BookUpdate{PublishedYear: &y}
			},
			wantErr: false,
		},
		{
			name: "valid current year passes",
			setup: func() internal.BookUpdate {
				y := currentYear
				return internal.BookUpdate{PublishedYear: &y}
			},
			wantErr: false,
		},
		{
			name: "whitespace-only summary becomes nil (no error)",
			setup: func() internal.BookUpdate {
				s := "   "
				return internal.BookUpdate{Summary: &s}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			u := tt.setup()
			err := u.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBookUpdate_WhitespaceStripping(t *testing.T) {
	title := "  Stripped Title  "
	author := "  Stripped Author  "
	summary := "  Stripped Summary  "
	u := internal