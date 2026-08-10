```go
package internal

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Book.String() — mirrors Python __str__
// ---------------------------------------------------------------------------

func TestBook_String(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "standard book with all fields",
			book: Book{
				ID:           1,
				Title:        "Dune",
				Author:       "Herbert",
				PublishedYear: 1965,
			},
			expected: "Dune by Herbert (1965)",
		},
		{
			name: "book with different values",
			book: Book{
				ID:           42,
				Title:        "Foundation",
				Author:       "Asimov",
				PublishedYear: 1951,
			},
			expected: "Foundation by Asimov (1951)",
		},
		{
			name: "book with empty title and author",
			book: Book{
				ID:           0,
				Title:        "",
				Author:       "",
				PublishedYear: 2000,
			},
			expected: " by  (2000)",
		},
		{
			name: "book with zero year",
			book: Book{
				ID:           5,
				Title:        "Unknown",
				Author:       "Anonymous",
				PublishedYear: 0,
			},
			expected: "Unknown by Anonymous (0)",
		},
		{
			name: "book with special characters in title and author",
			book: Book{
				ID:           99,
				Title:        "The Hitchhiker's Guide",
				Author:       "Douglas Adams",
				PublishedYear: 1979,
			},
			expected: "The Hitchhiker's Guide by Douglas Adams (1979)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.book.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Book.GoString() — mirrors Python __repr__
// ---------------------------------------------------------------------------

func TestBook_GoString(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "book with id=1, title=Dune, author=Herbert, year=1965",
			book: Book{
				ID:           1,
				Title:        "Dune",
				Author:       "Herbert",
				PublishedYear: 1965,
			},
			expected: "<Book(id=1, title='Dune', author='Herbert', year=1965)>",
		},
		{
			name: "book that has not been assigned an id yet (zero value)",
			book: Book{
				ID:           0,
				Title:        "Unknown",
				Author:       "Anonymous",
				PublishedYear: 2000,
			},
			expected: "<Book(id=0, title='Unknown', author='Anonymous', year=2000)>",
		},
		{
			name: "book with negative id (hypothetical unset sentinel)",
			book: Book{
				ID:           -1,
				Title:        "Draft",
				Author:       "Author",
				PublishedYear: 2024,
			},
			expected: "<Book(id=-1, title='Draft', author='Author', year=2024)>",
		},
		{
			name: "book with special characters",
			book: Book{
				ID:           7,
				Title:        "It's Complicated",
				Author:       "Some \"Author\"",
				PublishedYear: 2020,
			},
			expected: fmt.Sprintf("<Book(id=7, title='It's Complicated', author='Some \"Author\"', year=2020)>"),
		},
		{
			name: "book with large id",
			book: Book{
				ID:           9999999,
				Title:        "Big ID Book",
				Author:       "Writer",
				PublishedYear: 1900,
			},
			expected: "<Book(id=9999999, title='Big ID Book', author='Writer', year=1900)>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.book.GoString()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// coerceEmptyToNil
// ---------------------------------------------------------------------------

func TestCoerceEmptyToNil(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected sql.NullString
	}{
		{
			name:     "empty string becomes NULL",
			input:    "",
			expected: sql.NullString{String: "", Valid: false},
		},
		{
			name:     "non-empty string becomes valid NullString",
			input:    "A helpful summary",
			expected: sql.NullString{String: "A helpful summary", Valid: true},
		},
		{
			name:     "whitespace-only string is valid (not empty)",
			input:    "   ",
			expected: sql.NullString{String: "   ", Valid: true},
		},
		{
			name:     "single space is valid",
			input:    " ",
			expected: sql.NullString{String: " ", Valid: true},
		},
		{
			name:     "unicode content is valid",
			input:    "日本語のサマリ",
			expected: sql.NullString{String: "日本語のサマリ", Valid: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coerceEmptyToNil(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Book.SummaryOrEmpty
// ---------------------------------------------------------------------------

func TestBook_SummaryOrEmpty(t *testing.T) {
	tests := []struct {
		name     string
		book     Book
		expected string
	}{
		{
			name: "valid summary returns the string value",
			book: Book{
				Summary: sql.NullString{String: "A great book", Valid: true},
			},
			expected: "A great book",
		},
		{
			name: "NULL summary returns empty string",
			book: Book{
				Summary: sql.NullString{String: "", Valid: false},
			},
			expected: "",
		},
		{
			name: "NULL summary with non-empty internal string still returns empty (Valid=false)",
			book: Book{
				Summary: sql.NullString{String: "ignored", Valid: false},
			},
			expected: "",
		},
		{
			name: "summary with whitespace",
			book: Book{
				Summary: sql.NullString{String: "  spaces  ", Valid: true},
			},
			expected: "  spaces  ",
		},
		{
			name: "zero-value book has empty summary",
			book: Book{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.book.SummaryOrEmpty()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Book struct field validation (structural / unit level)
// These tests cover the behavioral specs around required fields, nullable
// summary, and the unique constraint semantics — exercised via an in-memory
// mock repository so that no real database is required.
// ---------------------------------------------------------------------------

// BookRepository is the interface that any Book persistence layer must satisfy.
type BookRepository interface {
	Create(b Book) (Book, error)
}

// mockRepo simulates basic constraint enforcement in memory.
type mockRepo struct {
	rows    []Book
	nextID  int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1}
}

var errConstraint = fmt.Errorf("constraint violation")
var errNotNull = fmt.Errorf("not-null constraint violation")
var errUnique = fmt.Errorf("unique constraint violation: unique_title_author")

func (r *mockRepo) Create(b Book) (Book, error) {
	// Enforce not-null constraints.
	if b.Title == "" {
		return Book{}, errNotNull
	}
	if b.Author == "" {
		return Book{}, errNotNull
	}
	// published_year 0 is treated as "not provided" per spec.
	if b.PublishedYear == 0 {
		return Book{}, errNotNull
	}

	// Enforce unique(title, author).
	for _, existing := range r.rows {
		if existing.Title == b.Title && existing.Author == b.Author {
			return Book{}, errUnique
		}
	}

	b.ID = r.nextID
	r.nextID++
	r.rows = append(r.rows, b)
	return b, nil
}

func TestBook_PersistenceBehaviors(t *testing.T) {
	tests := []struct {
		name          string
		books         []Book          // sequence of books to insert
		wantErr       []bool          // whether each insert should fail
		wantErrType   []error         // expected error for failing inserts
		wantIDs       []int64         // expected IDs for successful inserts (0 = skip check)
		wantNullSummary bool          // whether the last successful row has a null summary
	}{
		{
			name: "valid book is stored with auto-incremented id",
			books: []Book{
				{Title: "Dune", Author: "Herbert", PublishedYear: 1965},
			},
			wantErr:     []bool{false},
			wantErrType: []error{nil},
			wantIDs:     []int64{1},
		},
		{
			name: "book without title fails not-null constraint",
			books: []Book{
				{Title: "", Author: "Herbert", PublishedYear: 1965},
			},
			wantErr:     []bool{true},
			wantErrType: []error{errNotNull},
			wantIDs:     []int64{0},
		},
		{
			name: "book without author fails not-null constraint",
			books: []Book{
				{Title: "Dune", Author: "", PublishedYear: 1965},
			},
			wantErr:     []bool{true},
			wantErrType: []error{errNotNull},
			wantIDs:     []int64{0},
		},
		{
			name: "book without published_year fails not-null constraint",
			books: []Book{
				{Title: "Dune", Author: "Herbert", PublishedYear: 0},
			},
			wantErr:     []bool{true},
			wantErrType: []error{errNotNull},
			wantIDs:     []int64{0},
		},
		{
			name: "book with null summary is stored successfully",
			books: []Book{
				{Title: "Dune", Author: "Herbert", PublishedYear: 1965, Summary: sql.NullString{}},
			},
			wantErr:         []bool{false},
			wantErrType:     []error{nil},
			wantIDs:         []int64{1},
			wantNullSummary: true,
		},
		{
			name: "duplicate title+author fails unique constraint",
			books: []Book{
				{Title: "Dune", Author: "Herbert", PublishedYear: 1965},
				{Title: "Dune", Author: "Herbert", PublishedYear: 1966},
			},
			wantErr:     []bool{false, true},
			wantErrType: []error{nil, errUnique},
			wantIDs:     []int64{1, 0},
		},
		{
			name: "same title but different authors both succeed",
			books: []Book{
				{Title: "Dune", Author: "Herbert", PublishedYear: 1965},
				{Title: "Dune", Author: "NotHerbert", PublishedYear: 1999},
			},
			wantErr:     []bool{false, false},
			wantErrType: []error{nil, nil},
			wantIDs:     []int64{1, 2},
		},
		{
			name: "same author but different titles both succeed",
			books: []Book{
				{Title: "Dune", Author: "Herbert", PublishedYear: 1965},
				{Title: "Dune Messiah", Author: "Herbert", PublishedYear: 1969},
			},
			wantErr:     []bool{false, false},
			wantErrType: []error{nil, nil},
			wantIDs:     []int64{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()

			for i, book := range tt.books {
				got, err := repo.Create(book)
				if tt.wantErr[i] {
					assert.Error(t, err)
					assert.Equal(t, tt.wantErrType[i], err)
				} else {
					assert.NoError(t, err)
					if tt.wantIDs[i] != 0 {
						assert.Equal(t, tt.wantIDs[i], got.ID)
					}
					// Check null summary when requested (last successful insert).
					if tt.wantNullSummary && i == len(tt.books)-1 {
						assert.False(t, got.Summary.Valid, "expected summary to be NULL")
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Additional invariant tests
// ---------------------------------------------------------------------------

func TestBook_StructFields(t *testing.T) {
	b := Book{
		ID:           10,
		Title:        "Foundation",
		Author:       "Asimov",
		PublishedYear: 1951,
		Summary:      sql.NullString{String: "Epic saga", Valid: true},
	}

	assert.Equal(t, int64(10), b.ID)
	assert.Equal(t, "Foundation", b.Title)
	assert.Equal(t, "Asimov", b.Author)
	assert.Equal(t, 1951, b.PublishedYear)
	assert.True(t, b.Summary.Valid)
	assert.Equal(t, "Epic saga", b.Summary.String)
}

func TestBook_ZeroValue(t *testing.T) {
	var b Book
	assert.Equal(t, int64(0), b.ID)
	assert.Equal(t, "", b.Title)
	assert.Equal(t, "", b.Author)
	assert.Equal(t, 0, b.PublishedYear)
	assert.False(t, b.Summary.Valid)
	assert.Equal(t, "", b.SummaryOrEmpty())
}

// TestBook_GoStringFormat ensures the format string invariant: single-quoted
// title and author, unquoted id and year.
func TestBook_GoStringFormat(t *testing.T) {
	b := Book{ID: 1, Title: "Dune", Author: "Herbert", PublishedYear: 1965}
	repr := b.GoString()

	assert.Contains(t, repr, "id=1")
	assert.Contains(t, repr, "title='Dune'")
	assert.Contains(t, repr, "author='Herbert'")
	assert.Contains(t, repr, "year=1965")
	assert.True(t, len(repr) > 0)
	assert.Equal(t, '<', rune(repr[0]))
	assert.Equal(t, '>', rune(repr[len(repr)-1]))
}

// TestBook_StringFormat ensures the human-