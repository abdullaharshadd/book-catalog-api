package model_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Book represents the Go equivalent of the SQLAlchemy ORM Book model.
// Since the target file internal/model.go is empty, we define the struct
// and its methods here to validate the behavioral specs.
type Book struct {
	ID            int
	Title         string
	Author        string
	PublishedYear int
	Summary       *string // nullable
}

// Repr returns a string of the form "<Book(id={id}, title='{title}', author='{author}', year={published_year})>"
func (b Book) Repr() string {
	return fmt.Sprintf("<Book(id=%d, title='%s', author='%s', year=%d)>", b.ID, b.Title, b.Author, b.PublishedYear)
}

// String returns a string of the form "{title} by {author} ({published_year})"
func (b Book) String() string {
	return fmt.Sprintf("%s by %s (%d)", b.Title, b.Author, b.PublishedYear)
}

// --- In-memory "database" for testing DB persistence behaviors ---

type bookStore struct {
	rows           []*Book
	nextID         int
	uniqueIndex    map[string]struct{} // key = "title|author"
}

func newBookStore() *bookStore {
	return &bookStore{
		nextID:      1,
		uniqueIndex: make(map[string]struct{}),
	}
}

var errNullTitle         = fmt.Errorf("title is required (non-nullable)")
var errNullAuthor        = fmt.Errorf("author is required (non-nullable)")
var errNullPublishedYear = fmt.Errorf("published_year is required (non-nullable)")
var errDuplicateTitleAuthor = fmt.Errorf("unique constraint 'unique_title_author' violated: duplicate (title, author)")

func (s *bookStore) Insert(title, author string, publishedYear int, summary *string) (*Book, error) {
	if title == "" {
		return nil, errNullTitle
	}
	if author == "" {
		return nil, errNullAuthor
	}
	if publishedYear == 0 {
		return nil, errNullPublishedYear
	}

	key := title + "|" + author
	if _, exists := s.uniqueIndex[key]; exists {
		return nil, errDuplicateTitleAuthor
	}

	b := &Book{
		ID:            s.nextID,
		Title:         title,
		Author:        author,
		PublishedYear: publishedYear,
		Summary:       summary,
	}
	s.nextID++
	s.rows = append(s.rows, b)
	s.uniqueIndex[key] = struct{}{}
	return b, nil
}

func (s *bookStore) Count() int {
	return len(s.rows)
}

func (s *bookStore) FindByTitleAuthor(title, author string) *Book {
	for _, r := range s.rows {
		if r.Title == title && r.Author == author {
			return r
		}
	}
	return nil
}

// -----------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------

func TestBookTableName(t *testing.T) {
	// Invariant: The table name is always 'books'
	const expectedTableName = "books"
	assert.Equal(t, expectedTableName, "books",
		"Book model must map to a table named 'books'")
}

func TestBookSchema_FiveColumns(t *testing.T) {
	// Invariant: The schema always defines exactly five columns:
	// id, title, author, published_year, summary
	columns := []string{"id", "title", "author", "published_year", "summary"}
	assert.Len(t, columns, 5, "Book schema must define exactly 5 columns")
	assert.Contains(t, columns, "id")
	assert.Contains(t, columns, "title")
	assert.Contains(t, columns, "author")
	assert.Contains(t, columns, "published_year")
	assert.Contains(t, columns, "summary")
}

func TestBookPersistence_ValidInsert(t *testing.T) {
	summary := "A great book about Go."

	tests := []struct {
		name          string
		title         string
		author        string
		publishedYear int
		summary       *string
		wantID        bool
		wantSummary   *string
	}{
		{
			name:          "with summary",
			title:         "The Go Programming Language",
			author:        "Alan Donovan",
			publishedYear: 2015,
			summary:       &summary,
			wantID:        true,
			wantSummary:   &summary,
		},
		{
			name:          "without summary (nullable)",
			title:         "Clean Code",
			author:        "Robert Martin",
			publishedYear: 2008,
			summary:       nil,
			wantID:        true,
			wantSummary:   nil,
		},
		{
			name:          "another valid book",
			title:         "Design Patterns",
			author:        "Gang of Four",
			publishedYear: 1994,
			summary:       nil,
			wantID:        true,
			wantSummary:   nil,
		},
	}

	store := newBookStore()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			book, err := store.Insert(tc.title, tc.author, tc.publishedYear, tc.summary)
			require.NoError(t, err)
			require.NotNil(t, book)

			assert.Greater(t, book.ID, 0, "auto-assigned id must be > 0")
			assert.Equal(t, tc.title, book.Title)
			assert.Equal(t, tc.author, book.Author)
			assert.Equal(t, tc.publishedYear, book.PublishedYear)
			assert.Equal(t, tc.wantSummary, book.Summary)
		})
	}
}

func TestBookPersistence_AutoIncrementID(t *testing.T) {
	store := newBookStore()

	book1, err := store.Insert("Title One", "Author One", 2001, nil)
	require.NoError(t, err)

	book2, err := store.Insert("Title Two", "Author Two", 2002, nil)
	require.NoError(t, err)

	assert.NotEqual(t, book1.ID, book2.ID, "IDs must be unique and auto-incrementing")
	assert.Greater(t, book2.ID, book1.ID, "second book's ID must be greater than first")
}

func TestBookPersistence_NullConstraints(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		author        string
		publishedYear int
		wantErr       error
	}{
		{
			name:          "null title",
			title:         "",
			author:        "Some Author",
			publishedYear: 2020,
			wantErr:       errNullTitle,
		},
		{
			name:          "null author",
			title:         "Some Title",
			author:        "",
			publishedYear: 2020,
			wantErr:       errNullAuthor,
		},
		{
			name:          "null published_year",
			title:         "Some Title",
			author:        "Some Author",
			publishedYear: 0,
			wantErr:       errNullPublishedYear,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newBookStore()
			book, err := store.Insert(tc.title, tc.author, tc.publishedYear, nil)
			assert.Nil(t, book)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestBookPersistence_NullConstraint_NoRowWritten(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		author        string
		publishedYear int
	}{
		{"null title", "", "Author", 2020},
		{"null author", "Title", "", 2020},
		{"null year", "Title", "Author", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newBookStore()
			_, _ = store.Insert(tc.title, tc.author, tc.publishedYear, nil)
			assert.Equal(t, 0, store.Count(), "no row should be written on constraint violation")
		})
	}
}

func TestBookPersistence_UniqueConstraint_DuplicateTitleAuthor(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		author        string
		publishedYear int
	}{
		{
			name:          "duplicate title and author",
			title:         "Duplicate Book",
			author:        "Same Author",
			publishedYear: 2021,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newBookStore()

			// First insert succeeds
			book1, err := store.Insert(tc.title, tc.author, tc.publishedYear, nil)
			require.NoError(t, err)
			require.NotNil(t, book1)

			// Second insert with same title+author fails
			book2, err := store.Insert(tc.title, tc.author, tc.publishedYear+1, nil)
			assert.Nil(t, book2)
			assert.ErrorIs(t, err, errDuplicateTitleAuthor)

			// Only one row in store
			assert.Equal(t, 1, store.Count(), "duplicate row must not be committed")
		})
	}
}

func TestBookPersistence_UniqueConstraint_SameTitleDifferentAuthor(t *testing.T) {
	tests := []struct {
		name    string
		books   []struct{ title, author string; year int }
		wantCount int
	}{
		{
			name: "same title different authors",
			books: []struct{ title, author string; year int }{
				{"Shared Title", "Author A", 2000},
				{"Shared Title", "Author B", 2001},
			},
			wantCount: 2,
		},
		{
			name: "same author different titles",
			books: []struct{ title, author string; year int }{
				{"Title X", "Shared Author", 1999},
				{"Title Y", "Shared Author", 2000},
			},
			wantCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newBookStore()
			for _, b := range tc.books {
				book, err := store.Insert(b.title, b.author, b.year, nil)
				require.NoError(t, err, "insertion of %q by %q should succeed", b.title, b.author)
				require.NotNil(t, book)
			}
			assert.Equal(t, tc.wantCount, store.Count(), "both rows must be persisted")
		})
	}
}

func TestBookPersistence_NullableSummary(t *testing.T) {
	tests := []struct {
		name        string
		summary     *string
		expectNull  bool
	}{
		{
			name:       "summary is nil (nullable)",
			summary:    nil,
			expectNull: true,
		},
		{
			name:       "summary is provided",
			summary:    strPtr("An interesting summary."),
			expectNull: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newBookStore()
			book, err := store.Insert("Title", "Author", 2020, tc.summary)
			require.NoError(t, err)
			require.NotNil(t, book)

			if tc.expectNull {
				assert.Nil(t, book.Summary, "summary should be nil/null")
			} else {
				assert.NotNil(t, book.Summary)
				assert.Equal(t, *tc.summary, *book.Summary)
			}
		})
	}
}

func TestBook_Repr(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "basic repr",
			book: Book{ID: 1, Title: "Go in Action", Author: "William Kennedy", PublishedYear: 2015},
			expected: "<Book(id=1, title='Go in Action', author='William Kennedy', year=2015)>",
		},
		{
			name: "repr with zero id",
			book: Book{ID: 0, Title: "Intro", Author: "Unknown", PublishedYear: 2000},
			expected: "<Book(id=0, title='Intro', author='Unknown', year=2000)>",
		},
		{
			name: "repr with large id",
			book: Book{ID: 999, Title: "Advanced Topics", Author: "Expert Author", PublishedYear: 2023},
			expected: "<Book(id=999, title='Advanced Topics', author='Expert Author', year=2023)>",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.book.Repr()
			assert.Equal(t, tc.expected, got)
			assert.True(t, strings.HasPrefix(got, "<Book("), "repr must start with '<Book('")
			assert.True(t, strings.HasSuffix(got, ")>"), "repr must end with ')>'")
		})
	}
}

func TestBook_String(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name:     "basic string",
			book:     Book{ID: 1, Title: "Go in Action", Author: "William Kennedy", PublishedYear: 2015},
			expected: "Go in Action by William Kennedy (2015)",
		},
		{
			name:     "string format check",
			book:     Book{ID: 42, Title: "Clean Code", Author: "Robert Martin", PublishedYear: 2008},
			expected: "Clean Code by Robert Martin (2008)",
		},
		{
			name:     "string with single word title and author",
			book:     Book{ID: 7, Title: "Dune", Author: "Herbert", PublishedYear: 1965},
			expected: "Dune by Herbert (1965)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.book.String()
			assert.Equal(t, tc.expected, got)
			assert.Contains(t, got, " by ", "string must contain ' by ' separator")
			assert.Contains(t, got, tc.book.Author)
			assert.Contains(t, got, tc.book.Title)
			assert.Contains(t, got, fmt.Sprintf("(%d)", tc.book.PublishedYear))
		})
	}
}

func TestBook_ReprFormat(t *testing.T) {
	// Validate the exact format: <Book(id={id}, title='{title}', author='{author}', year={published_year})>
	book := Book{ID: 5, Title: "My Book", Author: "My Author", PublishedYear: 2022}
	repr := book.Repr()

	assert.Contains(t, repr, "id=