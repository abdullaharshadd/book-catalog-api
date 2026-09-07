package schemas_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	schemas "migrated-app/internal/schemas"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func currentYear() int { return time.Now().Year() }

// ---------------------------------------------------------------------------
// BookCreate – UnmarshalJSON (required-field enforcement)
// ---------------------------------------------------------------------------

func TestBookCreate_UnmarshalJSON_RequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		body      string
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "missing title",
			body:      `{"author":"Tolkien","published_year":1954}`,
			wantErr:   true,
			errSubstr: "title",
		},
		{
			name:      "missing author",
			body:      `{"title":"The Hobbit","published_year":1937}`,
			wantErr:   true,
			errSubstr: "author",
		},
		{
			name:      "missing published_year",
			body:      `{"title":"The Hobbit","author":"Tolkien"}`,
			wantErr:   true,
			errSubstr: "published_year",
		},
		{
			name:    "all required fields present",
			body:    `{"title":"The Hobbit","author":"Tolkien","published_year":1937}`,
			wantErr: false,
		},
		{
			name:    "all fields present including summary",
			body:    `{"title":"The Hobbit","author":"Tolkien","published_year":1937,"summary":"A classic"}`,
			wantErr: false,
		},
		{
			name:      "empty JSON object",
			body:      `{}`,
			wantErr:   true,
			errSubstr: "field required",
		},
		{
			name:      "invalid JSON",
			body:      `not-json`,
			wantErr:   true,
			errSubstr: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var bc schemas.BookCreate
			err := json.Unmarshal([]byte(tc.body), &bc)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BookCreate – Validate
// ---------------------------------------------------------------------------

func TestBookCreate_Validate_Title(t *testing.T) {
	t.Parallel()

	year := currentYear()

	tests := []struct {
		name      string
		title     string
		wantErr   bool
		errSubstr string
		wantTitle string
	}{
		{
			name:      "empty title",
			title:     "",
			wantErr:   true,
			errSubstr: "Title cannot be empty",
		},
		{
			name:      "whitespace-only title",
			title:     "   ",
			wantErr:   true,
			errSubstr: "Title cannot be empty",
		},
		{
			name:      "title exceeds 255 chars",
			title:     strings.Repeat("a", 256),
			wantErr:   true,
			errSubstr: "ensure this value has at most 255 characters",
		},
		{
			name:      "title exactly 255 chars",
			title:     strings.Repeat("a", 255),
			wantErr:   false,
			wantTitle: strings.Repeat("a", 255),
		},
		{
			name:      "title with surrounding whitespace stripped",
			title:     "  The Hobbit  ",
			wantErr:   false,
			wantTitle: "The Hobbit",
		},
		{
			name:      "normal title",
			title:     "The Hobbit",
			wantErr:   false,
			wantTitle: "The Hobbit",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bc := schemas.BookCreate{
				Title:         tc.title,
				Author:        "Tolkien",
				PublishedYear: year,
			}
			err := bc.Validate()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantTitle, bc.Title)
			}
		})
	}
}

func TestBookCreate_Validate_Author(t *testing.T) {
	t.Parallel()

	year := currentYear()

	tests := []struct {
		name       string
		author     string
		wantErr    bool
		errSubstr  string
		wantAuthor string
	}{
		{
			name:      "empty author",
			author:    "",
			wantErr:   true,
			errSubstr: "Author cannot be empty",
		},
		{
			name:      "whitespace-only author",
			author:    "   ",
			wantErr:   true,
			errSubstr: "Author cannot be empty",
		},
		{
			name:      "author exceeds 255 chars",
			author:    strings.Repeat("b", 256),
			wantErr:   true,
			errSubstr: "ensure this value has at most 255 characters",
		},
		{
			name:       "author exactly 255 chars",
			author:     strings.Repeat("b", 255),
			wantErr:    false,
			wantAuthor: strings.Repeat("b", 255),
		},
		{
			name:       "author with surrounding whitespace",
			author:     "  Tolkien  ",
			wantErr:    false,
			wantAuthor: "Tolkien",
		},
		{
			name:       "normal author",
			author:     "Tolkien",
			wantErr:    false,
			wantAuthor: "Tolkien",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bc := schemas.BookCreate{
				Title:         "The Hobbit",
				Author:        tc.author,
				PublishedYear: year,
			}
			err := bc.Validate()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantAuthor, bc.Author)
			}
		})
	}
}

func TestBookCreate_Validate_PublishedYear(t *testing.T) {
	t.Parallel()

	year := currentYear()

	tests := []struct {
		name      string
		year      int
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "year below 1000",
			year:      999,
			wantErr:   true,
			errSubstr: "Published year must be after year 1000",
		},
		{
			name:      "year zero",
			year:      0,
			wantErr:   true,
			errSubstr: "Published year must be after year 1000",
		},
		{
			name:      "year exactly 1000",
			year:      1000,
			wantErr:   false,
		},
		{
			name:      "future year",
			year:      year + 1,
			wantErr:   true,
			errSubstr: "Published year cannot be in the future",
		},
		{
			name:      "current year",
			year:      year,
			wantErr:   false,
		},
		{
			name:      "historical year",
			year:      1954,
			wantErr:   false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bc := schemas.BookCreate{
				Title:         "Test Book",
				Author:        "Author",
				PublishedYear: tc.year,
			}
			err := bc.Validate()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBookCreate_Validate_PublishedYear_FutureMessageContainsCurrentYear(t *testing.T) {
	t.Parallel()

	year := currentYear()
	bc := schemas.BookCreate{
		Title:         "Future Book",
		Author:        "Author",
		PublishedYear: year + 1,
	}
	err := bc.Validate()
	require.Error(t, err)
	// Message must contain the actual current year value.
	assert.Contains(t, err.Error(), fmt.Sprintf("%d", year))
}

func TestBookCreate_Validate_Summary(t *testing.T) {
	t.Parallel()

	year := currentYear()

	tests := []struct {
		name        string
		summary     *string
		wantErr     bool
		errSubstr   string
		wantNil     bool
		wantSummary string
	}{
		{
			name:    "nil summary",
			summary: nil,
			wantNil: true,
		},
		{
			name:    "empty summary normalized to nil",
			summary: strPtr(""),
			wantNil: true,
		},
		{
			name:    "whitespace-only summary normalized to nil",
			summary: strPtr("   "),
			wantNil: true,
		},
		{
			name:        "valid summary stripped",
			summary:     strPtr("  A classic tale  "),
			wantSummary: "A classic tale",
		},
		{
			name:      "summary exceeds 2000 chars after strip",
			summary:   strPtr(strings.Repeat("x", 2001)),
			wantErr:   true,
			errSubstr: "ensure this value has at most 2000 characters",
		},
		{
			name:        "summary exactly 2000 chars",
			summary:     strPtr(strings.Repeat("x", 2000)),
			wantSummary: strings.Repeat("x", 2000),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			bc := schemas.BookCreate{
				Title:         "Test Book",
				Author:        "Author",
				PublishedYear: year,
				Summary:       tc.summary,
			}
			err := bc.Validate()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
				return
			}
			require.NoError(t, err)
			if tc.wantNil {
				assert.Nil(t, bc.Summary)
			} else {
				require.NotNil(t, bc.Summary)
				assert.Equal(t, tc.wantSummary, *bc.Summary)
			}
		})
	}
}

func TestBookCreate_Validate_ValidComplete(t *testing.T) {
	t.Parallel()

	year := currentYear()
	summary := "  A great adventure  "
	bc := schemas.BookCreate{
		Title:         "  The Hobbit  ",
		Author:        "  J.R.R. Tolkien  ",
		PublishedYear: year,
		Summary:       &summary,
	}
	err := bc.Validate()
	require.NoError(t, err)
	assert.Equal(t, "The Hobbit", bc.Title)
	assert.Equal(t, "J.R.R. Tolkien", bc.Author)
	assert.Equal(t, year, bc.PublishedYear)
	require.NotNil(t, bc.Summary)
	assert.Equal(t, "A great adventure", *bc.Summary)
}

// ---------------------------------------------------------------------------
// BookCreate invariants: 255-char limit evaluated on pre-strip value
// ---------------------------------------------------------------------------

func TestBookCreate_Validate_TitleLimit_PreStrip(t *testing.T) {
	t.Parallel()

	// A title that is 256 chars total but would be ≤255 after stripping
	// the one leading space. Per spec the limit is on the pre-strip value.
	title := " " + strings.Repeat("a", 255) // 256 chars total
	bc := schemas.BookCreate{
		Title:         title,
		Author:        "Author",
		PublishedYear: currentYear(),
	}
	err := bc.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ensure this value has at most 255 characters")
}

func TestBookCreate_Validate_AuthorLimit_PreStrip(t *testing.T) {
	t.Parallel()

	author := " " + strings.Repeat("b", 255) // 256 chars total
	bc := schemas.BookCreate{
		Title:         "Test",
		Author:        author,
		PublishedYear: currentYear(),
	}
	err := bc.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ensure this value has at most 255 characters")
}

// ---------------------------------------------------------------------------
// BookUpdate – Validate (all fields optional)
// ---------------------------------------------------------------------------

func TestBookUpdate_Validate_AllNil(t *testing.T) {
	t.Parallel()

	bu := schemas.BookUpdate{}
	err := bu.Validate()
	require.NoError(t, err)
	assert.Nil(t, bu.Title)
	assert.Nil(t, bu.Author)
	assert.Nil(t, bu.PublishedYear)
	assert.Nil(t, bu.Summary)
}

func TestBookUpdate_Validate_Title(t *testing.T) {
	t.Parallel()

	year := currentYear()

	tests := []struct {
		name      string
		title     *string
		wantErr   bool
		errSubstr string
		wantTitle *string
	}{
		{
			name:      "nil title passes",
			title:     nil,
			wantErr:   false,
			wantTitle: nil,
		},
		{
			name:      "empty title errors",
			title:     strPtr(""),
			wantErr:   true,
			errSubstr: "Title cannot be empty",
		},
		{
			name:      "whitespace-only title errors",
			title:     strPtr("   "),
			wantErr:   true,
			errSubstr: "Title cannot be empty",
		},
		{
			name:      "title > 255 chars errors",
			title:     strPtr(strings.Repeat("a", 256)),
			wantErr:   true,
			errSubstr: "ensure this value has at most 255 characters",
		},
		{
			name:      "valid title stripped",
			title:     strPtr("  Go Programming  "),
			wantErr:   false,
			wantTitle: strPtr("Go Programming"),
		},
	}

	for _, tc := range tests {
		tc := tc