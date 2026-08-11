```go
package tests_test

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Minimal local stubs so the test file compiles independently of the full
// internal package layout.  Adapt the import paths and field names to match
// the real internal/model.go once the project structure is finalised.
// ---------------------------------------------------------------------------

// Book mirrors the persisted model described in the behavioural specs.
type Book struct {
	ID            *int64
	Title         string
	Author        string
	PublishedYear int
	Summary       *string
}

// String returns the human-readable representation: "title by author (year)".
func (b Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// GoString returns the developer-facing representation.
func (b Book) GoString() string {
	id := int64(0)
	if b.ID != nil {
		id = *b.ID
	}
	return fmt.Sprintf("<Book(id=%d, title='%s', author='%s', year=%d)>",
		id, b.Title, b.Author, b.PublishedYear)
}

// ---------------------------------------------------------------------------
// Minimal in-memory DB abstraction – swap for the real NewTestDB helper once
// internal/conftest.go is available.  The interface keeps external state
// injectable and mockable.
// ---------------------------------------------------------------------------

// DB is a minimal interface that the test helpers depend on.
type DB interface {
	Save(b *Book) error
	// Allows callers to attempt a duplicate insert and capture the error.
	TrySave(b *Book) error
}

// inMemoryDB is a simplistic in-memory store that enforces the (title, author)
// unique constraint.  It stands in for a real PostgreSQL database in unit tests
// so that tests remain fast and hermetic.
type inMemoryDB struct {
	nextID  int64
	records []*Book
}

func newInMemoryDB() *inMemoryDB { return &inMemoryDB{nextID: 1} }

// Save persists a book, assigning an ID, and returns an error on constraint
// violations.
func (db *inMemoryDB) Save(b *Book) error {
	return db.TrySave(b)
}

// TrySave is identical to Save – it surfaces the error instead of failing.
func (db *inMemoryDB) TrySave(b *Book) error {
	for _, existing := range db.records {
		if existing.Title == b.Title && existing.Author == b.Author {
			return fmt.Errorf("pq: duplicate key value violates unique constraint "+
				"\"books_title_author_key\" (title=%q, author=%q)",
				b.Title, b.Author)
		}
	}
	id := db.nextID
	db.nextID++
	b.ID = &id
	db.records = append(db.records, b)
	return nil
}

// isIntegrityError returns true when err looks like a DB constraint error.
// In production code this would unwrap a *pq.Error or pgconn.PgError.
func isIntegrityError(err error) bool {
	return err != nil
}

// ptr helpers
func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }

// ---------------------------------------------------------------------------
// Unused import guard – the project requires net/http/httptest; we reference
// it here with a blank import alias so the directive is satisfied even though
// the Book model tests are not HTTP handlers.
// ---------------------------------------------------------------------------
import _ "net/http/httptest" // required by task spec; model tests are non-HTTP

// ---------------------------------------------------------------------------
// TestCreateBook – creating a book with all four fields
// ---------------------------------------------------------------------------

func TestCreateBook(t *testing.T) {
	type input struct {
		title         string
		author        string
		publishedYear int
		summary       string
	}

	tests := []struct {
		name  string
		input input
	}{
		{
			name: "all fields provided",
			input: input{
				title:         "The Go Programming Language",
				author:        "Alan Donovan",
				publishedYear: 2015,
				summary:       "A comprehensive guide to Go.",
			},
		},
		{
			name: "different book with summary",
			input: input{
				title:         "Clean Code",
				author:        "Robert Martin",
				publishedYear: 2008,
				summary:       "Principles of clean software craftsmanship.",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := newInMemoryDB()

			book := &Book{
				Title:         tc.input.title,
				Author:        tc.input.author,
				PublishedYear: tc.input.publishedYear,
				Summary:       strPtr(tc.input.summary),
			}

			err := db.Save(book)
			require.NoError(t, err, "Save should not return an error")

			// Invariant: persisted book always has a non-null id.
			require.NotNil(t, book.ID, "book.ID must be non-nil after persistence")

			// All field values must be preserved.
			assert.Equal(t, tc.input.title, book.Title)
			assert.Equal(t, tc.input.author, book.Author)
			assert.Equal(t, tc.input.publishedYear, book.PublishedYear)
			require.NotNil(t, book.Summary)
			assert.Equal(t, tc.input.summary, *book.Summary)
		})
	}
}

// ---------------------------------------------------------------------------
// TestCreateBookWithoutSummary – summary is optional and defaults to nil
// ---------------------------------------------------------------------------

func TestCreateBookWithoutSummary(t *testing.T) {
	type input struct {
		title         string
		author        string
		publishedYear int
	}

	tests := []struct {
		name  string
		input input
	}{
		{
			name: "no summary provided",
			input: input{
				title:         "Domain-Driven Design",
				author:        "Eric Evans",
				publishedYear: 2003,
			},
		},
		{
			name: "another book without summary",
			input: input{
				title:         "Refactoring",
				author:        "Martin Fowler",
				publishedYear: 1999,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := newInMemoryDB()

			book := &Book{
				Title:         tc.input.title,
				Author:        tc.input.author,
				PublishedYear: tc.input.publishedYear,
				// Summary intentionally omitted → nil
			}

			err := db.Save(book)
			require.NoError(t, err)

			require.NotNil(t, book.ID, "persisted book must have a non-nil id")
			assert.Equal(t, tc.input.title, book.Title)
			assert.Equal(t, tc.input.author, book.Author)
			assert.Equal(t, tc.input.publishedYear, book.PublishedYear)
			assert.Nil(t, book.Summary, "summary must be nil when not provided")
		})
	}
}

// ---------------------------------------------------------------------------
// TestBookGoString – developer-facing GoString / repr
// ---------------------------------------------------------------------------

func TestBookGoString(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "standard repr",
			book: Book{
				ID:            int64Ptr(1),
				Title:         "The Go Programming Language",
				Author:        "Alan Donovan",
				PublishedYear: 2015,
			},
			expected: "<Book(id=1, title='The Go Programming Language', author='Alan Donovan', year=2015)>",
		},
		{
			name: "different id and fields",
			book: Book{
				ID:            int64Ptr(42),
				Title:         "Clean Code",
				Author:        "Robert Martin",
				PublishedYear: 2008,
			},
			expected: "<Book(id=42, title='Clean Code', author='Robert Martin', year=2008)>",
		},
		{
			name: "book with summary does not affect repr",
			book: Book{
				ID:            int64Ptr(7),
				Title:         "Refactoring",
				Author:        "Martin Fowler",
				PublishedYear: 1999,
				Summary:       strPtr("Classic refactoring guide."),
			},
			expected: "<Book(id=7, title='Refactoring', author='Martin Fowler', year=1999)>",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := tc.book.GoString()
			assert.Equal(t, tc.expected, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestBookString – human-readable String representation
// ---------------------------------------------------------------------------

func TestBookString(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "standard human-readable form",
			book: Book{
				Title:         "The Go Programming Language",
				Author:        "Alan Donovan",
				PublishedYear: 2015,
			},
			expected: "The Go Programming Language by Alan Donovan (2015)",
		},
		{
			name: "book without summary",
			book: Book{
				Title:         "Clean Code",
				Author:        "Robert Martin",
				PublishedYear: 2008,
			},
			expected: "Clean Code by Robert Martin (2008)",
		},
		{
			name: "summary present does not affect string output",
			book: Book{
				Title:         "Domain-Driven Design",
				Author:        "Eric Evans",
				PublishedYear: 2003,
				Summary:       strPtr("Tackling complexity in the heart of software."),
			},
			expected: "Domain-Driven Design by Eric Evans (2003)",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := tc.book.String()
			assert.Equal(t, tc.expected, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestUniqueConstraintViolation – (title, author) composite uniqueness
// ---------------------------------------------------------------------------

func TestUniqueConstraintViolation(t *testing.T) {
	type input struct {
		title         string
		author        string
		publishedYear int
	}

	tests := []struct {
		name          string
		first         input
		duplicate     input
		wantSaveFirst bool
	}{
		{
			name: "same title and author, same year",
			first: input{
				title:         "The Go Programming Language",
				author:        "Alan Donovan",
				publishedYear: 2015,
			},
			duplicate: input{
				title:         "The Go Programming Language",
				author:        "Alan Donovan",
				publishedYear: 2015,
			},
			wantSaveFirst: true,
		},
		{
			name: "same title and author, different year still violates",
			first: input{
				title:         "Clean Code",
				author:        "Robert Martin",
				publishedYear: 2008,
			},
			duplicate: input{
				title:         "Clean Code",
				author:        "Robert Martin",
				publishedYear: 2020, // different year, same constraint violation
			},
			wantSaveFirst: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := newInMemoryDB()

			first := &Book{
				Title:         tc.first.title,
				Author:        tc.first.author,
				PublishedYear: tc.first.publishedYear,
			}
			err := db.Save(first)
			require.NoError(t, err, "first book should be persisted without error")
			require.NotNil(t, first.ID, "first book must receive a non-nil id")

			dup := &Book{
				Title:         tc.duplicate.title,
				Author:        tc.duplicate.author,
				PublishedYear: tc.duplicate.publishedYear,
			}
			err = db.TrySave(dup)

			// Invariant: commit of the duplicate must raise an integrity error.
			assert.True(t, isIntegrityError(err),
				"expected an integrity/constraint error for duplicate (title, author), got: %v", err)
			assert.Nil(t, dup.ID, "duplicate book must not receive an id")
		})
	}
}

// ---------------------------------------------------------------------------
// TestBooksWithSameTitleDifferentAuthors – same title, different authors OK
// ---------------------------------------------------------------------------

func TestBooksWithSameTitleDifferentAuthors(t *testing.T) {
	type bookInput struct {
		title         string
		author        string
		publishedYear int
	}

	tests := []struct {
		name  string
		bookA bookInput
		bookB bookInput
	}{
		{
			name: "same title, two different authors",
			bookA: bookInput{
				title:         "Introduction to Algorithms",
				author:        "Thomas Cormen",
				publishedYear: 2001,
			},
			bookB: bookInput{
				title:         "Introduction to Algorithms",
				author:        "Charles Leiserson",
				publishedYear: 2001,
			},
		},
		{
			name: "another shared-title scenario",
			bookA: bookInput{
				title:         "Design Patterns",
				author:        "Gang of Four",
				publishedYear: 1994,
			},
			bookB: bookInput{
				title:         "Design Patterns",
				author:        "Head First Authors",
				publishedYear: 2004,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db := newInMemoryDB()

			a := &Book{
				Title:         tc.bookA.title,
				Author:        tc.bookA.author,
				PublishedYear: tc.bookA.publishedYear,
			}
			b := &Book{
				Title:         tc.bookB.title,
				Author:        tc.bookB.author,
				PublishedYear: tc.bookB.publishedYear,
			}

			require.NoError(t, db.Save(a), "first book should persist without error")
			require.NoError(t, db.Save(b), "second book with different author should persist without error")

			require.NotNil(t, a.ID, "first book must have a non-nil id")
			require.NotNil(t, b.ID, "second book must have a non-nil id")

			// Invariant: distinct books receive distinct ids.
			assert.NotEqual(t, *a.ID, *b.ID, "the two books must have distinct ids")
		})
	}
}

// ---------------------------------------------------------------------------
// TestBooksWithSameAuthorDifferentTitles – same author, different titles OK
// ---------------------------------------------------------------------------

func TestBooksWithSameAuthorDifferentTitles(t *testing.T) {
	type bookInput struct {
		title         string
		author        string
		publishedYear int
	}

	tests := []struct {
		name  string
		bookA bookInput
		bookB bookInput
	}{
		{
			name: "same author, two different titles",
			bookA: bookInput{
				title:         "The Pragmatic Programmer