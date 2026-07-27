```go
package internal

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Book.String() – mirrors Python's __str__
// ---------------------------------------------------------------------------

func TestBook_String(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "typical book",
			book: Book{
				ID:            1,
				Title:         "X",
				Author:        "Y",
				PublishedYear: 2000,
			},
			expected: "X by Y (2000)",
		},
		{
			name: "book with spaces in title and author",
			book: Book{
				ID:            42,
				Title:         "The Great Gatsby",
				Author:        "F. Scott Fitzgerald",
				PublishedYear: 1925,
			},
			expected: "The Great Gatsby by F. Scott Fitzgerald (1925)",
		},
		{
			name: "book with zero-value fields",
			book: Book{},
			expected: " by  (0)",
		},
		{
			name: "book with summary set",
			book: Book{
				ID:            10,
				Title:         "1984",
				Author:        "George Orwell",
				PublishedYear: 1949,
				Summary:       strPtr("A dystopian novel"),
			},
			expected: "1984 by George Orwell (1949)",
		},
		{
			name: "book with nil summary",
			book: Book{
				ID:            11,
				Title:         "Brave New World",
				Author:        "Aldous Huxley",
				PublishedYear: 1932,
				Summary:       nil,
			},
			expected: "Brave New World by Aldous Huxley (1932)",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := tc.book.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Book.GoString() – mirrors Python's __repr__
// ---------------------------------------------------------------------------

func TestBook_GoString(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "canonical repr example",
			book: Book{
				ID:            1,
				Title:         "X",
				Author:        "Y",
				PublishedYear: 2000,
			},
			expected: "<Book(id=1, title='X', author='Y', year=2000)>",
		},
		{
			name: "zero-value book (simulates unset/before persistence)",
			book: Book{},
			expected: "<Book(id=0, title='', author='', year=0)>",
		},
		{
			name: "book with large id",
			book: Book{
				ID:            9999999,
				Title:         "Dune",
				Author:        "Frank Herbert",
				PublishedYear: 1965,
			},
			expected: "<Book(id=9999999, title='Dune', author='Frank Herbert', year=1965)>",
		},
		{
			name: "output invariant: starts with <Book( and ends with )>",
			book: Book{
				ID:            5,
				Title:         "Clean Code",
				Author:        "Robert C. Martin",
				PublishedYear: 2008,
			},
			expected: "<Book(id=5, title='Clean Code', author='Robert C. Martin', year=2008)>",
		},
		{
			name: "book with summary does not affect GoString",
			book: Book{
				ID:            7,
				Title:         "Go Programming",
				Author:        "Alan Donovan",
				PublishedYear: 2016,
				Summary:       strPtr("A comprehensive Go book"),
			},
			expected: "<Book(id=7, title='Go Programming', author='Alan Donovan', year=2016)>",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := tc.book.GoString()
			assert.Equal(t, tc.expected, result)

			// Invariant checks
			assert.True(t, len(result) > 0, "GoString must not be empty")
			assert.Equal(t, '<', rune(result[0]), "must start with '<'")
			assert.Equal(t, '>', rune(result[len(result)-1]), "must end with '>'")

			// Must start with <Book(
			assert.Contains(t, result, "<Book(", "must contain '<Book('")
			// Must end with )>
			assert.True(t, result[len(result)-2:] == ")>", "must end with ')>'")
		})
	}
}

// ---------------------------------------------------------------------------
// Book GoString invariants: id, title, author, year are all present
// ---------------------------------------------------------------------------

func TestBook_GoString_ContainsRequiredFields(t *testing.T) {
	book := Book{
		ID:            3,
		Title:         "Effective Go",
		Author:        "Google",
		PublishedYear: 2009,
	}

	repr := book.GoString()

	assert.Contains(t, repr, fmt.Sprintf("id=%d", book.ID))
	assert.Contains(t, repr, fmt.Sprintf("title='%s'", book.Title))
	assert.Contains(t, repr, fmt.Sprintf("author='%s'", book.Author))
	assert.Contains(t, repr, fmt.Sprintf("year=%d", book.PublishedYear))
}

// ---------------------------------------------------------------------------
// Book String invariants: {title} by {author} ({published_year})
// ---------------------------------------------------------------------------

func TestBook_String_FormatInvariant(t *testing.T) {
	tests := []struct {
		name  string
		book  Book
		title string
		author string
		year   int
	}{
		{
			name:   "format invariant basic",
			book:   Book{Title: "A", Author: "B", PublishedYear: 1999},
			title:  "A",
			author: "B",
			year:   1999,
		},
		{
			name:   "format invariant with spaces",
			book:   Book{Title: "Gone with the Wind", Author: "Margaret Mitchell", PublishedYear: 1936},
			title:  "Gone with the Wind",
			author: "Margaret Mitchell",
			year:   1936,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := tc.book.String()
			expected := fmt.Sprintf("%s by %s (%d)", tc.title, tc.author, tc.year)
			assert.Equal(t, expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Book struct field: Summary is a pointer (nullable)
// ---------------------------------------------------------------------------

func TestBook_SummaryIsNullable(t *testing.T) {
	t.Run("summary nil maps to nil pointer", func(t *testing.T) {
		b := Book{
			ID:            1,
			Title:         "Test",
			Author:        "Author",
			PublishedYear: 2021,
			Summary:       nil,
		}
		assert.Nil(t, b.Summary)
	})

	t.Run("summary set maps to non-nil pointer", func(t *testing.T) {
		s := "A great book"
		b := Book{
			ID:            2,
			Title:         "Test2",
			Author:        "Author2",
			PublishedYear: 2022,
			Summary:       &s,
		}
		assert.NotNil(t, b.Summary)
		assert.Equal(t, s, *b.Summary)
	})
}

// ---------------------------------------------------------------------------
// Book struct tags: db and json tags are correct
// ---------------------------------------------------------------------------

func TestBook_JSONTags(t *testing.T) {
	s := "A summary"
	b := Book{
		ID:            99,
		Title:         "JSON Test",
		Author:        "Tester",
		PublishedYear: 2023,
		Summary:       &s,
	}

	data, err := json.Marshal(b)
	assert.NoError(t, err)

	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	assert.NoError(t, err)

	assert.Contains(t, m, "id", "json tag 'id' must be present")
	assert.Contains(t, m, "title", "json tag 'title' must be present")
	assert.Contains(t, m, "author", "json tag 'author' must be present")
	assert.Contains(t, m, "published_year", "json tag 'published_year' must be present")
	assert.Contains(t, m, "summary", "json tag 'summary' must be present")

	assert.Equal(t, float64(99), m["id"])
	assert.Equal(t, "JSON Test", m["title"])
	assert.Equal(t, "Tester", m["author"])
	assert.Equal(t, float64(2023), m["published_year"])
	assert.Equal(t, "A summary", m["summary"])
}

func TestBook_JSONTags_NullSummary(t *testing.T) {
	b := Book{
		ID:            1,
		Title:         "No Summary",
		Author:        "Ghost",
		PublishedYear: 2000,
		Summary:       nil,
	}

	data, err := json.Marshal(b)
	assert.NoError(t, err)

	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	assert.NoError(t, err)

	// null in JSON maps to nil in Go
	assert.Contains(t, m, "summary")
	assert.Nil(t, m["summary"])
}

// ---------------------------------------------------------------------------
// Book struct: immutability of String() and GoString() (no mutation)
// ---------------------------------------------------------------------------

func TestBook_String_DoesNotMutateState(t *testing.T) {
	b := Book{
		ID:            1,
		Title:         "Immutable",
		Author:        "Author",
		PublishedYear: 2020,
	}

	before := b
	_ = b.String()
	assert.Equal(t, before, b, "String() must not mutate the Book")
}

func TestBook_GoString_DoesNotMutateState(t *testing.T) {
	b := Book{
		ID:            1,
		Title:         "Immutable",
		Author:        "Author",
		PublishedYear: 2020,
	}

	before := b
	_ = b.GoString()
	assert.Equal(t, before, b, "GoString() must not mutate the Book")
}

// ---------------------------------------------------------------------------
// Book struct: uniqueness logic (simulated via domain logic)
// ---------------------------------------------------------------------------

// bookStore is a minimal in-memory store to simulate unique constraint behaviour
// (the DB enforces this in production; here we validate the domain-level invariant).
type bookStore struct {
	books []Book
}

func (s *bookStore) add(b Book) error {
	for _, existing := range s.books {
		if existing.Title == b.Title && existing.Author == b.Author {
			return fmt.Errorf("unique constraint 'unique_title_author' violation: duplicate (title, author) combination")
		}
	}
	s.books = append(s.books, b)
	return nil
}

func TestBook_UniqueConstraint(t *testing.T) {
	tests := []struct {
		name        string
		books       []Book
		expectError bool
		errContains string
	}{
		{
			name: "same title and same author – second insertion rejected",
			books: []Book{
				{ID: 1, Title: "Dune", Author: "Herbert", PublishedYear: 1965},
				{ID: 2, Title: "Dune", Author: "Herbert", PublishedYear: 1965},
			},
			expectError: true,
			errContains: "unique constraint 'unique_title_author'",
		},
		{
			name: "same title but different authors – both succeed",
			books: []Book{
				{ID: 1, Title: "Dune", Author: "Herbert", PublishedYear: 1965},
				{ID: 2, Title: "Dune", Author: "Someone Else", PublishedYear: 2000},
			},
			expectError: false,
		},
		{
			name: "same author but different titles – both succeed",
			books: []Book{
				{ID: 1, Title: "Dune", Author: "Herbert", PublishedYear: 1965},
				{ID: 2, Title: "Dune Messiah", Author: "Herbert", PublishedYear: 1969},
			},
			expectError: false,
		},
		{
			name: "single book – always succeeds",
			books: []Book{
				{ID: 1, Title: "Only Book", Author: "Sole Author", PublishedYear: 2001},
			},
			expectError: false,
		},
		{
			name: "different title and different author – both succeed",
			books: []Book{
				{ID: 1, Title: "Book A", Author: "Author A", PublishedYear: 2001},
				{ID: 2, Title: "Book B", Author: "Author B", PublishedYear: 2002},
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			store := &bookStore{}
			var lastErr error
			for _, b := range tc.books {
				lastErr = store.add(b)
				if lastErr != nil {
					break
				}
			}
			if tc.expectError {
				assert.Error(t, lastErr)
				assert.Contains(t, lastErr.Error(), tc.errContains)
			} else {
				assert.NoError(t, lastErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Book struct: field types match schema specification
// ---------------------------------------------------------------------------

func TestBook_FieldTypes(t *testing.T) {
	t.Run("ID is int64", func(t *testing.T) {
		var b Book
		b.ID = 9223372036854775807 // max int64
		assert.Equal(t, int64(9223372036854775807), b.ID)
	})

	t.Run("PublishedYear is int", func(t *testing.T) {
		var b Book
		b.PublishedYear = 2024
		assert.Equal(t, 2024, b.PublishedYear)
	})

	t.Run("Title is string", func(t *testing.T) {
		var b Book
		b.Title = "Test Title"
		assert.Equal(t, "Test Title", b.Title)
	})

	t.Run("Author is string", func(t *testing.T) {
		var b Book
		b.Author = "Test Author"
		assert.Equal(t, "Test Author", b.Author)
	})

	t.Run("Summary is *string", func(t *testing.T) {
		var b Book
		s := "Test summary"
		b.Summary = &s
		assert.IsType(t, (*string)(nil), b.Summary)
		assert.Equal(t, "Test summary", *b.Summary)
	})
}

// ---------------------------------------------------------------------------
// Book: JSON round-trip
// ---------------------------------------------------------------------------

func TestBook_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		book Book
	}{
		{
			name: "book with summary",
			book: Book{
				ID