package schemas_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"migrated-app/internal/model"
	"migrated-app/internal/schemas"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func currentYear() int { return time.Now().Year() }

// mustMarshal converts v to a JSON byte slice, failing the test on error.
func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ---------------------------------------------------------------------------
// BookCreate – UnmarshalJSON
// ---------------------------------------------------------------------------

func TestBookCreate_UnmarshalJSON(t *testing.T) {
	cy := currentYear()

	type input struct {
		Title         any `json:"title,omitempty"`
		Author        any `json:"author,omitempty"`
		PublishedYear any `json:"published_year,omitempty"`
		Summary       any `json:"summary,omitempty"`
	}

	// Build a raw map so we have precise control over key presence/absence and
	// null vs. absent distinction.
	type rawMap = map[string]any

	tests := []struct {
		name        string
		payload     rawMap
		wantErr     bool
		errContains string
		check       func(t *testing.T, got schemas.BookCreate)
	}{
		// ------------------------------------------------------------------ happy paths
		{
			name: "valid minimal (no summary)",
			payload: rawMap{
				"title":          "Clean Code",
				"author":         "Robert C. Martin",
				"published_year": 2008,
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				assert.Equal(t, "Clean Code", got.Title)
				assert.Equal(t, "Robert C. Martin", got.Author)
				assert.Equal(t, 2008, got.PublishedYear)
				assert.Nil(t, got.Summary)
			},
		},
		{
			name: "valid with summary",
			payload: rawMap{
				"title":          "The Go Programming Language",
				"author":         "Alan Donovan",
				"published_year": 2015,
				"summary":        "A comprehensive guide to Go.",
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				assert.Equal(t, "The Go Programming Language", got.Title)
				assert.Equal(t, "Alan Donovan", got.Author)
				assert.Equal(t, 2015, got.PublishedYear)
				require.NotNil(t, got.Summary)
				assert.Equal(t, "A comprehensive guide to Go.", *got.Summary)
			},
		},
		{
			name: "whitespace stripped from title and author",
			payload: rawMap{
				"title":          "  Whitespace Book  ",
				"author":         "  Jane Author  ",
				"published_year": 2000,
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				assert.Equal(t, "Whitespace Book", got.Title)
				assert.Equal(t, "Jane Author", got.Author)
			},
		},
		{
			name: "whitespace stripped from summary",
			payload: rawMap{
				"title":          "Some Book",
				"author":         "Some Author",
				"published_year": 2000,
				"summary":        "  A summary.  ",
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				require.NotNil(t, got.Summary)
				assert.Equal(t, "A summary.", *got.Summary)
			},
		},
		{
			name: "published_year equal to 1000 (boundary – accepted)",
			payload: rawMap{
				"title":          "Ancient Book",
				"author":         "Old Author",
				"published_year": 1000,
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				assert.Equal(t, 1000, got.PublishedYear)
			},
		},
		{
			name: "published_year equal to current year (boundary – accepted)",
			payload: rawMap{
				"title":          "New Book",
				"author":         "New Author",
				"published_year": cy,
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				assert.Equal(t, cy, got.PublishedYear)
			},
		},
		{
			name: "summary null → stored as nil",
			payload: rawMap{
				"title":          "Book",
				"author":         "Author",
				"published_year": 2000,
				"summary":        nil,
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				assert.Nil(t, got.Summary)
			},
		},
		{
			name: "summary empty string → normalized to nil",
			payload: rawMap{
				"title":          "Book",
				"author":         "Author",
				"published_year": 2000,
				"summary":        "",
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				assert.Nil(t, got.Summary)
			},
		},
		{
			name: "summary only whitespace → normalized to nil",
			payload: rawMap{
				"title":          "Book",
				"author":         "Author",
				"published_year": 2000,
				"summary":        "   ",
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				assert.Nil(t, got.Summary)
			},
		},
		{
			name: "summary exactly 2000 chars → accepted",
			payload: rawMap{
				"title":          "Book",
				"author":         "Author",
				"published_year": 2000,
				"summary":        strings.Repeat("a", 2000),
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				require.NotNil(t, got.Summary)
				assert.Len(t, *got.Summary, 2000)
			},
		},
		{
			name: "title exactly 255 chars → accepted",
			payload: rawMap{
				"title":          strings.Repeat("t", 255),
				"author":         "Author",
				"published_year": 2000,
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				assert.Len(t, got.Title, 255)
			},
		},
		{
			name: "author exactly 255 chars → accepted",
			payload: rawMap{
				"title":          "Book",
				"author":         strings.Repeat("a", 255),
				"published_year": 2000,
			},
			check: func(t *testing.T, got schemas.BookCreate) {
				assert.Len(t, got.Author, 255)
			},
		},

		// ------------------------------------------------------------------ title errors
		{
			name: "title missing → field required",
			payload: rawMap{
				"author":         "Author",
				"published_year": 2000,
			},
			wantErr:     true,
			errContains: "field required",
		},
		{
			name: "title empty string → cannot be empty",
			payload: rawMap{
				"title":          "",
				"author":         "Author",
				"published_year": 2000,
			},
			wantErr:     true,
			errContains: "Title cannot be empty",
		},
		{
			name: "title only whitespace → cannot be empty",
			payload: rawMap{
				"title":          "   ",
				"author":         "Author",
				"published_year": 2000,
			},
			wantErr:     true,
			errContains: "Title cannot be empty",
		},
		{
			name: "title 256 chars → too long",
			payload: rawMap{
				"title":          strings.Repeat("t", 256),
				"author":         "Author",
				"published_year": 2000,
			},
			wantErr:     true,
			errContains: "ensure this value has at most 255 characters",
		},

		// ------------------------------------------------------------------ author errors
		{
			name: "author missing → field required",
			payload: rawMap{
				"title":          "Book",
				"published_year": 2000,
			},
			wantErr:     true,
			errContains: "field required",
		},
		{
			name: "author empty string → cannot be empty",
			payload: rawMap{
				"title":          "Book",
				"author":         "",
				"published_year": 2000,
			},
			wantErr:     true,
			errContains: "Author cannot be empty",
		},
		{
			name: "author only whitespace → cannot be empty",
			payload: rawMap{
				"title":          "Book",
				"author":         "   ",
				"published_year": 2000,
			},
			wantErr:     true,
			errContains: "Author cannot be empty",
		},
		{
			name: "author 256 chars → too long",
			payload: rawMap{
				"title":          "Book",
				"author":         strings.Repeat("a", 256),
				"published_year": 2000,
			},
			wantErr:     true,
			errContains: "ensure this value has at most 255 characters",
		},

		// ------------------------------------------------------------------ published_year errors
		{
			name: "published_year missing → field required",
			payload: rawMap{
				"title":  "Book",
				"author": "Author",
			},
			wantErr:     true,
			errContains: "field required",
		},
		{
			name: "published_year 999 → before year 1000",
			payload: rawMap{
				"title":          "Book",
				"author":         "Author",
				"published_year": 999,
			},
			wantErr:     true,
			errContains: "Published year must be after year 1000",
		},
		{
			name: "published_year 0 → before year 1000",
			payload: rawMap{
				"title":          "Book",
				"author":         "Author",
				"published_year": 0,
			},
			wantErr:     true,
			errContains: "Published year must be after year 1000",
		},
		{
			name: "published_year future → cannot be in the future",
			payload: rawMap{
				"title":          "Book",
				"author":         "Author",
				"published_year": cy + 1,
			},
			wantErr:     true,
			errContains: "Published year cannot be in the future",
		},
		{
			name: "published_year future error message contains current year",
			payload: rawMap{
				"title":          "Book",
				"author":         "Author",
				"published_year": cy + 1,
			},
			wantErr:     true,
			errContains: "current year:",
		},
		{
			name: "published_year wrong type (string) → type error",
			payload: rawMap{
				"title":          "Book",
				"author":         "Author",
				"published_year": "not-a-number",
			},
			wantErr: true,
		},

		// ------------------------------------------------------------------ summary errors
		{
			name: "summary 2001 chars → too long",
			payload: rawMap{
				"title":          "Book",
				"author":         "Author",
				"published_year": 2000,
				"summary":        strings.Repeat("s", 2001),
			},
			wantErr:     true,
			errContains: "ensure this value has at most 2000 characters",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.payload)
			require.NoError(t, err)

			var got schemas.BookCreate
			err = json.Unmarshal(data, &got)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BookUpdate – UnmarshalJSON
// ---------------------------------------------------------------------------

func TestBookUpdate_UnmarshalJSON(t *testing.T) {
	cy := currentYear()

	type rawMap = map[string]any

	tests := []struct {
		name        string
		payload     rawMap
		wantErr     bool
		errContains string
		check       func(t *testing.T, got schemas.BookUpdate)
	}{
		// ------------------------------------------------------------------ happy paths
		{
			name:    "empty object → all fields nil",
			payload: rawMap{},
			check: func(t *testing.T, got schemas.BookUpdate) {
				assert.Nil(t, got.Title)
				assert.Nil(t, got.Author)
				assert.Nil(t, got.PublishedYear)
				assert.Nil(t, got.Summary)
			},
		},
		{
			name: "all fields valid",
			payload: rawMap{
				"title":          "Updated Title",
				"author":         "Updated Author",
				"published_year": 2021,
				"summary":        "Updated summary.",
			},
			check: func(t *testing.T, got schemas.BookUpdate) {
				require.NotNil(t, got.Title)
				assert.Equal(t, "Updated Title", *got.Title)
				require.NotNil(t, got.Author)
				assert.Equal(t, "Updated Author", *got.Author)
				require.NotNil(t, got.PublishedYear)
				assert.Equal(t, 2021, *got.PublishedYear)
				require.NotNil(t, got.Summary)
				assert.Equal(t, "Updated summary.", *got.Summary)
			},
		},
		{
			name: "title null → title stays nil",
			payload: rawMap{
				"title":          nil,
				"author":         "Author",
				"published_year": 2000,
			},
			check: func(t *testing.T, got schemas.BookUpdate) {
				assert.Nil(t, got.Title)
			},
		},
		{
			name: "author null → author stays nil",
			payload: rawMap{
				"title":          "Book",
				"author":         nil,
				"published_year": 2000,
			},
			check: func(t *testing.T, got schemas.BookUpdate) {
				assert.Nil(t, got.Author)
			},
		},
		{
			name: "published_year null → published_year stays nil",
			payload: rawMap{
				"title":          "Book",
				"author":         "Author",
				"published_year": nil,
			},
			check: func(t *testing.T, got schemas.BookUpdate) {
				assert.Nil(t, got.PublishedYear)
			},
		},
		{
			name: "summary null → summary stays nil",
			payload: rawMap{
				"title":   "Book",
				"author":  "Author",
				"summary": nil,
			},
			check: func(t *testing.T, got schemas.BookUpdate) {
				assert.Nil(t, got.Summary)
			},