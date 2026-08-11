```go
package internal

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Book.String() — mirrors Python __str__
// ---------------------------------------------------------------------------

func TestBook_String(t *testing.T) {
	tests := []struct {
		name string
		book Book
		want string
	}{
		{
			name: "standard book with title, author, and published year",
			book: Book{
				ID:            1,
				Title:         "Dune",
				Author:        "Frank Herbert",
				PublishedYear: 1965,
			},
			want: "Dune by Frank Herbert (1965)",
		},
		{
			name: "book with different title, author, and year",
			book: Book{
				ID:            42,
				Title:         "The Hitchhiker's Guide to the Galaxy",
				Author:        "Douglas Adams",
				PublishedYear: 1979,
			},
			want: "The Hitchhiker's Guide to the Galaxy by Douglas Adams (1979)",
		},
		{
			name: "zero-value book",
			book: Book{},
			want: " by  (0)",
		},
		{
			name: "book with only title set",
			book: Book{
				Title: "Orphan Title",
			},
			want: "Orphan Title by  (0)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.book.String()

			// behavioral spec invariants
			assert.IsType(t, "", got, "String() must return a string")
			assert.Equal(t, tc.want, got)

			// format invariant: "<title> by <author> (<published_year>)"
			assert.Contains(t, got, " by ", "String() must contain ' by '")
			assert.True(t, strings.HasSuffix(got, fmt.Sprintf("(%d)", tc.book.PublishedYear)),
				"String() must end with (published_year)")
		})
	}
}

// ---------------------------------------------------------------------------
// Book.GoString() — mirrors Python __repr__
// ---------------------------------------------------------------------------

func TestBook_GoString(t *testing.T) {
	tests := []struct {
		name string
		book Book
		want string
	}{
		{
			name: "persisted book with id=1",
			book: Book{
				ID:            1,
				Title:         "Dune",
				Author:        "Frank Herbert",
				PublishedYear: 1965,
			},
			want: `<Book(id=1, title="Dune", author="Frank Herbert", year=1965)>`,
		},
		{
			name: "unpersisted book with id=0 (Go zero value, equivalent to Python None)",
			book: Book{
				ID:            0,
				Title:         "Unknown",
				Author:        "Someone",
				PublishedYear: 2000,
			},
			want: `<Book(id=0, title="Unknown", author="Someone", year=2000)>`,
		},
		{
			name: "book with special characters in title and author",
			book: Book{
				ID:            99,
				Title:         "It's Complicated",
				Author:        "A. Author",
				PublishedYear: 2021,
			},
			want: `<Book(id=99, title="It's Complicated", author="A. Author", year=2021)>`,
		},
		{
			name: "zero-value book",
			book: Book{},
			want: `<Book(id=0, title="", author="", year=0)>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.book.GoString()

			// behavioral spec invariants
			assert.IsType(t, "", got, "GoString() must return a string")
			assert.True(t, strings.HasPrefix(got, "<Book("),
				"GoString() must start with '<Book('")
			assert.True(t, strings.HasSuffix(got, ")>"),
				"GoString() must end with ')>'")

			// must contain id, title, author, year in that order
			idIdx := strings.Index(got, "id=")
			titleIdx := strings.Index(got, "title=")
			authorIdx := strings.Index(got, "author=")
			yearIdx := strings.Index(got, "year=")

			assert.Greater(t, idIdx, -1, "GoString() must contain id field")
			assert.Greater(t, titleIdx, -1, "GoString() must contain title field")
			assert.Greater(t, authorIdx, -1, "GoString() must contain author field")
			assert.Greater(t, yearIdx, -1, "GoString() must contain year field")

			assert.Less(t, idIdx, titleIdx, "id must appear before title")
			assert.Less(t, titleIdx, authorIdx, "title must appear before author")
			assert.Less(t, authorIdx, yearIdx, "author must appear before year")

			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Book.GoString() via %#v verb (fmt.GoStringer interface)
// ---------------------------------------------------------------------------

func TestBook_GoStringer_Interface(t *testing.T) {
	b := Book{
		ID:            7,
		Title:         "Foundation",
		Author:        "Isaac Asimov",
		PublishedYear: 1951,
	}

	got := fmt.Sprintf("%#v", b)
	want := `<Book(id=7, title="Foundation", author="Isaac Asimov", year=1951)>`

	assert.Equal(t, want, got, "%#v must call GoString()")
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

func TestConstants(t *testing.T) {
	t.Run("BooksTable constant equals 'books'", func(t *testing.T) {
		assert.Equal(t, "books", BooksTable)
	})

	t.Run("UniqueTitleAuthorConstraint constant equals 'unique_title_author'", func(t *testing.T) {
		assert.Equal(t, "unique_title_author", UniqueTitleAuthorConstraint)
	})
}

// ---------------------------------------------------------------------------
// Book struct field tags
// ---------------------------------------------------------------------------

func TestBook_StructFields(t *testing.T) {
	b := Book{
		ID:            1,
		Title:         "Test Title",
		Author:        "Test Author",
		PublishedYear: 2024,
		Summary: sql.NullString{
			String: "A great book",
			Valid:  true,
		},
	}

	t.Run("ID field round-trips correctly", func(t *testing.T) {
		assert.Equal(t, int64(1), b.ID)
	})

	t.Run("Title field round-trips correctly", func(t *testing.T) {
		assert.Equal(t, "Test Title", b.Title)
	})

	t.Run("Author field round-trips correctly", func(t *testing.T) {
		assert.Equal(t, "Test Author", b.Author)
	})

	t.Run("PublishedYear field round-trips correctly", func(t *testing.T) {
		assert.Equal(t, 2024, b.PublishedYear)
	})

	t.Run("Summary field handles valid NullString", func(t *testing.T) {
		assert.True(t, b.Summary.Valid)
		assert.Equal(t, "A great book", b.Summary.String)
	})

	t.Run("Summary field handles null NullString", func(t *testing.T) {
		bNull := Book{Summary: sql.NullString{Valid: false}}
		assert.False(t, bNull.Summary.Valid)
		assert.Equal(t, "", bNull.Summary.String)
	})
}

// ---------------------------------------------------------------------------
// Summary nullable field
// ---------------------------------------------------------------------------

func TestBook_SummaryNullable(t *testing.T) {
	tests := []struct {
		name        string
		summary     sql.NullString
		expectValid bool
		expectStr   string
	}{
		{
			name:        "null summary (absent)",
			summary:     sql.NullString{Valid: false},
			expectValid: false,
			expectStr:   "",
		},
		{
			name:        "non-null summary",
			summary:     sql.NullString{String: "A wonderful tale", Valid: true},
			expectValid: true,
			expectStr:   "A wonderful tale",
		},
		{
			name:        "empty string summary (valid but empty)",
			summary:     sql.NullString{String: "", Valid: true},
			expectValid: true,
			expectStr:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := Book{
				ID:            1,
				Title:         "T",
				Author:        "A",
				PublishedYear: 2000,
				Summary:       tc.summary,
			}

			assert.Equal(t, tc.expectValid, b.Summary.Valid)
			assert.Equal(t, tc.expectStr, b.Summary.String)

			// Summary must not break String() or GoString()
			assert.NotPanics(t, func() { _ = b.String() })
			assert.NotPanics(t, func() { _ = b.GoString() })
		})
	}
}

// ---------------------------------------------------------------------------
// String() format invariants — table-driven
// ---------------------------------------------------------------------------

func TestBook_String_FormatInvariants(t *testing.T) {
	tests := []struct {
		name string
		book Book
	}{
		{
			name: "Dune by Frank Herbert 1965",
			book: Book{Title: "Dune", Author: "Frank Herbert", PublishedYear: 1965},
		},
		{
			name: "1984 by George Orwell 1949",
			book: Book{Title: "1984", Author: "George Orwell", PublishedYear: 1949},
		},
		{
			name: "empty fields",
			book: Book{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.book.String()
			expected := fmt.Sprintf("%s by %s (%d)",
				tc.book.Title, tc.book.Author, tc.book.PublishedYear)
			assert.Equal(t, expected, got)
		})
	}
}

// ---------------------------------------------------------------------------
// GoString() format invariants — table-driven
// ---------------------------------------------------------------------------

func TestBook_GoString_FormatInvariants(t *testing.T) {
	tests := []struct {
		name string
		book Book
	}{
		{
			name: "standard book",
			book: Book{ID: 1, Title: "Dune", Author: "Frank Herbert", PublishedYear: 1965},
		},
		{
			name: "zero id (unpersisted)",
			book: Book{ID: 0, Title: "Draft", Author: "Writer", PublishedYear: 2024},
		},
		{
			name: "negative id edge case",
			book: Book{ID: -1, Title: "Edge", Author: "Caser", PublishedYear: 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.book.GoString()
			expected := fmt.Sprintf(`<Book(id=%d, title=%q, author=%q, year=%d)>`,
				tc.book.ID, tc.book.Title, tc.book.Author, tc.book.PublishedYear)

			assert.Equal(t, expected, got)
			assert.True(t, strings.HasPrefix(got, "<Book("))
			assert.True(t, strings.HasSuffix(got, ")>"))
		})
	}
}
```