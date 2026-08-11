```go
package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBook_String covers the __str__ behavioral specs.
func TestBook_String(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "standard book with all fields set",
			book: Book{
				ID:            1,
				Title:         "Dune",
				Author:        "Herbert",
				PublishedYear: 1965,
				Summary:       "A science fiction masterpiece",
			},
			expected: "Dune by Herbert (1965)",
		},
		{
			name: "zero-value book (unset/None equivalent)",
			book: Book{},
			// Go zero values: Title="", Author="", PublishedYear=0
			expected: " by  (0)",
		},
		{
			name: "book with empty title and author",
			book: Book{
				ID:            0,
				Title:         "",
				Author:        "",
				PublishedYear: 0,
			},
			expected: " by  (0)",
		},
		{
			name: "book with special characters in title and author",
			book: Book{
				ID:            42,
				Title:         "The Lord of the Rings",
				Author:        "J.R.R. Tolkien",
				PublishedYear: 1954,
			},
			expected: "The Lord of the Rings by J.R.R. Tolkien (1954)",
		},
		{
			name: "book with negative published year",
			book: Book{
				Title:         "Ancient Text",
				Author:        "Unknown",
				PublishedYear: -500,
			},
			expected: "Ancient Text by Unknown (-500)",
		},
		{
			name: "book with unicode characters",
			book: Book{
				Title:         "Война и мир",
				Author:        "Толстой",
				PublishedYear: 1869,
			},
			expected: "Война и мир by Толстой (1869)",
		},
		{
			name: "book with large published year",
			book: Book{
				Title:         "Future Book",
				Author:        "Future Author",
				PublishedYear: 2099,
			},
			expected: "Future Book by Future Author (2099)",
		},
		{
			name: "format always follows title by author (year)",
			book: Book{
				Title:         "1984",
				Author:        "George Orwell",
				PublishedYear: 1949,
			},
			expected: "1984 by George Orwell (1949)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.book.String()
			assert.Equal(t, tt.expected, result)

			// Invariant: output always follows the format '{title} by {author} ({published_year})'
			formatted := fmt.Sprintf("%s by %s (%d)", tt.book.Title, tt.book.Author, tt.book.PublishedYear)
			assert.Equal(t, formatted, result)
		})
	}
}

// TestBook_GoString covers the __repr__ behavioral specs.
func TestBook_GoString(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "standard book with id=1, title=Dune, author=Herbert, year=1965",
			book: Book{
				ID:            1,
				Title:         "Dune",
				Author:        "Herbert",
				PublishedYear: 1965,
				Summary:       "A science fiction masterpiece",
			},
			expected: "<Book(id=1, title='Dune', author='Herbert', year=1965)>",
		},
		{
			name: "zero-value book (unset/None equivalent)",
			book: Book{},
			// Go zero values: ID=0, Title="", Author="", PublishedYear=0
			expected: "<Book(id=0, title='', author='', year=0)>",
		},
		{
			name: "book with id=0 and empty fields",
			book: Book{
				ID:            0,
				Title:         "",
				Author:        "",
				PublishedYear: 0,
			},
			expected: "<Book(id=0, title='', author='', year=0)>",
		},
		{
			name: "book with special characters",
			book: Book{
				ID:            10,
				Title:         "The Lord of the Rings",
				Author:        "J.R.R. Tolkien",
				PublishedYear: 1954,
			},
			expected: "<Book(id=10, title='The Lord of the Rings', author='J.R.R. Tolkien', year=1954)>",
		},
		{
			name: "book with large id",
			book: Book{
				ID:            999999,
				Title:         "Big ID Book",
				Author:        "Some Author",
				PublishedYear: 2000,
			},
			expected: "<Book(id=999999, title='Big ID Book', author='Some Author', year=2000)>",
		},
		{
			name: "book with negative year",
			book: Book{
				ID:            5,
				Title:         "Old Book",
				Author:        "Ancient Writer",
				PublishedYear: -100,
			},
			expected: "<Book(id=5, title='Old Book', author='Ancient Writer', year=-100)>",
		},
		{
			name: "book with unicode fields",
			book: Book{
				ID:            7,
				Title:         "Война и мир",
				Author:        "Толстой",
				PublishedYear: 1869,
			},
			expected: "<Book(id=7, title='Война и мир', author='Толстой', year=1869)>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.book.GoString()
			assert.Equal(t, tt.expected, result)

			// Invariant: output always begins with '<Book(' and ends with ')>'
			assert.True(t, len(result) >= len("<Book()>"))
			assert.Equal(t, "<Book(", result[:6])
			assert.Equal(t, ")>", result[len(result)-2:])

			// Invariant: title and author values are wrapped in single quotes
			expectedTitleWrapped := fmt.Sprintf("title='%s'", tt.book.Title)
			expectedAuthorWrapped := fmt.Sprintf("author='%s'", tt.book.Author)
			assert.Contains(t, result, expectedTitleWrapped)
			assert.Contains(t, result, expectedAuthorWrapped)

			// Invariant: id and year are not wrapped in quotes
			expectedID := fmt.Sprintf("id=%d", tt.book.ID)
			expectedYear := fmt.Sprintf("year=%d", tt.book.PublishedYear)
			assert.Contains(t, result, expectedID)
			assert.Contains(t, result, expectedYear)
		})
	}
}

// TestBook_GoString_VerbPercent tests that %#v uses GoString.
func TestBook_GoString_VerbPercent(t *testing.T) {
	book := Book{
		ID:            1,
		Title:         "Dune",
		Author:        "Herbert",
		PublishedYear: 1965,
	}

	// %#v should invoke GoString
	result := fmt.Sprintf("%#v", book)
	assert.Equal(t, "<Book(id=1, title='Dune', author='Herbert', year=1965)>", result)
}

// TestBook_String_VerbS tests that %s / default formatting uses String.
func TestBook_String_VerbS(t *testing.T) {
	book := Book{
		Title:         "Dune",
		Author:        "Herbert",
		PublishedYear: 1965,
	}

	result := fmt.Sprintf("%s", book)
	assert.Equal(t, "Dune by Herbert (1965)", result)
}

// TestBook_Invariants validates structural invariants on the Book type.
func TestBook_Invariants(t *testing.T) {
	t.Run("String produces no side effects", func(t *testing.T) {
		book := Book{
			ID:            1,
			Title:         "Dune",
			Author:        "Herbert",
			PublishedYear: 1965,
			Summary:       "Epic sci-fi",
		}

		// Call String multiple times — result must be identical and book unchanged
		first := book.String()
		second := book.String()
		assert.Equal(t, first, second)
		assert.Equal(t, int64(1), book.ID)
		assert.Equal(t, "Dune", book.Title)
		assert.Equal(t, "Herbert", book.Author)
		assert.Equal(t, 1965, book.PublishedYear)
		assert.Equal(t, "Epic sci-fi", book.Summary)
	})

	t.Run("GoString produces no side effects", func(t *testing.T) {
		book := Book{
			ID:            1,
			Title:         "Dune",
			Author:        "Herbert",
			PublishedYear: 1965,
			Summary:       "Epic sci-fi",
		}

		// Call GoString multiple times — result must be identical and book unchanged
		first := book.GoString()
		second := book.GoString()
		assert.Equal(t, first, second)
		assert.Equal(t, int64(1), book.ID)
		assert.Equal(t, "Dune", book.Title)
		assert.Equal(t, "Herbert", book.Author)
		assert.Equal(t, 1965, book.PublishedYear)
		assert.Equal(t, "Epic sci-fi", book.Summary)
	})

	t.Run("GoString always begins with <Book( and ends with )>", func(t *testing.T) {
		books := []Book{
			{},
			{ID: 1, Title: "A", Author: "B", PublishedYear: 2000},
			{ID: 999, Title: "Long Title Here", Author: "Many Authors", PublishedYear: 1},
		}
		for _, b := range books {
			result := b.GoString()
			assert.True(t, len(result) > 8, "result too short: %s", result)
			assert.Equal(t, "<Book(", result[:6])
			assert.Equal(t, ")>", result[len(result)-2:])
		}
	})

	t.Run("String format always follows title by author (year)", func(t *testing.T) {
		books := []Book{
			{Title: "T1", Author: "A1", PublishedYear: 2001},
			{Title: "", Author: "", PublishedYear: 0},
			{Title: "X", Author: "Y", PublishedYear: -1},
		}
		for _, b := range books {
			result := b.String()
			expected := fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
			assert.Equal(t, expected, result)
		}
	})
}

// TestBook_StructFields validates the Book struct fields and JSON/db tags
// using reflection-free field assignment to confirm field accessibility.
func TestBook_StructFields(t *testing.T) {
	t.Run("all fields are assignable", func(t *testing.T) {
		b := Book{
			ID:            42,
			Title:         "Test Title",
			Author:        "Test Author",
			PublishedYear: 2023,
			Summary:       "Test summary",
		}
		assert.Equal(t, int64(42), b.ID)
		assert.Equal(t, "Test Title", b.Title)
		assert.Equal(t, "Test Author", b.Author)
		assert.Equal(t, 2023, b.PublishedYear)
		assert.Equal(t, "Test summary", b.Summary)
	})

	t.Run("zero value book has expected defaults", func(t *testing.T) {
		b := Book{}
		assert.Equal(t, int64(0), b.ID)
		assert.Equal(t, "", b.Title)
		assert.Equal(t, "", b.Author)
		assert.Equal(t, 0, b.PublishedYear)
		assert.Equal(t, "", b.Summary)
	})

	t.Run("Summary is optional and defaults to empty string", func(t *testing.T) {
		b := Book{
			ID:            1,
			Title:         "No Summary Book",
			Author:        "Author",
			PublishedYear: 2020,
		}
		// Summary should be empty string (representing SQL NULL / absent summary)
		assert.Equal(t, "", b.Summary)
	})
}

// TestBook_GoString_ContainsAllFields validates the invariant that GoString
// always contains id, title, author, and published_year values.
func TestBook_GoString_ContainsAllFields(t *testing.T) {
	tests := []struct {
		name string
		book Book
	}{
		{
			name: "fully populated book",
			book: Book{ID: 1, Title: "Dune", Author: "Herbert", PublishedYear: 1965},
		},
		{
			name: "zero value book",
			book: Book{},
		},
		{
			name: "book with only ID set",
			book: Book{ID: 99},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.book.GoString()

			// Must contain all four fields
			assert.Contains(t, result, fmt.Sprintf("id=%d", tt.book.ID))
			assert.Contains(t, result, fmt.Sprintf("title='%s'", tt.book.Title))
			assert.Contains(t, result, fmt.Sprintf("author='%s'", tt.book.Author))
			assert.Contains(t, result, fmt.Sprintf("year=%d", tt.book.PublishedYear))
		})
	}
}
```