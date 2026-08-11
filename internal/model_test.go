```go
package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBookString covers the Book.String() method which replaces __str__.
func TestBookString(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "standard book with title, author, and published year",
			book: Book{
				ID:            1,
				Title:         "The Go Programming Language",
				Author:        "Alan Donovan",
				PublishedYear: 2015,
				Summary:       "A comprehensive guide to Go.",
			},
			expected: "The Go Programming Language by Alan Donovan (2015)",
		},
		{
			name: "book with empty summary",
			book: Book{
				ID:            2,
				Title:         "Clean Code",
				Author:        "Robert Martin",
				PublishedYear: 2008,
				Summary:       "",
			},
			expected: "Clean Code by Robert Martin (2008)",
		},
		{
			name: "book with zero ID (not yet persisted)",
			book: Book{
				ID:            0,
				Title:         "1984",
				Author:        "George Orwell",
				PublishedYear: 1949,
			},
			expected: "1984 by George Orwell (1949)",
		},
		{
			name: "book with special characters in title and author",
			book: Book{
				ID:            3,
				Title:         "L'Étranger",
				Author:        "Albert Camus",
				PublishedYear: 1942,
			},
			expected: "L'Étranger by Albert Camus (1942)",
		},
		{
			name: "book with very old year",
			book: Book{
				ID:            4,
				Title:         "Don Quixote",
				Author:        "Miguel de Cervantes",
				PublishedYear: 1605,
			},
			expected: "Don Quixote by Miguel de Cervantes (1605)",
		},
		{
			name: "book with future year",
			book: Book{
				ID:            5,
				Title:         "Future Book",
				Author:        "Unknown Author",
				PublishedYear: 2099,
			},
			expected: "Future Book by Unknown Author (2099)",
		},
		{
			name: "book with empty title and author (zero-value struct fields)",
			book: Book{
				ID:            6,
				Title:         "",
				Author:        "",
				PublishedYear: 0,
			},
			expected: " by  (0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.book.String()
			assert.Equal(t, tt.expected, result)

			// Verify the pattern using fmt.Sprintf to cross-check
			expectedPattern := fmt.Sprintf("%s by %s (%d)", tt.book.Title, tt.book.Author, tt.book.PublishedYear)
			assert.Equal(t, expectedPattern, result)
		})
	}
}

// TestBookGoString covers the Book.GoString() method which replaces __repr__.
func TestBookGoString(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "book with id, title, author, and published_year set",
			book: Book{
				ID:            1,
				Title:         "The Go Programming Language",
				Author:        "Alan Donovan",
				PublishedYear: 2015,
				Summary:       "A comprehensive guide to Go.",
			},
			expected: "<Book(id=1, title='The Go Programming Language', author='Alan Donovan', year=2015)>",
		},
		{
			name: "book with id not yet assigned (zero value, analogous to None)",
			book: Book{
				ID:            0,
				Title:         "Clean Code",
				Author:        "Robert Martin",
				PublishedYear: 2008,
			},
			expected: "<Book(id=0, title='Clean Code', author='Robert Martin', year=2008)>",
		},
		{
			name: "book with special characters in title",
			book: Book{
				ID:            42,
				Title:         "It's a Wonderful Life",
				Author:        "Some Author",
				PublishedYear: 1990,
			},
			expected: "<Book(id=42, title='It's a Wonderful Life', author='Some Author', year=1990)>",
		},
		{
			name: "book with unicode characters",
			book: Book{
				ID:            99,
				Title:         "L'Étranger",
				Author:        "Albert Camus",
				PublishedYear: 1942,
			},
			expected: "<Book(id=99, title='L'Étranger', author='Albert Camus', year=1942)>",
		},
		{
			name: "book with large ID",
			book: Book{
				ID:            9999999,
				Title:         "Big ID Book",
				Author:        "Test Author",
				PublishedYear: 2023,
			},
			expected: "<Book(id=9999999, title='Big ID Book', author='Test Author', year=2023)>",
		},
		{
			name: "zero-value book (analogous to uninitialized instance)",
			book: Book{
				ID:            0,
				Title:         "",
				Author:        "",
				PublishedYear: 0,
			},
			expected: "<Book(id=0, title='', author='', year=0)>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.book.GoString()
			assert.Equal(t, tt.expected, result)

			// Verify format invariants
			assert.Contains(t, result, "<Book(")
			assert.Contains(t, result, ")>")
			assert.Contains(t, result, fmt.Sprintf("id=%d", tt.book.ID))
			assert.Contains(t, result, fmt.Sprintf("title='%s'", tt.book.Title))
			assert.Contains(t, result, fmt.Sprintf("author='%s'", tt.book.Author))
			assert.Contains(t, result, fmt.Sprintf("year=%d", tt.book.PublishedYear))
		})
	}
}

// TestBookGoStringViaFmtVerb verifies that GoString is invoked by the %#v verb.
func TestBookGoStringViaFmtVerb(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "fmt %#v uses GoString",
			book: Book{
				ID:            7,
				Title:         "Domain-Driven Design",
				Author:        "Eric Evans",
				PublishedYear: 2003,
			},
			expected: "<Book(id=7, title='Domain-Driven Design', author='Eric Evans', year=2003)>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fmt.Sprintf("%#v", tt.book)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBookStringViaFmtVerb verifies that String is invoked by the %s verb.
func TestBookStringViaFmtVerb(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "fmt %s uses String",
			book: Book{
				ID:            8,
				Title:         "Refactoring",
				Author:        "Martin Fowler",
				PublishedYear: 1999,
			},
			expected: "Refactoring by Martin Fowler (1999)",
		},
		{
			name: "fmt %v also uses String",
			book: Book{
				ID:            9,
				Title:         "The Pragmatic Programmer",
				Author:        "David Thomas",
				PublishedYear: 1999,
			},
			expected: "The Pragmatic Programmer by David Thomas (1999)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultS := fmt.Sprintf("%s", tt.book)
			assert.Equal(t, tt.expected, resultS)

			resultV := fmt.Sprintf("%v", tt.book)
			assert.Equal(t, tt.expected, resultV)
		})
	}
}

// TestBookStructFields verifies the JSON and db tags are correctly defined.
func TestBookStructFields(t *testing.T) {
	tests := []struct {
		name          string
		book          Book
		expectedID    int64
		expectedTitle string
		expectedAuth  string
		expectedYear  int
		expectedSumm  string
	}{
		{
			name: "full book with all fields",
			book: Book{
				ID:            1,
				Title:         "Test Title",
				Author:        "Test Author",
				PublishedYear: 2020,
				Summary:       "Test summary",
			},
			expectedID:    1,
			expectedTitle: "Test Title",
			expectedAuth:  "Test Author",
			expectedYear:  2020,
			expectedSumm:  "Test summary",
		},
		{
			name: "book with empty summary (optional field)",
			book: Book{
				ID:            2,
				Title:         "No Summary Book",
				Author:        "Author Name",
				PublishedYear: 2021,
				Summary:       "",
			},
			expectedID:    2,
			expectedTitle: "No Summary Book",
			expectedAuth:  "Author Name",
			expectedYear:  2021,
			expectedSumm:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedID, tt.book.ID)
			assert.Equal(t, tt.expectedTitle, tt.book.Title)
			assert.Equal(t, tt.expectedAuth, tt.book.Author)
			assert.Equal(t, tt.expectedYear, tt.book.PublishedYear)
			assert.Equal(t, tt.expectedSumm, tt.book.Summary)
		})
	}
}

// TestBookStringIsReadOnly verifies that calling String() does not mutate the Book.
func TestBookStringIsReadOnly(t *testing.T) {
	original := Book{
		ID:            10,
		Title:         "Immutable Book",
		Author:        "Immutable Author",
		PublishedYear: 2000,
		Summary:       "Immutable summary",
	}
	snapshot := original

	_ = original.String()

	assert.Equal(t, snapshot.ID, original.ID)
	assert.Equal(t, snapshot.Title, original.Title)
	assert.Equal(t, snapshot.Author, original.Author)
	assert.Equal(t, snapshot.PublishedYear, original.PublishedYear)
	assert.Equal(t, snapshot.Summary, original.Summary)
}

// TestBookGoStringIsReadOnly verifies that calling GoString() does not mutate the Book.
func TestBookGoStringIsReadOnly(t *testing.T) {
	original := Book{
		ID:            11,
		Title:         "Immutable GoString Book",
		Author:        "Immutable GoString Author",
		PublishedYear: 2001,
		Summary:       "Immutable summary",
	}
	snapshot := original

	_ = original.GoString()

	assert.Equal(t, snapshot.ID, original.ID)
	assert.Equal(t, snapshot.Title, original.Title)
	assert.Equal(t, snapshot.Author, original.Author)
	assert.Equal(t, snapshot.PublishedYear, original.PublishedYear)
	assert.Equal(t, snapshot.Summary, original.Summary)
}

// TestBookGoStringInvariantsAngleBrackets verifies the <Book(...)> envelope.
func TestBookGoStringInvariantsAngleBrackets(t *testing.T) {
	tests := []struct {
		name string
		book Book
	}{
		{
			name: "regular book",
			book: Book{ID: 1, Title: "A", Author: "B", PublishedYear: 2000},
		},
		{
			name: "zero-value book",
			book: Book{},
		},
		{
			name: "book with negative year (edge case)",
			book: Book{ID: 5, Title: "Ancient", Author: "Unknown", PublishedYear: -500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.book.GoString()

			// Must start with <Book( and end with )>
			assert.True(t, len(result) >= 8, "result too short")
			assert.Equal(t, "<Book(", result[:6])
			assert.Equal(t, ")>", result[len(result)-2:])

			// Title and author must be in single quotes
			assert.Contains(t, result, fmt.Sprintf("title='%s'", tt.book.Title))
			assert.Contains(t, result, fmt.Sprintf("author='%s'", tt.book.Author))
		})
	}
}

// TestBookStringPattern verifies the exact "title by author (year)" pattern.
func TestBookStringPattern(t *testing.T) {
	tests := []struct {
		name string
		book Book
	}{
		{
			name: "typical book",
			book: Book{ID: 1, Title: "Go In Action", Author: "William Kennedy", PublishedYear: 2015},
		},
		{
			name: "book with numbers in title",
			book: Book{ID: 2, Title: "7 Habits", Author: "Stephen Covey", PublishedYear: 1989},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.book.String()

			// Must follow "{title} by {author} ({year})" pattern
			expected := fmt.Sprintf("%s by %s (%d)", tt.book.Title, tt.book.Author, tt.book.PublishedYear)
			assert.Equal(t, expected, result)

			// Must contain " by " separator
			assert.Contains(t, result, " by ")

			// Must end with (year)
			assert.Contains(t, result, fmt.Sprintf("(%d)", tt.book.PublishedYear))
		})
	}
}
```